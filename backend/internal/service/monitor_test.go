package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
)

type memoryMonitors struct{ monitors []model.Monitor }

func (m *memoryMonitors) Create(_ context.Context, monitor *model.Monitor) error {
	m.monitors = append(m.monitors, *monitor)
	return nil
}

func (m *memoryMonitors) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.Monitor, error) {
	var result []model.Monitor
	for _, monitor := range m.monitors {
		if monitor.UserID == userID {
			result = append(result, monitor)
		}
	}
	return result, nil
}

type fakeTokenVerifier struct {
	userID uuid.UUID
	err    error
}

func (v fakeTokenVerifier) UserIDFromAccessToken(string) (uuid.UUID, error) { return v.userID, v.err }

func TestCreateMonitorValidatesAndPersistsInput(t *testing.T) {
	userID := uuid.New()
	store := &memoryMonitors{}
	svc := NewMonitorService(store, fakeTokenVerifier{userID: userID})
	svc.now = func() time.Time { return time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC) }

	monitor, err := svc.Create(context.Background(), "access", MonitorInput{URL: " https://example.com/health ", IntervalSeconds: 300})
	if err != nil {
		t.Fatal(err)
	}
	if monitor.UserID != userID || monitor.URL != "https://example.com/health" || monitor.IntervalSeconds != 300 || len(store.monitors) != 1 {
		t.Fatalf("unexpected monitor: %#v", monitor)
	}
}

func TestCreateMonitorRejectsInvalidURLAndInterval(t *testing.T) {
	svc := NewMonitorService(&memoryMonitors{}, fakeTokenVerifier{userID: uuid.New()})
	for _, input := range []MonitorInput{
		{URL: "example.com", IntervalSeconds: 60},
		{URL: "ftp://example.com", IntervalSeconds: 60},
		{URL: "https://example.com", IntervalSeconds: 4},
		{URL: "https://example.com", IntervalSeconds: maxMonitorIntervalSeconds + 1},
	} {
		if _, err := svc.Create(context.Background(), "access", input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v: error = %v", input, err)
		}
	}
}

func TestMonitorRequiresValidAccessToken(t *testing.T) {
	svc := NewMonitorService(&memoryMonitors{}, fakeTokenVerifier{err: errors.New("bad token")})
	if _, err := svc.List(context.Background(), "access"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("list error = %v", err)
	}
	if _, err := svc.Create(context.Background(), "access", MonitorInput{URL: "https://example.com", IntervalSeconds: 60}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("create error = %v", err)
	}
}
