package repositories

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"boxengage/backend/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestAttendanceAccessRepositoryPersistsAuditedPresenceAndDeduplicates(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	adminDB := stdlib.OpenDB(*adminConfig)
	defer adminDB.Close()
	testDatabase := "spec008_repo_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+testDatabase); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if _, err := adminDB.ExecContext(cleanup, "DROP DATABASE IF EXISTS "+testDatabase+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	}()
	testConfig := adminConfig.Copy()
	testConfig.Database = testDatabase
	sqlDB := stdlib.OpenDB(*testConfig)
	if _, err := migrations.New(sqlDB).Up(ctx); err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	testURL, _ := url.Parse(databaseURL)
	testURL.Path = "/" + testDatabase
	db, err := postgres.Open(testURL.String())
	if err != nil {
		t.Fatal(err)
	}

	// 23:30 in the unit is already the next UTC day.
	now := time.Date(2026, 10, 10, 2, 30, 0, 0, time.UTC)
	box := &domain.Box{Name: "SPEC-008 Unit", Timezone: "America/Sao_Paulo", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner := &domain.User{Name: "Owner", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := NewBoxGormRepository(db).SaveWithOwner(ctx, box, owner); err != nil {
		t.Fatal(err)
	}
	personID, studentID := uuid.NewString(), uuid.NewString()
	person := &models.PersonModel{ID: personID, OrganizationID: string(box.OrganizationID), Name: "Pessoa Corporativa", ProfileStatus: "active", Revision: 1, CreatedAt: now, UpdatedAt: now}
	student := &models.StudentModel{ID: studentID, PersonID: &personID, BoxID: string(box.ID), Name: "Pessoa Corporativa", Source: "wellhub", ExternalID: "corporate-" + studentID, RiskStatus: "active", ContactStatus: "unknown", CreatedAt: now, UpdatedAt: now}
	relationship := &models.PersonUnitRelationshipModel{ID: uuid.NewString(), OrganizationID: string(box.OrganizationID), PersonID: personID, UnitID: string(box.ID), RelationshipStatus: "active", StartedAt: now, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(person).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(student).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(relationship).Error; err != nil {
		t.Fatal(err)
	}
	accessCandidates, err := NewEnrollmentGormRepository(db).ResolveMembershipAccessCandidates(ctx, domain.MembershipAccessRequest{
		OrganizationID: box.OrganizationID,
		UnitID:         box.ID,
		PersonID:       domain.ID(personID),
		OccurredAt:     now,
		Purpose:        domain.MembershipAccessCheckin,
	})
	if err != nil {
		t.Fatalf("membership access without enrollment: %v", err)
	}
	if len(accessCandidates) != 0 {
		t.Fatalf("expected no access candidates without enrollment, got %+v", accessCandidates)
	}

	repo := NewAttendanceGormRepository(db)
	subject, err := repo.ResolveAttendanceSubject(ctx, box.OrganizationID, box.ID, domain.ID(personID), now)
	if err != nil || subject.Source != domain.SourceWellhub || !subject.RelationshipActive {
		t.Fatalf("subject=%+v err=%v", subject, err)
	}
	entitlementID := domain.ID(uuid.NewString())
	remaining := 2
	decision := &domain.AttendanceDecision{ID: domain.ID(uuid.NewString()), OrganizationID: box.OrganizationID, UnitID: box.ID, PersonID: domain.ID(personID), StudentID: domain.ID(studentID), EntitlementID: &entitlementID, RemainingUsage: &remaining, Source: domain.SourceWellhub, Outcome: domain.AttendanceAllowed, ReasonCode: domain.AttendanceReasonCorporateRelationship, Reason: "Vínculo corporativo ativo; entrada liberada localmente.", ActorUserID: owner.ID, OccurredAt: now}
	if err := repo.CreateAttendanceDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	record := &domain.AttendanceRecord{ID: domain.ID(uuid.NewString()), OrganizationID: box.OrganizationID, UnitID: box.ID, DecisionID: decision.ID, PersonID: decision.PersonID, StudentID: decision.StudentID, LocalDate: now.In(mustLocation(t, "America/Sao_Paulo")).Format("2006-01-02"), OccurredAt: now}
	if err := repo.CreateAttendanceRecord(ctx, record, domain.SourceWellhub, owner.ID); err != nil {
		t.Fatal(err)
	}
	if record.CheckinID == nil || record.AlreadyRecorded {
		t.Fatalf("first record=%+v", record)
	}
	duplicate := &domain.AttendanceRecord{ID: domain.ID(uuid.NewString()), OrganizationID: box.OrganizationID, UnitID: box.ID, DecisionID: decision.ID, PersonID: decision.PersonID, StudentID: decision.StudentID, LocalDate: record.LocalDate, OccurredAt: now.Add(time.Minute)}
	if err := repo.CreateAttendanceRecord(ctx, duplicate, domain.SourceWellhub, owner.ID); err != nil {
		t.Fatal(err)
	}
	if !duplicate.AlreadyRecorded || duplicate.ID != record.ID {
		t.Fatalf("duplicate=%+v first=%+v", duplicate, record)
	}
	if err := repo.CreateAttendanceCommand(ctx, &domain.AttendanceCommandRequest{OrganizationID: box.OrganizationID, IdempotencyKey: "attendance.integration", Fingerprint: strings.Repeat("a", 64), DecisionID: decision.ID}); err != nil {
		t.Fatal(err)
	}
	command, err := repo.FindAttendanceCommand(ctx, box.OrganizationID, "attendance.integration")
	if err != nil || command == nil || command.DecisionID != decision.ID {
		t.Fatalf("command=%+v err=%v", command, err)
	}
	items, err := repo.ListAttendanceForDate(ctx, box.OrganizationID, box.ID, record.LocalDate)
	if err != nil || len(items) != 1 || items[0].Attendance == nil || items[0].Decision.EntitlementID == nil || *items[0].Decision.EntitlementID != entitlementID || items[0].Decision.RemainingUsage == nil || *items[0].Decision.RemainingUsage != remaining {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	for _, adjacentDate := range []string{"2026-10-08", "2026-10-10"} {
		items, err := repo.ListAttendanceForDate(ctx, box.OrganizationID, box.ID, adjacentDate)
		if err != nil || len(items) != 0 {
			t.Fatalf("adjacent date %s: items=%+v err=%v", adjacentDate, items, err)
		}
	}
}

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	value, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
