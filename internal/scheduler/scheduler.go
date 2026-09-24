package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
	"github.com/dr1m91/domain-expiry-exporter/internal/probe"
	"github.com/rs/zerolog/log"
)

// Scheduler periodically scans the store and re-checks domains whose
// NextCheckAt has passed, with bounded concurrency.
type Scheduler struct {
	Store        domain.Store
	Client       probe.Client
	Concurrency  int
	ScanInterval time.Duration
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.ScanInterval)
	defer ticker.Stop()

	s.scan(ctx)
	for {
		select {
		case <-ticker.C:
			s.scan(ctx)
		case <-ctx.Done():
			log.Info().Msg("scheduler is finished")
			return
		}
	}
}

func (s *Scheduler) scan(ctx context.Context) {
	entries, err := s.Store.List()
	if err != nil {
		log.Error().Err(err).Msg("failed to list domains")
		return
	}

	now := time.Now()
	sem := make(chan struct{}, s.Concurrency)
	var wg sync.WaitGroup

	for _, entry := range entries {
		if entry.NextCheckAt.After(now) {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			checkDomain(ctx, s.Store, s.Client, entry, time.Now())
		}()
	}

	wg.Wait()
}
