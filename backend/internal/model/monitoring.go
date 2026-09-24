package model

import "time"

// CheckTimeout leaves room before the next scheduled check even for five-second intervals.
func CheckTimeout(intervalSeconds int) time.Duration {
	return min(10*time.Second, time.Duration(intervalSeconds)*time.Second*4/5)
}

// CheckResult contains metadata only; response bodies are never persisted.
type CheckResult struct {
	CheckedAt  time.Time
	StatusCode *int
	Error      string
	DurationMS int64
}

type MonitorBucket struct {
	Start          time.Time `json:"start"`
	Successes      int64     `json:"successes"`
	Failures       int64     `json:"failures"`
	SuccessPercent *float64  `json:"successPercent"`
	FailurePercent *float64  `json:"failurePercent"`
}

type MonitorStats struct {
	Period         string          `json:"period"`
	From           time.Time       `json:"from"`
	To             time.Time       `json:"to"`
	BucketSeconds  int             `json:"bucketSeconds"`
	HistoryVersion int64           `json:"historyVersion"`
	Successes      int64           `json:"successes"`
	Failures       int64           `json:"failures"`
	SuccessPercent *float64        `json:"successPercent"`
	FailurePercent *float64        `json:"failurePercent"`
	Buckets        []MonitorBucket `json:"buckets"`
}

// Percentages uses counts rather than averages so sparse buckets carry the correct weight.
func Percentages(successes, failures int64) (*float64, *float64) {
	if successes+failures == 0 {
		return nil, nil
	}
	success := float64(successes) * 100 / float64(successes+failures)
	failure := 100 - success
	return &success, &failure
}
