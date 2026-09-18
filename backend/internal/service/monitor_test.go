package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/repository"
)

type memoryMonitors struct{ monitors []model.Monitor }

func (m *memoryMonitors) CreateIfBelowLimit(_ context.Context, monitor *model.Monitor, limit int) (bool, error) {
	count := 0
	for _, existing := range m.monitors {
		if existing.UserID == monitor.UserID {
			count++
		}
	}
	if count >= limit {
		return false, nil
	}
	m.monitors = append(m.monitors, *monitor)
	return true, nil
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

func (m *memoryMonitors) UpdateByIDAndUserID(_ context.Context, monitor *model.Monitor) (model.Monitor, error) {
	for index, existing := range m.monitors {
		if existing.ID == monitor.ID && existing.UserID == monitor.UserID {
			m.monitors[index].URL = monitor.URL
			m.monitors[index].IntervalSeconds = monitor.IntervalSeconds
			m.monitors[index].UpdatedAt = monitor.UpdatedAt
			return m.monitors[index], nil
		}
	}
	return model.Monitor{}, repository.ErrMonitorNotFound
}

func (m *memoryMonitors) DeleteByIDAndUserID(_ context.Context, monitorID, userID uuid.UUID) error {
	for index, existing := range m.monitors {
		if existing.ID == monitorID && existing.UserID == userID {
			m.monitors = append(m.monitors[:index], m.monitors[index+1:]...)
			return nil
		}
	}
	return repository.ErrMonitorNotFound
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
		{URL: "http://localhost", IntervalSeconds: 60},
		{URL: "http://127.0.0.1", IntervalSeconds: 60},
		{URL: "http://169.254.169.254", IntervalSeconds: 60},
		{URL: "http://[::1]", IntervalSeconds: 60},
		{URL: "https://example.com", IntervalSeconds: 4},
		{URL: "https://example.com", IntervalSeconds: maxMonitorIntervalSeconds + 1},
	} {
		if _, err := svc.Create(context.Background(), "access", input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v: error = %v", input, err)
		}
	}
}

func TestCreateMonitorEnforcesUserLimit(t *testing.T) {
	userID := uuid.New()
	store := &memoryMonitors{}
	for range maxMonitorsPerUser {
		store.monitors = append(store.monitors, model.Monitor{UserID: userID})
	}
	svc := NewMonitorService(store, fakeTokenVerifier{userID: userID})
	if _, err := svc.Create(context.Background(), "access", MonitorInput{URL: "https://example.com", IntervalSeconds: 60}); !errors.Is(err, ErrMonitorLimitReached) {
		t.Fatalf("create error = %v", err)
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
	if _, err := svc.Update(context.Background(), "access", uuid.New(), MonitorInput{URL: "https://example.com", IntervalSeconds: 60}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("update error = %v", err)
	}
	if err := svc.Delete(context.Background(), "access", uuid.New()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("delete error = %v", err)
	}
}

func TestUpdateMonitorValidatesAndScopesByUser(t *testing.T) {
	userID := uuid.New()
	monitor := model.Monitor{ID: uuid.New(), UserID: userID, URL: "https://example.com", IntervalSeconds: 60}
	store := &memoryMonitors{monitors: []model.Monitor{monitor}}
	svc := NewMonitorService(store, fakeTokenVerifier{userID: userID})
	svc.now = func() time.Time { return time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC) }

	updated, err := svc.Update(context.Background(), "access", monitor.ID, MonitorInput{URL: " https://updated.example.com ", IntervalSeconds: 300})
	if err != nil || updated.URL != "https://updated.example.com" || updated.IntervalSeconds != 300 || !updated.UpdatedAt.Equal(svc.now()) {
		t.Fatalf("updated monitor = %#v, err = %v", updated, err)
	}
	if _, err := svc.Update(context.Background(), "access", uuid.New(), MonitorInput{URL: "https://example.com", IntervalSeconds: 60}); !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("missing update error = %v", err)
	}
	for _, input := range []MonitorInput{
		{URL: "example.com", IntervalSeconds: 60},
		{URL: "https://example.com", IntervalSeconds: 4},
	} {
		if _, err := svc.Update(context.Background(), "access", monitor.ID, input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid update input %#v: error = %v", input, err)
		}
	}
	if err := svc.Delete(context.Background(), "access", monitor.ID); err != nil {
		t.Fatalf("delete error = %v", err)
	}
	if len(store.monitors) != 0 {
		t.Fatalf("monitors after delete = %#v", store.monitors)
	}
}
