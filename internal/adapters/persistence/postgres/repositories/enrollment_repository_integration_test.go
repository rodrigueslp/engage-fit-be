package repositories

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

func TestEnrollmentLifecycleIdempotencyConcurrencyAndPrivacy(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC)
	boxRepository := NewBoxGormRepository(db)
	boxes := []*domain.Box{{Name: "Enrollment Unit One " + uuid.NewString(), Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}, {Name: "Enrollment Unit Two " + uuid.NewString(), Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}}
	owners := []*domain.User{{Name: "Owner One", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}, {Name: "Owner Two", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}}
	for index := range boxes {
		if err := boxRepository.SaveWithOwner(ctx, boxes[index], owners[index]); err != nil {
			t.Fatal(err)
		}
	}
	organizationIDs := []string{string(boxes[0].OrganizationID), string(boxes[1].OrganizationID)}
	unitIDs := []string{string(boxes[0].ID), string(boxes[1].ID)}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM enrollment_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM enrollment_command_requests WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM enrollments WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_command_requests WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_audit_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("ALTER TABLE membership_plan_entitlements DISABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_versions DISABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM membership_plan_entitlements WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM membership_plan_versions WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("ALTER TABLE membership_plan_versions ENABLE TRIGGER USER").Error
		_ = db.Exec("ALTER TABLE membership_plan_entitlements ENABLE TRIGGER USER").Error
		_ = db.Exec("DELETE FROM membership_plans WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM catalog_audit_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM unit_modalities WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM modalities WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM people_audit_events WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM person_unit_relationships WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM students WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM persons WHERE organization_id IN ?", organizationIDs).Error
		_ = db.Exec("DELETE FROM users WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM boxes WHERE id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM organizations WHERE id IN ?", organizationIDs).Error
	})

	personID, studentID, relationshipID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if err := db.Exec(`INSERT INTO persons (id, organization_id, name, phone_e164) VALUES (?, ?, 'Ana Enrollment', '+5511999999999')`, personID, string(boxes[0].OrganizationID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO students (id, box_id, person_id, name, source, contact_status, membership_started_at) VALUES (?, ?, ?, 'Ana Enrollment', 'box_member', 'unknown', '2025-01-10')`, studentID, string(boxes[0].ID), personID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, started_at) VALUES (?, ?, ?, ?, NOW())`, relationshipID, string(boxes[0].OrganizationID), personID, string(boxes[0].ID)).Error; err != nil {
		t.Fatal(err)
	}

	transactionManager := NewGormTransactionManager(db)
	catalogRepository := NewCatalogGormRepository(db)
	catalogService := catalog.NewService(catalogRepository, transactionManager)
	modality, err := catalogService.CreateModality(ctx, boxes[0].ID, owners[0].ID, "CrossFit Enrollment "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatal(err)
	}
	membershipRepository := NewMembershipGormRepository(db)
	membershipService := memberships.NewService(membershipRepository, catalogRepository, boxRepository, transactionManager)
	monthly := memberships.PlanInput{Name: "Mensal Enrollment " + uuid.NewString()[:8], PriceCents: 25000, BillingCycle: domain.MembershipBillingMonthly, Entitlements: []memberships.EntitlementInput{{ModalityID: string(modality.ID), AccessType: domain.MembershipAccessOpen}}}
	monthlyPlan, _, err := membershipService.Create(ctx, boxes[0].ID, owners[0].ID, "enroll-plan-create-"+uuid.NewString(), monthly)
	if err != nil {
		t.Fatal(err)
	}
	monthlyPlan, _, err = membershipService.Publish(ctx, boxes[0].ID, owners[0].ID, monthlyPlan.ID, "enroll-plan-publish-"+uuid.NewString(), monthlyPlan.Draft.Revision)
	if err != nil {
		t.Fatal(err)
	}

	enrollmentRepository := NewEnrollmentGormRepository(db)
	peopleRepository := NewPeopleGormRepository(db)
	service := memberships.NewEnrollmentService(enrollmentRepository, peopleRepository, catalogRepository, boxRepository, transactionManager, memberships.WithEnrollmentClock(func() time.Time { return now }))
	createInput := memberships.CreateEnrollmentInput{PlanID: string(monthlyPlan.ID), PlanVersionID: string(monthlyPlan.Published.ID), StartsOn: "2026-08-19"}
	initialCreateKey := "enrollment." + uuid.NewString()
	created, replayed, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, initialCreateKey, createInput)
	if err != nil || replayed || created.ValidUntil != nil || created.EffectiveStatus != domain.EnrollmentEffectiveActive {
		t.Fatalf("recurring create=%+v replayed=%v err=%v", created, replayed, err)
	}
	roster, err := service.ListRoster(ctx, boxes[0].ID)
	if err != nil {
		t.Fatalf("list enrollment roster: %v", err)
	}
	var rosterItem *domain.EnrollmentRosterItem
	for index := range roster {
		if roster[index].EnrollmentID == created.ID {
			rosterItem = &roster[index]
			break
		}
	}
	if rosterItem == nil || rosterItem.PlanID != monthlyPlan.ID || len(rosterItem.Targets) != 1 || rosterItem.Targets[0].ID != modality.ID {
		t.Fatalf("roster item=%+v", rosterItem)
	}
	if rosterItem.AccessStatus.Eligible || len(rosterItem.AccessStatus.PendingReasonCodes) != 3 {
		t.Fatalf("roster access status=%+v", rosterItem.AccessStatus)
	}
	accessService := memberships.NewAccessService(enrollmentRepository, NewMembershipAccessUsageGormAdapter(db), NewAttendanceGormRepository(db))
	accessDecision, err := accessService.Evaluate(ctx, domain.MembershipAccessRequest{OrganizationID: boxes[0].OrganizationID, UnitID: boxes[0].ID, PersonID: domain.ID(personID), TargetKind: domain.ScheduleTargetModality, TargetID: modality.ID, OccurredAt: now, Purpose: domain.MembershipAccessCheckin})
	if err != nil || accessDecision.Outcome != domain.MembershipAccessAllowed || accessDecision.EnrollmentID != created.ID || accessDecision.EntitlementID == "" {
		t.Fatalf("membership access decision=%+v err=%v", accessDecision, err)
	}
	createdReplay, replayed, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, initialCreateKey, createInput)
	if err != nil || !replayed || createdReplay.ID != created.ID {
		t.Fatalf("create replay=%+v replayed=%v err=%v", createdReplay, replayed, err)
	}
	differentCreate := createInput
	differentCreate.StartsOn = "2026-08-20"
	if _, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, initialCreateKey, differentCreate); !errors.Is(err, memberships.ErrEnrollmentIdempotencyReused) {
		t.Fatalf("create key reuse error=%v", err)
	}
	createKey := "enrollment." + uuid.NewString()
	replayedEnrollment, replayed, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, createKey, memberships.CreateEnrollmentInput{PlanID: string(monthlyPlan.ID), PlanVersionID: string(monthlyPlan.Published.ID), StartsOn: "2026-08-20"})
	if !errors.Is(err, memberships.ErrActiveEnrollmentExists) || replayed || replayedEnrollment != nil {
		t.Fatalf("active conflict result=%+v replayed=%v err=%v", replayedEnrollment, replayed, err)
	}

	firstKey := "enrollment-end." + uuid.NewString()
	ended, replayed, err := service.End(ctx, boxes[0].ID, created.ID, owners[0].ID, firstKey, memberships.EndEnrollmentInput{EndedOn: "2026-08-19", Reason: "  Solicitação   do aluno  ", ExpectedRevision: 1})
	if err != nil || replayed || ended.Revision != 2 || ended.EndedReason != "Solicitação do aluno" {
		t.Fatalf("end=%+v replayed=%v err=%v", ended, replayed, err)
	}
	endedReplay, replayed, err := service.End(ctx, boxes[0].ID, created.ID, owners[0].ID, firstKey, memberships.EndEnrollmentInput{EndedOn: "2026-08-19", Reason: "Solicitação do aluno", ExpectedRevision: 1})
	if err != nil || !replayed || endedReplay.ID != ended.ID {
		t.Fatalf("end replay=%+v replayed=%v err=%v", endedReplay, replayed, err)
	}
	if _, _, err := service.End(ctx, boxes[0].ID, created.ID, owners[0].ID, "enrollment-end."+uuid.NewString(), memberships.EndEnrollmentInput{EndedOn: "2026-08-19", Reason: "Outro motivo", ExpectedRevision: 1}); !errors.Is(err, memberships.ErrEnrollmentAlreadyEnded) {
		t.Fatalf("second end error=%v", err)
	}

	if _, err := catalogService.SetModalityActive(ctx, boxes[0].ID, boxes[0].ID, owners[0].ID, modality.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, "enrollment."+uuid.NewString(), createInput); !errors.Is(err, memberships.ErrPlanTargetsInactive) {
		t.Fatalf("inactive target error=%v", err)
	}
	if _, err := catalogService.SetModalityActive(ctx, boxes[0].ID, boxes[0].ID, owners[0].ID, modality.ID, true); err != nil {
		t.Fatal(err)
	}

	validityCount, validityUnit := 30, domain.MembershipValidityDay
	oneTimeInput := memberships.PlanInput{Name: "Pacote Enrollment " + uuid.NewString()[:8], PriceCents: 30000, BillingCycle: domain.MembershipBillingOneTime, ValidityCount: &validityCount, ValidityUnit: &validityUnit, Entitlements: monthly.Entitlements}
	oneTimePlan, _, err := membershipService.Create(ctx, boxes[0].ID, owners[0].ID, "enroll-pack-create-"+uuid.NewString(), oneTimeInput)
	if err != nil {
		t.Fatal(err)
	}
	oneTimePlan, _, err = membershipService.Publish(ctx, boxes[0].ID, owners[0].ID, oneTimePlan.ID, "enroll-pack-publish-"+uuid.NewString(), oneTimePlan.Draft.Revision)
	if err != nil {
		t.Fatal(err)
	}
	expired, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, "enrollment."+uuid.NewString(), memberships.CreateEnrollmentInput{PlanID: string(oneTimePlan.ID), PlanVersionID: string(oneTimePlan.Published.ID), StartsOn: "2026-07-01"})
	if err != nil || expired.ValidUntil == nil || expired.ValidUntil.Format("2006-01-02") != "2026-07-30" || expired.EffectiveStatus != domain.EnrollmentEffectiveExpired {
		t.Fatalf("one-time expired=%+v err=%v", expired, err)
	}
	current, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, "enrollment."+uuid.NewString(), createInput)
	if err != nil || current.ID == expired.ID {
		t.Fatalf("expired reconciliation create=%+v err=%v", current, err)
	}
	expiredAfter, err := service.Get(ctx, boxes[0].ID, expired.ID)
	if err != nil || expiredAfter.LifecycleStatus != domain.EnrollmentLifecycleEnded || expiredAfter.EndedReason != "Validade natural encerrada" || expiredAfter.EndedOn == nil || expiredAfter.EndedOn.Format("2006-01-02") != "2026-07-30" {
		t.Fatalf("expired reconciliation=%+v err=%v", expiredAfter, err)
	}

	privacy, err := service.ListPrivacy(ctx, boxes[0].ID, domain.ID(personID))
	if err != nil || len(privacy) < 3 {
		t.Fatalf("privacy history=%+v err=%v", privacy, err)
	}
	foundReconciliation := false
	for _, item := range privacy {
		for _, event := range item.Events {
			if event.Action == "enrollment.expiration_reconciled" {
				foundReconciliation = true
			}
		}
	}
	if !foundReconciliation {
		t.Fatal("privacy export did not include reconciliation audit")
	}

	var membershipStartedAt time.Time
	var checkinCount int64
	if err := db.Raw(`SELECT membership_started_at FROM students WHERE id = ?`, studentID).Scan(&membershipStartedAt).Error; err != nil || membershipStartedAt.Format("2006-01-02") != "2025-01-10" {
		t.Fatalf("membership_started_at changed: %v err=%v", membershipStartedAt, err)
	}
	if err := db.Table("checkins").Where("student_id = ?", studentID).Count(&checkinCount).Error; err != nil || checkinCount != 0 {
		t.Fatalf("check-ins changed: count=%d err=%v", checkinCount, err)
	}
	if _, err := service.Get(ctx, boxes[1].ID, current.ID); !errors.Is(err, memberships.ErrEnrollmentNotFound) {
		t.Fatalf("cross-tenant get error=%v", err)
	}

	concurrentStudent, concurrentPerson := uuid.NewString(), uuid.NewString()
	if err := db.Exec(`INSERT INTO persons (id, organization_id, name, phone_e164) VALUES (?, ?, 'Concurrent Person', '+5511988888888')`, concurrentPerson, string(boxes[0].OrganizationID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO students (id, box_id, person_id, name, source, contact_status) VALUES (?, ?, ?, 'Concurrent Person', 'box_member', 'unknown')`, concurrentStudent, string(boxes[0].ID), concurrentPerson).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, started_at) VALUES (?, ?, ?, ?, NOW())`, uuid.NewString(), string(boxes[0].OrganizationID), concurrentPerson, string(boxes[0].ID)).Error; err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _, createErr := service.Create(ctx, boxes[0].ID, domain.ID(concurrentStudent), owners[0].ID, "enrollment."+uuid.NewString(), createInput)
			results <- createErr
		}()
	}
	wait.Wait()
	close(results)
	succeeded, conflicted := 0, 0
	for resultErr := range results {
		switch {
		case resultErr == nil:
			succeeded++
		case errors.Is(resultErr, memberships.ErrActiveEnrollmentExists):
			conflicted++
		default:
			t.Fatalf("unexpected concurrent create error: %v", resultErr)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent creates succeeded=%d conflicted=%d", succeeded, conflicted)
	}

	rollbackStudent, rollbackPerson := uuid.NewString(), uuid.NewString()
	if err := db.Exec(`INSERT INTO persons (id, organization_id, name, phone_e164) VALUES (?, ?, 'Rollback Person', '+5511977777777')`, rollbackPerson, string(boxes[0].OrganizationID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO students (id, box_id, person_id, name, source, contact_status) VALUES (?, ?, ?, 'Rollback Person', 'box_member', 'unknown')`, rollbackStudent, string(boxes[0].ID), rollbackPerson).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO person_unit_relationships (id, organization_id, person_id, unit_id, started_at) VALUES (?, ?, ?, ?, NOW())`, uuid.NewString(), string(boxes[0].OrganizationID), rollbackPerson, string(boxes[0].ID)).Error; err != nil {
		t.Fatal(err)
	}
	failingRepository := failingEnrollmentEventRepository{EnrollmentRepository: enrollmentRepository}
	failingService := memberships.NewEnrollmentService(failingRepository, peopleRepository, catalogRepository, boxRepository, transactionManager, memberships.WithEnrollmentClock(func() time.Time { return now }))
	if _, _, err := failingService.Create(ctx, boxes[0].ID, domain.ID(rollbackStudent), owners[0].ID, "enrollment."+uuid.NewString(), createInput); err == nil {
		t.Fatal("expected event failure")
	}
	var rollbackCount int64
	if err := db.Table("enrollments").Where("person_id = ?", rollbackPerson).Count(&rollbackCount).Error; err != nil || rollbackCount != 0 {
		t.Fatalf("event failure did not roll back enrollment: count=%d err=%v", rollbackCount, err)
	}

	if _, _, err := service.End(ctx, boxes[0].ID, current.ID, owners[0].ID, "enrollment-end."+uuid.NewString(), memberships.EndEnrollmentInput{EndedOn: "2026-08-19", Reason: "Troca de versão", ExpectedRevision: current.Revision}); err != nil {
		t.Fatal(err)
	}
	withDraft, _, err := membershipService.NewVersion(ctx, boxes[0].ID, owners[0].ID, monthlyPlan.ID, "enroll-new-version-"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	versionTwoPlan, _, err := membershipService.Publish(ctx, boxes[0].ID, owners[0].ID, monthlyPlan.ID, "enroll-publish-v2-"+uuid.NewString(), withDraft.Draft.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, "enrollment."+uuid.NewString(), createInput); !errors.Is(err, memberships.ErrPlanVersionChanged) {
		t.Fatalf("retired reviewed version error=%v", err)
	}
	versionTwoInput := memberships.CreateEnrollmentInput{PlanID: string(versionTwoPlan.ID), PlanVersionID: string(versionTwoPlan.Published.ID), StartsOn: "2026-08-19"}
	versionTwoEnrollment, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, "enrollment."+uuid.NewString(), versionTwoInput)
	if err != nil || versionTwoEnrollment.Terms.VersionNumber != 2 {
		t.Fatalf("version two enrollment=%+v err=%v", versionTwoEnrollment, err)
	}
	if _, _, err := service.End(ctx, boxes[0].ID, versionTwoEnrollment.ID, owners[0].ID, "enrollment-end."+uuid.NewString(), memberships.EndEnrollmentInput{EndedOn: "2026-08-19", Reason: "Teste de arquivo", ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := membershipService.SetArchived(ctx, boxes[0].ID, owners[0].ID, monthlyPlan.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Create(ctx, boxes[0].ID, domain.ID(studentID), owners[0].ID, "enrollment."+uuid.NewString(), versionTwoInput); !errors.Is(err, memberships.ErrPlanUnavailable) {
		t.Fatalf("archived plan error=%v", err)
	}

	if err := db.Exec(`UPDATE students SET anonymized_at=NOW() WHERE id = ?`, studentID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE persons SET profile_status='anonymized', name='Aluno anonimizado', phone_e164=NULL, email=NULL WHERE id = ?`, personID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListForStudent(ctx, boxes[0].ID, domain.ID(studentID)); !errors.Is(err, memberships.ErrEnrollmentSubjectIneligible) {
		t.Fatalf("anonymized subject error=%v", err)
	}
	privacyAfter, err := service.ListPrivacy(ctx, boxes[0].ID, domain.ID(personID))
	if err != nil || len(privacyAfter) != len(privacy)+1 {
		t.Fatalf("privacy history after anonymization count=%d want=%d err=%v", len(privacyAfter), len(privacy)+1, err)
	}
}

type failingEnrollmentEventRepository struct {
	portrepo.EnrollmentRepository
}

func (failingEnrollmentEventRepository) CreateEvent(context.Context, *domain.EnrollmentEvent) error {
	return errors.New("forced enrollment event failure")
}
