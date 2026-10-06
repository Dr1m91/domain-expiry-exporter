package domain

import (
	"testing"
	"time"
)

func TestEntry_IsStale(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)

	if !(Entry{LastRequestedAt: old}).IsStale(now, 24*time.Hour) {
		t.Error("expected old entry to be stale")
	}
	if (Entry{LastRequestedAt: old, Static: true}).IsStale(now, 24*time.Hour) {
		t.Error("static entry must never be stale")
	}
}
