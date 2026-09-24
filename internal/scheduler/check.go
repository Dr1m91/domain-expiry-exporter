package scheduler

import (
	"context"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
	"github.com/dr1m91/domain-expiry-exporter/internal/probe"
	"github.com/rs/zerolog/log"
)

func checkDomain(ctx context.Context, store domain.Store, client probe.Client, entry domain.Entry, now time.Time) {
	expireTime, err := client.ExpireTime(ctx, entry.Domain, "")
	entry.LastAttemptAt = now

	if err != nil {
		entry.ConsecutiveFailures++
		log.Error().Err(err).Msgf("failed to check %s", entry.Domain)
	} else {
		entry.ExpireTime = expireTime
		entry.LastSuccessAt = now
		entry.ConsecutiveFailures = 0
	}

	entry = planNextCheck(entry, now)

	if err := store.Set(entry); err != nil {
		log.Error().Err(err).Msgf("failed to save entry for %s", entry.Domain)
	}
}
