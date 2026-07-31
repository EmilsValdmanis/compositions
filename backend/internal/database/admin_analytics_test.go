package database

import (
	"context"
	"testing"
	"time"
)

func TestGetAdminAnalyticsRejectsUnconfiguredStore(t *testing.T) {
	dateRange := AdminAnalyticsRange{
		From: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC),
	}
	dateRange.PreviousTo = dateRange.From
	dateRange.PreviousFrom = dateRange.From.AddDate(0, 0, -30)
	if _, err := (*UserStore)(nil).GetAdminAnalytics(context.Background(), dateRange); err == nil {
		t.Fatal("GetAdminAnalytics() error = nil; want unconfigured store error")
	}
}
