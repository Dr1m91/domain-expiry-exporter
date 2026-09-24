package scheduler

import "time"

func nextCheckInterval(daysUntilExpiry int) time.Duration {
	switch {
	case daysUntilExpiry < 7:
		return time.Hour
	case daysUntilExpiry < 30:
		return 12 * time.Hour
	default:
		return 24 * time.Hour
	}
}
