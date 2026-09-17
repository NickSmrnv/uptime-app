package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MonitorRepository struct{ db *gorm.DB }

func NewMonitorRepository(db *gorm.DB) *MonitorRepository { return &MonitorRepository{db: db} }

func (r *MonitorRepository) CreateIfBelowLimit(ctx context.Context, monitor *model.Monitor, limit int) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", monitor.UserID).Take(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return fmt.Errorf("lock monitor user: %w", err)
		}
		var count int64
		if err := tx.Model(&model.Monitor{}).Where("user_id = ?", monitor.UserID).Count(&count).Error; err != nil {
			return fmt.Errorf("count monitors: %w", err)
		}
		if count >= int64(limit) {
			return nil
		}
		if err := tx.Create(monitor).Error; err != nil {
			return fmt.Errorf("create monitor: %w", err)
		}
		created = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return created, nil
}

func (r *MonitorRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Monitor, error) {
	var monitors []model.Monitor
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&monitors).Error; err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	return monitors, nil
}
