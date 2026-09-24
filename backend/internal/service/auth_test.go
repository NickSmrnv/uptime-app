package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/repository"
)

type memoryUsers struct{ byEmail map[string]model.User }

func (m *memoryUsers) Create(_ context.Context, user *model.User) error {
	if _, exists := m.byEmail[user.Email]; exists {
		return repository.ErrDuplicateEmail
	}
	m.byEmail[user.Email] = *user
	return nil
}
func (m *memoryUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, nil
	}
	return &user, nil
}
func (m *memoryUsers) FindByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	for _, user := range m.byEmail {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, nil
}
func (m *memoryUsers) UpdateName(_ context.Context, id uuid.UUID, name string) (*model.User, error) {
	for email, user := range m.byEmail {
		if user.ID == id {
			user.Name = name
			m.byEmail[email] = user
			return &user, nil
		}
	}
	return nil, nil
}
func (m *memoryUsers) UpdateAvatar(_ context.Context, id uuid.UUID, filename string) (*model.User, error) {
	for email, user := range m.byEmail {
		if user.ID == id {
			user.AvatarFilename = filename
			m.byEmail[email] = user
			return &user, nil
		}
	}
	return nil, nil
}

type memoryAvatars struct {
	filenames []string
	deleted   []string
}

func (m *memoryAvatars) Save(_ context.Context, category string, _ []byte, extension string) (string, error) {
	filename := category + "/new-avatar" + extension
	m.filenames = append(m.filenames, filename)
	return filename, nil
}
func (m *memoryAvatars) Delete(_ context.Context, filename string) error {
	m.deleted = append(m.deleted, filename)
	return nil
}

type memorySessions struct {
	byHash map[string]model.RefreshSession
}

func (m *memorySessions) Create(_ context.Context, session *model.RefreshSession) error {
	m.byHash[session.TokenHash] = *session
	return nil
}
func (m *memorySessions) Rotate(_ context.Context, hash string, replacement *model.RefreshSession, now time.Time) (model.RefreshSession, error) {
	previous, ok := m.byHash[hash]
	if !ok {
		return model.RefreshSession{}, repository.ErrSessionNotFound
	}
	if previous.RevokedAt != nil || !previous.ExpiresAt.After(now) {
		return model.RefreshSession{}, repository.ErrSessionInvalid
	}
	previous.RevokedAt = &now
	previous.ReplacedByID = &replacement.ID
	m.byHash[hash] = previous
	replacement.UserID = previous.UserID
	m.byHash[replacement.TokenHash] = *replacement
	return previous, nil
}
func (m *memorySessions) RevokeByTokenHash(_ context.Context, hash string, now time.Time) error {
	session, ok := m.byHash[hash]
	if ok && session.RevokedAt == nil {
		session.RevokedAt = &now
		m.byHash[hash] = session
	}
	return nil
}

func newTestService() (*AuthenticationService, *memoryUsers, *memorySessions, *memoryAvatars) {
	users := &memoryUsers{byEmail: map[string]model.User{}}
	sessions := &memorySessions{byHash: map[string]model.RefreshSession{}}
	avatars := &memoryAvatars{}
	svc := NewAuthenticationService(users, sessions, avatars, AuthConfig{JWTSecret: []byte("01234567890123456789012345678901"), JWTIssuer: "test", AccessTokenTTL: 24 * time.Hour, RefreshTokenTTL: 30 * 24 * time.Hour})
	// JWT verification uses the real clock; keep the fixture stable within a test, not tied to a past date.
	now := time.Now().UTC().Truncate(time.Second)
	svc.now = func() time.Time { return now }
	return svc, users, sessions, avatars
}

func TestRegisterNormalizesEmailAndIssuesTokens(t *testing.T) {
	svc, users, sessions, _ := newTestService()
	result, err := svc.Register(context.Background(), UserInput{Name: "  Alex  ", Email: "  PERSON@Example.com ", Password: "password"})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Name != "Alex" || result.User.Email != "person@example.com" || result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
	user := users.byEmail["person@example.com"]
	if user.PasswordHash == "password" {
		t.Fatal("password was not hashed")
	}
	if len(sessions.byHash) != 1 {
		t.Fatal("refresh session was not stored")
	}
	for hash := range sessions.byHash {
		if hash == result.RefreshToken {
			t.Fatal("raw refresh token was stored")
		}
	}
	parsed, err := jwt.Parse(result.AccessToken, func(token *jwt.Token) (any, error) { return []byte("01234567890123456789012345678901"), nil })
	if err != nil || !parsed.Valid {
		t.Fatalf("invalid jwt: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != result.User.ID.String() || claims["iss"] != "test" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
func TestRegisterRejectsWeakAndDuplicatePassword(t *testing.T) {
	svc, _, _, _ := newTestService()
	if _, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "person@example.com", Password: "short"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("weak password error = %v", err)
	}
	if _, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "person@example.com", Password: "password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "PERSON@example.com", Password: "password"}); !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Fatalf("duplicate error = %v", err)
	}
}
func TestLoginAndRefreshRotateSession(t *testing.T) {
	svc, _, _, _ := newTestService()
	registered, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "person@example.com", Password: "password"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), UserInput{Email: "person@example.com", Password: "wrong-password"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login error = %v", err)
	}
	refreshed, err := svc.Refresh(context.Background(), registered.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.RefreshToken == registered.RefreshToken || refreshed.AccessToken == "" || refreshed.User.Email != "person@example.com" {
		t.Fatal("tokens were not rotated")
	}
	if _, err := svc.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("old token error = %v", err)
	}
}
func TestLogoutRevokesSession(t *testing.T) {
	svc, _, _, _ := newTestService()
	registered, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "person@example.com", Password: "password"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(context.Background(), registered.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("refresh after logout = %v", err)
	}
}

func TestProfileUpdatesNameWithValidAccessToken(t *testing.T) {
	svc, _, _, _ := newTestService()
	registered, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "person@example.com", Password: "password"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateProfile(context.Background(), registered.AccessToken, "  Alexandra  ")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Alexandra" || updated.Email != "person@example.com" {
		t.Fatalf("unexpected updated profile: %#v", updated)
	}
	profile, err := svc.Profile(context.Background(), registered.AccessToken)
	if err != nil || profile.Name != "Alexandra" {
		t.Fatalf("profile = %#v, err = %v", profile, err)
	}
}

func TestProfileUploadsAvatarAndReplacesPreviousFile(t *testing.T) {
	svc, users, _, avatars := newTestService()
	registered, err := svc.Register(context.Background(), UserInput{Name: "Alex", Email: "person@example.com", Password: "password"})
	if err != nil {
		t.Fatal(err)
	}
	user := users.byEmail["person@example.com"]
	user.AvatarFilename = "old-avatar.jpg"
	users.byEmail["person@example.com"] = user

	updated, err := svc.UpdateAvatar(context.Background(), registered.AccessToken, AvatarInput{Data: []byte("image"), Extension: ".png"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AvatarURL != "/uploads/avatars/new-avatar.png" {
		t.Fatalf("avatar URL = %q", updated.AvatarURL)
	}
	if len(avatars.deleted) != 1 || avatars.deleted[0] != "old-avatar.jpg" {
		t.Fatalf("deleted avatars = %#v", avatars.deleted)
	}
}

var _ = uuid.Nil
