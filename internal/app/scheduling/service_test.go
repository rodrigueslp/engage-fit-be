package scheduling

import (
	"context"
	"errors"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
)

type mockSchedulingRepo struct {
	professionals map[domain.ID]*domain.Professional
	commands      map[string]*domain.SchedulingCommandRequest
	auditEvents   []domain.SchedulingAuditEvent
	unitActive    map[string]bool
}

func newMockRepo() *mockSchedulingRepo {
	return &mockSchedulingRepo{
		professionals: make(map[domain.ID]*domain.Professional),
		commands:      make(map[string]*domain.SchedulingCommandRequest),
		auditEvents:   nil,
		unitActive:    make(map[string]bool),
	}
}

func (m *mockSchedulingRepo) LockCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) error {
	return nil
}

func (m *mockSchedulingRepo) FindCommand(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.SchedulingCommandRequest, error) {
	key := string(organizationID) + ":" + command + ":" + idempotencyKey
	return m.commands[key], nil
}

func (m *mockSchedulingRepo) CreateCommand(ctx context.Context, request *domain.SchedulingCommandRequest) error {
	key := string(request.OrganizationID) + ":" + request.Command + ":" + request.IdempotencyKey
	m.commands[key] = request
	return nil
}

func (m *mockSchedulingRepo) LockProfessional(ctx context.Context, organizationID, professionalID domain.ID) error {
	return nil
}

func (m *mockSchedulingRepo) LockAccount(ctx context.Context, organizationID, userID domain.ID) error {
	return nil
}

func (m *mockSchedulingRepo) ListProfessionals(ctx context.Context, organizationID, unitID domain.ID, includeArchived bool, query string) ([]domain.Professional, error) {
	var list []domain.Professional
	for _, p := range m.professionals {
		if p.OrganizationID != organizationID {
			continue
		}
		if !includeArchived && p.Status != domain.ProfessionalStatusActive {
			continue
		}
		if query != "" && !stringsContains(p.DisplayNameKey, stringsToLower(query)) {
			continue
		}
		copy := *p
		copy.ActiveInUnit = m.unitActive[string(unitID)+":"+string(p.ID)]
		list = append(list, copy)
	}
	return list, nil
}

func (m *mockSchedulingRepo) GetByID(ctx context.Context, organizationID, unitID, professionalID domain.ID) (*domain.Professional, error) {
	p, ok := m.professionals[professionalID]
	if !ok || p.OrganizationID != organizationID {
		return nil, portrepo.ErrSchedulingProfessionalNotFound
	}
	copy := *p
	copy.ActiveInUnit = m.unitActive[string(unitID)+":"+string(p.ID)]
	return &copy, nil
}

func (m *mockSchedulingRepo) GetByIDForUpdate(ctx context.Context, organizationID, professionalID domain.ID) (*domain.Professional, error) {
	p, ok := m.professionals[professionalID]
	if !ok || p.OrganizationID != organizationID {
		return nil, portrepo.ErrSchedulingProfessionalNotFound
	}
	copy := *p
	return &copy, nil
}

func (m *mockSchedulingRepo) FindByLinkedUser(ctx context.Context, organizationID, userID domain.ID) (*domain.Professional, error) {
	for _, p := range m.professionals {
		if p.OrganizationID == organizationID && p.LinkedUserID != nil && *p.LinkedUserID == userID {
			copy := *p
			return &copy, nil
		}
	}
	return nil, nil
}

func (m *mockSchedulingRepo) CreateProfessional(ctx context.Context, professional *domain.Professional, unitID domain.ID) error {
	copy := *professional
	m.professionals[professional.ID] = &copy
	m.unitActive[string(unitID)+":"+string(professional.ID)] = true
	return nil
}

func (m *mockSchedulingRepo) UpdateProfessional(ctx context.Context, professional *domain.Professional) error {
	copy := *professional
	m.professionals[professional.ID] = &copy
	return nil
}

func (m *mockSchedulingRepo) SetUnitActive(ctx context.Context, organizationID, unitID, professionalID domain.ID, active bool) error {
	m.unitActive[string(unitID)+":"+string(professionalID)] = active
	return nil
}

func (m *mockSchedulingRepo) CreateAuditEvent(ctx context.Context, event *domain.SchedulingAuditEvent) error {
	m.auditEvents = append(m.auditEvents, *event)
	return nil
}

func (m *mockSchedulingRepo) ListAuditEventsForProfessional(ctx context.Context, organizationID, unitID, professionalID domain.ID) ([]domain.SchedulingAuditEvent, error) {
	var list []domain.SchedulingAuditEvent
	for _, ev := range m.auditEvents {
		if ev.OrganizationID == organizationID && ev.EntityID == professionalID {
			list = append(list, ev)
		}
	}
	return list, nil
}

func (m *mockSchedulingRepo) CountActiveInUnit(ctx context.Context, organizationID, unitID domain.ID) (int64, error) {
	var count int64
	for _, p := range m.professionals {
		if p.OrganizationID == organizationID && p.Status == domain.ProfessionalStatusActive && m.unitActive[string(unitID)+":"+string(p.ID)] {
			count++
		}
	}
	return count, nil
}

type mockContextResolver struct {
	orgID      domain.ID
	unitID     domain.ID
	role       string
	authorized bool
}

func (m mockContextResolver) ResolveSchedulingUnitContext(ctx context.Context, authenticatedUserID, unitID domain.ID) (*domain.SchedulingUnitContext, error) {
	if unitID != m.unitID {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}
	return &domain.SchedulingUnitContext{
		OrganizationID: m.orgID,
		UnitID:         m.unitID,
		UnitName:       "Unit 1",
		Role:           m.role,
		Authorized:     m.authorized,
	}, nil
}

type mockAccountResolver struct {
	candidates []domain.ProfessionalAccountCandidate
}

func (m mockAccountResolver) ListProfessionalAccountCandidates(ctx context.Context, organizationID, unitID domain.ID) ([]domain.ProfessionalAccountCandidate, error) {
	return m.candidates, nil
}

func (m mockAccountResolver) ResolveProfessionalAccount(ctx context.Context, organizationID, unitID, userID domain.ID) (*domain.ProfessionalAccountCandidate, error) {
	for _, c := range m.candidates {
		if c.UserID == userID {
			return &c, nil
		}
	}
	return nil, nil
}

type mockTxManager struct{}

func (m mockTxManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func stringsContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func setupTestService(role string) (*Service, *mockSchedulingRepo, domain.ID, domain.ID, domain.ID) {
	orgID := domain.ID("org-1")
	unitID := domain.ID("unit-1")
	userID := domain.ID("user-1")

	repo := newMockRepo()
	ctxResolver := mockContextResolver{orgID: orgID, unitID: unitID, role: role, authorized: true}
	accResolver := mockAccountResolver{
		candidates: []domain.ProfessionalAccountCandidate{
			{UserID: "coach-1", DisplayName: "Coach Pedro", Role: "COACH", Active: true, AuthorizedInUnit: true},
			{UserID: "coach-2", DisplayName: "Coach Inactive", Role: "COACH", Active: false, AuthorizedInUnit: true},
		},
	}
	tx := mockTxManager{}
	fixedTime := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	svc := NewService(repo, ctxResolver, accResolver, tx, WithClock(func() time.Time { return fixedTime }))

	return svc, repo, orgID, unitID, userID
}

func TestSchedulingLifecycle(t *testing.T) {
	svc, repo, _, unitID, userID := setupTestService("OWNER")
	ctx := context.Background()

	// 1. Create professional with name and linked account
	coachID := domain.ID("coach-1")
	created, isNew, err := svc.Create(ctx, userID, unitID, "idemp-key-create-001", CreateInput{
		DisplayName:  "  Marina  Souza  ",
		LinkedUserID: &coachID,
	})
	if err != nil {
		t.Fatalf("unexpected error creating professional: %v", err)
	}
	if !isNew {
		t.Fatalf("expected isNew to be true")
	}
	if created.DisplayName != "Marina Souza" || created.DisplayNameKey != "marina souza" {
		t.Errorf("unexpected display name: %q", created.DisplayName)
	}
	if created.Revision != 1 || created.Status != domain.ProfessionalStatusActive {
		t.Errorf("unexpected revision %d or status %s", created.Revision, created.Status)
	}
	if created.LinkedUserID == nil || *created.LinkedUserID != coachID {
		t.Errorf("expected linked user to be coach-1")
	}

	// 2. Replay create with same key and payload
	replay, isNewReplay, err := svc.Create(ctx, userID, unitID, "idemp-key-create-001", CreateInput{
		DisplayName:  "Marina Souza",
		LinkedUserID: &coachID,
	})
	if err != nil {
		t.Fatalf("unexpected error on replay create: %v", err)
	}
	if isNewReplay {
		t.Fatalf("expected isNew to be false on replay")
	}
	if replay.ID != created.ID {
		t.Errorf("expected same ID on replay")
	}

	// 3. Conflict on create with duplicate account
	otherName := "Marina Dois"
	_, _, err = svc.Create(ctx, userID, unitID, "idemp-key-create-002", CreateInput{
		DisplayName:  otherName,
		LinkedUserID: &coachID,
	})
	if !errors.Is(err, ErrSchedulingAccountConflict) {
		t.Fatalf("expected ErrSchedulingAccountConflict, got %v", err)
	}

	// 4. Create homonym without linked account (should succeed and have different ID)
	homonym, _, err := svc.Create(ctx, userID, unitID, "idemp-key-create-003", CreateInput{
		DisplayName: "Marina Souza",
	})
	if err != nil {
		t.Fatalf("unexpected error creating homonym: %v", err)
	}
	if homonym.ID == created.ID {
		t.Fatalf("homonym must have a different ID")
	}

	// 5. Update professional name
	newName := "Marina Souza Silva"
	updated, _, err := svc.Update(ctx, userID, unitID, created.ID, "idemp-key-update-001", UpdateInput{
		DisplayName:      &newName,
		ExpectedRevision: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}
	if updated.Revision != 2 || updated.DisplayName != newName {
		t.Errorf("unexpected update result: rev=%d, name=%q", updated.Revision, updated.DisplayName)
	}

	// 6. Update revision conflict
	_, _, err = svc.Update(ctx, userID, unitID, created.ID, "idemp-key-update-002", UpdateInput{
		DisplayName:      &newName,
		ExpectedRevision: 1, // outdated
	})
	if !errors.Is(err, ErrSchedulingRevisionConflict) {
		t.Fatalf("expected ErrSchedulingRevisionConflict, got %v", err)
	}

	// 7. Clear linked account
	updated2, _, err := svc.Update(ctx, userID, unitID, created.ID, "idemp-key-update-003", UpdateInput{
		ClearLinkedUser:  true,
		ExpectedRevision: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error clearing linked user: %v", err)
	}
	if updated2.Revision != 3 || updated2.LinkedUserID != nil {
		t.Errorf("expected cleared linked user and revision 3, got rev=%d, linked=%v", updated2.Revision, updated2.LinkedUserID)
	}

	// 8. Archive professional
	archived, _, err := svc.Archive(ctx, userID, unitID, created.ID, "idemp-key-arch-001", ArchiveInput{
		ExpectedRevision: 3,
		Reason:           "Mudança de cidade",
	})
	if err != nil {
		t.Fatalf("unexpected error archiving: %v", err)
	}
	if archived.Status != domain.ProfessionalStatusArchived || archived.Revision != 4 || archived.ActiveInUnit {
		t.Errorf("unexpected archive result: status=%s, rev=%d, activeInUnit=%v", archived.Status, archived.Revision, archived.ActiveInUnit)
	}

	// 9. Reactivate professional
	reactivated, _, err := svc.Reactivate(ctx, userID, unitID, created.ID, "idemp-key-react-001", ReactivateInput{
		ExpectedRevision: 4,
		Reason:           "Retorno ao trabalho",
	})
	if err != nil {
		t.Fatalf("unexpected error reactivating: %v", err)
	}
	if reactivated.Status != domain.ProfessionalStatusActive || reactivated.Revision != 5 || !reactivated.ActiveInUnit {
		t.Errorf("unexpected reactivate result: status=%s, rev=%d, activeInUnit=%v", reactivated.Status, reactivated.Revision, reactivated.ActiveInUnit)
	}

	// 10. Archive again before anonymizing
	_, _, err = svc.Archive(ctx, userID, unitID, created.ID, "idemp-key-arch-002", ArchiveInput{
		ExpectedRevision: 5,
		Reason:           "Desligamento permanente",
	})
	if err != nil {
		t.Fatalf("unexpected error archiving: %v", err)
	}

	// 11. Anonymize professional
	anonymized, _, err := svc.Anonymize(ctx, userID, unitID, created.ID, "idemp-key-anon-001", AnonymizeInput{
		ExpectedRevision: 6,
		Reason:           "Solicitação LGPD de eliminação",
	})
	if err != nil {
		t.Fatalf("unexpected error anonymizing: %v", err)
	}
	if anonymized.DisplayName != domain.AnonymizedProfessionalName || anonymized.Revision != 7 || anonymized.AnonymizedAt == nil {
		t.Errorf("unexpected anonymize result: name=%q, rev=%d, anonAt=%v", anonymized.DisplayName, anonymized.Revision, anonymized.AnonymizedAt)
	}

	// 12. Attempt to reactivate or update anonymized professional (must fail)
	_, _, err = svc.Reactivate(ctx, userID, unitID, created.ID, "idemp-key-react-002", ReactivateInput{
		ExpectedRevision: 7,
		Reason:           "Tentativa inválida",
	})
	if !errors.Is(err, ErrSchedulingProfessionalAnonymized) {
		t.Fatalf("expected ErrSchedulingProfessionalAnonymized, got %v", err)
	}

	// 13. Check audit events do not contain names or linked user IDs in data
	for _, ev := range repo.auditEvents {
		if ev.BeforeData != nil {
			if _, ok := ev.BeforeData["display_name"]; ok {
				t.Errorf("audit BeforeData must not contain display_name")
			}
			if _, ok := ev.BeforeData["linked_user_id"]; ok {
				t.Errorf("audit BeforeData must not contain linked_user_id")
			}
		}
		if ev.AfterData != nil {
			if _, ok := ev.AfterData["display_name"]; ok {
				t.Errorf("audit AfterData must not contain display_name")
			}
			if _, ok := ev.AfterData["linked_user_id"]; ok {
				t.Errorf("audit AfterData must not contain linked_user_id")
			}
		}
	}

	// 14. Export privacy
	export, err := svc.ExportPrivacy(ctx, userID, unitID, created.ID)
	if err != nil {
		t.Fatalf("unexpected error exporting privacy: %v", err)
	}
	if export.Professional.ID != created.ID || len(export.Events) == 0 {
		t.Errorf("unexpected privacy export: %v", export)
	}
}

func TestCoachReadOnly(t *testing.T) {
	svc, repo, orgID, unitID, userID := setupTestService("COACH")
	ctx := context.Background()

	// Seed one active professional with linked user and one archived professional
	coachID := domain.ID("coach-1")
	profActive := domain.Professional{
		ID:             "prof-active",
		OrganizationID: orgID,
		DisplayName:    "Coach Ativo",
		DisplayNameKey: "coach ativo",
		LinkedUserID:   &coachID,
		Status:         domain.ProfessionalStatusActive,
		Revision:       1,
		ActiveInUnit:   true,
	}
	profArchived := domain.Professional{
		ID:             "prof-archived",
		OrganizationID: orgID,
		DisplayName:    "Coach Arquivado",
		DisplayNameKey: "coach arquivado",
		Status:         domain.ProfessionalStatusArchived,
		Revision:       1,
		ActiveInUnit:   false,
	}
	_ = repo.CreateProfessional(ctx, &profActive, unitID)
	_ = repo.CreateProfessional(ctx, &profArchived, unitID)
	_ = repo.SetUnitActive(ctx, orgID, unitID, "prof-archived", false)

	// Coach lists professionals
	list, err := svc.ListProfessionals(ctx, userID, unitID, true, "")
	if err != nil {
		t.Fatalf("coach list failed: %v", err)
	}
	// Coach should only see active, and linked user ID must be nil
	if len(list) != 1 {
		t.Fatalf("expected 1 active professional for coach, got %d", len(list))
	}
	if list[0].ID != "prof-active" {
		t.Errorf("expected prof-active, got %s", list[0].ID)
	}
	if list[0].LinkedUserID != nil {
		t.Errorf("coach response must not expose LinkedUserID")
	}

	// Coach mutations must be rejected
	_, _, err = svc.Create(ctx, userID, unitID, "idemp-coach-create", CreateInput{
		DisplayName: "Coach Tentativa",
	})
	if !errors.Is(err, ErrSchedulingUnitNotFound) {
		t.Errorf("expected coach create to be rejected with ErrSchedulingUnitNotFound, got %v", err)
	}

	_, err = svc.ListCandidates(ctx, userID, unitID)
	if !errors.Is(err, ErrSchedulingUnitNotFound) {
		t.Errorf("expected coach list candidates to be rejected with ErrSchedulingUnitNotFound, got %v", err)
	}
}
