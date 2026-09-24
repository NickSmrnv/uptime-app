package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
)

func TestMonitorStatsWeightsCountsAndLeavesGaps(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 30, 0, time.UTC)
	monitor := model.Monitor{ID: uuid.New(), UserID: uuid.New(), HistoryVersion: 2}
	store := &memoryMonitors{monitors: []model.Monitor{monitor}, minutes: []model.MonitorBucket{
		{Start: now.Truncate(time.Minute), Successes: 9, Failures: 3},
		{Start: now.Truncate(time.Minute).Add(-time.Minute), Successes: 1, Failures: 0},
	}}
	svc := NewMonitorService(store, fakeTokenVerifier{userID: monitor.UserID})
	svc.now = func() time.Time { return now }
	for _, period := range []string{"1h", "24h", "7d", "30d"} {
		stats, err := svc.Stats(context.Background(), "token", monitor.ID, period)
		if err != nil {
			t.Fatal(err)
		}
		if stats.Successes != 10 || stats.Failures != 3 || *stats.SuccessPercent != float64(10)*100/13 {
			t.Fatalf("weighted stats = %+v", stats)
		}
		if stats.Buckets[0].SuccessPercent != nil || stats.HistoryVersion != 2 {
			t.Fatal("gap/version lost")
		}
	}
	if _, err := svc.Stats(context.Background(), "token", uuid.New(), "1h"); !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("foreign/missing stats: %v", err)
	}
	if _, err := svc.Stats(context.Background(), "token", monitor.ID, "all"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid period: %v", err)
	}
	svc.tokens = fakeTokenVerifier{err: errors.New("expired")}
	if _, err := svc.Stats(context.Background(), "token", monitor.ID, "1h"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("invalid token: %v", err)
	}
}

func TestMonitorStatusDistinguishesMissingChecks(t *testing.T) {
	now := time.Now().UTC()
	code := 200
	monitor := model.Monitor{IntervalSeconds: 5, NextCheckAt: now}
	if MonitorStatus(monitor, now) != "pending" {
		t.Fatal("new monitor should be pending")
	}
	monitor.LastCheckedAt, monitor.LastStatusCode = &now, &code
	if MonitorStatus(monitor, now) != "up" {
		t.Fatal("200 should be up")
	}
	code = 302
	if MonitorStatus(monitor, now) != "down" {
		t.Fatal("redirect should be down")
	}
	code = 200
	monitor.LastError = "timeout"
	if MonitorStatus(monitor, now) != "down" {
		t.Fatal("network error should be down")
	}
	monitor.LastError = ""
	if MonitorStatus(monitor, now.Add(9*time.Second)) != "up" {
		t.Fatal("premature stale status")
	}
	if MonitorStatus(monitor, now.Add(10*time.Second)) != "stale" {
		t.Fatal("missing check should be stale")
	}
	if model.CheckTimeout(5) != 4*time.Second || model.CheckTimeout(60) != 10*time.Second {
		t.Fatal("wrong timeout")
	}
}
