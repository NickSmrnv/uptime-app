package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptime-app/backend/internal/model"
	"github.com/uptime-app/backend/internal/repository"
)

// Stats returns only the active URL history; empty buckets remain null rather than implying uptime.
func (s *MonitorService) Stats(ctx context.Context, token string, id uuid.UUID, period string) (model.MonitorStats, error) {
	userID, err := s.tokens.UserIDFromAccessToken(token)
	if err != nil {
		return model.MonitorStats{}, ErrUnauthorized
	}
	var duration, step time.Duration
	switch period {
	case "1h":
		duration, step = time.Hour, time.Minute
	case "24h":
		duration, step = 24*time.Hour, 5*time.Minute
	case "7d":
		duration, step = 7*24*time.Hour, time.Hour
	case "30d":
		duration, step = 30*24*time.Hour, 6*time.Hour
	default:
		return model.MonitorStats{}, ErrInvalidInput
	}
	to := s.now()
	// Whole-minute storage cannot split a boundary minute; exclude the partial oldest minute.
	from := to.Add(-duration).Truncate(time.Minute)
	if from.Before(to.Add(-duration)) {
		from = from.Add(time.Minute)
	}
	version, minutes, err := s.monitors.ReadBuckets(ctx, repository.StatsQuery{
		MonitorID: id, UserID: userID, From: from, To: to, BucketSeconds: int(step.Seconds()),
	})
	if errors.Is(err, repository.ErrMonitorNotFound) {
		return model.MonitorStats{}, ErrMonitorNotFound
	}
	if err != nil {
		return model.MonitorStats{}, err
	}
	stats := model.MonitorStats{
		Period: period, From: from, To: to, BucketSeconds: int(step.Seconds()),
		HistoryVersion: version, Buckets: []model.MonitorBucket{},
	}
	first := from.Truncate(step)
	for start := first; !start.After(to); start = start.Add(step) {
		stats.Buckets = append(stats.Buckets, model.MonitorBucket{Start: start})
	}
	for _, minute := range minutes {
		index := int(minute.Start.Sub(first) / step)
		if index < 0 || index >= len(stats.Buckets) {
			continue
		}
		stats.Buckets[index].Successes += minute.Successes
		stats.Buckets[index].Failures += minute.Failures
		stats.Successes += minute.Successes
		stats.Failures += minute.Failures
	}
	for i := range stats.Buckets {
		bucket := &stats.Buckets[i]
		bucket.SuccessPercent, bucket.FailurePercent = model.Percentages(bucket.Successes, bucket.Failures)
	}
	stats.SuccessPercent, stats.FailurePercent = model.Percentages(stats.Successes, stats.Failures)
	return stats, nil
}

// MonitorStatus separates missing observations from actual failed probes.
func MonitorStatus(monitor model.Monitor, now time.Time) string {
	if now.After(monitor.NextCheckAt.Add(model.CheckTimeout(monitor.IntervalSeconds) + 5*time.Second)) {
		return "stale"
	}
	if monitor.LastCheckedAt == nil {
		return "pending"
	}
	if monitor.LastError == "" && monitor.LastStatusCode != nil && *monitor.LastStatusCode == 200 {
		return "up"
	}
	return "down"
}
