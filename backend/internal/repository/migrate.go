package repository

import (
	"context"
	"fmt"
	"github.com/uptime-app/backend/internal/model"
	"gorm.io/gorm"
)

// Migrate creates the initial schema. Future changes require explicit new steps.
func Migrate(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		migrator := tx.Migrator()
		if !migrator.HasTable(&model.User{}) {
			if err := migrator.CreateTable(&model.User{}); err != nil {
				return fmt.Errorf("create users table: %w", err)
			}
		}
		if !migrator.HasColumn(&model.User{}, "Name") {
			if err := tx.Exec(`ALTER TABLE users ADD COLUMN name varchar(100) NOT NULL DEFAULT 'Пользователь'`).Error; err != nil {
				return fmt.Errorf("add users.name column: %w", err)
			}
			if err := tx.Exec(`ALTER TABLE users ALTER COLUMN name DROP DEFAULT`).Error; err != nil {
				return fmt.Errorf("remove users.name default: %w", err)
			}
		}
		if !migrator.HasColumn(&model.User{}, "AvatarFilename") {
			if err := tx.Exec(`ALTER TABLE users ADD COLUMN avatar_filename varchar(80)`).Error; err != nil {
				return fmt.Errorf("add users.avatar_filename column: %w", err)
			}
		}
		if err := tx.Exec(`UPDATE users SET avatar_filename = 'avatars/' || avatar_filename WHERE avatar_filename <> '' AND position('/' in avatar_filename) = 0`).Error; err != nil {
			return fmt.Errorf("migrate users.avatar_filename values: %w", err)
		}
		if !migrator.HasTable(&model.RefreshSession{}) {
			if err := migrator.CreateTable(&model.RefreshSession{}); err != nil {
				return fmt.Errorf("create refresh_sessions table: %w", err)
			}
		}
		return nil
	})
}
