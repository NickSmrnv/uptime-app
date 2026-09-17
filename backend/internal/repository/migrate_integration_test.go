package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestMigratePostgreSQLAndRotateSession requires a disposable database because it drops tables.
func TestMigratePostgreSQLAndRotateSession(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.RefreshSession{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Migrator().DropTable(&model.RefreshSession{}, &model.User{}) })
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	users := NewUserRepository(db)
	user := &model.User{ID: uuid.New(), Email: "person@example.com", PasswordHash: "hash"}
	if err := users.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(context.Background(), &model.User{ID: uuid.New(), Email: user.Email, PasswordHash: "other"}); !errors.Is(err, ErrDuplicateEmail) {
		t.Fatalf("duplicate error = %v", err)
	}

	sessions := NewSessionRepository(db)
	now := time.Now().UTC()
	if err := sessions.Create(context.Background(), &model.RefreshSession{ID: uuid.New(), UserID: user.ID, TokenHash: "old", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	replacement := &model.RefreshSession{ID: uuid.New(), TokenHash: "new", ExpiresAt: now.Add(time.Hour)}
	if _, err := sessions.Rotate(context.Background(), "old", replacement, now); err != nil {
		t.Fatal(err)
	}
	if replacement.UserID != user.ID {
		t.Fatalf("replacement user ID = %s, want %s", replacement.UserID, user.ID)
	}
	if _, err := sessions.Rotate(context.Background(), "old", &model.RefreshSession{ID: uuid.New(), TokenHash: "another", ExpiresAt: now.Add(time.Hour)}, now); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("old session rotation error = %v", err)
	}
}
