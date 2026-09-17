package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"gorm.io/gorm"
)

type MonitorRepository struct{ db *gorm.DB }

func NewMonitorRepository(db *gorm.DB) *MonitorRepository { return &MonitorRepository{db: db} }

func (r *MonitorRepository) Create(ctx context.Context, monitor *model.Monitor) error {
	if err := r.db.WithContext(ctx).Create(monitor).Error; err != nil {
		return fmt.Errorf("create monitor: %w", err)
	}
	return nil
}

func (r *MonitorRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Monitor, error) {
	var monitors []model.Monitor
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&monitors).Error; err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	return monitors, nil
}
