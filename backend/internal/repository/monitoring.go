package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"gorm.io/gorm"
)

// ClaimChecks commits short leases before any network I/O. SKIP LOCKED allows multiple API processes.
func (r *MonitorRepository) ClaimChecks(ctx context.Context, limit int) ([]model.Monitor, error) {
	monitors := []model.Monitor{}
	if limit <= 0 {
		return monitors, nil
	}
	err := r.db.WithContext(ctx).Raw(`
        WITH due AS (
            SELECT id FROM monitors
            WHERE next_check_at <= now() AND (lease_until IS NULL OR lease_until <= now())
            ORDER BY next_check_at, id LIMIT ? FOR UPDATE SKIP LOCKED
        )
        UPDATE monitors m SET attempt_id = gen_random_uuid(), lease_until = now() + interval '30 seconds'
        FROM due WHERE m.id = due.id RETURNING m.*`, limit).Scan(&monitors).Error
	return monitors, err
}

// CompleteCheck fences expired/replaced attempts and atomically updates both status and aggregates.
func (r *MonitorRepository) CompleteCheck(ctx context.Context, check model.Monitor, result model.CheckResult) (bool, error) {
	var accepted bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Monitor
		updated := tx.Raw(`
            UPDATE monitors SET
                next_check_at = next_check_at +
                    (floor(greatest(0, extract(epoch FROM (?::timestamptz - next_check_at))) / interval_seconds) + 1)
                    * interval_seconds * interval '1 second',
                attempt_id = NULL, lease_until = NULL,
                last_checked_at = ?, last_status_code = ?, last_error = ?, last_duration_ms = ?
            WHERE id = ? AND config_version = ? AND attempt_id = ? AND lease_until > now()
            RETURNING *`,
			result.CheckedAt, result.CheckedAt, result.StatusCode, result.Error, result.DurationMS,
			check.ID, check.ConfigVersion, check.AttemptID,
		).Scan(&current)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return nil
		}
		var successes, failures int
		if result.StatusCode != nil && *result.StatusCode == 200 && result.Error == "" {
			successes = 1
		} else {
			failures = 1
		}
		if err := tx.Exec(`
            INSERT INTO monitor_minutes (monitor_id, history_version, minute, successes, failures)
            VALUES (?, ?, ?, ?, ?)
            ON CONFLICT (monitor_id, history_version, minute) DO UPDATE SET
                successes = monitor_minutes.successes + EXCLUDED.successes,
                failures = monitor_minutes.failures + EXCLUDED.failures`,
			check.ID, current.HistoryVersion, result.CheckedAt.UTC().Truncate(time.Minute), successes, failures,
		).Error; err != nil {
			return err
		}
		accepted = true
		return nil
	})
	return accepted && err == nil, err
}

// DeleteExpiredMinutes bounds each transaction so retention does not block normal check writes.
func (r *MonitorRepository) DeleteExpiredMinutes(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
        DELETE FROM monitor_minutes WHERE ctid IN (
            SELECT ctid FROM monitor_minutes WHERE minute < now() - interval '30 days'
            ORDER BY minute LIMIT 5000 FOR UPDATE SKIP LOCKED
        )`)
	return result.RowsAffected, result.Error
}

type StatsQuery struct {
	BucketSeconds int
	MonitorID     uuid.UUID
	UserID        uuid.UUID
	From          time.Time
	To            time.Time
}

// ReadBuckets keeps ownership, the current history version and its samples in one database snapshot.
func (r *MonitorRepository) ReadBuckets(ctx context.Context, query StatsQuery) (int64, []model.MonitorBucket, error) {
	type row struct {
		HistoryVersion int64
		Minute         *time.Time
		Successes      int64
		Failures       int64
	}
	rows := []row{}
	err := r.db.WithContext(ctx).Raw(`
        SELECT m.history_version,
            to_timestamp(floor(extract(epoch FROM b.minute) / ?) * ?) AS minute,
            coalesce(sum(b.successes), 0) successes, coalesce(sum(b.failures), 0) failures
        FROM monitors m LEFT JOIN monitor_minutes b
            ON b.monitor_id = m.id AND b.history_version = m.history_version
            AND b.minute >= ? AND b.minute <= ?
            AND b.minute >= now() - interval '30 days'
        WHERE m.id = ? AND m.user_id = ? GROUP BY m.history_version, 2 ORDER BY 2`,
		max(60, query.BucketSeconds), max(60, query.BucketSeconds),
		query.From, query.To, query.MonitorID, query.UserID,
	).Scan(&rows).Error
	if err != nil {
		return 0, nil, err
	}
	if len(rows) == 0 {
		return 0, nil, ErrMonitorNotFound
	}
	buckets := []model.MonitorBucket{}
	for _, row := range rows {
		if row.Minute != nil {
			buckets = append(buckets, model.MonitorBucket{
				Start: *row.Minute, Successes: row.Successes, Failures: row.Failures,
			})
		}
	}
	return rows[0].HistoryVersion, buckets, nil
}
