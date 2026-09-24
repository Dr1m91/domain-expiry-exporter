package scheduler

import (
	"testing"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
)

func TestPlanNextCheck(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	entry := domain.Entry{
		Domain:     "example.com",
		ExpireTime: now.Add(5 * 24 * time.Hour), // 5 days left
	}

	got := planNextCheck(entry, now)

	wantNextCheck := now.Add(time.Hour)
	if !got.NextCheckAt.Equal(wantNextCheck) {
		t.Errorf("NextCheckAt = %v, want %v", got.NextCheckAt, wantNextCheck)
	}
}
