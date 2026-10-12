package repositories

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type importedPersonParser struct {
	rows []services.ParsedCheckin
}

func (p importedPersonParser) Parse(context.Context, io.Reader, domain.Source, string) ([]services.ParsedCheckin, error) {
	return p.rows, nil
}

func importedPersonFixture(t *testing.T) (*gorm.DB, domain.ID) {
	t.Helper()
	db := openImportTestDatabase(t).Begin()
	if db.Error != nil {
		t.Fatal(db.Error)
	}
	t.Cleanup(func() { _ = db.Rollback().Error })
	now := time.Now().UTC().Truncate(time.Second)
	box := models.BoxModel{ID: uuid.NewString(), Name: "Import unit", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&box).Error; err != nil {
		t.Fatal(err)
	}
	return db, domain.ID(box.ID)
}

func importedPersonUseCase(db *gorm.DB, rows []services.ParsedCheckin) imports.ImportCheckinsUseCase {
	return imports.NewImportCheckinsUseCase(importedPersonParser{rows}, NewImportHistoryGormRepository(db),
		NewStudentGormRepository(db), NewCheckinGormRepository(db), NewCampaignGormRepository(db),
		NewRewardGormRepository(db), NewPrivacyGormRepository(db), NewGormTransactionManager(db), nil, NewPeopleGormRepository(db))
}

func TestImportCreatesPersonAndUnitRelationshipAndPreservesReimports(t *testing.T) {
	for _, source := range []domain.Source{domain.SourceWellhub, domain.SourceTotalPass} {
		t.Run(string(source), func(t *testing.T) {
			db, unitID := importedPersonFixture(t)
			now := time.Now().UTC().Truncate(time.Second)
			row := services.ParsedCheckin{StudentName: "  Pessoa   Importada  ", StudentExternalID: "provider-123", StudentPhone: "(11) 99999-9999", StudentEmail: "IMPORT@example.test", CheckinDate: now, CheckinTime: &now}
			rows := []services.ParsedCheckin{row, row}
			useCase := importedPersonUseCase(db, rows)
			input := imports.ImportCheckinsInput{BoxID: unitID, Source: source, Filename: "fixture.csv", File: strings.NewReader("fixture")}
			output, err := useCase.Execute(context.Background(), input)
			if err != nil || output.Students != 1 || output.Checkins != 1 {
				t.Fatalf("import counts: output=%+v err=%v", output, err)
			}
			var student models.StudentModel
			if err := db.Where("box_id = ?", unitID).Take(&student).Error; err != nil {
				t.Fatal(err)
			}
			if student.PersonID == nil || *student.PersonID != student.ID || student.ContactStatus != "unknown" {
				t.Fatal("import did not link its person or changed consent")
			}
			var person models.PersonModel
			if err := db.Where("id = ? AND organization_id = ?", student.PersonID, unitID).Take(&person).Error; err != nil {
				t.Fatal(err)
			}
			if person.Name != "Pessoa Importada" || person.PhoneE164 == nil || person.Email == nil || person.CPFEncrypted != nil {
				t.Fatal("import did not normalize its incomplete person profile")
			}
			var relationship models.PersonUnitRelationshipModel
			if err := db.Where("person_id = ? AND unit_id = ?", student.PersonID, unitID).Take(&relationship).Error; err != nil {
				t.Fatal(err)
			}
			if relationship.OrganizationID != string(unitID) || relationship.RelationshipStatus != "active" {
				t.Fatal("person was not linked to the importing unit")
			}
			// Existing human-edited data and access restrictions survive another file.
			if err := db.Model(&person).Updates(map[string]any{"name": "Perfil revisado", "phone_e164": nil}).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&relationship).Update("relationship_status", "blocked").Error; err != nil {
				t.Fatal(err)
			}
			replayed, err := useCase.Execute(context.Background(), input)
			if err != nil || replayed.Students != 0 || replayed.Checkins != 0 {
				t.Fatalf("reimport counts: output=%+v err=%v", replayed, err)
			}
			var peopleCount, relationshipCount int64
			if err := db.Model(&models.PersonModel{}).Where("organization_id = ?", unitID).Count(&peopleCount).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.PersonUnitRelationshipModel{}).Where("unit_id = ?", unitID).Count(&relationshipCount).Error; err != nil {
				t.Fatal(err)
			}
			if peopleCount != 1 || relationshipCount != 1 {
				t.Fatal("reimport duplicated people or unit relationships")
			}
			var currentPerson models.PersonModel
			var currentRelationship models.PersonUnitRelationshipModel
			if err := db.Where("id = ?", person.ID).Take(&currentPerson).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("id = ?", relationship.ID).Take(&currentRelationship).Error; err != nil {
				t.Fatal(err)
			}
			if currentPerson.Name != "Perfil revisado" || currentPerson.PhoneE164 != nil || currentRelationship.RelationshipStatus != "blocked" {
				t.Fatal("reimport overwrote the person profile or restored unit access")
			}
		})
	}
}

type failedImportedPersonCheckins struct {
	portrepo.CheckinRepository
	err error
}

func (r failedImportedPersonCheckins) SaveMany(context.Context, []domain.Checkin) (int, error) {
	return 0, r.err
}

func TestImportRollsBackPeopleWhenCheckinPersistenceFails(t *testing.T) {
	db, unitID := importedPersonFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	row := services.ParsedCheckin{StudentName: "Pessoa da importação", StudentExternalID: "failed-123", CheckinDate: now}
	expected := errors.New("checkin persistence failed")
	useCase := imports.NewImportCheckinsUseCase(importedPersonParser{[]services.ParsedCheckin{row}}, NewImportHistoryGormRepository(db),
		NewStudentGormRepository(db), failedImportedPersonCheckins{NewCheckinGormRepository(db), expected}, NewCampaignGormRepository(db),
		NewRewardGormRepository(db), NewPrivacyGormRepository(db), NewGormTransactionManager(db), nil, NewPeopleGormRepository(db))
	output, err := useCase.Execute(context.Background(), imports.ImportCheckinsInput{BoxID: unitID, Source: domain.SourceTotalPass, Filename: "failed.csv"})
	if !errors.Is(err, expected) || output != nil {
		t.Fatalf("failed import: output=%+v err=%v", output, err)
	}
	for table, column := range map[string]string{"students": "box_id", "persons": "organization_id", "person_unit_relationships": "unit_id", "checkins": "box_id"} {
		var count int64
		if err := db.Table(table).Where(column+" = ?", unitID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("failed import leaked %s: count=%d err=%v", table, count, err)
		}
	}
	var history models.ImportHistoryModel
	if err := db.Where("box_id = ?", unitID).Take(&history).Error; err != nil || history.Status != "failed" {
		t.Fatalf("failed history status: status=%s err=%v", history.Status, err)
	}
}

func TestImportedPeoplePrivacyExportAnonymizationAndSuppression(t *testing.T) {
	db, unitID := importedPersonFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	owner := models.UserModel{ID: uuid.NewString(), BoxID: stringPointer(string(unitID)), Name: "Import owner", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: "OWNER", Active: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	row := services.ParsedCheckin{StudentName: "Pessoa da privacidade", StudentExternalID: "privacy-import-123", StudentPhone: "(11) 99999-9999", StudentEmail: "privacy-import@example.test", CheckinDate: now, CheckinTime: &now}
	useCase := importedPersonUseCase(db, []services.ParsedCheckin{row})
	input := imports.ImportCheckinsInput{BoxID: unitID, Source: domain.SourceTotalPass, Filename: "privacy.csv"}
	if _, err := useCase.Execute(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	student, err := NewStudentGormRepository(db).FindByExternalID(context.Background(), unitID, input.Source, row.StudentExternalID)
	if err != nil {
		t.Fatal(err)
	}
	privacy := NewPrivacyGormRepository(db)
	exported, err := privacy.ExportStudent(context.Background(), unitID, student.ID)
	if err != nil || exported.Person == nil || len(exported.Person.Relationships) != 1 {
		t.Fatal("privacy export did not include the imported person and unit relationship")
	}
	if err := privacy.AnonymizeStudent(context.Background(), unitID, student.ID, domain.ID(owner.ID), "requested erasure"); err != nil {
		t.Fatal(err)
	}
	output, err := useCase.Execute(context.Background(), input)
	if err != nil || output.Students != 0 || output.Checkins != 0 {
		t.Fatal("reimport restored a suppressed identity")
	}
	var person models.PersonModel
	if err := db.Where("id = ?", student.PersonID).Take(&person).Error; err != nil {
		t.Fatal(err)
	}
	if person.ProfileStatus != "anonymized" || person.PhoneE164 != nil || person.Email != nil {
		t.Fatal("imported person retained contact after anonymization")
	}
}

func TestImportedPeopleDoNotMergeNamesSourcesOrUnits(t *testing.T) {
	db, firstUnit := importedPersonFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	secondBox := models.BoxModel{ID: uuid.NewString(), OrganizationID: stringPointer(string(firstUnit)), Name: "Second import unit", Status: "active", RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&secondBox).Error; err != nil {
		t.Fatal(err)
	}
	row := services.ParsedCheckin{StudentName: "Mesmo nome", StudentExternalID: "same-external-123", CheckinDate: now, CheckinTime: &now}
	useCase := importedPersonUseCase(db, []services.ParsedCheckin{row})
	for _, input := range []imports.ImportCheckinsInput{
		{BoxID: firstUnit, Source: domain.SourceTotalPass, Filename: "first.csv"},
		{BoxID: firstUnit, Source: domain.SourceWellhub, Filename: "other-source.csv"},
		{BoxID: domain.ID(secondBox.ID), Source: domain.SourceTotalPass, Filename: "other-unit.csv"},
	} {
		if _, err := useCase.Execute(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&models.PersonModel{}).Where("organization_id = ?", firstUnit).Count(&count).Error; err != nil || count != 3 {
		t.Fatal("import merged students across sources or units")
	}
	student, err := NewStudentGormRepository(db).FindByExternalID(context.Background(), firstUnit, domain.SourceTotalPass, row.StudentExternalID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewPeopleGormRepository(db).EnsureImportedStudent(context.Background(), domain.ID(secondBox.ID), student.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("linker accepted a student from a different existing unit")
	}
}

func TestImportRepairsLegacyPersonAndRetainsLinkOnEarlierMembershipSave(t *testing.T) {
	db, unitID := importedPersonFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	student := models.StudentModel{ID: uuid.NewString(), BoxID: string(unitID), Name: "Aluno antigo", Source: "totalpass", ExternalID: "legacy-123", RiskStatus: "active", ContactStatus: "opted_out", MembershipStartedAt: &now, MembershipStartedSource: stringPointer("first_checkin_inferred"), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&student).Error; err != nil {
		t.Fatal(err)
	}
	previous := now.AddDate(0, 0, -3)
	row := services.ParsedCheckin{StudentName: "Aluno antigo", StudentExternalID: student.ExternalID, CheckinDate: previous, CheckinTime: &previous}
	useCase := importedPersonUseCase(db, []services.ParsedCheckin{row})
	output, err := useCase.Execute(context.Background(), imports.ImportCheckinsInput{BoxID: unitID, Source: domain.SourceTotalPass, Filename: "legacy.csv"})
	if err != nil || output.Students != 0 || output.Checkins != 1 {
		t.Fatalf("legacy reimport: output=%+v err=%v", output, err)
	}
	var repaired models.StudentModel
	if err := db.Where("id = ?", student.ID).Take(&repaired).Error; err != nil {
		t.Fatal(err)
	}
	if repaired.PersonID == nil || *repaired.PersonID != student.ID || repaired.ContactStatus != "opted_out" {
		t.Fatal("legacy repair lost its person ID or changed consent")
	}
	if repaired.MembershipStartedAt == nil || repaired.MembershipStartedAt.Format("2006-01-02") != previous.Format("2006-01-02") {
		t.Fatal("legacy import stopped correcting its inferred membership date")
	}
	var person models.PersonModel
	if err := db.Where("id = ?", student.ID).Take(&person).Error; err != nil {
		t.Fatal(err)
	}
	if person.PhoneE164 != nil || person.CPFEncrypted != nil {
		t.Fatal("legacy import invented missing contact or identity data")
	}
}

func TestImportedPersonLinkerIsScopedAndRollsBackWithTheImport(t *testing.T) {
	db, unitID := importedPersonFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	student := &domain.Student{BoxID: unitID, Name: "Aluno transacional", Source: domain.SourceTotalPass, ExternalID: "rollback-person", CreatedAt: now, UpdatedAt: now}
	linker := NewPeopleGormRepository(db)
	expected := errors.New("later import failure")
	err := NewGormTransactionManager(db).WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := NewStudentGormRepository(db).Save(ctx, student); err != nil {
			return err
		}
		if _, err := linker.EnsureImportedStudent(ctx, unitID, student.ID); err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatal(err)
	}
	for _, table := range []string{"students", "persons", "person_unit_relationships"} {
		var count int64
		column := "organization_id"
		if table == "students" {
			column = "box_id"
		}
		if err := db.Table(table).Where(column+" = ?", unitID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("rollback leaked %s: count=%d err=%v", table, count, err)
		}
	}
	if err := NewStudentGormRepository(db).Save(context.Background(), student); err != nil {
		t.Fatal(err)
	}
	if _, err := linker.EnsureImportedStudent(context.Background(), domain.ID(uuid.NewString()), student.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("linker accepted a student from another unit")
	}
}

func TestImportedPersonLinkerPreservesExistingPersonAndAnonymization(t *testing.T) {
	db, unitID := importedPersonFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	person := models.PersonModel{ID: uuid.NewString(), OrganizationID: string(unitID), Name: "Pessoa existente", ProfileStatus: "active", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&person).Error; err != nil {
		t.Fatal(err)
	}
	student := models.StudentModel{ID: uuid.NewString(), PersonID: &person.ID, BoxID: string(unitID), Name: "Aluno existente", Source: "wellhub", ExternalID: "existing-person", RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&student).Error; err != nil {
		t.Fatal(err)
	}
	linker := NewPeopleGormRepository(db)
	if id, err := linker.EnsureImportedStudent(context.Background(), unitID, domain.ID(student.ID)); err != nil || id != domain.ID(person.ID) {
		t.Fatalf("existing person link: id=%s err=%v", id, err)
	}
	anonymized := models.StudentModel{ID: uuid.NewString(), BoxID: string(unitID), Name: "Aluno anonimizado", Source: "totalpass", ExternalID: "anonymized:" + uuid.NewString(), RiskStatus: "paused", ContactStatus: "opted_out", AnonymizedAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&anonymized).Error; err != nil {
		t.Fatal(err)
	}
	if id, err := linker.EnsureImportedStudent(context.Background(), unitID, domain.ID(anonymized.ID)); err != nil || id != "" {
		t.Fatal("linker restored an anonymized student")
	}
	var count int64
	if err := db.Model(&models.PersonModel{}).Where("organization_id = ?", unitID).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("linker duplicated an existing person or recreated an anonymized one")
	}
}
