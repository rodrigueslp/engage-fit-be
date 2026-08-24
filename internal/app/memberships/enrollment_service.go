package memberships

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/observability"
	"boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"sort"
)

var (
	ErrEnrollmentSubjectNotFound   = errors.New("enrollment subject not found")
	ErrEnrollmentSubjectIneligible = errors.New("enrollment subject ineligible")
	ErrEnrollmentNotFound          = repositories.ErrEnrollmentNotFound
	ErrActiveEnrollmentExists      = repositories.ErrActiveEnrollmentExists
	ErrPlanVersionChanged          = repositories.ErrPlanVersionChanged
	ErrEnrollmentRevisionConflict  = repositories.ErrEnrollmentRevisionConflict
	ErrEnrollmentAlreadyEnded      = repositories.ErrEnrollmentAlreadyEnded
	ErrEnrollmentIdempotencyReused = repositories.ErrEnrollmentIdempotencyReused
	ErrEnrollmentAddonNotFound     = repositories.ErrEnrollmentAddonNotFound
	ErrEnrollmentAddonDuplicate    = repositories.ErrEnrollmentAddonDuplicate
	ErrPlanUnavailable             = errors.New("plan unavailable")
	ErrPlanTargetsInactive         = errors.New("plan targets inactive")
	ErrInvalidStartDate            = errors.New("invalid enrollment start date")
	ErrInvalidEnd                  = errors.New("invalid enrollment end")
	ErrInvalidEnrollmentRequest    = errors.New("invalid enrollment request")
)

type CreateEnrollmentInput struct {
	PlanID        string   `json:"plan_id"`
	PlanVersionID string   `json:"plan_version_id"`
	StartsOn      string   `json:"starts_on"`
	AddonIDs      []string `json:"addon_ids"`
}

type EndEnrollmentInput struct {
	EndedOn          string `json:"ended_on"`
	Reason           string `json:"reason"`
	ExpectedRevision int    `json:"expected_revision"`
}

type EnrollmentOption func(*EnrollmentService)

func WithEnrollmentClock(now func() time.Time) EnrollmentOption {
	return func(service *EnrollmentService) { service.now = now }
}

func WithEnrollmentLocation(location *time.Location) EnrollmentOption {
	return func(service *EnrollmentService) { service.location = location }
}

type EnrollmentService struct {
	repository   repositories.EnrollmentRepository
	people       repositories.EnrollmentSubjectResolver
	catalog      repositories.CatalogPlanReferenceResolver
	boxes        repositories.BoxRepository
	transactions repositories.TransactionManager
	now          func() time.Time
	location     *time.Location
}

func NewEnrollmentService(repository repositories.EnrollmentRepository, people repositories.EnrollmentSubjectResolver, catalog repositories.CatalogPlanReferenceResolver, boxes repositories.BoxRepository, transactions repositories.TransactionManager, options ...EnrollmentOption) *EnrollmentService {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		location = time.FixedZone("America/Sao_Paulo", -3*60*60)
	}
	service := &EnrollmentService{repository: repository, people: people, catalog: catalog, boxes: boxes, transactions: transactions, now: func() time.Time { return time.Now().UTC() }, location: location}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *EnrollmentService) ListForStudent(ctx context.Context, unitID, studentID domain.ID) (*domain.EnrollmentHistory, error) {
	if !validEnrollmentID(studentID) {
		return nil, ErrEnrollmentSubjectNotFound
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, ErrEnrollmentSubjectNotFound
	}
	subject, err := s.resolveSubject(ctx, organizationID, unitID, studentID)
	if err != nil {
		return nil, err
	}
	result, err := s.repository.ListForPerson(ctx, organizationID, unitID, subject.PersonID)
	if err != nil {
		return nil, err
	}
	s.projectStatuses(result)
	return result, nil
}

func (s *EnrollmentService) Get(ctx context.Context, unitID, enrollmentID domain.ID) (*domain.Enrollment, error) {
	if !validEnrollmentID(enrollmentID) {
		return nil, ErrEnrollmentNotFound
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, ErrEnrollmentNotFound
	}
	item, err := s.repository.GetByID(ctx, organizationID, unitID, enrollmentID)
	if err != nil {
		return nil, err
	}
	item.EffectiveStatus = domain.EnrollmentStatus(*item, s.today())
	return item, nil
}

func (s *EnrollmentService) Create(ctx context.Context, unitID, studentID, actorID domain.ID, idempotencyKey string, input CreateEnrollmentInput) (*domain.Enrollment, bool, error) {
	startedAt := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || !validEnrollmentID(unitID) || !validEnrollmentID(studentID) || !validEnrollmentID(actorID) {
		return nil, false, ErrInvalidEnrollmentRequest
	}
	startsOn, err := s.parseStart(input.StartsOn)
	if err != nil {
		return nil, false, err
	}
	planID, versionID := domain.ID(input.PlanID), domain.ID(input.PlanVersionID)
	if !validEnrollmentID(planID) || !validEnrollmentID(versionID) {
		return nil, false, ErrInvalidEnrollmentRequest
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, false, ErrEnrollmentSubjectNotFound
	}
	normalizedAddonIDs := make([]string, 0, len(input.AddonIDs))
	seenAddonIDs := map[string]struct{}{}
	for _, rawID := range input.AddonIDs {
		if !validEnrollmentID(domain.ID(rawID)) {
			return nil, false, ErrInvalidEnrollmentRequest
		}
		if _, exists := seenAddonIDs[rawID]; exists {
			return nil, false, ErrEnrollmentAddonDuplicate
		}
		seenAddonIDs[rawID] = struct{}{}
		normalizedAddonIDs = append(normalizedAddonIDs, rawID)
	}
	sort.Strings(normalizedAddonIDs)

	fingerprint := enrollmentFingerprint(struct {
		UnitID        domain.ID `json:"unit_id"`
		StudentID     domain.ID `json:"student_id"`
		PlanID        domain.ID `json:"plan_id"`
		PlanVersionID domain.ID `json:"plan_version_id"`
		StartsOn      string    `json:"starts_on"`
		AddonIDs      []string  `json:"addon_ids"`
	}{unitID, studentID, planID, versionID, startsOn.Format("2006-01-02"), normalizedAddonIDs})
	commandTime := s.now().UTC()
	var result *domain.Enrollment
	var replayed bool
	var expirationReconciled bool
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repository.LockCommand(txctx, organizationID, "create", idempotencyKey); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(txctx, organizationID, "create", idempotencyKey)
		if err != nil {
			return err
		}
		if previous != nil {
			if !enrollmentFingerprintsEqual(previous.Fingerprint, fingerprint) {
				return ErrEnrollmentIdempotencyReused
			}
			result, err = enrollmentCommandResult(previous)
			if err == nil && (result == nil || result.HomeUnitID != unitID) {
				return ErrEnrollmentNotFound
			}
			replayed = true
			return err
		}
		subject, err := s.resolveSubject(txctx, organizationID, unitID, studentID)
		if err != nil {
			return err
		}
		if err := s.repository.LockSubject(txctx, organizationID, subject.PersonID, unitID); err != nil {
			return err
		}
		open, err := s.repository.GetOpenForUpdate(txctx, organizationID, unitID, subject.PersonID)
		if err != nil {
			return err
		}
		if open != nil {
			open.EffectiveStatus = domain.EnrollmentStatus(*open, s.today())
			if open.EffectiveStatus != domain.EnrollmentEffectiveExpired {
				return ErrActiveEnrollmentExists
			}
			ended, err := s.repository.EndEnrollment(txctx, open.ID, organizationID, unitID, open.Revision, *open.ValidUntil, commandTime, "Validade natural encerrada", actorID)
			if err != nil {
				return err
			}
			if err := s.repository.CreateEvent(txctx, &domain.EnrollmentEvent{OrganizationID: organizationID, UnitID: unitID, PersonID: subject.PersonID, EnrollmentID: open.ID, ActorUserID: actorID, Action: "enrollment.expiration_reconciled", BeforeData: enrollmentAudit(*open), AfterData: enrollmentAudit(*ended), Reason: "Validade natural encerrada", CreatedAt: commandTime}); err != nil {
				return err
			}
			expirationReconciled = true
		}
		_, version, err := s.repository.GetPlanVersionForEnrollment(txctx, organizationID, unitID, planID, versionID)
		if errors.Is(err, repositories.ErrMembershipPlanNotFound) {
			return ErrPlanUnavailable
		}
		if err != nil {
			return err
		}
		if err := s.validateTargets(txctx, organizationID, unitID, *version); err != nil {
			return err
		}
		addonsByID := map[domain.ID]domain.MembershipPlanVersionAddon{}
		for _, a := range version.Addons {
			addonsByID[a.ID] = a
		}
		enrollmentAddons := make([]domain.EnrollmentAddon, 0, len(input.AddonIDs))
		addonsPriceCents := 0
		for _, rawID := range input.AddonIDs {
			addonID := domain.ID(rawID)
			versionAddon, exists := addonsByID[addonID]
			if !exists {
				return ErrEnrollmentAddonNotFound
			}
			if err := s.validateAddonTargets(txctx, organizationID, unitID, versionAddon); err != nil {
				return err
			}
			enrollmentAddon := domain.EnrollmentAddon{
				PlanVersionAddonID: versionAddon.ID,
				Name:               versionAddon.Name,
				PriceDeltaCents:    versionAddon.PriceDeltaCents,
				Summary:            versionAddon.Summary,
				CreatedAt:          commandTime,
			}
			enrollmentAddons = append(enrollmentAddons, enrollmentAddon)
			addonsPriceCents += versionAddon.PriceDeltaCents
		}
		basePriceCents := version.PriceCents
		totalPriceCents := basePriceCents + addonsPriceCents
		validUntil := calculateValidUntil(startsOn, *version)
		enrollment := &domain.Enrollment{
			OrganizationID:   organizationID,
			PersonID:         subject.PersonID,
			HomeUnitID:       unitID,
			PlanVersionID:    version.ID,
			LifecycleStatus:  domain.EnrollmentLifecycleActive,
			StartsOn:         startsOn,
			ValidUntil:       validUntil,
			Revision:         1,
			BasePriceCents:   basePriceCents,
			AddonsPriceCents: addonsPriceCents,
			TotalPriceCents:  totalPriceCents,
			Addons:           enrollmentAddons,
			CreatedByUserID:  actorID,
			CreatedAt:        commandTime,
			UpdatedAt:        commandTime,
			Terms:            termsForVersion(planID, *version, basePriceCents, addonsPriceCents, totalPriceCents),
		}
		if err := s.repository.CreateEnrollment(txctx, enrollment); err != nil {
			return err
		}
		enrollment.EffectiveStatus = domain.EnrollmentStatus(*enrollment, s.today())
		if err := s.repository.CreateEvent(txctx, &domain.EnrollmentEvent{OrganizationID: organizationID, UnitID: unitID, PersonID: subject.PersonID, EnrollmentID: enrollment.ID, ActorUserID: actorID, Action: "enrollment.created", AfterData: enrollmentAudit(*enrollment), CreatedAt: commandTime}); err != nil {
			return err
		}
		request := &domain.EnrollmentCommandRequest{OrganizationID: organizationID, Command: "create", IdempotencyKey: idempotencyKey, Fingerprint: fingerprint, EnrollmentID: enrollment.ID, ResponseData: enrollmentCommandResponse(enrollment), CreatedAt: commandTime}
		if err := s.repository.CreateCommand(txctx, request); err != nil {
			return err
		}
		result = enrollment
		return nil
	})
	elapsed := time.Since(startedAt)
	observability.RecordEnrollmentCommand(ctx, "create", enrollmentCommandResultLabel(err, replayed), enrollmentConflictReason(err), elapsed, expirationReconciled && err == nil)
	logEnrollmentCommand("create", organizationID, unitID, actorID, result, replayed, elapsed, err)
	return result, replayed, err
}

func (s *EnrollmentService) End(ctx context.Context, unitID, enrollmentID, actorID domain.ID, idempotencyKey string, input EndEnrollmentInput) (*domain.Enrollment, bool, error) {
	startedAt := time.Now()
	if !idempotencyKeyPattern.MatchString(idempotencyKey) || input.ExpectedRevision < 1 || !validEnrollmentID(unitID) || !validEnrollmentID(enrollmentID) || !validEnrollmentID(actorID) {
		return nil, false, ErrInvalidEnrollmentRequest
	}
	endedOn, err := time.ParseInLocation("2006-01-02", input.EndedOn, s.location)
	if err != nil {
		return nil, false, ErrInvalidEnd
	}
	reason := strings.Join(strings.Fields(input.Reason), " ")
	if utf8.RuneCountInString(reason) < 3 || utf8.RuneCountInString(reason) > 240 {
		return nil, false, ErrInvalidEnd
	}
	organizationID, err := s.organizationForUnit(ctx, unitID)
	if err != nil {
		return nil, false, ErrEnrollmentNotFound
	}
	fingerprint := enrollmentFingerprint(struct {
		UnitID           domain.ID `json:"unit_id"`
		EnrollmentID     domain.ID `json:"enrollment_id"`
		EndedOn          string    `json:"ended_on"`
		Reason           string    `json:"reason"`
		ExpectedRevision int       `json:"expected_revision"`
	}{unitID, enrollmentID, endedOn.Format("2006-01-02"), reason, input.ExpectedRevision})
	commandTime := s.now().UTC()
	var result *domain.Enrollment
	var replayed bool
	err = s.transactions.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repository.LockCommand(txctx, organizationID, "end", idempotencyKey); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(txctx, organizationID, "end", idempotencyKey)
		if err != nil {
			return err
		}
		if previous != nil {
			if !enrollmentFingerprintsEqual(previous.Fingerprint, fingerprint) {
				return ErrEnrollmentIdempotencyReused
			}
			result, err = enrollmentCommandResult(previous)
			if err == nil && (result == nil || result.HomeUnitID != unitID) {
				return ErrEnrollmentNotFound
			}
			replayed = true
			return err
		}
		item, err := s.repository.GetByID(txctx, organizationID, unitID, enrollmentID)
		if err != nil {
			return err
		}
		if err := s.repository.LockSubject(txctx, organizationID, item.PersonID, unitID); err != nil {
			return err
		}
		open, err := s.repository.GetOpenForUpdate(txctx, organizationID, unitID, item.PersonID)
		if err != nil {
			return err
		}
		if open == nil || open.ID != enrollmentID {
			return ErrEnrollmentAlreadyEnded
		}
		if open.Revision != input.ExpectedRevision {
			return ErrEnrollmentRevisionConflict
		}
		if err := s.validateEndDate(*open, endedOn); err != nil {
			return err
		}
		result, err = s.repository.EndEnrollment(txctx, enrollmentID, organizationID, unitID, input.ExpectedRevision, endedOn, commandTime, reason, actorID)
		if err != nil {
			return err
		}
		result.EffectiveStatus = domain.EnrollmentEffectiveEnded
		if err := s.repository.CreateEvent(txctx, &domain.EnrollmentEvent{OrganizationID: organizationID, UnitID: unitID, PersonID: open.PersonID, EnrollmentID: enrollmentID, ActorUserID: actorID, Action: "enrollment.ended", BeforeData: enrollmentAudit(*open), AfterData: enrollmentAudit(*result), Reason: reason, CreatedAt: commandTime}); err != nil {
			return err
		}
		return s.repository.CreateCommand(txctx, &domain.EnrollmentCommandRequest{OrganizationID: organizationID, Command: "end", IdempotencyKey: idempotencyKey, Fingerprint: fingerprint, EnrollmentID: enrollmentID, ResponseData: enrollmentCommandResponse(result), CreatedAt: commandTime})
	})
	elapsed := time.Since(startedAt)
	observability.RecordEnrollmentCommand(ctx, "end", enrollmentCommandResultLabel(err, replayed), enrollmentConflictReason(err), elapsed, false)
	logEnrollmentCommand("end", organizationID, unitID, actorID, result, replayed, elapsed, err)
	return result, replayed, err
}

func (s *EnrollmentService) ListPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.EnrollmentPrivacyExport, error) {
	items, err := s.repository.ListPrivacy(ctx, unitID, personID)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Enrollment.EffectiveStatus = domain.EnrollmentStatus(items[index].Enrollment, s.today())
	}
	return items, nil
}

func (s *EnrollmentService) organizationForUnit(ctx context.Context, unitID domain.ID) (domain.ID, error) {
	if !validEnrollmentID(unitID) {
		return "", ErrEnrollmentNotFound
	}
	box, err := s.boxes.FindByID(ctx, unitID)
	if err != nil || box.OrganizationID == "" {
		return "", ErrEnrollmentNotFound
	}
	return box.OrganizationID, nil
}

func validEnrollmentID(value domain.ID) bool {
	_, err := uuid.Parse(string(value))
	return err == nil
}

func (s *EnrollmentService) resolveSubject(ctx context.Context, organizationID, unitID, studentID domain.ID) (*domain.EnrollmentSubject, error) {
	subject, err := s.people.ResolveEnrollmentSubject(ctx, organizationID, unitID, studentID)
	if err != nil {
		return nil, err
	}
	if subject == nil || subject.PersonID == "" {
		return nil, ErrEnrollmentSubjectNotFound
	}
	if !subject.Eligible {
		return nil, ErrEnrollmentSubjectIneligible
	}
	return subject, nil
}

func (s *EnrollmentService) validateTargets(ctx context.Context, organizationID, unitID domain.ID, version domain.MembershipPlanVersion) error {
	modalities, services := []domain.ID{}, []domain.ID{}
	for _, item := range version.Entitlements {
		if item.ModalityID != "" {
			modalities = append(modalities, item.ModalityID)
		} else if item.ServiceOfferingID != "" {
			services = append(services, item.ServiceOfferingID)
		}
	}
	references, err := s.catalog.ResolvePlanReferences(ctx, organizationID, unitID, modalities, services)
	if err != nil {
		return err
	}
	available := map[string]bool{}
	for _, reference := range references {
		available[reference.Type+":"+string(reference.ID)] = reference.Status == domain.CatalogStatusActive && reference.ActiveInUnit
	}
	for _, item := range version.Entitlements {
		key := "modality:" + string(item.ModalityID)
		if item.ServiceOfferingID != "" {
			key = "service_offering:" + string(item.ServiceOfferingID)
		}
		if !available[key] {
			return ErrPlanTargetsInactive
		}
	}
	return nil
}

func (s *EnrollmentService) parseStart(raw string) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02", raw, s.location)
	if err != nil {
		return time.Time{}, ErrInvalidStartDate
	}
	today := s.today()
	if value.Before(today.AddDate(-10, 0, 0)) || value.After(today.AddDate(0, 0, 365)) {
		return time.Time{}, ErrInvalidStartDate
	}
	return value, nil
}

func (s *EnrollmentService) validateEndDate(enrollment domain.Enrollment, endedOn time.Time) error {
	today := s.today()
	if dateKey(enrollment.StartsOn) > dateKey(today) {
		if dateKey(endedOn) != dateKey(today) {
			return ErrInvalidEnd
		}
		return nil
	}
	if dateKey(endedOn) < dateKey(enrollment.StartsOn) || dateKey(endedOn) > dateKey(today) {
		return ErrInvalidEnd
	}
	return nil
}

func dateKey(value time.Time) int {
	return value.Year()*10000 + int(value.Month())*100 + value.Day()
}

func (s *EnrollmentService) today() time.Time {
	local := s.now().In(s.location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.location)
}

func (s *EnrollmentService) projectStatuses(history *domain.EnrollmentHistory) {
	if history.Current != nil {
		history.Current.EffectiveStatus = domain.EnrollmentStatus(*history.Current, s.today())
	}
	for index := range history.History {
		history.History[index].EffectiveStatus = domain.EnrollmentStatus(history.History[index], s.today())
	}
}

func calculateValidUntil(startsOn time.Time, version domain.MembershipPlanVersion) *time.Time {
	if version.BillingCycle != domain.MembershipBillingOneTime || version.ValidityCount == nil || version.ValidityUnit == nil {
		return nil
	}
	count := *version.ValidityCount
	var exclusive time.Time
	switch *version.ValidityUnit {
	case domain.MembershipValidityDay:
		exclusive = startsOn.AddDate(0, 0, count)
	case domain.MembershipValidityWeek:
		exclusive = startsOn.AddDate(0, 0, 7*count)
	case domain.MembershipValidityMonth:
		exclusive = addCalendarClamped(startsOn, 0, count)
	case domain.MembershipValidityYear:
		exclusive = addCalendarClamped(startsOn, count, 0)
	default:
		return nil
	}
	result := exclusive.AddDate(0, 0, -1)
	return &result
}

func addCalendarClamped(value time.Time, years, months int) time.Time {
	targetMonth := time.Date(value.Year()+years, value.Month()+time.Month(months), 1, 0, 0, 0, 0, value.Location())
	lastDay := time.Date(targetMonth.Year(), targetMonth.Month()+1, 0, 0, 0, 0, 0, value.Location()).Day()
	day := value.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetMonth.Year(), targetMonth.Month(), day, 0, 0, 0, 0, value.Location())
}

func (s *EnrollmentService) validateAddonTargets(ctx context.Context, organizationID, unitID domain.ID, addon domain.MembershipPlanVersionAddon) error {
	modalities, services := []domain.ID{}, []domain.ID{}
	for _, item := range addon.Entitlements {
		if item.ModalityID != "" {
			modalities = append(modalities, item.ModalityID)
		} else if item.ServiceOfferingID != "" {
			services = append(services, item.ServiceOfferingID)
		}
	}
	references, err := s.catalog.ResolvePlanReferences(ctx, organizationID, unitID, modalities, services)
	if err != nil {
		return err
	}
	available := map[string]bool{}
	for _, reference := range references {
		available[reference.Type+":"+string(reference.ID)] = reference.Status == domain.CatalogStatusActive && reference.ActiveInUnit
	}
	for _, item := range addon.Entitlements {
		key := "modality:" + string(item.ModalityID)
		if item.ServiceOfferingID != "" {
			key = "service_offering:" + string(item.ServiceOfferingID)
		}
		if !available[key] {
			return ErrPlanTargetsInactive
		}
	}
	return nil
}

func termsForVersion(planID domain.ID, version domain.MembershipPlanVersion, basePriceCents, addonsPriceCents, totalPriceCents int) domain.EnrollmentTerms {
	return domain.EnrollmentTerms{
		PlanID:           planID,
		PlanVersionID:    version.ID,
		VersionNumber:    version.VersionNumber,
		Name:             version.Name,
		Description:      version.Description,
		PriceCents:       totalPriceCents,
		BasePriceCents:   basePriceCents,
		AddonsPriceCents: addonsPriceCents,
		TotalPriceCents:  totalPriceCents,
		Currency:         version.Currency,
		BillingCycle:     version.BillingCycle,
		Summary:          version.Summary,
		BaseSummary:      version.Summary,
	}
}

func enrollmentFingerprint(value any) string {
	payload, _ := json.Marshal(value)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func enrollmentFingerprintsEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func enrollmentCommandResponse(value *domain.Enrollment) map[string]any {
	payload, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	result := map[string]any{}
	if json.Unmarshal(payload, &result) != nil {
		return map[string]any{}
	}
	return map[string]any{"result": result}
}

func enrollmentCommandResult(request *domain.EnrollmentCommandRequest) (*domain.Enrollment, error) {
	if request == nil || request.ResponseData == nil {
		return nil, nil
	}
	payload, err := json.Marshal(request.ResponseData["result"])
	if err != nil {
		return nil, err
	}
	var result domain.Enrollment
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func enrollmentAudit(value domain.Enrollment) map[string]any {
	addons := make([]map[string]any, 0, len(value.Addons))
	for _, a := range value.Addons {
		addons = append(addons, map[string]any{
			"plan_version_addon_id": a.PlanVersionAddonID,
			"addon_name":            a.Name,
			"price_delta_cents":     a.PriceDeltaCents,
			"entitlement_summary":   a.Summary,
		})
	}
	return map[string]any{
		"lifecycle_status":   value.LifecycleStatus,
		"starts_on":          value.StartsOn.Format("2006-01-02"),
		"valid_until":        formatOptionalDate(value.ValidUntil),
		"ended_on":           formatOptionalDate(value.EndedOn),
		"revision":           value.Revision,
		"plan_version_id":    value.PlanVersionID,
		"base_price_cents":   value.BasePriceCents,
		"addons_price_cents": value.AddonsPriceCents,
		"total_price_cents":  value.TotalPriceCents,
		"addons":             addons,
	}
}

func formatOptionalDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format("2006-01-02")
}

func enrollmentResult(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, ErrActiveEnrollmentExists) || errors.Is(err, ErrPlanVersionChanged) || errors.Is(err, ErrEnrollmentRevisionConflict) || errors.Is(err, ErrEnrollmentIdempotencyReused) || errors.Is(err, ErrEnrollmentAlreadyEnded) || errors.Is(err, ErrEnrollmentAddonDuplicate) || errors.Is(err, ErrEnrollmentAddonNotFound) {
		return "conflict"
	}
	if errors.Is(err, ErrInvalidEnrollmentRequest) || errors.Is(err, ErrInvalidStartDate) || errors.Is(err, ErrInvalidEnd) || errors.Is(err, ErrPlanUnavailable) || errors.Is(err, ErrPlanTargetsInactive) || errors.Is(err, ErrEnrollmentSubjectIneligible) {
		return "invalid"
	}
	return "error"
}

func enrollmentCommandResultLabel(err error, replayed bool) string {
	if err == nil && replayed {
		return "replay"
	}
	return enrollmentResult(err)
}

func enrollmentConflictReason(err error) string {
	switch {
	case errors.Is(err, ErrActiveEnrollmentExists):
		return "active_enrollment_exists"
	case errors.Is(err, ErrPlanVersionChanged):
		return "plan_version_changed"
	case errors.Is(err, ErrEnrollmentRevisionConflict):
		return "enrollment_revision_conflict"
	case errors.Is(err, ErrEnrollmentIdempotencyReused):
		return "idempotency_key_reused"
	case errors.Is(err, ErrEnrollmentAlreadyEnded):
		return "enrollment_already_ended"
	case errors.Is(err, ErrEnrollmentAddonNotFound):
		return "membership_plan_addon_not_found"
	case errors.Is(err, ErrEnrollmentAddonDuplicate):
		return "membership_plan_addon_duplicate"
	default:
		return ""
	}
}

func logEnrollmentCommand(action string, organizationID, unitID, actorID domain.ID, enrollment *domain.Enrollment, replayed bool, elapsed time.Duration, err error) {
	enrollmentID, planVersionID := domain.ID(""), domain.ID("")
	if enrollment != nil {
		enrollmentID, planVersionID = enrollment.ID, enrollment.PlanVersionID
	}
	slog.Info("enrollment_command", "action", action, "organization_id", organizationID, "unit_id", unitID, "actor_user_id", actorID, "enrollment_id", enrollmentID, "plan_version_id", planVersionID, "result", enrollmentCommandResultLabel(err, replayed), "conflict_reason", enrollmentConflictReason(err), "replayed", replayed, "rolled_back", err != nil, "duration_ms", elapsed.Milliseconds())
}
