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

func TestAttendanceRepositoryCreatesOneBoxMemberCheckinPerDay(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	boxID, userID, studentID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	fixtures := []any{
		&models.BoxModel{ID: boxID, Name: "Attendance Test", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		&models.UserModel{ID: userID, BoxID: &boxID, Name: "Owner", Email: uuid.NewString() + "@example.com", PasswordHash: "hash", AuthVersion: 1, Role: "OWNER", Active: true, CreatedAt: now, UpdatedAt: now},
		&models.StudentModel{ID: studentID, BoxID: boxID, Name: "Maria Mensalista", Phone: "5511999999999", Source: "box_member", ExternalID: "self-registration:" + studentID, RiskStatus: "active", ContactStatus: "opted_in", CreatedAt: now, UpdatedAt: now},
	}
	for _, fixture := range fixtures {
		if err := db.Create(fixture).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error })

	repository := NewAttendanceGormRepository(db)
	session := &domain.SelfCheckinSession{
		BoxID: domain.ID(boxID), CreatedByUserID: domain.ID(userID), TokenHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now,
	}
	if err := repository.SaveSelfCheckinSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	loaded, boxName, err := repository.FindValidSelfCheckinSession(context.Background(), session.TokenHash, now)
	if err != nil || loaded.BoxID != domain.ID(boxID) || boxName != "Attendance Test" {
		t.Fatalf("unexpected session: %+v %q %v", loaded, boxName, err)
	}
	students, err := repository.FindActiveBoxMembersByPhone(context.Background(), domain.ID(boxID), "5511999999999")
	if err != nil || len(students) != 1 || students[0].ID != domain.ID(studentID) {
		t.Fatalf("unexpected students: %+v %v", students, err)
	}

	checkinTime := now
	checkin := &domain.Checkin{
		BoxID: domain.ID(boxID), StudentID: domain.ID(studentID), CheckinDate: now,
		CheckinTime: &checkinTime, Source: domain.SourceBoxMember, EntryMethod: domain.CheckinEntrySelfService,
		SelfCheckinSessionID: session.ID, CreatedAt: now,
	}
	created, err := repository.SaveBoxMemberCheckin(context.Background(), checkin)
	if err != nil || !created {
		t.Fatalf("first checkin: created=%v err=%v", created, err)
	}
	checkin.ID = ""
	created, err = repository.SaveBoxMemberCheckin(context.Background(), checkin)
	if err != nil || created {
		t.Fatalf("duplicate checkin: created=%v err=%v", created, err)
	}
}
