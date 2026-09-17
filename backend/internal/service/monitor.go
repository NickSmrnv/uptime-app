package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
)

const (
	minMonitorIntervalSeconds = 5
	maxMonitorIntervalSeconds = 7 * 24 * 60 * 60
)

type MonitorStore interface {
	Create(context.Context, *model.Monitor) error
	ListByUserID(context.Context, uuid.UUID) ([]model.Monitor, error)
}

type AccessTokenVerifier interface {
	UserIDFromAccessToken(string) (uuid.UUID, error)
}

type MonitorInput struct {
	URL             string
	IntervalSeconds int
}

type MonitorService struct {
	monitors MonitorStore
	tokens   AccessTokenVerifier
	now      func() time.Time
}

func NewMonitorService(monitors MonitorStore, tokens AccessTokenVerifier) *MonitorService {
	return &MonitorService{monitors: monitors, tokens: tokens, now: func() time.Time { return time.Now().UTC() }}
}

func (s *MonitorService) Create(ctx context.Context, accessToken string, input MonitorInput) (model.Monitor, error) {
	userID, err := s.tokens.UserIDFromAccessToken(accessToken)
	if err != nil {
		return model.Monitor{}, ErrUnauthorized
	}
	monitorURL, err := normalizeMonitorURL(input.URL)
	if err != nil || input.IntervalSeconds < minMonitorIntervalSeconds || input.IntervalSeconds > maxMonitorIntervalSeconds {
		return model.Monitor{}, ErrInvalidInput
	}
	now := s.now()
	monitor := model.Monitor{
		ID:              uuid.New(),
		UserID:          userID,
		URL:             monitorURL,
		IntervalSeconds: input.IntervalSeconds,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.monitors.Create(ctx, &monitor); err != nil {
		return model.Monitor{}, err
	}
	return monitor, nil
}

func (s *MonitorService) List(ctx context.Context, accessToken string) ([]model.Monitor, error) {
	userID, err := s.tokens.UserIDFromAccessToken(accessToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return s.monitors.ListByUserID(ctx, userID)
}

func normalizeMonitorURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 2048 {
		return "", errors.New("invalid monitor URL")
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return "", errors.New("invalid monitor URL")
	}
	return parsed.String(), nil
}
