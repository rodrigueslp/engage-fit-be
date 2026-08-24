package repositories

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
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
