package messages

import (
	"strings"
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

func TestTemplateTypeForAudienceUsesOfficialTypes(t *testing.T) {
	tests := map[domain.MessageAudience]domain.MessageTemplateType{
		domain.MessageAudienceAlmostThere: domain.MessageTemplateAlmostThere,
		domain.MessageAudienceNearGoal:    domain.MessageTemplateAlmostThere,
		domain.MessageAudienceAchieved:    domain.MessageTemplateGoalReached,
		domain.MessageAudienceInactive:    domain.MessageTemplateWeMissYou,
	}

	for audience, want := range tests {
		if got := templateTypeForAudience(audience); got != want {
			t.Fatalf("templateTypeForAudience(%s) = %s, want %s", audience, got, want)
		}
	}
}

func TestRenderOfficialWhatsappTemplate(t *testing.T) {
	body, err := domain.RenderOfficialWhatsappTemplate(domain.MessageTemplateGoalReached, map[string]string{
		"student_name":     "Luiz",
		"campaign_name":    "Desafio Julho",
		"box_name":         "EngageFit Box",
		"current_checkins": "12",
		"reward_name":      "Camiseta",
	})
	if err != nil {
		t.Fatalf("RenderOfficialWhatsappTemplate() error = %v", err)
	}
	if !strings.Contains(body, "Parabéns, Luiz!") || !strings.Contains(body, "Camiseta") {
		t.Fatalf("rendered body did not include expected variables: %s", body)
	}
}

func TestApplyEffectiveTemplateMetadataUsesPlatformContentSID(t *testing.T) {
	template := domain.MessageTemplate{
		TemplateType:   domain.MessageTemplateGoalReached,
		ContentSID:     "HX-tenant",
		ApprovalStatus: domain.MessageTemplateRejected,
	}
	settings := domain.WhatsappSettings{
		ConnectionMode: domain.WhatsappConnectionPlatform,
		ContentSIDs: map[domain.MessageTemplateType]string{
			domain.MessageTemplateGoalReached: "HX-platform",
		},
	}

	applyEffectiveTemplateMetadata(&template, settings)

	if template.ContentSID != "HX-platform" || template.ApprovalStatus != domain.MessageTemplateApproved {
		t.Fatalf("effective template = %#v, want approved platform metadata", template)
	}
}

func TestApplyEffectiveTemplateMetadataKeepsDedicatedContentSID(t *testing.T) {
	template := domain.MessageTemplate{
		TemplateType:   domain.MessageTemplateGoalReached,
		ContentSID:     "HX-dedicated",
		ApprovalStatus: domain.MessageTemplateApproved,
	}

	applyEffectiveTemplateMetadata(&template, domain.WhatsappSettings{ConnectionMode: domain.WhatsappConnectionDedicated})

	if template.ContentSID != "HX-dedicated" || template.ApprovalStatus != domain.MessageTemplateApproved {
		t.Fatalf("effective template = %#v, want dedicated metadata unchanged", template)
	}
}
