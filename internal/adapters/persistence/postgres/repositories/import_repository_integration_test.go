package repositories

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestCheckinRepositoryPersistsLargeImportInSafeBatches(t *testing.T) {
	db := openImportTestDatabase(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	boxID, studentID, importID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	createImportFixtures(t, db, boxID, studentID, importID, now)

	const total = 5986
	checkins := make([]domain.Checkin, 0, total)
	for index := 0; index < total; index++ {
		checkinTime := time.Date(2026, time.August, 4, 0, 0, index, 0, time.UTC)
		checkins = append(checkins, domain.Checkin{
			BoxID: domain.ID(boxID), StudentID: domain.ID(studentID), CheckinDate: checkinTime,
			CheckinTime: &checkinTime, Source: domain.SourceTotalPass,
			ImportHistoryID: domain.ID(importID), CreatedAt: now,
		})
	}

	inserted, err := NewCheckinGormRepository(db).SaveMany(ctx, checkins)
	if err != nil {
		t.Fatal(err)
	}
	if inserted != total {
		t.Fatalf("expected %d inserted checkins, got %d", total, inserted)
	}
	var stored int64
	if err := db.Model(&models.CheckinModel{}).Where("import_history_id = ?", importID).Count(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored != total {
		t.Fatalf("expected %d stored checkins, got %d", total, stored)
	}
}

func TestTransactionManagerRollsBackImportSideEffects(t *testing.T) {
	db := openImportTestDatabase(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	boxID, importID := uuid.NewString(), uuid.NewString()
	if err := db.Create(&models.BoxModel{ID: boxID, Name: "Import rollback", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error })
	if err := db.Create(&models.ImportHistoryModel{ID: importID, BoxID: boxID, Filename: "rollback.xlsx", Source: "totalpass", Status: "processing", TotalRecords: 1, ImportedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	studentRepository := NewStudentGormRepository(db)
	checkinRepository := NewCheckinGormRepository(db)
	transactionManager := NewGormTransactionManager(db)
	student := &domain.Student{
		BoxID: domain.ID(boxID), Name: "Rollback Student", Source: domain.SourceTotalPass,
		ExternalID: uuid.NewString(), RiskStatus: domain.StudentRiskStatusActive,
		ContactStatus: domain.ContactStatusUnknown,
		CreatedAt:     now, UpdatedAt: now,
	}
	expectedError := errors.New("force rollback")
	err := transactionManager.WithinTransaction(ctx, func(transactionContext context.Context) error {
		if err := studentRepository.Save(transactionContext, student); err != nil {
			return err
		}
		_, err := checkinRepository.SaveMany(transactionContext, []domain.Checkin{{
			BoxID: domain.ID(boxID), StudentID: student.ID, CheckinDate: now,
			Source: domain.SourceTotalPass, ImportHistoryID: domain.ID(importID), CreatedAt: now,
		}})
		if err != nil {
			return err
		}
		return expectedError
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected rollback error, got %v", err)
	}

	var students, checkins int64
	if err := db.Model(&models.StudentModel{}).Where("box_id = ?", boxID).Count(&students).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.CheckinModel{}).Where("box_id = ?", boxID).Count(&checkins).Error; err != nil {
		t.Fatal(err)
	}
	if students != 0 || checkins != 0 {
		t.Fatalf("transaction leaked side effects: students=%d checkins=%d", students, checkins)
	}
}

func openImportTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
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
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func createImportFixtures(t *testing.T, db *gorm.DB, boxID, studentID, importID string, now time.Time) {
	t.Helper()
	fixtures := []any{
		&models.BoxModel{ID: boxID, Name: "Large import", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now},
		&models.StudentModel{ID: studentID, BoxID: boxID, Name: "Large Import Student", Source: "totalpass", ExternalID: uuid.NewString(), RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now},
		&models.ImportHistoryModel{ID: importID, BoxID: boxID, Filename: "large.xlsx", Source: "totalpass", Status: "processing", TotalRecords: 5986, ImportedAt: now},
	}
	for _, fixture := range fixtures {
		if err := db.Create(fixture).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error })
}
