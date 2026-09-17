package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"time"

	"github.com/uptime-app/backend/internal/service"
)

const refreshCookieName = "refresh_token"

const (
	maxAvatarSize   = 5 << 20
	maxAvatarPixels = 16_000_000
)

type AuthenticationService interface {
	Register(context.Context, service.UserInput) (service.AuthenticationResult, error)
	Login(context.Context, service.UserInput) (service.AuthenticationResult, error)
	Refresh(context.Context, string) (service.AuthenticationResult, error)
	Logout(context.Context, string) error
	Profile(context.Context, string) (service.PublicUser, error)
	UpdateProfile(context.Context, string, string) (service.PublicUser, error)
	UpdateAvatar(context.Context, string, service.AvatarInput) (service.PublicUser, error)
}

type CookieConfig struct {
	Secure          bool
	RefreshTokenTTL time.Duration
}
type AuthHandler struct {
	auth   AuthenticationService
	cookie CookieConfig
}

func NewAuthHandler(auth AuthenticationService, cookie CookieConfig) *AuthHandler {
	return &AuthHandler{auth: auth, cookie: cookie}
}
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.HandleFunc("POST /auth/refresh", h.refresh)
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("GET /users/me", h.profile)
	mux.HandleFunc("PATCH /users/me", h.updateProfile)
	mux.HandleFunc("PUT /users/me/avatar", h.updateAvatar)
}

type CredentialsRequest struct {
	Name     string `json:"name" example:"Alex"`
	Email    string `json:"email" example:"alex@example.com"`
	Password string `json:"password" example:"correct-horse-battery-staple"`
}
type UpdateProfileRequest struct {
	Name string `json:"name" example:"Alexandra"`
}
type UserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}
type AuthResponse struct {
	AccessToken string       `json:"accessToken"`
	User        UserResponse `json:"user"`
}
type ErrorResponse struct {
	Error string `json:"error" example:"invalid input"`
}

// register creates an account and starts an authenticated session.
// @Summary Register a user
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body CredentialsRequest true "Registration credentials"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	result, err := h.auth.Register(r.Context(), input)
	if !h.writeAuthError(w, err) {
		return
	}
	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusCreated, responseFrom(result))
}

// login authenticates an existing user and starts an authenticated session.
// @Summary Log in
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body CredentialsRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	result, err := h.auth.Login(r.Context(), input)
	if !h.writeAuthError(w, err) {
		return
	}
	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, responseFrom(result))
}

// refresh rotates the refresh cookie and returns a new access token.
// @Summary Refresh an access token
// @Tags authentication
// @Produce json
// @Security RefreshCookie
// @Success 200 {object} AuthResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie(refreshCookieName)
	if err != nil || token.Value == "" {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	result, err := h.auth.Refresh(r.Context(), token.Value)
	if !h.writeAuthError(w, err) {
		return
	}
	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, responseFrom(result))
}

// logout revokes the current refresh session and clears its cookie.
// @Summary Log out
// @Tags authentication
// @Security RefreshCookie
// @Success 204
// @Failure 500 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(refreshCookieName)
	if cookie != nil {
		if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (service.UserInput, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	var request CredentialsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return service.UserInput{}, false
	}
	return service.UserInput{Name: request.Name, Email: request.Email, Password: request.Password}, true
}

// profile returns the authenticated user's public profile.
// @Summary Get the current user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/me [get]
func (h *AuthHandler) profile(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.Profile(r.Context(), accessToken(r))
	if !h.writeProfileError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, userFrom(user))
}

// updateProfile changes only the authenticated user's display name.
// @Summary Update the current user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileRequest true "Profile changes"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/me [patch]
func (h *AuthHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	var request UpdateProfileRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.auth.UpdateProfile(r.Context(), accessToken(r), request.Name)
	if !h.writeProfileError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, userFrom(user))
}

// updateAvatar validates bytes at the HTTP boundary to reject oversized, malformed, and mislabeled files
// before they reach storage.
// @Summary Replace the current user's avatar
// @Tags users
// @Accept mpfd
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "JPEG or PNG image up to 5 MB and 16 megapixels"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/me/avatar [put]
func (h *AuthHandler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+(1<<20))
	defer r.Body.Close()
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "avatar must not exceed 5 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
	if err != nil || len(data) == 0 {
		writeError(w, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	if len(data) > maxAvatarSize {
		writeError(w, http.StatusRequestEntityTooLarge, "avatar must not exceed 5 MB")
		return
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width*config.Height > maxAvatarPixels {
		writeError(w, http.StatusBadRequest, "invalid avatar image")
		return
	}
	extension := ""
	switch format {
	case "jpeg":
		extension = ".jpg"
	case "png":
		extension = ".png"
	default:
		writeError(w, http.StatusBadRequest, "avatar must be a JPEG or PNG image")
		return
	}
	user, err := h.auth.UpdateAvatar(r.Context(), accessToken(r), service.AvatarInput{Data: data, Extension: extension})
	if !h.writeProfileError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, userFrom(user))
}
func (h *AuthHandler) writeAuthError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrEmailAlreadyUsed):
		writeError(w, http.StatusConflict, "email already used")
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrInvalidRefresh):
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return false
}
func (h *AuthHandler) writeProfileError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return false
}
func responseFrom(result service.AuthenticationResult) AuthResponse {
	return AuthResponse{AccessToken: result.AccessToken, User: userFrom(result.User)}
}

func userFrom(user service.PublicUser) UserResponse {
	return UserResponse{ID: user.ID.String(), Name: user.Name, Email: user.Email, AvatarURL: user.AvatarURL}
}
func accessToken(r *http.Request) string {
	const prefix = "Bearer "
	value := r.Header.Get("Authorization")
	if len(value) <= len(prefix) || value[:len(prefix)] != prefix {
		return ""
	}
	return value[len(prefix):]
}
func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: value, Path: "/auth", MaxAge: int(h.cookie.RefreshTokenTTL.Seconds()), HttpOnly: true, Secure: h.cookie.Secure, SameSite: http.SameSiteStrictMode})
}
func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/auth", MaxAge: -1, HttpOnly: true, Secure: h.cookie.Secure, SameSite: http.SameSiteStrictMode})
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
