package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrEmailAlreadyUsed   = errors.New("email already used")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrUnauthorized       = errors.New("unauthorized")
)

type UserStore interface {
	Create(context.Context, *model.User) error
	FindByEmail(context.Context, string) (*model.User, error)
	FindByID(context.Context, uuid.UUID) (*model.User, error)
	UpdateName(context.Context, uuid.UUID, string) (*model.User, error)
	UpdateAvatar(context.Context, uuid.UUID, string) (*model.User, error)
}

type SessionStore interface {
	Create(context.Context, *model.RefreshSession) error
	Rotate(context.Context, string, *model.RefreshSession, time.Time) (model.RefreshSession, error)
	RevokeByTokenHash(context.Context, string, time.Time) error
}

type AuthConfig struct {
	JWTSecret       []byte
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type AuthenticationService struct {
	users    UserStore
	sessions SessionStore
	uploads  FileStore
	config   AuthConfig
	now      func() time.Time
}

type UserInput struct{ Name, Email, Password string }

type PublicUser struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
}

type AvatarInput struct {
	Data      []byte
	Extension string
}

type AuthenticationResult struct {
	AccessToken  string
	RefreshToken string
	User         PublicUser
}

// NewAuthenticationService uses a UTC clock so token and session expiry timestamps are consistent
// across storage and JWT claims.
func NewAuthenticationService(users UserStore, sessions SessionStore, uploads FileStore, cfg AuthConfig) *AuthenticationService {
	return &AuthenticationService{users: users, sessions: sessions, uploads: uploads, config: cfg, now: func() time.Time { return time.Now().UTC() }}
}

// Register uses a canonical email and bcrypt hash so equivalent addresses share one identity and
// the original password never reaches the database.
func (s *AuthenticationService) Register(ctx context.Context, input UserInput) (AuthenticationResult, error) {
	email, err := normalizeEmail(input.Email)
	name, errName := normalizeName(input.Name)
	if err != nil || errName != nil || !validPassword(input.Password) {
		return AuthenticationResult{}, ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthenticationResult{}, fmt.Errorf("hash password: %w", err)
	}
	now := s.now()
	user := model.User{ID: uuid.New(), Name: name, Email: email, PasswordHash: string(hash), CreatedAt: now, UpdatedAt: now}
	if err := s.users.Create(ctx, &user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return AuthenticationResult{}, ErrEmailAlreadyUsed
		}
		return AuthenticationResult{}, err
	}
	return s.issue(ctx, user)
}

// Login returns the same credential error for an unknown email and a wrong password to avoid
// disclosing which accounts exist.
func (s *AuthenticationService) Login(ctx context.Context, input UserInput) (AuthenticationResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil || input.Password == "" {
		return AuthenticationResult{}, ErrInvalidCredentials
	}
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return AuthenticationResult{}, err
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return AuthenticationResult{}, ErrInvalidCredentials
	}
	return s.issue(ctx, *user)
}

// Refresh atomically replaces a valid refresh session and returns a new access token,
// preventing a previously used refresh token from being replayed.
func (s *AuthenticationService) Refresh(ctx context.Context, refreshToken string) (AuthenticationResult, error) {
	if refreshToken == "" {
		return AuthenticationResult{}, ErrInvalidRefresh
	}
	now := s.now()
	newToken, newHash, err := generateRefreshToken()
	if err != nil {
		return AuthenticationResult{}, err
	}
	replacement := model.RefreshSession{ID: uuid.New(), TokenHash: newHash, ExpiresAt: now.Add(s.config.RefreshTokenTTL), CreatedAt: now, UpdatedAt: now}
	previous, err := s.sessions.Rotate(ctx, hashToken(refreshToken), &replacement, now)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) || errors.Is(err, repository.ErrSessionInvalid) {
			return AuthenticationResult{}, ErrInvalidRefresh
		}
		return AuthenticationResult{}, err
	}
	user, err := s.users.FindByID(ctx, previous.UserID)
	if err != nil {
		return AuthenticationResult{}, err
	}
	if user == nil {
		return AuthenticationResult{}, ErrInvalidRefresh
	}
	accessToken, err := s.signAccessToken(previous.UserID, now)
	if err != nil {
		return AuthenticationResult{}, err
	}
	return AuthenticationResult{AccessToken: accessToken, RefreshToken: newToken, User: publicUser(*user)}, nil
}

// Profile reloads the user after token validation so deleted users and changed profile data are
// never inferred from stale JWT claims.
func (s *AuthenticationService) Profile(ctx context.Context, accessToken string) (PublicUser, error) {
	userID, err := s.userIDFromAccessToken(accessToken)
	if err != nil {
		return PublicUser{}, ErrUnauthorized
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	if user == nil {
		return PublicUser{}, ErrUnauthorized
	}
	return publicUser(*user), nil
}

// UpdateProfile persists only the validated display name to avoid accepting uncontrolled user attributes.
func (s *AuthenticationService) UpdateProfile(ctx context.Context, accessToken, name string) (PublicUser, error) {
	userID, err := s.userIDFromAccessToken(accessToken)
	if err != nil {
		return PublicUser{}, ErrUnauthorized
	}
	name, err = normalizeName(name)
	if err != nil {
		return PublicUser{}, ErrInvalidInput
	}
	user, err := s.users.UpdateName(ctx, userID, name)
	if err != nil {
		return PublicUser{}, err
	}
	if user == nil {
		return PublicUser{}, ErrUnauthorized
	}
	return publicUser(*user), nil
}

// UpdateAvatar replaces the authenticated user's avatar after validating the accepted image type.
// It removes a newly stored file on a database failure and the old file only after a successful update.
func (s *AuthenticationService) UpdateAvatar(ctx context.Context, accessToken string, avatar AvatarInput) (PublicUser, error) {
	userID, err := s.userIDFromAccessToken(accessToken)
	if err != nil {
		return PublicUser{}, ErrUnauthorized
	}
	if s.uploads == nil || (avatar.Extension != ".jpg" && avatar.Extension != ".png") || len(avatar.Data) == 0 {
		return PublicUser{}, ErrInvalidInput
	}
	current, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	if current == nil {
		return PublicUser{}, ErrUnauthorized
	}
	filename, err := s.uploads.Save(ctx, "avatars", avatar.Data, avatar.Extension)
	if err != nil {
		return PublicUser{}, err
	}
	updated, err := s.users.UpdateAvatar(ctx, userID, filename)
	if err != nil {
		_ = s.uploads.Delete(ctx, filename)
		return PublicUser{}, err
	}
	if updated == nil {
		_ = s.uploads.Delete(ctx, filename)
		return PublicUser{}, ErrUnauthorized
	}
	if current.AvatarFilename != "" && current.AvatarFilename != filename {
		_ = s.uploads.Delete(ctx, current.AvatarFilename)
	}
	return publicUser(*updated), nil
}

// Logout revokes the supplied refresh session when present; an absent cookie is treated as a
// successful logout so clients can safely repeat the request.
func (s *AuthenticationService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.sessions.RevokeByTokenHash(ctx, hashToken(refreshToken), s.now())
}

func (s *AuthenticationService) issue(ctx context.Context, user model.User) (AuthenticationResult, error) {
	now := s.now()
	refreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		return AuthenticationResult{}, err
	}
	if err := s.sessions.Create(ctx, &model.RefreshSession{ID: uuid.New(), UserID: user.ID, TokenHash: tokenHash, ExpiresAt: now.Add(s.config.RefreshTokenTTL), CreatedAt: now, UpdatedAt: now}); err != nil {
		return AuthenticationResult{}, err
	}
	accessToken, err := s.signAccessToken(user.ID, now)
	if err != nil {
		return AuthenticationResult{}, err
	}
	return AuthenticationResult{AccessToken: accessToken, RefreshToken: refreshToken, User: publicUser(user)}, nil
}

// signAccessToken creates an HS256 JWT with standard identity and expiry claims; HS256 is used
// because this service signs and verifies with the same configured server secret.
func (s *AuthenticationService) signAccessToken(userID uuid.UUID, now time.Time) (string, error) {
	claims := jwt.MapClaims{"sub": userID.String(), "iss": s.config.JWTIssuer, "iat": now.Unix(), "exp": now.Add(s.config.AccessTokenTTL).Unix(), "jti": uuid.NewString()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.config.JWTSecret)
}

// userIDFromAccessToken verifies the configured issuer and exact signing method before reading
// the subject UUID, preventing algorithm substitution and untrusted identity claims.
func (s *AuthenticationService) userIDFromAccessToken(accessToken string) (uuid.UUID, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.config.JWTSecret, nil
	}, jwt.WithIssuer(s.config.JWTIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid {
		return uuid.Nil, ErrUnauthorized
	}
	subject, err := claims.GetSubject()
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	userID, err := uuid.Parse(subject)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	return userID, nil
}

// normalizeEmail trims and lowercases an address before strict parsing so the unique email key
// represents one canonical value instead of case or whitespace variants.
func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 320 {
		return "", ErrInvalidInput
	}
	return email, nil
}

func validPassword(password string) bool { return utf8.RuneCountInString(password) >= 8 }

func normalizeName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return "", ErrInvalidInput
	}
	return name, nil
}

// publicUser prevents password and session fields from crossing the API boundary.
func publicUser(user model.User) PublicUser {
	public := PublicUser{ID: user.ID, Name: user.Name, Email: user.Email}
	if user.AvatarFilename != "" {
		public.AvatarURL = "/uploads/" + user.AvatarFilename
	}
	return public
}

// hashToken returns a deterministic SHA-256 digest so refresh tokens can be matched without
// storing their bearer value in the database.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum[:])
}

// generateRefreshToken creates 256 bits of cryptographic randomness and returns both its URL-safe
// bearer value and database-safe hash.
func generateRefreshToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, hashToken(token), nil
}
