package scheduler

import (
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
)

func planNextCheck(entry domain.Entry, now time.Time) domain.Entry {
	if entry.ConsecutiveFailures > 0 {
		entry.NextCheckAt = now.Add(backoffInterval(entry.ConsecutiveFailures))
		return entry
	}

	daysUntilExpiry := int(entry.ExpireTime.Sub(now).Hours() / 24)
	entry.NextCheckAt = now.Add(nextCheckInterval(daysUntilExpiry))
	return entry
}

func backoffInterval(failures int) time.Duration {
	const maxBackoff = time.Hour

	if failures > 6 {
		return maxBackoff
	}

	return time.Minute * time.Duration(1<<uint(failures-1))
}
