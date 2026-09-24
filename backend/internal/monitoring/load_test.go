package monitoring

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type loadProbe struct {
	delay time.Duration
	calls atomic.Int64
}

func (p *loadProbe) Check(ctx context.Context, _ model.Monitor) model.CheckResult {
	p.calls.Add(1)
	select {
	case <-ctx.Done():
	case <-time.After(p.delay):
	}
	code := 200
	return model.CheckResult{CheckedAt: time.Now().UTC(), StatusCode: &code, DurationMS: p.delay.Milliseconds()}
}

// TestMonitorLoad uses real scheduling/storage and controlled probe latency, never external websites.
func TestMonitorLoad(t *testing.T) {
	if os.Getenv("MONITOR_LOAD_TEST") != "1" {
		t.Skip("set MONITOR_LOAD_TEST=1 and TEST_DATABASE_URL for the load scenario")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := repository.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, delay := range []time.Duration{20 * time.Millisecond, 3 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			user := model.User{ID: uuid.New(), Email: uuid.NewString() + "@load.invalid", PasswordHash: "test"}
			if err := db.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			defer db.Delete(&user)
			repo := repository.NewMonitorRepository(db)
			for range 100 {
				monitor := model.Monitor{ID: uuid.New(), UserID: user.ID, URL: "https://example.com", IntervalSeconds: 5}
				if ok, err := repo.CreateIfBelowLimit(context.Background(), &monitor, 100); err != nil || !ok {
					t.Fatal(err)
				}
			}
			probe := &loadProbe{delay: delay}
			runner := NewRunner(repo, probe, 20)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { defer close(done); runner.Run(ctx) }()
			defer func() { cancel(); <-done }()
			deadline := time.Now().Add(35 * time.Second)
			var checked int64
			for time.Now().Before(deadline) {
				err := db.Model(&model.Monitor{}).Where("user_id = ? AND last_checked_at IS NOT NULL", user.ID).Count(&checked).Error
				if err != nil {
					t.Fatal(err)
				}
				if checked == 100 {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if checked != 100 || runner.writeErrors.Load() != 0 || runner.claimErrors.Load() != 0 {
				t.Fatalf("checked=%d writeErrors=%d claimErrors=%d", checked, runner.writeErrors.Load(), runner.claimErrors.Load())
			}
			t.Logf("100 monitors, interval=5s, latency=%s, concurrency=20: completed=%d, max_schedule_lag_ms=%d",
				delay, runner.completed.Load(), runner.maxLagMS.Load())
		})
	}
}
