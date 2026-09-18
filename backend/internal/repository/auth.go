package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var (
	ErrDuplicateEmail  = errors.New("email already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("refresh session not found")
	ErrSessionInvalid  = errors.New("refresh session is invalid")
)

type UserRepository struct{ db *gorm.DB }

// NewUserRepository keeps user persistence behind the shared configured GORM connection.
func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

// Create inserts a user and translates the database's unique-email violation into a domain error
// so callers do not depend on PostgreSQL error details.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// FindByEmail returns nil for an absent user, which distinguishes an ordinary login miss from a
// database failure without requiring callers to interpret GORM errors.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

// FindByID lets callers reject a valid but stale token after its user was removed.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by ID: %w", err)
	}
	return &user, nil
}

// UpdateName updates only the display name and reloads the row so its result reflects database-managed fields.
func (r *UserRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) (*model.User, error) {
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("name", name)
	if result.Error != nil {
		return nil, fmt.Errorf("update user name: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return r.FindByID(ctx, id)
}

// UpdateAvatar changes only the stored avatar key and reloads the user for a complete public response.
func (r *UserRepository) UpdateAvatar(ctx context.Context, id uuid.UUID, filename string) (*model.User, error) {
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("avatar_filename", filename)
	if result.Error != nil {
		return nil, fmt.Errorf("update user avatar: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return r.FindByID(ctx, id)
}

type SessionRepository struct{ db *gorm.DB }

// NewSessionRepository keeps refresh-session persistence behind the shared configured GORM connection.
func NewSessionRepository(db *gorm.DB) *SessionRepository { return &SessionRepository{db: db} }

// Create persists a hashed refresh session after the service has generated its bearer counterpart.
func (r *SessionRepository) Create(ctx context.Context, session *model.RefreshSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

// Rotate locks the previous session and revokes it before inserting its replacement in one transaction.
// The row lock ensures concurrent refreshes cannot both succeed with the same token.
func (r *SessionRepository) Rotate(ctx context.Context, tokenHash string, replacement *model.RefreshSession, now time.Time) (model.RefreshSession, error) {
	var previous model.RefreshSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", tokenHash).Take(&previous).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		if err != nil {
			return fmt.Errorf("lock refresh session: %w", err)
		}
		if previous.RevokedAt != nil || !previous.ExpiresAt.After(now) {
			return ErrSessionInvalid
		}
		replacement.UserID = previous.UserID
		if err := tx.Model(&previous).Updates(map[string]any{"revoked_at": now, "replaced_by_id": replacement.ID}).Error; err != nil {
			return fmt.Errorf("revoke refresh session: %w", err)
		}
		if err := tx.Create(replacement).Error; err != nil {
			return fmt.Errorf("create replacement refresh session: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.RefreshSession{}, err
	}
	return previous, nil
}

// RevokeByTokenHash marks an active session revoked without treating an already absent or revoked
// token as an error, which makes logout repeatable.
func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.RefreshSession{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh session: %w", result.Error)
	}
	return nil
}

// isUniqueViolation hides the ORM-specific duplicate-key sentinel behind the repository boundary.
func isUniqueViolation(err error) bool { return errors.Is(err, gorm.ErrDuplicatedKey) }
