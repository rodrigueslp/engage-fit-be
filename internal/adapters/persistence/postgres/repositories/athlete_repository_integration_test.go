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

func TestAthleteRepositoryLoadsInvitationByTokenHash(t *testing.T) {
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
		&models.BoxModel{ID: boxID, Name: "Athlete Invitation Test", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		&models.UserModel{ID: userID, BoxID: &boxID, Name: "Owner", Email: uuid.NewString() + "@example.com", PasswordHash: "hash", AuthVersion: 1, Role: "OWNER", Active: true, CreatedAt: now, UpdatedAt: now},
		&models.StudentModel{ID: studentID, BoxID: boxID, Name: "Atleta Teste", Source: "box_member", ExternalID: "athlete-test:" + studentID, RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
	}
	for _, fixture := range fixtures {
		if err := db.Create(fixture).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error })

	repository := NewAthleteGormRepository(db)
	invitation := &domain.AthleteInvitation{
		BoxID: domain.ID(boxID), StudentID: domain.ID(studentID), CreatedByUserID: domain.ID(userID),
		TokenHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now,
	}
	if err := repository.SaveInvitation(context.Background(), invitation); err != nil {
		t.Fatal(err)
	}

	loaded, err := repository.FindInvitationByTokenHash(context.Background(), invitation.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != invitation.ID || loaded.BoxID != invitation.BoxID || loaded.StudentID != invitation.StudentID || loaded.BoxName != "Athlete Invitation Test" || loaded.StudentName != "Atleta Teste" || !loaded.ExpiresAt.Equal(invitation.ExpiresAt) {
		t.Fatalf("unexpected invitation: %+v", loaded)
	}
}
