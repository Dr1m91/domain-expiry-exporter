package domain

import "sync"

// InMemoryStore is a Store backed by a plain map, safe for concurrent use.
// Data does not survive a restart.
type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

// NewInMemoryStore returns an empty InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entries: make(map[string]Entry),
	}
}

func (s *InMemoryStore) Get(domain string) (Entry, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.entries[domain]
	return e, ok, nil
}

func (s *InMemoryStore) Set(entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[entry.Domain] = entry
	return nil
}

func (s *InMemoryStore) Delete(domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, domain)
	return nil
}

func (s *InMemoryStore) List() ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	return out, nil
}
