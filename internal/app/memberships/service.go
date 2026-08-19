package memberships

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/observability"
	"boxengage/backend/internal/ports/repositories"
)

var (
	ErrPlanNotFound      = repositories.ErrMembershipPlanNotFound
	ErrNameConflict      = repositories.ErrMembershipPlanNameConflict
	ErrDraftNotFound     = repositories.ErrMembershipDraftNotFound
	ErrDraftExists       = repositories.ErrMembershipDraftExists
	ErrRevisionConflict  = repositories.ErrMembershipRevisionConflict
	ErrAlreadyPublished  = repositories.ErrMembershipAlreadyPublished
	ErrPlanArchived      = repositories.ErrMembershipPlanArchived
	ErrIdempotencyReused = repositories.ErrMembershipIdempotencyReused
	ErrInvalid           = errors.New("membership plan invalid")
	ErrTargetInvalid     = errors.New("membership plan target invalid")
	ErrInvalidKey        = errors.New("membership idempotency key invalid")
	ErrUnitNotFound      = errors.New("membership unit not found")
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type EntitlementInput struct {
	ModalityID        string                         `json:"modality_id"`
	ServiceOfferingID string                         `json:"service_offering_id"`
	AccessType        domain.MembershipAccessType    `json:"access_type"`
	Quantity          *int                           `json:"quantity"`
	PeriodUnit        *domain.MembershipPeriodUnit   `json:"period_unit"`
	DailyLimit        *int                           `json:"daily_limit"`
	DeliveryMode      *domain.MembershipDeliveryMode `json:"delivery_mode"`
	BookingRequired   bool                           `json:"booking_required"`
}

type PlanInput struct {
	Name             string                         `json:"name"`
	Description      string                         `json:"description"`
	PriceCents       int                            `json:"price_cents"`
	BillingCycle     domain.MembershipBillingCycle  `json:"billing_cycle"`
	ValidityCount    *int                           `json:"validity_count"`
	ValidityUnit     *domain.MembershipValidityUnit `json:"validity_unit"`
	Entitlements     []EntitlementInput             `json:"entitlements"`
	ExpectedRevision int                            `json:"expected_revision,omitempty"`
}

type Service struct {
	repository   repositories.MembershipRepository
	catalog      repositories.CatalogPlanReferenceResolver
	boxes        repositories.BoxRepository
	transactions repositories.TransactionManager
	now          func() time.Time
}

func NewService(repository repositories.MembershipRepository, catalog repositories.CatalogPlanReferenceResolver, boxes repositories.BoxRepository, transactions repositories.TransactionManager) *Service {
	return &Service{repository: repository, catalog: catalog, boxes: boxes, transactions: transactions, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) List(ctx context.Context, unitID domain.ID, includeArchived bool) ([]domain.MembershipPlan, error) {
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListPlans(ctx, organizationID, unitID, includeArchived)
}

func (s *Service) Get(ctx context.Context, unitID, planID domain.ID) (*domain.MembershipPlan, error) {
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	return s.repository.GetPlan(ctx, organizationID, unitID, planID)
}

func (s *Service) Create(ctx context.Context, unitID, actorID domain.ID, idempotencyKey string, input PlanInput) (*domain.MembershipPlan, bool, error) {
	startedAt := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrInvalidKey
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, false, err
	}
	now := s.now()
	version, nameKey, err := normalizedVersion(input, now)
	if err != nil {
		return nil, false, err
	}
	fingerprint, err := inputFingerprint(version)
	if err != nil {
		return nil, false, err
	}
	var result *domain.MembershipPlan
	var replayed bool
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repository.LockCommand(txctx, organizationID, "create", idempotencyKey); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(txctx, organizationID, "create", idempotencyKey)
		if err != nil {
			return err
		}
		if previous != nil {
			if !fingerprintsEqual(previous.Fingerprint, fingerprint) {
				return ErrIdempotencyReused
			}
			result, err = commandResultForScope(previous, organizationID, unitID)
			if err == nil && result == nil {
				result, err = s.repository.GetPlan(txctx, organizationID, unitID, previous.PlanID)
			}
			replayed = true
			return err
		}
		if err := s.resolveReferences(txctx, organizationID, unitID, &version); err != nil {
			return err
		}
		plan := domain.MembershipPlan{OrganizationID: organizationID, HomeUnitID: unitID, Name: version.Name, NameKey: nameKey, Status: domain.MembershipPlanActive, CreatedAt: now, UpdatedAt: now}
		if err := s.repository.CreatePlan(txctx, &plan, &version, version.Entitlements); err != nil {
			return err
		}
		if err := s.audit(txctx, organizationID, unitID, actorID, "membership_plan.draft_created", plan.ID, version.ID, nil, versionAudit(version), now); err != nil {
			return err
		}
		result, err = s.repository.GetPlan(txctx, organizationID, unitID, plan.ID)
		if err != nil {
			return err
		}
		request := domain.MembershipCommandRequest{OrganizationID: organizationID, Command: "create", IdempotencyKey: idempotencyKey, Fingerprint: fingerprint, PlanID: plan.ID, PlanVersionID: version.ID, ResponseData: commandResponse(result), CreatedAt: now}
		if err := s.repository.CreateCommand(txctx, &request); err != nil {
			return err
		}
		return nil
	})
	observability.RecordMembershipCommand(ctx, "create", membershipResult(err), time.Since(startedAt))
	logMembershipCommand("create", organizationID, unitID, actorID, result, err)
	return result, replayed, err
}

func (s *Service) UpdateDraft(ctx context.Context, unitID, actorID, planID domain.ID, input PlanInput) (*domain.MembershipPlan, error) {
	startedAt := time.Now()
	if input.ExpectedRevision < 1 {
		return nil, ErrInvalid
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	version, _, err := normalizedVersion(input, now)
	if err != nil {
		return nil, err
	}
	var result *domain.MembershipPlan
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		before, err := s.repository.GetPlan(txctx, organizationID, unitID, planID)
		if err != nil {
			return err
		}
		if before.Draft == nil {
			return ErrDraftNotFound
		}
		if err := s.resolveReferences(txctx, organizationID, unitID, &version); err != nil {
			return err
		}
		version.ID, version.PlanID, version.OrganizationID = before.Draft.ID, planID, organizationID
		result, err = s.repository.UpdateDraft(txctx, organizationID, unitID, planID, input.ExpectedRevision, version, version.Entitlements)
		if err != nil {
			return err
		}
		return s.audit(txctx, organizationID, unitID, actorID, "membership_plan.draft_updated", planID, version.ID, versionAudit(*before.Draft), versionAudit(*result.Draft), now)
	})
	observability.RecordMembershipCommand(ctx, "update_draft", membershipResult(err), time.Since(startedAt))
	logMembershipCommand("update_draft", organizationID, unitID, actorID, result, err)
	return result, err
}

func (s *Service) Publish(ctx context.Context, unitID, actorID, planID domain.ID, idempotencyKey string, draftRevision int) (*domain.MembershipPlan, bool, error) {
	startedAt := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrInvalidKey
	}
	if draftRevision < 1 {
		return nil, false, ErrInvalid
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, false, err
	}
	fingerprint := hashJSON(struct {
		PlanID   domain.ID `json:"plan_id"`
		Revision int       `json:"draft_revision"`
	}{planID, draftRevision})
	now := s.now()
	var result *domain.MembershipPlan
	var replayed bool
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repository.LockCommand(txctx, organizationID, "publish", idempotencyKey); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(txctx, organizationID, "publish", idempotencyKey)
		if err != nil {
			return err
		}
		if previous != nil {
			if !fingerprintsEqual(previous.Fingerprint, fingerprint) {
				return ErrIdempotencyReused
			}
			result, err = commandResultForScope(previous, organizationID, unitID)
			if err == nil && result == nil {
				result, err = s.repository.GetPlan(txctx, organizationID, unitID, previous.PlanID)
			}
			replayed = true
			return err
		}
		before, err := s.repository.GetPlan(txctx, organizationID, unitID, planID)
		if err != nil {
			return err
		}
		if before.Status == domain.MembershipPlanArchived {
			return ErrPlanArchived
		}
		if before.Draft == nil {
			return ErrDraftNotFound
		}
		if before.Draft.Revision != draftRevision {
			return ErrRevisionConflict
		}
		if !domain.ValidMembershipVersion(*before.Draft) {
			return ErrInvalid
		}
		if err := s.resolveReferences(txctx, organizationID, unitID, before.Draft); err != nil {
			return err
		}
		result, err = s.repository.PublishDraft(txctx, organizationID, unitID, planID, draftRevision, now)
		if err != nil {
			return err
		}
		published := result.Published
		if published == nil {
			return ErrAlreadyPublished
		}
		if err := s.audit(txctx, organizationID, unitID, actorID, "membership_plan.published", planID, published.ID, versionAudit(*before.Draft), versionAudit(*published), now); err != nil {
			return err
		}
		request := domain.MembershipCommandRequest{OrganizationID: organizationID, Command: "publish", IdempotencyKey: idempotencyKey, Fingerprint: fingerprint, PlanID: planID, PlanVersionID: published.ID, ResponseData: commandResponse(result), CreatedAt: now}
		return s.repository.CreateCommand(txctx, &request)
	})
	observability.RecordMembershipCommand(ctx, "publish", membershipResult(err), time.Since(startedAt))
	logMembershipCommand("publish", organizationID, unitID, actorID, result, err)
	return result, replayed, err
}

func (s *Service) NewVersion(ctx context.Context, unitID, actorID, planID domain.ID, idempotencyKey string) (*domain.MembershipPlan, bool, error) {
	startedAt := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrInvalidKey
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, false, err
	}
	fingerprint := hashJSON(struct {
		PlanID domain.ID `json:"plan_id"`
	}{planID})
	now := s.now()
	var result *domain.MembershipPlan
	var replayed bool
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repository.LockCommand(txctx, organizationID, "new_version", idempotencyKey); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(txctx, organizationID, "new_version", idempotencyKey)
		if err != nil {
			return err
		}
		if previous != nil {
			if !fingerprintsEqual(previous.Fingerprint, fingerprint) {
				return ErrIdempotencyReused
			}
			result, err = commandResultForScope(previous, organizationID, unitID)
			if err == nil && result == nil {
				result, err = s.repository.GetPlan(txctx, organizationID, unitID, previous.PlanID)
			}
			replayed = true
			return err
		}
		result, err = s.repository.CreateNewVersion(txctx, organizationID, unitID, planID, now)
		if err != nil {
			return err
		}
		if result.Draft == nil {
			return ErrDraftNotFound
		}
		if err := s.audit(txctx, organizationID, unitID, actorID, "membership_plan.new_version_created", planID, result.Draft.ID, nil, versionAudit(*result.Draft), now); err != nil {
			return err
		}
		request := domain.MembershipCommandRequest{OrganizationID: organizationID, Command: "new_version", IdempotencyKey: idempotencyKey, Fingerprint: fingerprint, PlanID: planID, PlanVersionID: result.Draft.ID, ResponseData: commandResponse(result), CreatedAt: now}
		return s.repository.CreateCommand(txctx, &request)
	})
	observability.RecordMembershipCommand(ctx, "new_version", membershipResult(err), time.Since(startedAt))
	logMembershipCommand("new_version", organizationID, unitID, actorID, result, err)
	return result, replayed, err
}

func (s *Service) SetArchived(ctx context.Context, unitID, actorID, planID domain.ID, archived bool) (*domain.MembershipPlan, error) {
	startedAt := time.Now()
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	status := domain.MembershipPlanActive
	action := "membership_plan.reactivated"
	if archived {
		status = domain.MembershipPlanArchived
		action = "membership_plan.archived"
	}
	var result *domain.MembershipPlan
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		before, err := s.repository.GetPlan(txctx, organizationID, unitID, planID)
		if err != nil {
			return err
		}
		result, err = s.repository.SetPlanStatus(txctx, organizationID, unitID, planID, status, now)
		if err != nil {
			return err
		}
		if before.Status == status {
			return nil
		}
		return s.audit(txctx, organizationID, unitID, actorID, action, planID, "", map[string]any{"status": before.Status}, map[string]any{"status": status}, now)
	})
	observability.RecordMembershipCommand(ctx, action, membershipResult(err), time.Since(startedAt))
	logMembershipCommand(action, organizationID, unitID, actorID, result, err)
	return result, err
}

func (s *Service) organizationForUnit(ctx context.Context, unitID domain.ID) (domain.ID, error) {
	if unitID == "" {
		return "", ErrUnitNotFound
	}
	box, err := s.boxes.FindByID(ctx, unitID)
	if err != nil || box.OrganizationID == "" {
		return "", ErrUnitNotFound
	}
	return box.OrganizationID, nil
}

func (s *Service) resolveReferences(ctx context.Context, organizationID, unitID domain.ID, version *domain.MembershipPlanVersion) error {
	modalities, services := []domain.ID{}, []domain.ID{}
	for _, item := range version.Entitlements {
		if item.ModalityID != "" {
			modalities = append(modalities, item.ModalityID)
		} else {
			services = append(services, item.ServiceOfferingID)
		}
	}
	references, err := s.catalog.ResolvePlanReferences(ctx, organizationID, unitID, modalities, services)
	if err != nil {
		return err
	}
	byKey := map[string]domain.CatalogPlanReference{}
	for _, reference := range references {
		byKey[reference.Type+":"+string(reference.ID)] = reference
	}
	for index := range version.Entitlements {
		item := &version.Entitlements[index]
		key := "modality:" + string(item.ModalityID)
		if item.ServiceOfferingID != "" {
			key = "service_offering:" + string(item.ServiceOfferingID)
		}
		reference, exists := byKey[key]
		if !exists || reference.Status != domain.CatalogStatusActive || !reference.ActiveInUnit {
			return ErrTargetInvalid
		}
		item.TargetName, item.TargetType = reference.Name, reference.Type
	}
	version.Summary = domain.MembershipPlanSummary(*version)
	return nil
}

func normalizedVersion(input PlanInput, now time.Time) (domain.MembershipPlanVersion, string, error) {
	name, key, ok := domain.NormalizeMembershipPlanName(input.Name)
	if !ok {
		return domain.MembershipPlanVersion{}, "", ErrInvalid
	}
	description, ok := domain.NormalizeMembershipDescription(input.Description)
	if !ok {
		return domain.MembershipPlanVersion{}, "", ErrInvalid
	}
	version := domain.MembershipPlanVersion{VersionNumber: 1, Revision: 1, Status: domain.MembershipVersionDraft, Name: name, Description: description, PriceCents: input.PriceCents, Currency: "BRL", BillingCycle: input.BillingCycle, ValidityCount: input.ValidityCount, ValidityUnit: input.ValidityUnit, CreatedAt: now, UpdatedAt: now, Entitlements: make([]domain.MembershipPlanEntitlement, 0, len(input.Entitlements))}
	for position, item := range input.Entitlements {
		version.Entitlements = append(version.Entitlements, domain.MembershipPlanEntitlement{ModalityID: domain.ID(item.ModalityID), ServiceOfferingID: domain.ID(item.ServiceOfferingID), AccessType: item.AccessType, Quantity: item.Quantity, PeriodUnit: item.PeriodUnit, DailyLimit: item.DailyLimit, DeliveryMode: item.DeliveryMode, BookingRequired: item.BookingRequired, Position: position, CreatedAt: now, UpdatedAt: now})
	}
	if !domain.ValidMembershipVersion(version) {
		return domain.MembershipPlanVersion{}, "", ErrInvalid
	}
	return version, key, nil
}

func inputFingerprint(version domain.MembershipPlanVersion) (string, error) {
	canonical := version
	canonical.ID, canonical.OrganizationID, canonical.PlanID = "", "", ""
	canonical.CreatedAt, canonical.UpdatedAt = time.Time{}, time.Time{}
	canonical.Summary, canonical.Warnings = "", nil
	for index := range canonical.Entitlements {
		canonical.Entitlements[index].CreatedAt = time.Time{}
		canonical.Entitlements[index].UpdatedAt = time.Time{}
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func hashJSON(value any) string {
	payload, _ := json.Marshal(value)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func commandResponse(plan *domain.MembershipPlan) map[string]any {
	if plan == nil {
		return map[string]any{}
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return map[string]any{}
	}
	result := map[string]any{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return map[string]any{}
	}
	return map[string]any{"result": result}
}

func commandResult(request *domain.MembershipCommandRequest) (*domain.MembershipPlan, error) {
	if request == nil || request.ResponseData == nil {
		return nil, nil
	}
	value, exists := request.ResponseData["result"]
	if !exists {
		return nil, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var plan domain.MembershipPlan
	if err := json.Unmarshal(payload, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func commandResultForScope(request *domain.MembershipCommandRequest, organizationID, unitID domain.ID) (*domain.MembershipPlan, error) {
	plan, err := commandResult(request)
	if err != nil || plan == nil {
		return plan, err
	}
	if plan.OrganizationID != organizationID || plan.HomeUnitID != unitID {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

func fingerprintsEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func membershipResult(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, ErrNameConflict) || errors.Is(err, ErrDraftExists) || errors.Is(err, ErrRevisionConflict) || errors.Is(err, ErrIdempotencyReused) || errors.Is(err, ErrAlreadyPublished) || errors.Is(err, ErrPlanArchived) {
		return "conflict"
	}
	if errors.Is(err, ErrInvalid) || errors.Is(err, ErrInvalidKey) || errors.Is(err, ErrTargetInvalid) {
		return "invalid"
	}
	return "error"
}

func logMembershipCommand(action string, organizationID, unitID, actorID domain.ID, plan *domain.MembershipPlan, err error) {
	planID, versionNumber := domain.ID(""), 0
	if plan != nil {
		planID = plan.ID
		if plan.Draft != nil {
			versionNumber = plan.Draft.VersionNumber
		} else if plan.Published != nil {
			versionNumber = plan.Published.VersionNumber
		}
	}
	slog.Info("membership_plan_command", "action", action, "organization_id", organizationID, "unit_id", unitID, "actor_user_id", actorID, "plan_id", planID, "version_number", versionNumber, "result", membershipResult(err))
}

func versionAudit(value domain.MembershipPlanVersion) map[string]any {
	targets := make([]map[string]any, 0, len(value.Entitlements))
	for _, item := range value.Entitlements {
		targets = append(targets, map[string]any{"modality_id": item.ModalityID, "service_offering_id": item.ServiceOfferingID, "access_type": item.AccessType, "quantity": item.Quantity, "period_unit": item.PeriodUnit, "daily_limit": item.DailyLimit, "delivery_mode": item.DeliveryMode, "booking_required": item.BookingRequired, "position": item.Position})
	}
	return map[string]any{"version_number": value.VersionNumber, "revision": value.Revision, "status": value.Status, "name": value.Name, "description": value.Description, "price_cents": value.PriceCents, "currency": value.Currency, "billing_cycle": value.BillingCycle, "validity_count": value.ValidityCount, "validity_unit": value.ValidityUnit, "entitlements": targets}
}

func (s *Service) audit(ctx context.Context, organizationID, unitID, actorID domain.ID, action string, planID, versionID domain.ID, before, after map[string]any, createdAt time.Time) error {
	event := domain.MembershipPlanAuditEvent{OrganizationID: organizationID, UnitID: unitID, ActorUserID: actorID, Action: action, PlanID: planID, PlanVersionID: versionID, BeforeData: before, AfterData: after, CreatedAt: createdAt}
	return s.repository.CreateAuditEvent(ctx, &event)
}
