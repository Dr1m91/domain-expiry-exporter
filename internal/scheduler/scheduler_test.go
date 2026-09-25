package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
)

type fakeClient struct {
	expireTime time.Time
}

func (f fakeClient) ExpireTime(ctx context.Context, domain, host string) (time.Time, error) {
	return f.expireTime, nil
}

func TestScheduler_scan(t *testing.T) {
	store := domain.NewInMemoryStore()
	now := time.Now()

	if err := store.Set(domain.Entry{Domain: "due.com", NextCheckAt: now.Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(domain.Entry{Domain: "not-due.com", NextCheckAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	s := &Scheduler{
		Store:       store,
		Client:      fakeClient{expireTime: now.Add(60 * 24 * time.Hour)},
		Concurrency: 5,
	}
	s.scan(context.Background())

	due, _, _ := store.Get("due.com")
	if due.LastSuccessAt.IsZero() {
		t.Error("expected due.com to be checked")
	}

	notDue, _, _ := store.Get("not-due.com")
	if !notDue.LastSuccessAt.IsZero() {
		t.Error("expected not-due.com to be skipped")
	}
}
