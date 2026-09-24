package monitoring

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uptime-app/backend/internal/model"
)

type Store interface {
	ClaimChecks(context.Context, int) ([]model.Monitor, error)
	CompleteCheck(context.Context, model.Monitor, model.CheckResult) (bool, error)
	DeleteExpiredMinutes(context.Context) (int64, error)
}

type Probe interface {
	Check(context.Context, model.Monitor) model.CheckResult
}

type Runner struct {
	store       Store
	probe       Probe
	concurrency int
	active      atomic.Int64
	completed   atomic.Int64
	writeErrors atomic.Int64
	claimErrors atomic.Int64
	maxLagMS    atomic.Int64
}

func NewRunner(store Store, probe Probe, concurrency int) *Runner {
	return &Runner{store: store, probe: probe, concurrency: max(1, concurrency)}
}

// Run joins all checks before returning so the application can close its database safely.
func (r *Runner) Run(ctx context.Context) {
	var checks sync.WaitGroup
	defer checks.Wait()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	metrics := time.NewTicker(time.Minute)
	defer metrics.Stop()
	cleanup := time.NewTicker(time.Hour)
	defer cleanup.Stop()
	cleanupDone := make(chan struct{}, 1)
	startCleanup := func() {
		select {
		case cleanupDone <- struct{}{}:
		default:
			return
		}
		checks.Add(1)
		go func() {
			defer checks.Done()
			defer func() { <-cleanupDone }()
			r.cleanup(ctx)
		}()
	}
	startCleanup()
	for {
		if ctx.Err() != nil {
			return
		}
		r.dispatch(ctx, &checks)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-cleanup.C:
			startCleanup()
		case <-metrics.C:
			slog.Info("monitoring_metrics", "active", r.active.Load(), "completed", r.completed.Swap(0),
				"write_errors", r.writeErrors.Swap(0), "claim_errors", r.claimErrors.Swap(0),
				"max_schedule_lag_ms", r.maxLagMS.Swap(0))
		}
	}
}

func (r *Runner) dispatch(ctx context.Context, checks *sync.WaitGroup) {
	available := r.concurrency - int(r.active.Load())
	if available <= 0 {
		return
	}
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	monitors, err := r.store.ClaimChecks(dbCtx, available)
	cancel()
	if err != nil {
		r.claimErrors.Add(1)
		slog.Error("claim monitor checks", "error", err)
		return
	}
	for _, monitor := range monitors {
		r.active.Add(1)
		checks.Add(1)
		go func() {
			defer checks.Done()
			defer r.active.Add(-1)
			lag := max(0, time.Since(monitor.NextCheckAt).Milliseconds())
			for old := r.maxLagMS.Load(); lag > old; old = r.maxLagMS.Load() {
				if r.maxLagMS.CompareAndSwap(old, lag) {
					break
				}
			}
			result := r.probe.Check(ctx, monitor)
			// Shutdown is a missing observation, never a fabricated site failure.
			if ctx.Err() != nil {
				return
			}
			dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			accepted, err := r.store.CompleteCheck(dbCtx, monitor, result)
			if err != nil {
				r.writeErrors.Add(1)
				slog.Error("save monitor check", "monitor_id", monitor.ID, "error", err)
				return
			}
			if accepted {
				r.completed.Add(1)
			}
		}()
	}
}

func (r *Runner) cleanup(ctx context.Context) {
	for ctx.Err() == nil {
		batchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		deleted, err := r.store.DeleteExpiredMinutes(batchCtx)
		cancel()
		if err != nil {
			slog.Error("clean monitor history", "error", err)
			return
		}
		if deleted < 5000 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}
