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
	ErrSessionNotFound = errors.New("refresh session not found")
	ErrSessionInvalid  = errors.New("refresh session is invalid")
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
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

func NewSessionRepository(db *gorm.DB) *SessionRepository { return &SessionRepository{db: db} }
func (r *SessionRepository) Create(ctx context.Context, session *model.RefreshSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}
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
func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.RefreshSession{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh session: %w", result.Error)
	}
	return nil
}
func isUniqueViolation(err error) bool { return errors.Is(err, gorm.ErrDuplicatedKey) }
