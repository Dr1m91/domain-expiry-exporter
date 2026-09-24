package domain

import "time"

// Entry holds everything the exporter knows about a single monitored domain.
type Entry struct {
	Domain string

	ExpireTime          time.Time
	LastSuccessAt       time.Time
	LastAttemptAt       time.Time
	ConsecutiveFailures int

	LastRequestedAt time.Time
	NextCheckAt     time.Time
}

// IsStale reports whether domain hasn't been scraped in longer than maxAge.
func (e Entry) IsStale(now time.Time, maxAge time.Duration) bool {
	return now.Sub(e.LastRequestedAt) > maxAge
}

// HasEverSucceeded reports whether we have any expiry data for this domain.
func (e Entry) HasEverSucceeded() bool {
	return !e.LastSuccessAt.IsZero()
}
