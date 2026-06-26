package messages

import (
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

func TestIsAlmostThere(t *testing.T) {
	tests := []struct {
		name     string
		progress domain.CampaignProgress
		daysLeft int
		want     bool
	}{
		{
			name:     "one checkin remaining with enough days",
			progress: domain.NewCampaignProgress("campaign", "student", 9, 10),
			daysLeft: 3,
			want:     true,
		},
		{
			name:     "two checkins remaining with enough days",
			progress: domain.NewCampaignProgress("campaign", "student", 8, 10),
			daysLeft: 2,
			want:     true,
		},
		{
			name:     "below eighty percent",
			progress: domain.NewCampaignProgress("campaign", "student", 7, 10),
			daysLeft: 5,
			want:     false,
		},
		{
			name:     "not enough days left",
			progress: domain.NewCampaignProgress("campaign", "student", 8, 10),
			daysLeft: 1,
			want:     false,
		},
		{
			name:     "already achieved",
			progress: domain.NewCampaignProgress("campaign", "student", 10, 10),
			daysLeft: 5,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAlmostThere(tt.progress, tt.daysLeft); got != tt.want {
				t.Fatalf("isAlmostThere() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCampaignDaysLeft(t *testing.T) {
	campaign := domain.Campaign{
		EndDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	}

	got := campaignDaysLeft(campaign, time.Date(2026, 6, 26, 15, 0, 0, 0, time.FixedZone("BRT", -3*60*60)))
	if got != 5 {
		t.Fatalf("campaignDaysLeft() = %d, want 5", got)
	}
}
