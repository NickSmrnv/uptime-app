package repository

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func monitoringDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL must point to a disposable database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	return db
}

func createMonitoringFixture(t *testing.T, db *gorm.DB) (*MonitorRepository, model.Monitor) {
	t.Helper()
	now := time.Now().UTC()
	user := model.User{ID: uuid.New(), Email: uuid.NewString() + "@test.invalid", Name: "Test", PasswordHash: "test", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Delete(&user) })
	monitor := model.Monitor{ID: uuid.New(), UserID: user.ID, URL: "https://example.com", IntervalSeconds: 5, CreatedAt: now, UpdatedAt: now}
	repo := NewMonitorRepository(db)
	if ok, err := repo.CreateIfBelowLimit(context.Background(), &monitor, 100); !ok || err != nil {
		t.Fatalf("create: %v", err)
	}
	return repo, monitor
}

func TestMonitoringAtomicCompletionAndHistory(t *testing.T) {
	db := monitoringDB(t)
	repo, monitor := createMonitoringFixture(t, db)
	ctx := context.Background()
	claim := func() model.Monitor {
		t.Helper()
		checks, err := repo.ClaimChecks(ctx, 1)
		if err != nil || len(checks) != 1 {
			t.Fatalf("claim: %v %+v", err, checks)
		}
		return checks[0]
	}
	check := claim()
	code := 200
	result := model.CheckResult{CheckedAt: time.Now().UTC(), StatusCode: &code, DurationMS: 15}
	if ok, err := repo.CompleteCheck(ctx, check, result); !ok || err != nil {
		t.Fatalf("complete: %t %v", ok, err)
	}
	if ok, err := repo.CompleteCheck(ctx, check, result); ok || err != nil {
		t.Fatalf("duplicate: %t %v", ok, err)
	}
	var saved model.Monitor
	db.First(&saved, "id = ?", monitor.ID)
	if !saved.NextCheckAt.After(result.CheckedAt) || saved.LastCheckedAt == nil {
		t.Fatal("schedule/status not updated")
	}
	query := StatsQuery{MonitorID: monitor.ID, UserID: monitor.UserID, From: time.Now().Add(-time.Hour), To: time.Now()}
	_, minutes, err := repo.ReadBuckets(ctx, query)
	if err != nil || len(minutes) != 1 || minutes[0].Successes != 1 {
		t.Fatalf("minutes: %+v %v", minutes, err)
	}
	query.UserID = uuid.New()
	if _, _, err := repo.ReadBuckets(ctx, query); !errors.Is(err, ErrMonitorNotFound) {
		t.Fatal("foreign history exposed")
	}
	query.UserID = monitor.UserID
	// Interval edits retain history, but fence in-flight attempts and restart immediately.
	db.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("next_check_at", time.Now().Add(-time.Second))
	old := claim()
	monitor.IntervalSeconds = 60
	updated, err := repo.UpdateByIDAndUserID(ctx, &monitor)
	if err != nil || updated.HistoryVersion != 1 || updated.LastCheckedAt != nil {
		t.Fatalf("interval edit: %+v %v", updated, err)
	}
	if ok, err := repo.CompleteCheck(ctx, old, result); ok || err != nil {
		t.Fatal("old config accepted")
	}
	check = claim()
	code = 503
	result.CheckedAt = time.Now().UTC()
	if ok, err := repo.CompleteCheck(ctx, check, result); !ok || err != nil {
		t.Fatal(err)
	}
	query.To = time.Now()
	_, minutes, err = repo.ReadBuckets(ctx, query)
	var successes, failures int64
	for _, minute := range minutes {
		successes += minute.Successes
		failures += minute.Failures
	}
	if err != nil || successes != 1 || failures != 1 {
		t.Fatalf("aggregates: %+v %v", minutes, err)
	}
	monitor.URL = "https://changed.example.com"
	updated, err = repo.UpdateByIDAndUserID(ctx, &monitor)
	if err != nil || updated.HistoryVersion != 2 {
		t.Fatal("URL did not create new history")
	}
	_, minutes, err = repo.ReadBuckets(ctx, query)
	if err != nil || len(minutes) != 0 {
		t.Fatal("old URL history leaked")
	}
	check = claim()
	if err := repo.DeleteByIDAndUserID(ctx, monitor.ID, monitor.UserID); err != nil {
		t.Fatal(err)
	}
	if ok, err := repo.CompleteCheck(ctx, check, result); ok || err != nil {
		t.Fatal("deleted result accepted")
	}
	var count int64
	db.Table("monitor_minutes").Where("monitor_id = ?", monitor.ID).Count(&count)
	if count != 0 {
		t.Fatal("history was not cascaded")
	}
}

func TestMonitoringLeasesAndRetention(t *testing.T) {
	db := monitoringDB(t)
	repo, monitor := createMonitoringFixture(t, db)
	ctx := context.Background()
	var group sync.WaitGroup
	claimed := make(chan []model.Monitor, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			checks, err := repo.ClaimChecks(ctx, 1)
			if err != nil {
				t.Error(err)
			}
			claimed <- checks
		}()
	}
	group.Wait()
	close(claimed)
	all := []model.Monitor{}
	for checks := range claimed {
		all = append(all, checks...)
	}
	if len(all) != 1 {
		t.Fatalf("concurrent claims: %d", len(all))
	}
	db.Model(&model.Monitor{}).Where("id = ?", monitor.ID).Update("lease_until", time.Now().Add(-time.Second))
	newClaims, err := repo.ClaimChecks(ctx, 1)
	if err != nil || len(newClaims) != 1 || *newClaims[0].AttemptID == *all[0].AttemptID {
		t.Fatal("restart did not reclaim lease")
	}
	code := 200
	result := model.CheckResult{CheckedAt: time.Now().UTC(), StatusCode: &code}
	if ok, err := repo.CompleteCheck(ctx, all[0], result); ok || err != nil {
		t.Fatal("expired attempt accepted")
	}
	if ok, err := repo.CompleteCheck(ctx, newClaims[0], result); !ok || err != nil {
		t.Fatal("reclaimed attempt failed", err)
	}
	if err := db.Exec(`INSERT INTO monitor_minutes VALUES (?, 1, now()-interval '31 days', 1, 0)`, monitor.ID).Error; err != nil {
		t.Fatal(err)
	}
	deleted, err := repo.DeleteExpiredMinutes(ctx)
	if err != nil || deleted != 1 {
		t.Fatalf("retention: %d %v", deleted, err)
	}
	var count int64
	db.Table("monitor_minutes").Where("monitor_id = ?", monitor.ID).Count(&count)
	if count != 1 {
		t.Fatal("retention removed recent data")
	}
}

func TestMonitoringUpgradeSchedulesExistingRows(t *testing.T) {
	db := monitoringDB(t)
	rollback := errors.New("rollback isolated upgrade schema")
	err := db.Transaction(func(tx *gorm.DB) error {
		schema := "upgrade_" + uuid.New().String()
		if err := tx.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`SET LOCAL search_path TO "` + schema + `"`).Error; err != nil {
			return err
		}
		for _, name := range []string{"000001_initial_schema.sql", "000002_add_test_table.sql"} {
			sql, err := migrations.FS.ReadFile(name)
			if err != nil {
				return err
			}
			if err := tx.Exec(string(sql)).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec(`CREATE TABLE schema_migrations (version bigint PRIMARY KEY, name text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now());
            INSERT INTO schema_migrations (version, name) VALUES (1, 'initial_schema'), (2, 'add_test_table')`).Error; err != nil {
			return err
		}
		userID, monitorID := uuid.New(), uuid.New()
		if err := tx.Exec(`INSERT INTO users (id, name, email, password_hash, created_at, updated_at)
            VALUES (?, 'Upgrade', 'upgrade@test.invalid', 'test', now(), now())`, userID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO monitors (id, user_id, url, interval_seconds, created_at, updated_at)
            VALUES (?, ?, 'https://example.com', 60, now(), now())`, monitorID, userID).Error; err != nil {
			return err
		}
		if err := Migrate(context.Background(), tx); err != nil {
			return err
		}
		if err := Migrate(context.Background(), tx); err != nil {
			return err
		}
		checks, err := NewMonitorRepository(tx).ClaimChecks(context.Background(), 10)
		if err != nil {
			return err
		}
		if len(checks) != 1 || checks[0].ID != monitorID || checks[0].HistoryVersion != 1 {
			t.Fatalf("upgraded monitor was not scheduled: %+v", checks)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("upgrade failed: %v", err)
	}
}
