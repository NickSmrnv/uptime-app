package monitoring

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
)

type runnerStore struct {
	mu        sync.Mutex
	remaining []model.Monitor
	saved     int
	claimErr  error
	saveErr   error
}

func (s *runnerStore) ClaimChecks(_ context.Context, limit int) ([]model.Monitor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	n := min(limit, len(s.remaining))
	checks := append([]model.Monitor{}, s.remaining[:n]...)
	s.remaining = s.remaining[n:]
	return checks, nil
}

func (s *runnerStore) CompleteCheck(_ context.Context, _ model.Monitor, _ model.CheckResult) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveErr != nil {
		return false, s.saveErr
	}
	s.saved++
	return true, nil
}

func (s *runnerStore) DeleteExpiredMinutes(context.Context) (int64, error) { return 0, nil }

type blockingProbe struct {
	started chan struct{}
	release chan struct{}
}

func (p *blockingProbe) Check(ctx context.Context, _ model.Monitor) model.CheckResult {
	p.started <- struct{}{}
	select {
	case <-ctx.Done():
	case <-p.release:
	}
	code := 200
	return model.CheckResult{CheckedAt: time.Now(), StatusCode: &code}
}

func TestRunnerBoundsConcurrencyAndJoinsShutdown(t *testing.T) {
	store := &runnerStore{remaining: []model.Monitor{}}
	for range 100 {
		store.remaining = append(store.remaining, model.Monitor{ID: uuid.New(), IntervalSeconds: 5, NextCheckAt: time.Now()})
	}
	probe := &blockingProbe{started: make(chan struct{}, 100), release: make(chan struct{})}
	runner := NewRunner(store, probe, 20)
	ctx, cancel := context.WithCancel(context.Background())
	var group sync.WaitGroup
	runner.dispatch(ctx, &group)
	for range 20 {
		select {
		case <-probe.started:
		case <-time.After(time.Second):
			t.Fatal("check not started")
		}
	}
	runner.dispatch(ctx, &group)
	if runner.active.Load() != 20 {
		t.Fatal("concurrency not bounded")
	}
	cancel()
	group.Wait()
	if store.saved != 0 || runner.active.Load() != 0 {
		t.Fatal("shutdown generated fake failures or leaked checks")
	}
}

func TestRunnerDatabaseErrorsDoNotBecomeResults(t *testing.T) {
	store := &runnerStore{claimErr: errors.New("offline")}
	probe := &blockingProbe{started: make(chan struct{}, 1), release: make(chan struct{})}
	close(probe.release)
	runner := NewRunner(store, probe, 1)
	var group sync.WaitGroup
	runner.dispatch(context.Background(), &group)
	if runner.claimErrors.Load() != 1 || len(probe.started) != 0 {
		t.Fatal("probe ran without claim")
	}
	store.claimErr = nil
	store.saveErr = errors.New("offline")
	store.remaining = []model.Monitor{{NextCheckAt: time.Now(), IntervalSeconds: 5}}
	runner.dispatch(context.Background(), &group)
	group.Wait()
	if runner.writeErrors.Load() != 1 || runner.completed.Load() != 0 || store.saved != 0 {
		t.Fatal("database error recorded as observation")
	}
}
