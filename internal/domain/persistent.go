package domain

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Backend persists entries outside of process memory.
type Backend interface {
	LoadAll(ctx context.Context) ([]Entry, error)
	Save(ctx context.Context, entries []Entry) error
	Remove(ctx context.Context, domains []string) error
}

// PersistentStore keeps entries in memory and flushes changes to a Backend
// in the background, so reads and writes never wait on the network.
type PersistentStore struct {
	mem     *InMemoryStore
	backend Backend

	mu      sync.Mutex
	dirty   map[string]struct{}
	removed map[string]struct{}
}

func NewPersistentStore(ctx context.Context, backend Backend) *PersistentStore {
	s := &PersistentStore{
		mem:     NewInMemoryStore(),
		backend: backend,
		dirty:   make(map[string]struct{}),
		removed: make(map[string]struct{}),
	}

	entries, err := backend.LoadAll(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load entries from backend, starting empty")
		return s
	}
	for _, e := range entries {
		_ = s.mem.Set(e)
	}
	log.Info().Msgf("loaded %d entries from backend", len(entries))
	return s
}

func (s *PersistentStore) Get(domain string) (Entry, bool, error) {
	return s.mem.Get(domain)
}

func (s *PersistentStore) List() ([]Entry, error) {
	return s.mem.List()
}

func (s *PersistentStore) Set(entry Entry) error {
	if err := s.mem.Set(entry); err != nil {
		return err
	}
	s.mu.Lock()
	s.dirty[entry.Domain] = struct{}{}
	delete(s.removed, entry.Domain)
	s.mu.Unlock()
	return nil
}

func (s *PersistentStore) Delete(domain string) error {
	if err := s.mem.Delete(domain); err != nil {
		return err
	}
	s.mu.Lock()
	s.removed[domain] = struct{}{}
	delete(s.dirty, domain)
	s.mu.Unlock()
	return nil
}

// Run flushes pending changes every interval and once more on shutdown.
func (s *PersistentStore) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flush(ctx)
		case <-ctx.Done():
			final, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			s.flush(final)
			cancel()
			return
		}
	}
}

func (s *PersistentStore) flush(ctx context.Context) {
	s.mu.Lock()
	dirty, removed := s.dirty, s.removed
	s.dirty = make(map[string]struct{})
	s.removed = make(map[string]struct{})
	s.mu.Unlock()

	entries := make([]Entry, 0, len(dirty))
	for d := range dirty {
		if e, ok, _ := s.mem.Get(d); ok {
			entries = append(entries, e)
		}
	}
	gone := make([]string, 0, len(removed))
	for d := range removed {
		gone = append(gone, d)
	}

	if len(entries) > 0 {
		if err := s.backend.Save(ctx, entries); err != nil {
			log.Warn().Err(err).Msg("failed to save entries to backend, will retry")
			s.requeueDirty(dirty)
		}
	}
	if len(gone) > 0 {
		if err := s.backend.Remove(ctx, gone); err != nil {
			log.Warn().Err(err).Msg("failed to remove entries from backend, will retry")
			s.requeueRemoved(gone)
		}
	}
}

func (s *PersistentStore) requeueDirty(domains map[string]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for d := range domains {
		if _, gone := s.removed[d]; !gone {
			s.dirty[d] = struct{}{}
		}
	}
}

func (s *PersistentStore) requeueRemoved(domains []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range domains {
		if _, back := s.dirty[d]; !back {
			s.removed[d] = struct{}{}
		}
	}
}
