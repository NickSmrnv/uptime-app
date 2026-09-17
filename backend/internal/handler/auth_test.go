package handler

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/service"
)

type fakeAuth struct {
	result    service.AuthenticationResult
	err       error
	refresh   string
	loggedOut string
	profile   service.PublicUser
	access    string
}

func (f *fakeAuth) Register(_ context.Context, _ service.UserInput) (service.AuthenticationResult, error) {
	return f.result, f.err
}
func (f *fakeAuth) Login(_ context.Context, _ service.UserInput) (service.AuthenticationResult, error) {
	return f.result, f.err
}
func (f *fakeAuth) Refresh(_ context.Context, token string) (service.AuthenticationResult, error) {
	f.refresh = token
	return f.result, f.err
}
func (f *fakeAuth) Logout(_ context.Context, token string) error { f.loggedOut = token; return f.err }
func (f *fakeAuth) Profile(_ context.Context, token string) (service.PublicUser, error) {
	f.access = token
	return f.profile, f.err
}
func (f *fakeAuth) UpdateProfile(_ context.Context, token, name string) (service.PublicUser, error) {
	f.access = token
	f.profile.Name = name
	return f.profile, f.err
}
func (f *fakeAuth) UpdateAvatar(_ context.Context, token string, avatar service.AvatarInput) (service.PublicUser, error) {
	f.access = token
	f.profile.AvatarURL = "/uploads/avatars/avatar" + avatar.Extension
	return f.profile, f.err
}

func TestRegisterSetsSecureRefreshCookie(t *testing.T) {
	auth := &fakeAuth{result: service.AuthenticationResult{AccessToken: "access", RefreshToken: "refresh", User: service.PublicUser{ID: uuid.New(), Email: "person@example.com"}}}
	mux := http.NewServeMux()
	NewAuthHandler(auth, CookieConfig{Secure: true, RefreshTokenTTL: 30 * 24 * time.Hour}).RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"person@example.com","password":"password"}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	cookie := recorder.Result().Cookies()[0]
	if cookie.Name != refreshCookieName || cookie.Value != "refresh" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/auth" {
		t.Fatalf("unexpected cookie: %#v", cookie)
	}
	if strings.Contains(recorder.Body.String(), "refresh") {
		t.Fatal("refresh token leaked into response body")
	}
}
func TestRefreshAndLogout(t *testing.T) {
	auth := &fakeAuth{result: service.AuthenticationResult{AccessToken: "new-access", RefreshToken: "new-refresh", User: service.PublicUser{ID: uuid.New(), Email: "person@example.com"}}}
	mux := http.NewServeMux()
	NewAuthHandler(auth, CookieConfig{RefreshTokenTTL: 30 * 24 * time.Hour}).RegisterRoutes(mux)
	refresh := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	refresh.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "old-refresh"})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, refresh)
	if recorder.Code != http.StatusOK || auth.refresh != "old-refresh" || !strings.Contains(recorder.Body.String(), "new-access") || !strings.Contains(recorder.Body.String(), "person@example.com") || strings.Contains(recorder.Body.String(), "new-refresh") {
		t.Fatalf("refresh failed: %d %s", recorder.Code, recorder.Body.String())
	}
	logout := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logout.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "new-refresh"})
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, logout)
	if recorder.Code != http.StatusNoContent || auth.loggedOut != "new-refresh" {
		t.Fatalf("logout failed: %d %q", recorder.Code, auth.loggedOut)
	}
	if recorder.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatal("logout did not clear cookie")
	}
}
func TestAuthErrors(t *testing.T) {
	auth := &fakeAuth{err: service.ErrEmailAlreadyUsed}
	mux := http.NewServeMux()
	NewAuthHandler(auth, CookieConfig{}).RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"person@example.com","password":"password"}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d", recorder.Code)
	}
	auth.err = service.ErrInvalidCredentials
	if !errors.Is(auth.err, service.ErrInvalidCredentials) {
		t.Fatal("test setup")
	}
}

func TestProfileReadsAndUpdatesName(t *testing.T) {
	auth := &fakeAuth{profile: service.PublicUser{ID: uuid.New(), Name: "Alex", Email: "person@example.com"}}
	mux := http.NewServeMux()
	NewAuthHandler(auth, CookieConfig{}).RegisterRoutes(mux)

	profile := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	profile.Header.Set("Authorization", "Bearer access-token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, profile)
	if recorder.Code != http.StatusOK || auth.access != "access-token" || !strings.Contains(recorder.Body.String(), "Alex") {
		t.Fatalf("profile failed: %d %s", recorder.Code, recorder.Body.String())
	}

	update := httptest.NewRequest(http.MethodPatch, "/users/me", strings.NewReader(`{"name":"Alexandra"}`))
	update.Header.Set("Authorization", "Bearer access-token")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, update)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Alexandra") {
		t.Fatalf("profile update failed: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestProfileUploadsAvatar(t *testing.T) {
	auth := &fakeAuth{profile: service.PublicUser{ID: uuid.New(), Name: "Alex", Email: "person@example.com"}}
	mux := http.NewServeMux()
	NewAuthHandler(auth, CookieConfig{}).RegisterRoutes(mux)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "profile.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(part, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/users/me/avatar", &body)
	request.Header.Set("Authorization", "Bearer access-token")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || auth.access != "access-token" || !strings.Contains(recorder.Body.String(), "avatarUrl") {
		t.Fatalf("avatar upload failed: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestProfileRejectsUnsupportedAvatar(t *testing.T) {
	auth := &fakeAuth{}
	mux := http.NewServeMux()
	NewAuthHandler(auth, CookieConfig{}).RegisterRoutes(mux)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "profile.gif")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("not an image"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/users/me/avatar", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
