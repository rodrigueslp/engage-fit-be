package contracts

import (
	"context"
	"errors"
	"testing"
	"time"

	contractsadapter "boxengage/backend/internal/adapters/contracts"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
)

type mockContractRepository struct {
	templates        map[string]domain.ContractTemplate
	templateVersions map[string]domain.ContractTemplateVersion
	instances        map[string]domain.ContractInstance
	sessions         map[string]domain.ContractAcceptanceSession
	sessionByHash    map[string]domain.ContractAcceptanceSession
	commands         map[string]domain.ContractCommandRequest
	events           []domain.ContractEvent
	auditEvents      []domain.ContractTemplateAuditEvent
}

func newMockContractRepository() *mockContractRepository {
	return &mockContractRepository{
		templates:        make(map[string]domain.ContractTemplate),
		templateVersions: make(map[string]domain.ContractTemplateVersion),
		instances:        make(map[string]domain.ContractInstance),
		sessions:         make(map[string]domain.ContractAcceptanceSession),
		sessionByHash:    make(map[string]domain.ContractAcceptanceSession),
		commands:         make(map[string]domain.ContractCommandRequest),
	}
}

func (m *mockContractRepository) FindTemplateByID(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) (*domain.ContractTemplate, error) {
	t, ok := m.templates[string(templateID)]
	if !ok || t.OrganizationID != organizationID {
		return nil, portrepo.ErrContractTemplateNotFound
	}
	// attach versions
	copyT := t
	copyT.Versions = nil
	for _, v := range m.templateVersions {
		if v.TemplateID == t.ID {
			copyT.Versions = append(copyT.Versions, v)
			if v.Status == domain.ContractTemplateVersionStatusDraft {
				copyV := v
				copyT.CurrentDraft = &copyV
			}
			if v.Status == domain.ContractTemplateVersionStatusPublished {
				copyV := v
				copyT.CurrentVersion = &copyV
			}
		}
	}
	return &copyT, nil
}

func (m *mockContractRepository) FindTemplateByNameKey(ctx context.Context, organizationID domain.ID, nameKey string) (*domain.ContractTemplate, error) {
	for _, t := range m.templates {
		if t.OrganizationID == organizationID && t.NameKey == nameKey {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockContractRepository) ListTemplates(ctx context.Context, organizationID, homeUnitID domain.ID, includeArchived bool) ([]domain.ContractTemplate, error) {
	var list []domain.ContractTemplate
	for _, t := range m.templates {
		if t.OrganizationID == organizationID && t.HomeUnitID == homeUnitID {
			if !includeArchived && t.Status == domain.ContractTemplateStatusArchived {
				continue
			}
			full, _ := m.FindTemplateByID(ctx, organizationID, homeUnitID, t.ID)
			list = append(list, *full)
		}
	}
	return list, nil
}

func (m *mockContractRepository) SaveTemplateDraft(ctx context.Context, template *domain.ContractTemplate, draft *domain.ContractTemplateVersion) error {
	m.templates[string(template.ID)] = *template
	m.templateVersions[string(draft.ID)] = *draft
	return nil
}

func (m *mockContractRepository) UpdateTemplateDraft(ctx context.Context, organizationID, homeUnitID, templateID domain.ID, draft *domain.ContractTemplateVersion, expectedRevision int) error {
	for id, v := range m.templateVersions {
		if v.TemplateID == templateID && v.Status == domain.ContractTemplateVersionStatusDraft {
			if v.Revision != expectedRevision {
				return portrepo.ErrContractTemplateRevisionConflict
			}
			v.Title = draft.Title
			v.BodyMarkdown = draft.BodyMarkdown
			v.AcceptanceStatement = draft.AcceptanceStatement
			v.Revision++
			m.templateVersions[id] = v
			return nil
		}
	}
	return portrepo.ErrContractTemplateNotFound
}

func (m *mockContractRepository) PublishTemplateVersion(ctx context.Context, organizationID, homeUnitID, templateID, versionID domain.ID, draftRevision int, publishedAt time.Time, contentSHA256 string) error {
	for id, v := range m.templateVersions {
		if v.TemplateID == templateID && v.Status == domain.ContractTemplateVersionStatusPublished {
			v.Status = domain.ContractTemplateVersionStatusRetired
			m.templateVersions[id] = v
		}
	}
	v, ok := m.templateVersions[string(versionID)]
	if !ok || v.Revision != draftRevision {
		return portrepo.ErrContractTemplateRevisionConflict
	}
	v.Status = domain.ContractTemplateVersionStatusPublished
	v.PublishedAt = &publishedAt
	v.ContentSHA256 = contentSHA256
	v.Revision++
	m.templateVersions[string(versionID)] = v
	return nil
}

func (m *mockContractRepository) CreateNewTemplateVersion(ctx context.Context, organizationID, homeUnitID, templateID domain.ID, newDraft *domain.ContractTemplateVersion) error {
	m.templateVersions[string(newDraft.ID)] = *newDraft
	return nil
}

func (m *mockContractRepository) ArchiveTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) error {
	t, ok := m.templates[string(templateID)]
	if !ok {
		return portrepo.ErrContractTemplateNotFound
	}
	t.Status = domain.ContractTemplateStatusArchived
	m.templates[string(templateID)] = t
	return nil
}

func (m *mockContractRepository) ReactivateTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) error {
	t, ok := m.templates[string(templateID)]
	if !ok {
		return portrepo.ErrContractTemplateNotFound
	}
	t.Status = domain.ContractTemplateStatusActive
	m.templates[string(templateID)] = t
	return nil
}

func (m *mockContractRepository) FindTemplateVersionByID(ctx context.Context, organizationID, versionID domain.ID) (*domain.ContractTemplateVersion, error) {
	v, ok := m.templateVersions[string(versionID)]
	if !ok || v.OrganizationID != organizationID {
		return nil, portrepo.ErrContractTemplateNotFound
	}
	return &v, nil
}

func (m *mockContractRepository) FindInstanceByID(ctx context.Context, organizationID, unitID, instanceID domain.ID) (*domain.ContractInstance, error) {
	inst, ok := m.instances[string(instanceID)]
	if !ok || inst.OrganizationID != organizationID || inst.UnitID != unitID {
		return nil, portrepo.ErrContractInstanceNotFound
	}
	return &inst, nil
}

func (m *mockContractRepository) FindActiveInstanceByEnrollmentID(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.ContractInstance, error) {
	for _, inst := range m.instances {
		if inst.OrganizationID == organizationID && inst.UnitID == unitID && inst.EnrollmentID == enrollmentID && inst.Status != domain.ContractInstanceStatusDiscarded {
			return &inst, nil
		}
	}
	return nil, nil
}

func (m *mockContractRepository) ListInstancesForEnrollment(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractInstance, error) {
	var list []domain.ContractInstance
	for _, inst := range m.instances {
		if inst.OrganizationID == organizationID && inst.UnitID == unitID && inst.EnrollmentID == enrollmentID {
			list = append(list, inst)
		}
	}
	return list, nil
}

func (m *mockContractRepository) ListInstancesForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) ([]domain.ContractInstance, error) {
	var list []domain.ContractInstance
	for _, inst := range m.instances {
		if inst.OrganizationID == organizationID && inst.UnitID == unitID && inst.PersonID == personID {
			list = append(list, inst)
		}
	}
	return list, nil
}

func (m *mockContractRepository) SaveInstanceDraft(ctx context.Context, instance *domain.ContractInstance) error {
	m.instances[string(instance.ID)] = *instance
	return nil
}

func (m *mockContractRepository) IssueInstance(ctx context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, issuedAt time.Time) (*domain.ContractInstance, error) {
	inst, ok := m.instances[string(instanceID)]
	if !ok {
		return nil, portrepo.ErrContractInstanceNotFound
	}
	if inst.Revision != expectedRevision {
		return nil, portrepo.ErrContractRevisionConflict
	}
	inst.Status = domain.ContractInstanceStatusPendingAcceptance
	inst.IssuedAt = &issuedAt
	inst.IssuedByUserID = &actorUserID
	inst.Revision++
	m.instances[string(instanceID)] = inst
	return &inst, nil
}

func (m *mockContractRepository) DiscardInstance(ctx context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, discardedAt time.Time) (*domain.ContractInstance, error) {
	inst, ok := m.instances[string(instanceID)]
	if !ok {
		return nil, portrepo.ErrContractInstanceNotFound
	}
	if inst.Revision != expectedRevision {
		return nil, portrepo.ErrContractRevisionConflict
	}
	inst.Status = domain.ContractInstanceStatusDiscarded
	inst.DiscardedAt = &discardedAt
	inst.DiscardedByUserID = &actorUserID
	inst.Revision++
	m.instances[string(instanceID)] = inst
	return &inst, nil
}

func (m *mockContractRepository) AcceptInstance(ctx context.Context, organizationID, unitID, instanceID, sessionID domain.ID, acceptedAt time.Time, evidence map[string]any, adapter, adapterRef, docSHA256 string, statementVersion int, personID domain.ID) (*domain.ContractInstance, error) {
	inst, ok := m.instances[string(instanceID)]
	if !ok {
		return nil, portrepo.ErrContractInstanceNotFound
	}
	inst.Status = domain.ContractInstanceStatusAccepted
	inst.AcceptedAt = &acceptedAt
	inst.Revision++
	m.instances[string(instanceID)] = inst
	return &inst, nil
}

func (m *mockContractRepository) FindActiveAcceptanceSessionByInstanceID(ctx context.Context, organizationID, instanceID domain.ID) (*domain.ContractAcceptanceSession, error) {
	for _, s := range m.sessions {
		if s.OrganizationID == organizationID && s.ContractInstanceID == instanceID && s.Status == domain.ContractAcceptanceSessionStatusActive {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockContractRepository) FindAcceptanceSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.ContractAcceptanceSession, *domain.ContractInstance, error) {
	s, ok := m.sessionByHash[tokenHash]
	if !ok {
		return nil, nil, portrepo.ErrContractAcceptanceSessionInvalid
	}
	inst, ok := m.instances[string(s.ContractInstanceID)]
	if !ok {
		return nil, nil, portrepo.ErrContractInstanceNotFound
	}
	return &s, &inst, nil
}

func (m *mockContractRepository) SaveAcceptanceSession(ctx context.Context, session *domain.ContractAcceptanceSession) error {
	m.sessions[string(session.ID)] = *session
	m.sessionByHash[session.TokenHash] = *session
	return nil
}

func (m *mockContractRepository) SupersedeActiveAcceptanceSessions(ctx context.Context, organizationID, instanceID domain.ID) error {
	for id, s := range m.sessions {
		if s.OrganizationID == organizationID && s.ContractInstanceID == instanceID && s.Status == domain.ContractAcceptanceSessionStatusActive {
			s.Status = domain.ContractAcceptanceSessionStatusSuperseded
			m.sessions[id] = s
			m.sessionByHash[s.TokenHash] = s
		}
	}
	return nil
}

func (m *mockContractRepository) FindCommandRequest(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.ContractCommandRequest, error) {
	k := string(organizationID) + ":" + command + ":" + idempotencyKey
	req, ok := m.commands[k]
	if !ok {
		return nil, nil
	}
	return &req, nil
}

func (m *mockContractRepository) SaveCommandRequest(ctx context.Context, req *domain.ContractCommandRequest) error {
	k := string(req.OrganizationID) + ":" + req.Command + ":" + req.IdempotencyKey
	m.commands[k] = *req
	return nil
}

func (m *mockContractRepository) RecordEvent(ctx context.Context, event *domain.ContractEvent) error {
	m.events = append(m.events, *event)
	return nil
}

func (m *mockContractRepository) RecordTemplateAuditEvent(ctx context.Context, event *domain.ContractTemplateAuditEvent) error {
	m.auditEvents = append(m.auditEvents, *event)
	return nil
}

func (m *mockContractRepository) ListPrivacyContracts(ctx context.Context, organizationID, personID domain.ID) ([]domain.ContractPrivacyExport, error) {
	var list []domain.ContractPrivacyExport
	for _, inst := range m.instances {
		if inst.OrganizationID == organizationID && inst.PersonID == personID {
			tmpl := m.templateVersions[string(inst.TemplateVersionID)]
			list = append(list, domain.ContractPrivacyExport{
				Instance: inst,
				Template: tmpl,
			})
		}
	}
	return list, nil
}

func (m *mockContractRepository) PurgeExpiredSimulatorRecords(ctx context.Context, now time.Time, apply bool) (int64, error) {
	var count int64
	for id, inst := range m.instances {
		if inst.RetentionClass == domain.ContractRetentionClassLocalSimulator && inst.RetentionDueAt != nil && !inst.RetentionDueAt.After(now) {
			count++
			if apply {
				delete(m.instances, id)
			}
		}
	}
	return count, nil
}

type mockEnrollmentResolver struct {
	snapshot *domain.ContractEnrollmentSnapshot
	err      error
}

func (m *mockEnrollmentResolver) ResolveContractEnrollmentSnapshot(ctx context.Context, organizationID, unitID, enrollmentID domain.ID, asOf time.Time) (*domain.ContractEnrollmentSnapshot, error) {
	return m.snapshot, m.err
}

type mockPartyResolver struct {
	party *domain.ContractPartyResolution
	err   error
}

func (m *mockPartyResolver) ResolveContractParty(ctx context.Context, organizationID, personID domain.ID) (*domain.ContractPartyResolution, error) {
	return m.party, m.err
}

type mockBoxRepository struct {
	box domain.Box
}

func (m *mockBoxRepository) FindByID(ctx context.Context, id domain.ID) (*domain.Box, error) {
	return &m.box, nil
}
func (m *mockBoxRepository) ListAll(ctx context.Context) ([]domain.Box, error) {
	return []domain.Box{m.box}, nil
}
func (m *mockBoxRepository) Save(ctx context.Context, box *domain.Box) error { return nil }
func (m *mockBoxRepository) SaveWithOwner(ctx context.Context, box *domain.Box, owner *domain.User) error {
	return nil
}
func (m *mockBoxRepository) Update(ctx context.Context, box domain.Box) error { return nil }

type mockTransactionManager struct{}

func (m *mockTransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func setupTestContractsService() (*Service, *mockContractRepository, *mockEnrollmentResolver, *mockPartyResolver) {
	repo := newMockContractRepository()
	enrResolver := &mockEnrollmentResolver{
		snapshot: &domain.ContractEnrollmentSnapshot{
			PersonID:           "40000000-0000-4000-8000-000000000001",
			EnrollmentRevision: 1,
			EffectiveStatus:    domain.EnrollmentEffectiveActive,
			PlanID:             "70000000-0000-4000-8000-000000000001",
			PlanVersionID:      "80000000-0000-4000-8000-000000000001",
			PlanVersionNumber:  1,
			PlanName:           "CrossFit Pro",
			BillingCycle:       domain.MembershipBillingMonthly,
			Currency:           "BRL",
			BasePriceCents:     18900,
			AddonsPriceCents:   5000,
			TotalPriceCents:    23900,
			StartsOn:           time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC),
			BaseSummary:        "CrossFit · livre",
			Addons: []domain.ContractEnrollmentAddonSnapshot{
				{
					PlanVersionAddonID: "b0000000-0000-4000-8000-000000000001",
					Name:               "Recovery",
					PriceDeltaCents:    5000,
					Summary:            "Recovery · 2x",
				},
			},
		},
	}
	partyResolver := &mockPartyResolver{
		party: &domain.ContractPartyResolution{
			PersonID:      "40000000-0000-4000-8000-000000000001",
			DisplayName:   "Ana Souza",
			ProfileStatus: domain.PersonProfileActive,
			Eligible:      true,
		},
	}
	boxes := &mockBoxRepository{
		box: domain.Box{
			ID:             "10000000-0000-4000-8000-000000000001",
			OrganizationID: "10000000-0000-4000-8000-000000000001",
			Name:           "Alados Box",
			Status:         domain.BoxStatusActive,
		},
	}
	simAdapter := contractsadapter.NewLocalSimulatorAdapter("development", true)
	tx := &mockTransactionManager{}

	svc := NewService(repo, enrResolver, partyResolver, boxes, simAdapter, tx)
	return svc, repo, enrResolver, partyResolver
}

func TestContractsServiceLifecycle(t *testing.T) {
	svc, _, _, _ := setupTestContractsService()
	ctx := context.Background()

	orgID := domain.ID("10000000-0000-4000-8000-000000000001")
	unitID := domain.ID("10000000-0000-4000-8000-000000000001")
	ownerID := domain.ID("20000000-0000-4000-8000-000000000001")
	enrollmentID := domain.ID("a0000000-0000-4000-8000-000000000001")

	// 1. Create Template
	tmpl, replayed, err := svc.CreateTemplate(ctx, orgID, unitID, ownerID, "idem_tmpl_01", CreateTemplateInput{
		Name:                "Termo de Adesão",
		Title:               "Termo de Adesão — {{plan_name}}",
		BodyMarkdown:        "# Contrato\n\nAluno: {{member_name}}\nPlano: {{plan_name}} por {{total_price}}\nInício: {{starts_on}}",
		AcceptanceStatement: "Li e concordo com todos os termos deste contrato.",
	})
	if err != nil {
		t.Fatalf("CreateTemplate() err = %v", err)
	}
	if replayed || tmpl == nil || tmpl.CurrentDraft == nil {
		t.Fatalf("expected created draft template, got tmpl=%+v", tmpl)
	}

	// 2. Publish Template
	pubTmpl, _, err := svc.PublishTemplate(ctx, orgID, unitID, tmpl.ID, ownerID, "idem_pub_01", PublishTemplateInput{
		DraftRevision: tmpl.CurrentDraft.Revision,
	})
	if err != nil {
		t.Fatalf("PublishTemplate() err = %v", err)
	}
	if pubTmpl.CurrentVersion == nil || pubTmpl.CurrentVersion.Status != domain.ContractTemplateVersionStatusPublished {
		t.Fatalf("expected published template version")
	}

	// 3. Generate Preview for Enrollment
	preview, err := svc.PreviewEnrollmentContract(ctx, orgID, unitID, enrollmentID, PreviewContractInput{
		TemplateID:        string(tmpl.ID),
		TemplateVersionID: string(pubTmpl.CurrentVersion.ID),
	})
	if err != nil {
		t.Fatalf("PreviewEnrollmentContract() err = %v", err)
	}
	if preview.PreviewSHA256 == "" || preview.DocumentTitle != "Termo de Adesão — CrossFit Pro" {
		t.Fatalf("unexpected preview: %+v", preview)
	}

	// 4. Create Instance Draft
	inst, _, err := svc.CreateInstanceDraft(ctx, orgID, unitID, enrollmentID, ownerID, "idem_inst_01", CreateInstanceDraftInput{
		TemplateID:            string(tmpl.ID),
		TemplateVersionID:     string(pubTmpl.CurrentVersion.ID),
		ExpectedPreviewSHA256: preview.PreviewSHA256,
	})
	if err != nil {
		t.Fatalf("CreateInstanceDraft() err = %v", err)
	}
	if inst.Status != domain.ContractInstanceStatusDraft {
		t.Fatalf("expected draft status, got %s", inst.Status)
	}

	// 5. Issue Instance
	issued, _, err := svc.IssueInstance(ctx, orgID, unitID, inst.ID, ownerID, "idem_issue_01", IssueInstanceInput{
		ExpectedRevision: inst.Revision,
	})
	if err != nil {
		t.Fatalf("IssueInstance() err = %v", err)
	}
	if issued.Status != domain.ContractInstanceStatusPendingAcceptance {
		t.Fatalf("expected pending_acceptance status, got %s", issued.Status)
	}

	// 6. Create Acceptance Session
	launch, _, err := svc.CreateAcceptanceSession(ctx, orgID, unitID, inst.ID, ownerID, "idem_sess_01")
	if err != nil {
		t.Fatalf("CreateAcceptanceSession() err = %v", err)
	}
	if launch.LaunchToken == "" {
		t.Fatalf("expected launch token in session launch")
	}

	// 7. Get Public Acceptance Session
	pubSession, err := svc.GetPublicAcceptanceSession(ctx, launch.LaunchToken)
	if err != nil {
		t.Fatalf("GetPublicAcceptanceSession() err = %v", err)
	}
	if pubSession.DocumentSHA256 != inst.DocumentSHA256 {
		t.Fatalf("hash mismatch in public session: %s vs %s", pubSession.DocumentSHA256, inst.DocumentSHA256)
	}

	// 8. Accept Contract
	accepted, _, err := svc.AcceptContract(ctx, launch.LaunchToken, "idem_acc_01", AcceptContractInput{
		DocumentSHA256:             pubSession.DocumentSHA256,
		AcceptanceStatementVersion: pubSession.AcceptanceStatementVersion,
		Accepted:                   true,
	})
	if err != nil {
		t.Fatalf("AcceptContract() err = %v", err)
	}
	if accepted.Status != domain.ContractInstanceStatusAccepted {
		t.Fatalf("expected accepted status, got %s", accepted.Status)
	}
}

func TestContractsServiceValidationAndErrors(t *testing.T) {
	svc, _, enrResolver, _ := setupTestContractsService()
	ctx := context.Background()

	orgID := domain.ID("10000000-0000-4000-8000-000000000001")
	unitID := domain.ID("10000000-0000-4000-8000-000000000001")
	ownerID := domain.ID("20000000-0000-4000-8000-000000000001")
	enrollmentID := domain.ID("a0000000-0000-4000-8000-000000000001")

	// Disallow HTML tags
	_, _, err := svc.CreateTemplate(ctx, orgID, unitID, ownerID, "idem_inv_01", CreateTemplateInput{
		Name:                "Termo HTML",
		Title:               "Título",
		BodyMarkdown:        "Corpo com <script>alert(1)</script>",
		AcceptanceStatement: "Li e concordo com os termos.",
	})
	if err == nil || !errors.Is(err, ErrInvalidContractInput) {
		t.Fatalf("expected ErrInvalidContractInput for HTML script, got %v", err)
	}

	// Disallow unknown placeholders
	_, _, err = svc.CreateTemplate(ctx, orgID, unitID, ownerID, "idem_inv_02", CreateTemplateInput{
		Name:                "Termo Placeholders",
		Title:               "Título",
		BodyMarkdown:        "Corpo com {{unknown_token}}",
		AcceptanceStatement: "Li e concordo com os termos.",
	})
	if err == nil || !errors.Is(err, ErrInvalidContractInput) {
		t.Fatalf("expected ErrInvalidContractInput for unknown placeholder, got %v", err)
	}

	// Ineligible enrollment (expired)
	enrResolver.snapshot.EffectiveStatus = domain.EnrollmentEffectiveExpired
	tmpl, _, _ := svc.CreateTemplate(ctx, orgID, unitID, ownerID, "idem_tmpl_02", CreateTemplateInput{
		Name:                "Termo Válido",
		Title:               "Termo",
		BodyMarkdown:        "Corpo",
		AcceptanceStatement: "Li e concordo com os termos.",
	})
	pubTmpl, _, _ := svc.PublishTemplate(ctx, orgID, unitID, tmpl.ID, ownerID, "idem_pub_02", PublishTemplateInput{DraftRevision: 1})

	_, err = svc.PreviewEnrollmentContract(ctx, orgID, unitID, enrollmentID, PreviewContractInput{
		TemplateID:        string(tmpl.ID),
		TemplateVersionID: string(pubTmpl.CurrentVersion.ID),
	})
	if err == nil || !errors.Is(err, ErrContractEnrollmentUnavailable) {
		t.Fatalf("expected ErrContractEnrollmentUnavailable for expired enrollment, got %v", err)
	}
}
