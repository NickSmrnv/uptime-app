package model

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name           string    `gorm:"size:100;not null"`
	Email          string    `gorm:"size:320;not null;uniqueIndex:idx_users_email"`
	AvatarFilename string    `gorm:"size:80"`
	PasswordHash   string    `gorm:"not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

type RefreshSession struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_refresh_sessions_user_id"`
	User         User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:UserID"`
	TokenHash    string     `gorm:"size:64;not null;uniqueIndex:idx_refresh_sessions_token_hash"`
	ExpiresAt    time.Time  `gorm:"not null;index:idx_refresh_sessions_expires_at"`
	RevokedAt    *time.Time `gorm:"index"`
	ReplacedByID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

type Monitor struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index:idx_monitors_user_created,priority:1"`
	User            User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:UserID"`
	URL             string    `gorm:"size:2048;not null"`
	IntervalSeconds int       `gorm:"not null"`
	CreatedAt       time.Time `gorm:"not null;index:idx_monitors_user_created,priority:2"`
	UpdatedAt       time.Time `gorm:"not null"`
}
