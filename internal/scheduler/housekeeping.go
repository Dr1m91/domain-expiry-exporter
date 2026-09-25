package scheduler

import "time"

func (s *Scheduler) evictStale(now time.Time) {
	entries, err := s.Store.List()
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsStale(now, s.StaleAfter) {
			_ = s.Store.Delete(entry.Domain)
		}
	}
}
