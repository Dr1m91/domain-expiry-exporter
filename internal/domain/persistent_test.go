package domain

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type fakeBackend struct {
	mu       sync.Mutex
	stored   map[string]Entry
	failSave bool
}

func newFakeBackend(initial ...Entry) *fakeBackend {
	b := &fakeBackend{stored: make(map[string]Entry)}
	for _, e := range initial {
		b.stored[e.Domain] = e
	}
	return b
}

func (b *fakeBackend) LoadAll(context.Context) ([]Entry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Entry, 0, len(b.stored))
	for _, e := range b.stored {
		out = append(out, e)
	}
	return out, nil
}

func (b *fakeBackend) Save(_ context.Context, entries []Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failSave {
		return errors.New("backend down")
	}
	for _, e := range entries {
		b.stored[e.Domain] = e
	}
	return nil
}

func (b *fakeBackend) Remove(_ context.Context, domains []string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, d := range domains {
		delete(b.stored, d)
	}
	return nil
}

func (b *fakeBackend) has(domain string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.stored[domain]
	return ok
}

func TestPersistentStore_LoadsFromBackend(t *testing.T) {
	b := newFakeBackend(Entry{Domain: "a.com", ConsecutiveFailures: 2})
	s := NewPersistentStore(context.Background(), b)

	got, ok, _ := s.Get("a.com")
	if !ok || got.ConsecutiveFailures != 2 {
		t.Fatalf("expected a.com loaded from backend, got %+v ok=%v", got, ok)
	}
}

func TestPersistentStore_FlushesChanges(t *testing.T) {
	ctx := context.Background()
	b := newFakeBackend()
	s := NewPersistentStore(ctx, b)

	if err := s.Set(Entry{Domain: "a.com"}); err != nil {
		t.Fatal(err)
	}
	if b.has("a.com") {
		t.Fatal("Set must not reach the backend before flush")
	}

	s.flush(ctx)
	if !b.has("a.com") {
		t.Fatal("expected a.com in backend after flush")
	}

	if err := s.Delete("a.com"); err != nil {
		t.Fatal(err)
	}
	s.flush(ctx)
	if b.has("a.com") {
		t.Fatal("expected a.com removed from backend after flush")
	}
}

func TestPersistentStore_RetriesAfterFailure(t *testing.T) {
	ctx := context.Background()
	b := newFakeBackend()
	s := NewPersistentStore(ctx, b)

	if err := s.Set(Entry{Domain: "a.com"}); err != nil {
		t.Fatal(err)
	}

	b.failSave = true
	s.flush(ctx)
	if b.has("a.com") {
		t.Fatal("save should have failed")
	}

	b.failSave = false
	s.flush(ctx)
	if !b.has("a.com") {
		t.Fatal("expected retry to succeed on the next flush")
	}
}
