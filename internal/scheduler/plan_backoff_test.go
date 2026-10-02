package scheduler

import (
	"testing"
	"time"
)

func TestBackoffInterval(t *testing.T) {
	tests := []struct {
		failures int
		want     time.Duration
	}{
		{failures: 1, want: time.Minute},
		{failures: 2, want: 2 * time.Minute},
		{failures: 3, want: 4 * time.Minute},
		{failures: 6, want: 32 * time.Minute},
		{failures: 7, want: time.Hour},
		{failures: 100, want: time.Hour},
	}

	for _, tt := range tests {
		got := backoffInterval(tt.failures)
		if got != tt.want {
			t.Errorf("backoffInterval(%d) = %v, want %v", tt.failures, got, tt.want)
		}
	}
}
