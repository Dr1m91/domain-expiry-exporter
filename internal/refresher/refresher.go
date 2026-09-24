package refresher

import (
	"context"
	"sync"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/probe"
	"github.com/dr1m91/domain-expiry-exporter/internal/safeconfig"
	"github.com/rs/zerolog/log"
)

type Refresher struct {
	ticker      *time.Ticker
	client      probe.Client
	domains     []safeconfig.Domain
	timeout     time.Duration
	concurrency int
}

func New(interval time.Duration, client probe.Client, timeout time.Duration, domains ...safeconfig.Domain) Refresher {
	ticker := time.NewTicker(interval)
	return Refresher{
		ticker:      ticker,
		client:      client,
		domains:     domains,
		timeout:     timeout,
		concurrency: 20,
	}
}

func (r Refresher) Stop() {
	r.ticker.Stop()
}

func (r Refresher) Run(ctx context.Context) {
	log.Info().Msg("run refresher")
	r.Refresh(ctx)

	for {
		select {
		case <-r.ticker.C:
			r.Refresh(ctx)
		case <-ctx.Done():
			log.Info().Msg("refresher is finished")
			return
		}
	}
}

func (r Refresher) Refresh(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	sem := make(chan struct{}, r.concurrency)
	var wg sync.WaitGroup

	for _, domain := range r.domains {
		domain := domain
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := r.client.ExpireTime(ctx, domain.Name, domain.Host); err != nil {
				log.Error().Err(err).Msgf("failed to get expire time for %s", domain)
			}
		}()
	}

	wg.Wait()
	log.Debug().Msg("refresh is done")
}
