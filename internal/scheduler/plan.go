package scheduler

import (
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
)

// planNextCheck returns a copy of entry with NextCheckAt updated based on
// how close ExpireTime is, relative to now.
//
// TODO: factor in entry.ConsecutiveFailures for exponential backoff on
// repeated probe failures, instead of relying solely on ExpireTime.
func planNextCheck(entry domain.Entry, now time.Time) domain.Entry {
	daysUntilExpiry := int(entry.ExpireTime.Sub(now).Hours() / 24)
	entry.NextCheckAt = now.Add(nextCheckInterval(daysUntilExpiry))
	return entry
}
