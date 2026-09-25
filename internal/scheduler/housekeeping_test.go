package scheduler

import (
	"testing"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
)

func TestScheduler_evictStale(t *testing.T) {
	store := domain.NewInMemoryStore()
	now := time.Now()

	if err := store.Set(domain.Entry{Domain: "fresh.com", LastRequestedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(domain.Entry{Domain: "stale.com", LastRequestedAt: now.Add(-25 * time.Hour)}); err != nil {
		t.Fatal(err)
	}

	s := &Scheduler{Store: store, StaleAfter: 24 * time.Hour}
	s.evictStale(now)

	if _, ok, _ := store.Get("fresh.com"); !ok {
		t.Error("expected fresh.com to survive")
	}
	if _, ok, _ := store.Get("stale.com"); ok {
		t.Error("expected stale.com to be evicted")
	}
}
