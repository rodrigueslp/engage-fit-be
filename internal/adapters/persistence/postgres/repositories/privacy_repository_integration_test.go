package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

func TestPrivacyAnonymizationSuppressesIdentityAndContact(t *testing.T) {
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

	now := time.Now().UTC()
	boxID, studentID := uuid.NewString(), uuid.NewString()
	if err := db.Create(&models.BoxModel{ID: boxID, Name: "privacy-test", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Where("id = ?", boxID).Delete(&models.BoxModel{})
	if err := db.Create(&models.StudentModel{ID: studentID, BoxID: boxID, Name: "Personal Name", Email: "person@example.test", Phone: "+5511999999999", Source: "totalpass", ExternalID: "external-person", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	privacy := NewPrivacyGormRepository(db)
	exported, err := privacy.ExportStudent(context.Background(), domain.ID(boxID), domain.ID(studentID))
	if err != nil || exported.Student.Email != "person@example.test" {
		t.Fatalf("unexpected export: %+v err=%v", exported, err)
	}
	if err := privacy.AnonymizeStudent(context.Background(), domain.ID(boxID), domain.ID(studentID), "", "requested erasure"); err != nil {
		t.Fatal(err)
	}
	suppressed, err := privacy.IsIdentitySuppressed(context.Background(), domain.ID(boxID), domain.SourceTotalPass, "external-person")
	if err != nil || !suppressed {
		t.Fatalf("identity should be suppressed: suppressed=%v err=%v", suppressed, err)
	}

	student, err := NewStudentGormRepository(db).FindByID(context.Background(), domain.ID(boxID), domain.ID(studentID))
	if err != nil {
		t.Fatal(err)
	}
	if student.Name != "Aluno anonimizado" || student.Email != "" || student.Phone != "" || student.AnonymizedAt == nil || student.ContactStatus != domain.ContactStatusOptedOut {
		t.Fatalf("student PII was not anonymized: %+v", student)
	}
	contactable, err := NewStudentGormRepository(db).List(context.Background(), domain.ID(boxID), portrepo.StudentFilters{ContactableOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(contactable) != 0 {
		t.Fatalf("anonymized student remained contactable: %+v", contactable)
	}
}
