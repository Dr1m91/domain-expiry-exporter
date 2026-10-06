package domain

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestBackend(t *testing.T) (*RedisBackend, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisBackend(client, "test:entries"), mr
}

func TestRedisBackend_RoundTrip(t *testing.T) {
	ctx := context.Background()
	b, _ := newTestBackend(t)

	want := Entry{
		Domain:              "a.com",
		ExpireTime:          time.Date(2028, 1, 2, 3, 4, 5, 0, time.UTC),
		ConsecutiveFailures: 3,
		Static:              true,
	}
	if err := b.Save(ctx, []Entry{want, {Domain: "b.com"}}); err != nil {
		t.Fatal(err)
	}

	got, err := b.LoadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	for _, e := range got {
		if e.Domain == "a.com" && (!e.ExpireTime.Equal(want.ExpireTime) || e.ConsecutiveFailures != 3 || !e.Static) {
			t.Fatalf("a.com did not survive the round trip: %+v", e)
		}
	}

	if err := b.Remove(ctx, []string{"a.com"}); err != nil {
		t.Fatal(err)
	}
	got, _ = b.LoadAll(ctx)
	if len(got) != 1 || got[0].Domain != "b.com" {
		t.Fatalf("expected only b.com left, got %+v", got)
	}
}

func TestRedisBackend_SkipsCorruptEntries(t *testing.T) {
	ctx := context.Background()
	b, mr := newTestBackend(t)

	mr.HSet("test:entries", "broken.com", "not-json")
	if err := b.Save(ctx, []Entry{{Domain: "ok.com"}}); err != nil {
		t.Fatal(err)
	}

	got, err := b.LoadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Domain != "ok.com" {
		t.Fatalf("expected only ok.com, got %+v", got)
	}
}

func TestPersistentStore_SurvivesRestartViaRedis(t *testing.T) {
	ctx := context.Background()
	b, _ := newTestBackend(t)

	first := NewPersistentStore(ctx, b)
	if err := first.Set(Entry{Domain: "a.com", ConsecutiveFailures: 5}); err != nil {
		t.Fatal(err)
	}
	first.flush(ctx)

	second := NewPersistentStore(ctx, b)
	got, ok, _ := second.Get("a.com")
	if !ok || got.ConsecutiveFailures != 5 {
		t.Fatalf("expected a.com restored after restart, got %+v ok=%v", got, ok)
	}
}
