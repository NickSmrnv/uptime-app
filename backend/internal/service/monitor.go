package service

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/repository"
)

var ErrMonitorLimitReached = errors.New("monitor limit reached")

const (
	minMonitorIntervalSeconds = 5
	maxMonitorIntervalSeconds = 7 * 24 * 60 * 60
	maxMonitorsPerUser        = 100
)

type MonitorStore interface {
	CreateIfBelowLimit(context.Context, *model.Monitor, int) (bool, error)
	ListByUserID(context.Context, uuid.UUID) ([]model.Monitor, error)
}

type AccessTokenVerifier interface {
	UserIDFromAccessToken(string) (uuid.UUID, error)
}

type MonitorUserStore interface {
	FindByID(context.Context, uuid.UUID) (*model.User, error)
}

type MonitorInput struct {
	URL             string
	IntervalSeconds int
}

type MonitorService struct {
	monitors MonitorStore
	users    MonitorUserStore
	tokens   AccessTokenVerifier
	now      func() time.Time
}

func NewMonitorService(monitors MonitorStore, users MonitorUserStore, tokens AccessTokenVerifier) *MonitorService {
	return &MonitorService{monitors: monitors, users: users, tokens: tokens, now: func() time.Time { return time.Now().UTC() }}
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
	created, err := s.monitors.CreateIfBelowLimit(ctx, &monitor, maxMonitorsPerUser)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.Monitor{}, ErrUnauthorized
		}
		return model.Monitor{}, err
	}
	if !created {
		return model.Monitor{}, ErrMonitorLimitReached
	}
	return monitor, nil
}

func (s *MonitorService) List(ctx context.Context, accessToken string) ([]model.Monitor, error) {
	userID, err := s.tokens.UserIDFromAccessToken(accessToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
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
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil || isBlockedMonitorHost(parsed.Hostname()) {
		return "", errors.New("invalid monitor URL")
	}
	return parsed.String(), nil
}

func isBlockedMonitorHost(host string) bool {
	// A future monitor executor must repeat this check after resolving DNS and
	// before every redirect, since this validation only sees the submitted URL.
	host = strings.ToLower(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.Contains(host, "%") {
		return true
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified()
}
