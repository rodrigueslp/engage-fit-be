package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestOperationalReportsRewardsAndRecipients(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	boxID, studentID := uuid.NewString(), uuid.NewString()
	importID, checkinID := uuid.NewString(), uuid.NewString()
	campaignID, progressID := uuid.NewString(), uuid.NewString()
	rewardID, deliveryID := uuid.NewString(), uuid.NewString()
	templateID, messageCampaignID := uuid.NewString(), uuid.NewString()

	fixtures := []any{
		&models.BoxModel{ID: boxID, Name: "operational-flow", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		&models.StudentModel{ID: studentID, BoxID: boxID, Name: "Student Reports", Phone: "+5511999999999", Source: "totalpass", ExternalID: uuid.NewString(), RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
		&models.ImportHistoryModel{ID: importID, BoxID: boxID, Filename: "report.csv", Source: "totalpass", TotalRecords: 1, ImportedAt: now},
		&models.CheckinModel{ID: checkinID, BoxID: boxID, StudentID: studentID, CheckinDate: now, Source: "totalpass", ImportHistoryID: importID, CreatedAt: now},
		&models.CampaignModel{ID: campaignID, BoxID: boxID, Name: "Report Campaign", StartDate: now.AddDate(0, 0, -1), EndDate: now.AddDate(0, 0, 1), Active: true, CreatedAt: now, UpdatedAt: now},
		&models.CampaignProgressModel{ID: progressID, CampaignID: campaignID, StudentID: studentID, CurrentCheckins: 1, TargetCheckins: 1, ProgressPercentage: 100, Achieved: true, UpdatedAt: now},
		&models.RewardModel{ID: rewardID, CampaignID: campaignID, Name: "Report Reward", Quantity: 1},
		&models.RewardDeliveryModel{ID: deliveryID, RewardID: rewardID, StudentID: studentID, Delivered: false},
		&models.MessageTemplateModel{ID: templateID, BoxID: boxID, Name: "Operational Template", Content: "Hello", TemplateType: "manual", Provider: "twilio", ApprovalStatus: "approved", Language: "pt_BR", CreatedAt: now, UpdatedAt: now},
		&models.MessageCampaignModel{ID: messageCampaignID, BoxID: boxID, CampaignID: campaignID, Name: "Operational Message", Audience: "all", TemplateID: templateID, TemplateType: "manual", CreatedAt: now},
	}
	for _, fixture := range fixtures {
		if err := db.Create(fixture).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer db.Delete(&models.BoxModel{}, "id = ?", boxID)

	campaigns := NewCampaignGormRepository(db)
	eligible, err := campaigns.ListEligibleReportRows(ctx, domain.ID(boxID))
	if err != nil {
		t.Fatal(err)
	}
	if len(eligible) != 1 || eligible[0].StudentID != domain.ID(studentID) || eligible[0].RewardName != "Report Reward" || eligible[0].RemainingCheckins != 0 {
		t.Fatalf("unexpected eligible report: %+v", eligible)
	}

	checkins := NewCheckinGormRepository(db)
	frequency, err := checkins.ListMonthlyFrequency(ctx, domain.ID(boxID), domain.TimeRange{Start: now.AddDate(0, 0, -1), End: now.AddDate(0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(frequency) != 1 || frequency[0].StudentID != domain.ID(studentID) || frequency[0].Checkins != 1 {
		t.Fatalf("unexpected frequency report: %+v", frequency)
	}

	rewards := NewRewardGormRepository(db)
	pending, err := rewards.ListPendingDeliveries(ctx, domain.ID(boxID))
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != domain.ID(deliveryID) || pending[0].RewardName != "Report Reward" {
		t.Fatalf("unexpected pending deliveries: %+v", pending)
	}
	if err := rewards.MarkDelivered(ctx, domain.ID(boxID), domain.ID(deliveryID)); err != nil {
		t.Fatal(err)
	}
	deliveries, err := rewards.ListDeliveries(ctx, domain.ID(boxID))
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || !deliveries[0].Delivered || deliveries[0].DeliveredAt == nil {
		t.Fatalf("reward delivery was not persisted: %+v", deliveries)
	}

	messages := NewMessageGormRepository(db)
	recipients := []domain.MessageRecipient{{
		MessageCampaignID: domain.ID(messageCampaignID), StudentID: domain.ID(studentID), Phone: "+5511999999999",
		Status: domain.MessageRecipientPending, CreatedAt: now,
	}}
	if err := messages.SaveRecipients(ctx, recipients); err != nil {
		t.Fatal(err)
	}
	stored, err := messages.ListRecipients(ctx, domain.ID(messageCampaignID))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].ID == "" || stored[0].Status != domain.MessageRecipientPending {
		t.Fatalf("unexpected stored recipients: %+v", stored)
	}
	sentAt := now.Add(time.Minute)
	stored[0].Status = domain.MessageRecipientSent
	stored[0].ProviderMessageSID = "SM-test"
	stored[0].ProviderStatus = "queued"
	stored[0].SentAt = &sentAt
	if err := messages.UpdateRecipient(ctx, stored[0]); err != nil {
		t.Fatal(err)
	}
	updated, err := messages.ListRecipients(ctx, domain.ID(messageCampaignID))
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 || updated[0].Status != domain.MessageRecipientSent || updated[0].ProviderMessageSID != "SM-test" || updated[0].SentAt == nil {
		t.Fatalf("recipient update was not persisted: %+v", updated)
	}
}
