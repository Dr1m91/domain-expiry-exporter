package scheduler

import (
	"testing"
	"time"
)

func TestNextCheckInterval(t *testing.T) {
	tests := []struct {
		days int
		want time.Duration
	}{
		{days: 0, want: time.Hour},
		{days: 6, want: time.Hour},
		{days: 7, want: 12 * time.Hour},
		{days: 29, want: 12 * time.Hour},
		{days: 30, want: 24 * time.Hour},
		{days: 365, want: 24 * time.Hour},
	}

	for _, tt := range tests {
		got := nextCheckInterval(tt.days)
		if got != tt.want {
			t.Errorf("nextCheckInterval(%d) = %v, want %v", tt.days, got, tt.want)
		}
	}
}
