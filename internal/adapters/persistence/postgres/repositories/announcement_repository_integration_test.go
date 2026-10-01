package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"boxengage/backend/migrations"
	"github.com/google/uuid"
)

func TestAnnouncementRepositorySnapshotsAudienceAndTracksViews(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	if _, err = migrations.New(sqlDB).Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	organizationID, unitID, personID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	studentID, athleteID, membershipID, ownerID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if err := tx.Create(&models.OrganizationModel{ID: organizationID, Name: "Announcement Org", Status: "active", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&models.BoxModel{ID: unitID, OrganizationID: &organizationID, Name: "Announcement Unit", Status: "active", Timezone: "America/Sao_Paulo", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&models.UserModel{ID: ownerID, BoxID: &unitID, Name: "Owner", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: "OWNER", Active: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&models.PersonModel{ID: personID, OrganizationID: organizationID, Name: "Atleta", ProfileStatus: "active", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&models.StudentModel{ID: studentID, PersonID: &personID, BoxID: unitID, Name: "Atleta", Source: "box_member", ExternalID: "announcement:" + studentID, RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO athlete_accounts (id,name,email,password_hash,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", athleteID, "Atleta", uuid.NewString()+"@example.test", "hash", "active", now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO athlete_box_memberships (id,athlete_account_id,box_id,status,joined_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", membershipID, athleteID, unitID, "active", now, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO athlete_student_links (id,membership_id,student_id,link_method,linked_at,created_at) VALUES (?,?,?,?,?,?)", uuid.NewString(), membershipID, studentID, "individual_invite", now, now).Error; err != nil {
		t.Fatal(err)
	}

	repository := NewAnnouncementGormRepository(tx)
	value := &domain.Announcement{ID: domain.ID(uuid.NewString()), OrganizationID: domain.ID(organizationID), UnitID: domain.ID(unitID), Title: "Aula especial", Body: "Confira a programação deste sábado.", Priority: "important", Status: "draft", AudienceKind: "source", AudienceValue: "box_member", CreatedByUserID: domain.ID(ownerID), CreatedAt: now, UpdatedAt: now}
	if err := repository.Create(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	value.Status, value.PublishAt, value.PublishedAt = "published", &now, &now
	if err := repository.Publish(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	items, err := repository.ListForAthlete(context.Background(), domain.ID(athleteID), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].MembershipID != domain.ID(membershipID) || items[0].ViewedAt != nil {
		t.Fatalf("unexpected athlete announcements: %+v", items)
	}
	if err := repository.MarkViewed(context.Background(), domain.ID(athleteID), value.ID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	history, err := repository.List(context.Background(), domain.ID(organizationID), domain.ID(unitID))
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].RecipientCount != 1 || history[0].ViewedCount != 1 {
		t.Fatalf("unexpected history: %+v", history)
	}
}
