package repositories

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	catalogapp "boxengage/backend/internal/app/catalog"
	schedulingapp "boxengage/backend/internal/app/scheduling"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestSchedulingPostgresIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	boxRepository := NewBoxGormRepository(db)
	boxes := []*domain.Box{
		{Name: "Scheduling Box 1 " + uuid.NewString()[:8], Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now},
		{Name: "Scheduling Box 2 " + uuid.NewString()[:8], Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now},
	}
	owners := []*domain.User{
		{Name: "Owner One", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now},
		{Name: "Owner Two", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for i := range boxes {
		if err := boxRepository.SaveWithOwner(ctx, boxes[i], owners[i]); err != nil {
			t.Fatal(err)
		}
	}
	orgIDs := []string{string(boxes[0].OrganizationID), string(boxes[1].OrganizationID)}
	unitIDs := []string{string(boxes[0].ID), string(boxes[1].ID)}

	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM scheduling_audit_events WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM scheduling_command_requests WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM scheduling_professional_units WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM scheduling_professionals WHERE organization_id IN ?", orgIDs).Error
		_ = db.Exec("DELETE FROM users WHERE box_id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM boxes WHERE id IN ?", unitIDs).Error
		_ = db.Exec("DELETE FROM organizations WHERE id IN ?", orgIDs).Error
	})

	// Seed coach user in Box 1
	coachID := domain.ID(uuid.NewString())
	coachUser := domain.User{
		ID:           coachID,
		BoxID:        boxes[0].ID,
		Name:         "Coach Carlos",
		Email:        uuid.NewString() + "@example.test",
		PasswordHash: "hash",
		Role:         domain.UserRoleCoach,
		Active:       true,
		AuthVersion:  1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	userRepo := NewUserGormRepository(db)
	if err := userRepo.Save(ctx, &coachUser); err != nil {
		t.Fatal(err)
	}

	txManager := NewGormTransactionManager(db)
	schedulingRepo := NewSchedulingGormRepository(db)
	accessAdapter := NewSchedulingAccessGormAdapter(db)
	service := schedulingapp.NewService(schedulingRepo, accessAdapter, accessAdapter, txManager, schedulingapp.WithClock(func() time.Time { return now }))

	// 1. Create professional in Box 1 linked to Coach Carlos
	created, isNew, err := service.Create(ctx, owners[0].ID, boxes[0].ID, "prof.create."+uuid.NewString(), schedulingapp.CreateInput{
		DisplayName:  "  Carlos Silva  ",
		LinkedUserID: &coachID,
	})
	if err != nil {
		t.Fatalf("unexpected error creating professional: %v", err)
	}
	if !isNew || created.DisplayName != "Carlos Silva" || created.DisplayNameKey != "carlos silva" || created.Revision != 1 {
		t.Fatalf("unexpected created professional: %+v", created)
	}

	// 2. Conflict: attempt to link another professional to same coach user in same organization
	_, _, err = service.Create(ctx, owners[0].ID, boxes[0].ID, "prof.create."+uuid.NewString(), schedulingapp.CreateInput{
		DisplayName:  "Carlos Silva Junior",
		LinkedUserID: &coachID,
	})
	if !errors.Is(err, schedulingapp.ErrSchedulingAccountConflict) {
		t.Fatalf("expected ErrSchedulingAccountConflict, got %v", err)
	}

	// 3. Homonym: create another professional with same name without linked account
	homonym, _, err := service.Create(ctx, owners[0].ID, boxes[0].ID, "prof.create."+uuid.NewString(), schedulingapp.CreateInput{
		DisplayName: "Carlos Silva",
	})
	if err != nil {
		t.Fatalf("unexpected error creating homonym: %v", err)
	}
	if homonym.ID == created.ID {
		t.Fatalf("homonym must have a different ID")
	}

	// 4. Cross-tenant isolation: Box 2 owner cannot see Box 1 professionals
	box2List, err := service.ListProfessionals(ctx, owners[1].ID, boxes[1].ID, true, "")
	if err != nil {
		t.Fatalf("box2 list failed: %v", err)
	}
	if len(box2List) != 0 {
		t.Fatalf("box2 must see 0 professionals, got %d", len(box2List))
	}

	// 5. Cross-tenant mutation: Box 2 owner cannot update Box 1 professional
	newName := "Carlos Alterado"
	_, _, err = service.Update(ctx, owners[1].ID, boxes[1].ID, created.ID, "prof.update."+uuid.NewString(), schedulingapp.UpdateInput{
		DisplayName:      &newName,
		ExpectedRevision: 1,
	})
	if !errors.Is(err, schedulingapp.ErrSchedulingProfessionalNotFound) {
		t.Fatalf("expected ErrSchedulingProfessionalNotFound across tenant, got %v", err)
	}

	// 6. Concurrency: concurrent updates with same expected revision (only one succeeds)
	var wg sync.WaitGroup
	results := make([]error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			updateName := "Carlos Concurrent"
			_, _, updateErr := service.Update(ctx, owners[0].ID, boxes[0].ID, created.ID, "prof.update."+uuid.NewString(), schedulingapp.UpdateInput{
				DisplayName:      &updateName,
				ExpectedRevision: 1,
			})
			results[idx] = updateErr
		}()
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for _, res := range results {
		if res == nil {
			successCount++
		} else if errors.Is(res, schedulingapp.ErrSchedulingRevisionConflict) {
			conflictCount++
		}
	}
	if successCount != 1 || conflictCount != 4 {
		t.Fatalf("expected exactly 1 success and 4 conflicts, got %d success, %d conflicts", successCount, conflictCount)
	}

	// Fetch current state
	current, err := service.GetProfessional(ctx, owners[0].ID, boxes[0].ID, created.ID)
	if err != nil {
		t.Fatalf("failed to fetch current professional: %v", err)
	}
	if current.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", current.Revision)
	}

	// 7. Clear linked account
	cleared, _, err := service.Update(ctx, owners[0].ID, boxes[0].ID, created.ID, "prof.clear."+uuid.NewString(), schedulingapp.UpdateInput{
		ClearLinkedUser:  true,
		ExpectedRevision: current.Revision,
	})
	if err != nil {
		t.Fatalf("unexpected error clearing linked account: %v", err)
	}
	if cleared.LinkedUserID != nil || cleared.Revision != 3 {
		t.Fatalf("expected cleared account and revision 3, got %+v", cleared)
	}

	// 8. Archive professional
	archived, _, err := service.Archive(ctx, owners[0].ID, boxes[0].ID, created.ID, "prof.arch."+uuid.NewString(), schedulingapp.ArchiveInput{
		ExpectedRevision: 3,
		Reason:           "Desligamento da academia",
	})
	if err != nil {
		t.Fatalf("unexpected error archiving: %v", err)
	}
	if archived.Status != domain.ProfessionalStatusArchived || archived.Revision != 4 {
		t.Fatalf("expected archived status and revision 4, got %+v", archived)
	}

	// 9. Anonymize professional
	anonymized, _, err := service.Anonymize(ctx, owners[0].ID, boxes[0].ID, created.ID, "prof.anon."+uuid.NewString(), schedulingapp.AnonymizeInput{
		ExpectedRevision: 4,
		Reason:           "Solicitação LGPD de anonimização",
	})
	if err != nil {
		t.Fatalf("unexpected error anonymizing: %v", err)
	}
	if anonymized.DisplayName != domain.AnonymizedProfessionalName || anonymized.Revision != 5 || anonymized.AnonymizedAt == nil {
		t.Fatalf("expected anonymized professional: %+v", anonymized)
	}

	// 10. Privacy export
	export, err := service.ExportPrivacy(ctx, owners[0].ID, boxes[0].ID, created.ID)
	if err != nil {
		t.Fatalf("unexpected error exporting privacy: %v", err)
	}
	if export.Professional.ID != created.ID || len(export.Events) < 4 {
		t.Fatalf("unexpected privacy export: %+v", export)
	}

	// Verify no PII in audit data
	for _, ev := range export.Events {
		if ev.BeforeData != nil {
			if _, ok := ev.BeforeData["display_name"]; ok {
				t.Fatalf("audit event BeforeData contained display_name: %+v", ev)
			}
			if _, ok := ev.BeforeData["linked_user_id"]; ok {
				t.Fatalf("audit event BeforeData contained linked_user_id: %+v", ev)
			}
		}
		if ev.AfterData != nil {
			if _, ok := ev.AfterData["display_name"]; ok {
				t.Fatalf("audit event AfterData contained display_name: %+v", ev)
			}
			if _, ok := ev.AfterData["linked_user_id"]; ok {
				t.Fatalf("audit event AfterData contained linked_user_id: %+v", ev)
			}
		}
	}
}

func TestSchedulingGridPostgresIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	if err := db.Exec("DELETE FROM scheduling_materialization_runs WHERE execution_key = ?", "materialize:2026-09-15").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM scheduling_materialization_runs WHERE execution_key = ?", "materialize:2026-09-15").Error
	})
	box := &domain.Box{Name: "Schedule Grid " + uuid.NewString()[:8], Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner := &domain.User{Name: "Owner Grid", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := NewBoxGormRepository(db).SaveWithOwner(ctx, box, owner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		org := string(box.OrganizationID)
		unit := string(box.ID)
		_ = db.Exec("DELETE FROM scheduling_session_events WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_audit_events WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_command_requests WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_sessions WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_series_weekdays WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_series WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_professional_units WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM scheduling_professionals WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM catalog_audit_events WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM unit_modalities WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM modalities WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM user_unit_access WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM organization_user_memberships WHERE organization_id = ?", org).Error
		_ = db.Exec("DELETE FROM users WHERE box_id = ?", unit).Error
		_ = db.Exec("DELETE FROM boxes WHERE id = ?", unit).Error
		_ = db.Exec("DELETE FROM organizations WHERE id = ?", org).Error
	})
	tx := NewGormTransactionManager(db)
	catalogRepo := NewCatalogGormRepository(db)
	modality, err := catalogapp.NewService(catalogRepo, tx).CreateModality(ctx, box.ID, owner.ID, "CrossFit", "#0F766E")
	if err != nil {
		t.Fatal(err)
	}
	schedulingRepo := NewSchedulingGormRepository(db)
	access := NewSchedulingAccessGormAdapter(db)
	professionalService := schedulingapp.NewService(schedulingRepo, access, access, tx, schedulingapp.WithClock(func() time.Time { return now }))
	professional, _, err := professionalService.Create(ctx, owner.ID, box.ID, "prof.grid."+uuid.NewString(), schedulingapp.CreateInput{DisplayName: "Marina"})
	if err != nil {
		t.Fatal(err)
	}
	grid := schedulingapp.NewGridService(schedulingRepo, access, catalogRepo, tx, schedulingapp.WithGridClock(func() time.Time { return now }))
	input := schedulingapp.SeriesMutationInput{TargetKind: domain.ScheduleTargetModality, TargetID: modality.ID, ProfessionalID: professional.ID, DeliveryMode: domain.ScheduleDeliveryGroup, Capacity: 16, DurationMinutes: 60, Weekdays: []int{0, 1, 2, 3, 4, 5, 6}, LocalStartTime: "18:30", Timezone: "America/Sao_Paulo", EffectiveFrom: "2026-09-15"}
	series, fresh, err := grid.CreateSeries(ctx, owner.ID, box.ID, "series.create."+uuid.NewString(), input)
	if err != nil || !fresh {
		t.Fatalf("create series fresh=%v err=%v", fresh, err)
	}
	preview, err := grid.PreviewGrid(ctx, owner.ID, box.ID, "2026-09-15", "2026-09-28")
	if err != nil {
		t.Fatal(err)
	}
	if !preview.ReadyForFuturePublication || len(preview.Occurrences) != 14 {
		t.Fatalf("preview=%+v", preview)
	}
	published, fresh, err := grid.PublishSeries(ctx, owner.ID, box.ID, series.ID, "series.publish."+uuid.NewString(), series.Revision)
	if err != nil || !fresh {
		t.Fatalf("publish fresh=%v err=%v", fresh, err)
	}
	if published.Series.Status != domain.ScheduleSeriesPublished || len(published.Sessions) != 56 {
		t.Fatalf("published status=%s sessions=%d", published.Series.Status, len(published.Sessions))
	}
	materialized, err := grid.Materialize(ctx)
	if err != nil || materialized != 0 {
		t.Fatalf("idempotent materialization created=%d err=%v", materialized, err)
	}
	materialized, err = grid.Materialize(ctx)
	if err != nil || materialized != 0 {
		t.Fatalf("replayed materialization created=%d err=%v", materialized, err)
	}
	if err := db.Exec("UPDATE scheduling_materialization_runs SET status = 'failed', error_code = 'retry_test' WHERE execution_key = ?", "materialize:2026-09-15").Error; err != nil {
		t.Fatal(err)
	}
	materialized, err = grid.Materialize(ctx)
	if err != nil || materialized != 0 {
		t.Fatalf("retried materialization created=%d err=%v", materialized, err)
	}
	changeInput := input
	changeInput.LocalStartTime = "19:30"
	changeInput.ExpectedRevision = published.Series.Revision
	changedSeries, fresh, err := grid.ChangeSeries(ctx, owner.ID, box.ID, series.ID, "series.change."+uuid.NewString(), schedulingapp.SeriesChangeInput{SeriesMutationInput: changeInput, EffectiveOn: "2026-09-16"})
	if err != nil || !fresh {
		t.Fatalf("change series fresh=%v err=%v", fresh, err)
	}
	if changedSeries.Series.Revision != published.Series.Revision+1 || len(changedSeries.Sessions) != 55 {
		t.Fatalf("changed series revision=%d sessions=%d", changedSeries.Series.Revision, len(changedSeries.Sessions))
	}
	conflicting, _, err := grid.CreateSeries(ctx, owner.ID, box.ID, "series.conflict."+uuid.NewString(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = grid.PublishSeries(ctx, owner.ID, box.ID, conflicting.ID, "series.conflict.publish."+uuid.NewString(), conflicting.Revision); !errors.Is(err, schedulingapp.ErrScheduleConflict) {
		t.Fatalf("expected publication conflict, got %v", err)
	}
	concurrentInput := input
	concurrentInput.LocalStartTime = "10:00"
	firstConcurrent, _, err := grid.CreateSeries(ctx, owner.ID, box.ID, "series.concurrent.first."+uuid.NewString(), concurrentInput)
	if err != nil {
		t.Fatal(err)
	}
	secondConcurrent, _, err := grid.CreateSeries(ctx, owner.ID, box.ID, "series.concurrent.second."+uuid.NewString(), concurrentInput)
	if err != nil {
		t.Fatal(err)
	}
	type publishAttempt struct {
		result *schedulingapp.PublishResult
		err    error
	}
	attempts := make(chan publishAttempt, 2)
	for _, candidate := range []*domain.ScheduleSeries{firstConcurrent, secondConcurrent} {
		candidate := candidate
		go func() {
			result, _, publishErr := grid.PublishSeries(ctx, owner.ID, box.ID, candidate.ID, "series.concurrent.publish."+uuid.NewString(), candidate.Revision)
			attempts <- publishAttempt{result: result, err: publishErr}
		}()
	}
	successes, conflicts := 0, 0
	for range 2 {
		attempt := <-attempts
		switch {
		case attempt.err == nil && attempt.result != nil && len(attempt.result.Sessions) == 56:
			successes++
		case errors.Is(attempt.err, schedulingapp.ErrScheduleConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent publication result=%+v err=%v", attempt.result, attempt.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent publications successes=%d conflicts=%d", successes, conflicts)
	}
	listed, err := grid.ListSessions(ctx, owner.ID, box.ID, now, now.AddDate(0, 0, 7), false)
	if err != nil || len(listed) == 0 {
		t.Fatalf("list sessions=%d err=%v", len(listed), err)
	}
	first := listed[0]
	changed, _, err := grid.ChangeSession(ctx, owner.ID, box.ID, first.ID, "session.change."+uuid.NewString(), schedulingapp.SessionMutationInput{TargetKind: first.TargetKind, TargetID: first.TargetID, ProfessionalID: first.ProfessionalID, SpaceID: first.SpaceID, DeliveryMode: first.DeliveryMode, Capacity: first.Capacity, StartsAt: first.StartsAt.Add(30 * time.Minute), DurationMinutes: 60, Timezone: first.Timezone, ExpectedRevision: first.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if !changed.ManualOverride || changed.Revision != 2 {
		t.Fatalf("changed=%+v", changed)
	}
	cancelled, _, err := grid.CancelSession(ctx, owner.ID, box.ID, changed.ID, "session.cancel."+uuid.NewString(), schedulingapp.RevisionReasonInput{ExpectedRevision: changed.Revision, Reason: "Ajuste operacional"})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != domain.ScheduleSessionCancelled {
		t.Fatalf("cancelled=%+v", cancelled)
	}
	events, err := grid.ListSessionEvents(ctx, box.OrganizationID, box.ID, 0, 100)
	if err != nil || len(events) < 58 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
}
