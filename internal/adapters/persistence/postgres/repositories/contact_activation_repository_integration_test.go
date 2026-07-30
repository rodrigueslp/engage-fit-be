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

func TestContactActivationMatchesAndConfirmsStudent(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	checkinDate := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
	boxID, studentID, importID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if err := db.Create(&models.BoxModel{ID: boxID, Name: "Activation Test", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error })
	if err := db.Create(&models.ImportHistoryModel{ID: importID, BoxID: boxID, Filename: "activation.csv", Source: "totalpass", TotalRecords: 1, ImportedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.StudentModel{ID: studentID, BoxID: boxID, Name: "Adriana  Segatelli", Source: "totalpass", ExternalID: "adriana segatelli", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CheckinModel{ID: uuid.NewString(), BoxID: boxID, StudentID: studentID, CheckinDate: checkinDate, Source: "totalpass", ImportHistoryID: importID, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	repository := NewContactActivationGormRepository(db)
	activationCode, err := repository.ActivationCode(context.Background(), domain.ID(boxID))
	if err != nil || activationCode == "" {
		t.Fatalf("load activation code: %q, %v", activationCode, err)
	}
	publicBoxID, publicBoxName, err := repository.FindPublicBox(context.Background(), activationCode)
	if err != nil || publicBoxID != domain.ID(boxID) || publicBoxName != "Activation Test" {
		t.Fatalf("load public box: %s %q, %v", publicBoxID, publicBoxName, err)
	}
	matches, err := repository.FindMatchingStudents(context.Background(), domain.ID(boxID), domain.SourceTotalPass, "  ADRIANA SEGATELLI ", checkinDate)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != domain.ID(studentID) {
		t.Fatalf("unexpected matches: %+v", matches)
	}

	activation := &domain.ContactActivationRequest{
		ID: domain.ID(uuid.NewString()), BoxID: domain.ID(boxID), StudentID: domain.ID(studentID),
		ClaimedName: "Adriana Segatelli", Source: domain.SourceTotalPass, RecentCheckinDate: &checkinDate,
		SenderPhone: "5511999999999", TokenHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Status: domain.ContactActivationAwaitingMessage, ConsentVersion: "v1", ConsentText: "consent",
		ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	if err := repository.CreateActivation(context.Background(), activation); err != nil {
		t.Fatal(err)
	}
	confirmed, err := repository.ConfirmActivation(context.Background(), activation.ID, "5511988887777", now)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != domain.ContactActivationConfirmed {
		t.Fatalf("unexpected status: %s", confirmed.Status)
	}
	student, err := NewStudentGormRepository(db).FindByID(context.Background(), domain.ID(boxID), domain.ID(studentID))
	if err != nil {
		t.Fatal(err)
	}
	if student.Phone != "5511988887777" || student.ContactStatus != domain.ContactStatusOptedIn || student.ContactStatusSource != "whatsapp_self_activation" {
		t.Fatalf("student was not activated: %+v", student)
	}
	var consentCount int64
	if err := db.Table("contact_consent_events").Where("box_id = ? AND student_id = ? AND action = 'opted_in'", boxID, studentID).Count(&consentCount).Error; err != nil {
		t.Fatal(err)
	}
	if consentCount != 1 {
		t.Fatalf("expected one consent event, got %d", consentCount)
	}
}
