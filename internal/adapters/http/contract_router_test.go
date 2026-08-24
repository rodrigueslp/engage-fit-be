package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	contractsadapter "boxengage/backend/internal/adapters/contracts"
	"boxengage/backend/internal/adapters/http/middleware"
	contractsapp "boxengage/backend/internal/app/contracts"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type contractRepoStub struct {
	templates        map[string]domain.ContractTemplate
	templateVersions map[string]domain.ContractTemplateVersion
	instances        map[string]domain.ContractInstance
	sessions         map[string]domain.ContractAcceptanceSession
	sessionByHash    map[string]domain.ContractAcceptanceSession
	commands         map[string]domain.ContractCommandRequest
}

func newContractRepoStub() *contractRepoStub {
	return &contractRepoStub{
		templates:        make(map[string]domain.ContractTemplate),
		templateVersions: make(map[string]domain.ContractTemplateVersion),
		instances:        make(map[string]domain.ContractInstance),
		sessions:         make(map[string]domain.ContractAcceptanceSession),
		sessionByHash:    make(map[string]domain.ContractAcceptanceSession),
		commands:         make(map[string]domain.ContractCommandRequest),
	}
}

func (s *contractRepoStub) FindTemplateByID(_ context.Context, organizationID, homeUnitID, templateID domain.ID) (*domain.ContractTemplate, error) {
	t, ok := s.templates[string(templateID)]
	if !ok || t.OrganizationID != organizationID {
		return nil, portrepo.ErrContractTemplateNotFound
	}
	copyT := t
	copyT.Versions = nil
	for _, v := range s.templateVersions {
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

func (s *contractRepoStub) FindTemplateByNameKey(_ context.Context, organizationID domain.ID, nameKey string) (*domain.ContractTemplate, error) {
	for _, t := range s.templates {
		if t.OrganizationID == organizationID && t.NameKey == nameKey {
			return &t, nil
		}
	}
	return nil, nil
}

func (s *contractRepoStub) ListTemplates(ctx context.Context, organizationID, homeUnitID domain.ID, includeArchived bool) ([]domain.ContractTemplate, error) {
	var list []domain.ContractTemplate
	for _, t := range s.templates {
		if t.OrganizationID == organizationID && t.HomeUnitID == homeUnitID {
			if !includeArchived && t.Status == domain.ContractTemplateStatusArchived {
				continue
			}
			full, _ := s.FindTemplateByID(ctx, organizationID, homeUnitID, t.ID)
			list = append(list, *full)
		}
	}
	return list, nil
}

func (s *contractRepoStub) SaveTemplateDraft(_ context.Context, template *domain.ContractTemplate, draft *domain.ContractTemplateVersion) error {
	s.templates[string(template.ID)] = *template
	s.templateVersions[string(draft.ID)] = *draft
	return nil
}

func (s *contractRepoStub) UpdateTemplateDraft(_ context.Context, organizationID, homeUnitID, templateID domain.ID, draft *domain.ContractTemplateVersion, expectedRevision int) error {
	for id, v := range s.templateVersions {
		if v.TemplateID == templateID && v.Status == domain.ContractTemplateVersionStatusDraft {
			if v.Revision != expectedRevision {
				return portrepo.ErrContractTemplateRevisionConflict
			}
			v.Title = draft.Title
			v.BodyMarkdown = draft.BodyMarkdown
			v.AcceptanceStatement = draft.AcceptanceStatement
			v.Revision++
			s.templateVersions[id] = v
			return nil
		}
	}
	return portrepo.ErrContractTemplateNotFound
}

func (s *contractRepoStub) PublishTemplateVersion(_ context.Context, organizationID, homeUnitID, templateID, versionID domain.ID, draftRevision int, publishedAt time.Time, contentSHA256 string) error {
	for id, v := range s.templateVersions {
		if v.TemplateID == templateID && v.Status == domain.ContractTemplateVersionStatusPublished {
			v.Status = domain.ContractTemplateVersionStatusRetired
			s.templateVersions[id] = v
		}
	}
	v, ok := s.templateVersions[string(versionID)]
	if !ok || v.Revision != draftRevision {
		return portrepo.ErrContractTemplateRevisionConflict
	}
	v.Status = domain.ContractTemplateVersionStatusPublished
	v.PublishedAt = &publishedAt
	v.ContentSHA256 = contentSHA256
	v.Revision++
	s.templateVersions[string(versionID)] = v
	return nil
}

func (s *contractRepoStub) CreateNewTemplateVersion(_ context.Context, organizationID, homeUnitID, templateID domain.ID, newDraft *domain.ContractTemplateVersion) error {
	s.templateVersions[string(newDraft.ID)] = *newDraft
	return nil
}

func (s *contractRepoStub) ArchiveTemplate(_ context.Context, organizationID, homeUnitID, templateID domain.ID) error {
	t, ok := s.templates[string(templateID)]
	if !ok {
		return portrepo.ErrContractTemplateNotFound
	}
	t.Status = domain.ContractTemplateStatusArchived
	s.templates[string(templateID)] = t
	return nil
}

func (s *contractRepoStub) ReactivateTemplate(_ context.Context, organizationID, homeUnitID, templateID domain.ID) error {
	t, ok := s.templates[string(templateID)]
	if !ok {
		return portrepo.ErrContractTemplateNotFound
	}
	t.Status = domain.ContractTemplateStatusActive
	s.templates[string(templateID)] = t
	return nil
}

func (s *contractRepoStub) FindTemplateVersionByID(_ context.Context, organizationID, versionID domain.ID) (*domain.ContractTemplateVersion, error) {
	v, ok := s.templateVersions[string(versionID)]
	if !ok || v.OrganizationID != organizationID {
		return nil, portrepo.ErrContractTemplateNotFound
	}
	return &v, nil
}

func (s *contractRepoStub) FindInstanceByID(_ context.Context, organizationID, unitID, instanceID domain.ID) (*domain.ContractInstance, error) {
	inst, ok := s.instances[string(instanceID)]
	if !ok || inst.OrganizationID != organizationID || inst.UnitID != unitID {
		return nil, portrepo.ErrContractInstanceNotFound
	}
	return &inst, nil
}

func (s *contractRepoStub) FindActiveInstanceByEnrollmentID(_ context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.ContractInstance, error) {
	for _, inst := range s.instances {
		if inst.OrganizationID == organizationID && inst.UnitID == unitID && inst.EnrollmentID == enrollmentID && inst.Status != domain.ContractInstanceStatusDiscarded {
			return &inst, nil
		}
	}
	return nil, nil
}

func (s *contractRepoStub) ListInstancesForEnrollment(_ context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractInstance, error) {
	var list []domain.ContractInstance
	for _, inst := range s.instances {
		if inst.OrganizationID == organizationID && inst.UnitID == unitID && inst.EnrollmentID == enrollmentID {
			list = append(list, inst)
		}
	}
	return list, nil
}

func (s *contractRepoStub) ListInstancesForPerson(_ context.Context, organizationID, unitID, personID domain.ID) ([]domain.ContractInstance, error) {
	var list []domain.ContractInstance
	for _, inst := range s.instances {
		if inst.OrganizationID == organizationID && inst.UnitID == unitID && inst.PersonID == personID {
			list = append(list, inst)
		}
	}
	return list, nil
}

func (s *contractRepoStub) SaveInstanceDraft(_ context.Context, instance *domain.ContractInstance) error {
	s.instances[string(instance.ID)] = *instance
	return nil
}

func (s *contractRepoStub) IssueInstance(_ context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, issuedAt time.Time) (*domain.ContractInstance, error) {
	inst, ok := s.instances[string(instanceID)]
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
	s.instances[string(instanceID)] = inst
	return &inst, nil
}

func (s *contractRepoStub) DiscardInstance(_ context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, discardedAt time.Time) (*domain.ContractInstance, error) {
	inst, ok := s.instances[string(instanceID)]
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
	s.instances[string(instanceID)] = inst
	return &inst, nil
}

func (s *contractRepoStub) AcceptInstance(_ context.Context, organizationID, unitID, instanceID, sessionID domain.ID, acceptedAt time.Time, evidence map[string]any, adapter, adapterRef, docSHA256 string, statementVersion int, personID domain.ID) (*domain.ContractInstance, error) {
	inst, ok := s.instances[string(instanceID)]
	if !ok {
		return nil, portrepo.ErrContractInstanceNotFound
	}
	inst.Status = domain.ContractInstanceStatusAccepted
	inst.AcceptedAt = &acceptedAt
	inst.Revision++
	s.instances[string(instanceID)] = inst
	return &inst, nil
}

func (s *contractRepoStub) FindActiveAcceptanceSessionByInstanceID(_ context.Context, organizationID, instanceID domain.ID) (*domain.ContractAcceptanceSession, error) {
	for _, sess := range s.sessions {
		if sess.OrganizationID == organizationID && sess.ContractInstanceID == instanceID && sess.Status == domain.ContractAcceptanceSessionStatusActive {
			return &sess, nil
		}
	}
	return nil, nil
}

func (s *contractRepoStub) FindAcceptanceSessionByTokenHash(_ context.Context, tokenHash string) (*domain.ContractAcceptanceSession, *domain.ContractInstance, error) {
	sess, ok := s.sessionByHash[tokenHash]
	if !ok {
		return nil, nil, portrepo.ErrContractAcceptanceSessionInvalid
	}
	inst, ok := s.instances[string(sess.ContractInstanceID)]
	if !ok {
		return nil, nil, portrepo.ErrContractInstanceNotFound
	}
	return &sess, &inst, nil
}

func (s *contractRepoStub) SaveAcceptanceSession(_ context.Context, session *domain.ContractAcceptanceSession) error {
	s.sessions[string(session.ID)] = *session
	s.sessionByHash[session.TokenHash] = *session
	return nil
}

func (s *contractRepoStub) SupersedeActiveAcceptanceSessions(_ context.Context, organizationID, instanceID domain.ID) error {
	for id, sess := range s.sessions {
		if sess.OrganizationID == organizationID && sess.ContractInstanceID == instanceID && sess.Status == domain.ContractAcceptanceSessionStatusActive {
			sess.Status = domain.ContractAcceptanceSessionStatusSuperseded
			s.sessions[id] = sess
			s.sessionByHash[sess.TokenHash] = sess
		}
	}
	return nil
}

func (s *contractRepoStub) FindCommandRequest(_ context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.ContractCommandRequest, error) {
	k := string(organizationID) + ":" + command + ":" + idempotencyKey
	req, ok := s.commands[k]
	if !ok {
		return nil, nil
	}
	return &req, nil
}

func (s *contractRepoStub) SaveCommandRequest(_ context.Context, req *domain.ContractCommandRequest) error {
	k := string(req.OrganizationID) + ":" + req.Command + ":" + req.IdempotencyKey
	s.commands[k] = *req
	return nil
}

func (s *contractRepoStub) RecordEvent(context.Context, *domain.ContractEvent) error { return nil }
func (s *contractRepoStub) RecordTemplateAuditEvent(context.Context, *domain.ContractTemplateAuditEvent) error {
	return nil
}
func (s *contractRepoStub) ListPrivacyContracts(context.Context, domain.ID, domain.ID) ([]domain.ContractPrivacyExport, error) {
	return nil, nil
}
func (s *contractRepoStub) PurgeExpiredSimulatorRecords(context.Context, time.Time, bool) (int64, error) {
	return 0, nil
}

type contractEnrollmentResolverStub struct{}

func (contractEnrollmentResolverStub) ResolveContractEnrollmentSnapshot(_ context.Context, organizationID, unitID, enrollmentID domain.ID, asOf time.Time) (*domain.ContractEnrollmentSnapshot, error) {
	return &domain.ContractEnrollmentSnapshot{
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
	}, nil
}

type contractPartyResolverStub struct{}

func (contractPartyResolverStub) ResolveContractParty(_ context.Context, organizationID, personID domain.ID) (*domain.ContractPartyResolution, error) {
	return &domain.ContractPartyResolution{
		PersonID:      personID,
		DisplayName:   "Ana Souza",
		ProfileStatus: domain.PersonProfileActive,
		Eligible:      true,
	}, nil
}

type contractTxManagerStub struct{}

func (contractTxManagerStub) WithinTransaction(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}

func buildContractTestRouter(role domain.UserRole, contractsEnabled, simulatorEnabled bool, repo *contractRepoStub) http.Handler {
	unitID := domain.ID("10000000-0000-4000-8000-000000000001")
	userID := domain.ID("20000000-0000-4000-8000-000000000001")

	box := domain.Box{
		ID:             unitID,
		OrganizationID: unitID,
		Name:           "Alados Box",
		Status:         domain.BoxStatusActive,
	}
	user := domain.User{
		ID:          userID,
		BoxID:       unitID,
		Role:        role,
		AuthVersion: 1,
		Active:      true,
	}
	simAdapter := contractsadapter.NewLocalSimulatorAdapter("development", simulatorEnabled)
	contractService := contractsapp.NewService(
		repo,
		contractEnrollmentResolverStub{},
		contractPartyResolverStub{},
		catalogBoxStub{box: box},
		simAdapter,
		contractTxManagerStub{},
	)

	return NewRouter(RouterDependencies{
		AppEnv:         "development",
		TokenService:   catalogTokenStub{claims: services.AuthClaims{UserID: userID, BoxID: unitID, Role: role, AuthVersion: 1}},
		UserRepository: catalogUserStub{user: user},
		BoxRepository:  catalogBoxStub{box: box},
		SessionConfig: middleware.SessionConfig{
			CookieName:     "engagefit_session",
			CSRFCookieName: "engagefit_session_csrf",
			MaxAgeSeconds:  3600,
		},
		Capabilities: middleware.Capabilities{
			Memberships:                 true,
			Enrollments:                 true,
			Contracts:                   contractsEnabled,
			ContractAcceptanceSimulator: simulatorEnabled,
		},
		ContractService: contractService,
	})
}

func TestContractHTTPCapabilitiesAndRoles(t *testing.T) {
	repo := newContractRepoStub()

	// 1. Capability disabled -> 404 capability_disabled
	disabledRouter := buildContractTestRouter(domain.UserRoleOwner, false, false, repo)
	res := catalogRequest(disabledRouter, http.MethodGet, "/api/v1/contract-templates", "", true, false)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "capability_disabled") {
		t.Fatalf("expected 404 capability_disabled, got status %d body %s", res.Code, res.Body.String())
	}

	// 2. Coach role -> 403 forbidden
	coachRouter := buildContractTestRouter(domain.UserRoleCoach, true, true, repo)
	res = catalogRequest(coachRouter, http.MethodGet, "/api/v1/contract-templates", "", true, false)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 forbidden for coach role, got status %d", res.Code)
	}

	// 3. Different unit ID -> 404 contract_not_found
	ownerRouter := buildContractTestRouter(domain.UserRoleOwner, true, true, repo)
	res = catalogRequest(ownerRouter, http.MethodGet, "/api/v1/units/99999999-9999-4999-8999-999999999999/contracts/some-id", "", true, false)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong unit, got status %d", res.Code)
	}

	// 4. Public route when simulator is disabled -> 404 capability_disabled
	simDisabledRouter := buildContractTestRouter(domain.UserRoleOwner, true, false, repo)
	res = catalogRequest(simDisabledRouter, http.MethodGet, "/api/v1/public/contract-acceptance/test-token", "", false, false)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "capability_disabled") {
		t.Fatalf("expected 404 capability_disabled for public simulator when disabled, got status %d", res.Code)
	}
}

func TestContractHTTPLifecycleEndToEnd(t *testing.T) {
	repo := newContractRepoStub()
	router := buildContractTestRouter(domain.UserRoleOwner, true, true, repo)
	unitID := "10000000-0000-4000-8000-000000000001"
	enrollmentID := "a0000000-0000-4000-8000-000000000001"

	// 1. Create Template
	createTmplBody := `{"name":"Termo de Adesão Padrão","title":"Termo de Adesão — {{plan_name}}","body_markdown":"# Termo\n\nAluno: {{member_name}}\nValor: {{total_price}}","acceptance_statement":"Li e aceito os termos do contrato."}`
	res := membershipBearerRequest(router, http.MethodPost, "/api/v1/contract-templates", createTmplBody, "idem_tmpl_http_01")
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for template creation, got %d body %s", res.Code, res.Body.String())
	}
	var tmplResp struct {
		ID           string `json:"id"`
		CurrentDraft struct {
			ID       string `json:"id"`
			Revision int    `json:"revision"`
		} `json:"current_draft"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &tmplResp); err != nil {
		t.Fatalf("unmarshal template response: %v", err)
	}

	// 2. Publish Template
	pubBody := `{"draft_revision":1}`
	res = membershipBearerRequest(router, http.MethodPost, "/api/v1/contract-templates/"+tmplResp.ID+"/publish", pubBody, "idem_pub_http_01")
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for publish, got %d body %s", res.Code, res.Body.String())
	}
	var pubResp struct {
		CurrentVersion struct {
			ID string `json:"id"`
		} `json:"current_version"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &pubResp); err != nil {
		t.Fatalf("unmarshal publish response: %v", err)
	}

	// 3. Preview Contract
	prevBody := `{"template_id":"` + tmplResp.ID + `","template_version_id":"` + pubResp.CurrentVersion.ID + `"}`
	res = catalogRequest(router, http.MethodPost, "/api/v1/units/"+unitID+"/enrollments/"+enrollmentID+"/contracts/preview", prevBody, true, true)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for preview, got %d body %s", res.Code, res.Body.String())
	}
	var prevResp struct {
		PreviewSHA256 string `json:"preview_sha256"`
		DocumentTitle string `json:"document_title"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &prevResp); err != nil {
		t.Fatalf("unmarshal preview response: %v", err)
	}

	// 4. Create Contract Draft
	instDraftBody := `{"template_id":"` + tmplResp.ID + `","template_version_id":"` + pubResp.CurrentVersion.ID + `","expected_preview_sha256":"` + prevResp.PreviewSHA256 + `"}`
	res = membershipBearerRequest(router, http.MethodPost, "/api/v1/units/"+unitID+"/enrollments/"+enrollmentID+"/contracts", instDraftBody, "idem_inst_http_01")
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for instance draft, got %d body %s", res.Code, res.Body.String())
	}
	var instResp struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		Revision       int    `json:"revision"`
		DocumentSHA256 string `json:"document_sha256"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &instResp); err != nil {
		t.Fatalf("unmarshal instance response: %v", err)
	}

	// 5. Issue Contract
	issueBody := `{"expected_revision":1}`
	res = membershipBearerRequest(router, http.MethodPost, "/api/v1/units/"+unitID+"/contracts/"+instResp.ID+"/issue", issueBody, "idem_issue_http_01")
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for issue, got %d body %s", res.Code, res.Body.String())
	}

	// 6. Create Acceptance Session
	res = membershipBearerRequest(router, http.MethodPost, "/api/v1/units/"+unitID+"/contracts/"+instResp.ID+"/acceptance-sessions", "", "idem_sess_http_01")
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for acceptance session, got %d body %s", res.Code, res.Body.String())
	}
	var launchResp struct {
		LaunchToken string `json:"launch_token"`
		LaunchPath  string `json:"launch_path"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &launchResp); err != nil {
		t.Fatalf("unmarshal launch response: %v", err)
	}
	if launchResp.LaunchToken == "" {
		t.Fatalf("expected launch token in response")
	}

	// 7. Public GET Acceptance Session (no auth required)
	res = catalogRequest(router, http.MethodGet, "/api/v1/public/contract-acceptance/"+launchResp.LaunchToken, "", false, false)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for public acceptance session, got %d body %s", res.Code, res.Body.String())
	}
	if res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %s", res.Header().Get("Cache-Control"))
	}
	var pubSessResp struct {
		DocumentSHA256             string `json:"document_sha256"`
		AcceptanceStatementVersion int    `json:"acceptance_statement_version"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &pubSessResp); err != nil {
		t.Fatalf("unmarshal public session response: %v", err)
	}

	// 8. Public Accept (no auth required)
	acceptBody := `{"document_sha256":"` + pubSessResp.DocumentSHA256 + `","acceptance_statement_version":` + string(rune('0'+pubSessResp.AcceptanceStatementVersion)) + `,"accepted":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/contract-acceptance/"+launchResp.LaunchToken+"/accept", strings.NewReader(acceptBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem_acc_http_01")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for accept, got %d body %s", rec.Code, rec.Body.String())
	}
	var acceptedInst struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &acceptedInst); err != nil {
		t.Fatalf("unmarshal accepted instance response: %v", err)
	}
	if acceptedInst.Status != "accepted" {
		t.Fatalf("expected status accepted, got %s", acceptedInst.Status)
	}
}
