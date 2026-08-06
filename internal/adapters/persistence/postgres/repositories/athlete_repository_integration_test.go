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

func TestAthleteRepositoryStoresResultRecalculatesAndConfirmsPersonalRecord(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	boxID, athleteID, membershipID, workoutID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if err := db.Create(&models.BoxModel{ID: boxID, Name: "Athlete Results Test", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error })
	if err := db.Exec("INSERT INTO athlete_accounts (id,name,email,password_hash,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", athleteID, "Atleta Resultado", uuid.NewString()+"@example.com", "hash", "active", now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO athlete_box_memberships (id,athlete_account_id,box_id,status,joined_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", membershipID, athleteID, boxID, "active", now, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.WorkoutModel{ID: workoutID, BoxID: boxID, WorkoutDate: now, Title: "Back squat", RawText: "5x3 back squat", Classification: []byte(`{"version":"rules-v1","sections":[],"formats":[],"movement_mentions":["Back squat"]}`), Status: "published", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	repository := NewAthleteGormRepository(db)
	ctx := context.Background()
	result := &domain.AthleteWorkoutResult{AthleteID: domain.ID(athleteID), WorkoutID: domain.ID(workoutID), MembershipID: domain.ID(membershipID), Scale: "rx", Entries: []domain.AthleteResultEntry{{Movement: "Back squat", ScoreType: "load", LoadKG: 100}}, RPE: 8, PerformedAt: now, UpdatedAt: now}
	records, err := repository.UpsertWorkoutResult(ctx, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].BestValue != 100 || records[0].Status != "estimated" {
		t.Fatalf("unexpected records: %+v", records)
	}
	result.Entries[0].LoadKG = 90
	result.UpdatedAt = now.Add(time.Minute)
	records, err = repository.UpsertWorkoutResult(ctx, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("editing the source must recalculate the PR: %+v", records)
	}
	stored, err := repository.ListPersonalRecords(ctx, domain.ID(athleteID))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].BestValue != 90 {
		t.Fatalf("PR was not recalculated: %+v", stored)
	}
	if err := repository.ConfirmPersonalRecord(ctx, domain.ID(athleteID), stored[0].ID, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	stored, _ = repository.ListPersonalRecords(ctx, domain.ID(athleteID))
	if stored[0].Status != "confirmed" {
		t.Fatalf("PR not confirmed: %+v", stored[0])
	}
	result.Entries = []domain.AthleteResultEntry{{ScoreType: "completed", Completed: true}}
	result.UpdatedAt = now.Add(3 * time.Minute)
	if _, err := repository.UpsertWorkoutResult(ctx, result); err != nil {
		t.Fatal(err)
	}
	stored, _ = repository.ListPersonalRecords(ctx, domain.ID(athleteID))
	if len(stored) != 0 {
		t.Fatalf("removed movement left a stale PR: %+v", stored)
	}
}
