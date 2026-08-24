package scheduling

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

var (
	ErrSchedulingUnitNotFound           = portrepo.ErrSchedulingUnitNotFound
	ErrSchedulingProfessionalNotFound   = portrepo.ErrSchedulingProfessionalNotFound
	ErrSchedulingRevisionConflict       = portrepo.ErrSchedulingRevisionConflict
	ErrSchedulingAccountConflict        = portrepo.ErrSchedulingAccountConflict
	ErrSchedulingProfessionalArchived   = portrepo.ErrSchedulingProfessionalArchived
	ErrSchedulingProfessionalAnonymized = portrepo.ErrSchedulingProfessionalAnonymized
	ErrSchedulingIdempotencyKeyReused   = portrepo.ErrSchedulingIdempotencyKeyReused
	ErrSchedulingProfessionalInvalid    = portrepo.ErrSchedulingProfessionalInvalid
	ErrSchedulingAccountUnavailable     = portrepo.ErrSchedulingAccountUnavailable
	ErrSchedulingAnonymizationBlocked   = portrepo.ErrSchedulingAnonymizationBlocked
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type CreateInput struct {
	DisplayName  string     `json:"display_name"`
	LinkedUserID *domain.ID `json:"linked_user_id"`
}

type UpdateInput struct {
	DisplayName      *string    `json:"display_name"`
	LinkedUserID     *domain.ID `json:"linked_user_id"`
	ClearLinkedUser  bool       `json:"clear_linked_user"`
	ExpectedRevision int        `json:"expected_revision"`
}

type ArchiveInput struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type ReactivateInput struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type AnonymizeInput struct {
	ExpectedRevision int    `json:"expected_revision"`
	Reason           string `json:"reason"`
}

type Service struct {
	repo            portrepo.SchedulingRepository
	contextResolver portrepo.SchedulingUnitContextResolver
	accountResolver portrepo.SchedulingAccountResolver
	txManager       portrepo.TransactionManager
	now             func() time.Time
}

type Option func(*Service)

func WithClock(clock func() time.Time) Option {
	return func(s *Service) {
		if clock != nil {
			s.now = clock
		}
	}
}

func NewService(
	repo portrepo.SchedulingRepository,
	contextResolver portrepo.SchedulingUnitContextResolver,
	accountResolver portrepo.SchedulingAccountResolver,
	txManager portrepo.TransactionManager,
	opts ...Option,
) *Service {
	s := &Service{
		repo:            repo,
		contextResolver: contextResolver,
		accountResolver: accountResolver,
		txManager:       txManager,
		now:             func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) ListProfessionals(ctx context.Context, authenticatedUserID, requestedUnitID domain.ID, includeArchived bool, query string) ([]domain.Professional, error) {
	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" && unitCtx.Role != "COACH" {
		return nil, ErrSchedulingUnitNotFound
	}

	// Coach only sees active professionals
	if unitCtx.Role == "COACH" {
		includeArchived = false
	}

	trimmedQuery := strings.TrimSpace(query)
	if utf8.RuneCountInString(trimmedQuery) > 100 {
		return nil, ErrSchedulingProfessionalInvalid
	}

	list, err := s.repo.ListProfessionals(ctx, unitCtx.OrganizationID, requestedUnitID, includeArchived, trimmedQuery)
	if err != nil {
		return nil, err
	}

	if unitCtx.Role == "COACH" {
		for i := range list {
			list[i].LinkedUserID = nil
		}
	}

	return list, nil
}

func (s *Service) GetProfessional(ctx context.Context, authenticatedUserID, requestedUnitID, professionalID domain.ID) (*domain.Professional, error) {
	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" && unitCtx.Role != "COACH" {
		return nil, ErrSchedulingUnitNotFound
	}

	prof, err := s.repo.GetByID(ctx, unitCtx.OrganizationID, requestedUnitID, professionalID)
	if err != nil {
		return nil, err
	}

	if unitCtx.Role == "COACH" {
		if prof.Status != domain.ProfessionalStatusActive {
			return nil, ErrSchedulingProfessionalNotFound
		}
		prof.LinkedUserID = nil
	}

	return prof, nil
}

func (s *Service) ListCandidates(ctx context.Context, authenticatedUserID, requestedUnitID domain.ID) ([]domain.ProfessionalAccountCandidate, error) {
	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, ErrSchedulingUnitNotFound
	}

	candidates, err := s.accountResolver.ListProfessionalAccountCandidates(ctx, unitCtx.OrganizationID, requestedUnitID)
	if err != nil {
		return nil, err
	}

	for i := range candidates {
		linkedProf, err := s.repo.FindByLinkedUser(ctx, unitCtx.OrganizationID, candidates[i].UserID)
		if err == nil && linkedProf != nil {
			candidates[i].IsLinked = true
		}
	}

	return candidates, nil
}

func (s *Service) Create(ctx context.Context, authenticatedUserID, requestedUnitID domain.ID, idempotencyKey string, input CreateInput) (*domain.Professional, bool, error) {
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrSchedulingProfessionalInvalid
	}

	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, false, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, false, ErrSchedulingUnitNotFound
	}

	displayName, displayNameKey, valid := domain.NormalizeProfessionalName(input.DisplayName)
	if !valid {
		return nil, false, ErrSchedulingProfessionalInvalid
	}

	fingerprint := computeFingerprint("professional_create", map[string]any{
		"display_name":   displayName,
		"linked_user_id": input.LinkedUserID,
	})

	var result *domain.Professional
	var isNew bool

	txErr := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.LockCommand(txCtx, unitCtx.OrganizationID, "professional_create", idempotencyKey); err != nil {
			return err
		}

		existingCmd, err := s.repo.FindCommand(txCtx, unitCtx.OrganizationID, "professional_create", idempotencyKey)
		if err != nil {
			return err
		}
		if existingCmd != nil {
			if existingCmd.Fingerprint != fingerprint {
				return ErrSchedulingIdempotencyKeyReused
			}
			data, err := json.Marshal(existingCmd.ResponseData)
			if err != nil {
				return err
			}
			var cached domain.Professional
			if err := json.Unmarshal(data, &cached); err != nil {
				return err
			}
			result = &cached
			isNew = false
			return nil
		}

		if input.LinkedUserID != nil && *input.LinkedUserID != "" {
			if err := s.repo.LockAccount(txCtx, unitCtx.OrganizationID, *input.LinkedUserID); err != nil {
				return err
			}
			account, err := s.accountResolver.ResolveProfessionalAccount(txCtx, unitCtx.OrganizationID, requestedUnitID, *input.LinkedUserID)
			if err != nil || account == nil || !account.Active || !account.AuthorizedInUnit || (account.Role != "OWNER" && account.Role != "COACH") {
				return ErrSchedulingAccountUnavailable
			}
			existingProf, err := s.repo.FindByLinkedUser(txCtx, unitCtx.OrganizationID, *input.LinkedUserID)
			if err != nil && !errors.Is(err, ErrSchedulingProfessionalNotFound) {
				return err
			}
			if existingProf != nil {
				return ErrSchedulingAccountConflict
			}
		}

		now := s.now()
		profID := domain.ID(uuid.NewString())
		prof := domain.Professional{
			ID:              profID,
			OrganizationID:  unitCtx.OrganizationID,
			DisplayName:     displayName,
			DisplayNameKey:  displayNameKey,
			LinkedUserID:    input.LinkedUserID,
			Status:          domain.ProfessionalStatusActive,
			Revision:        1,
			CreatedByUserID: authenticatedUserID,
			UpdatedByUserID: authenticatedUserID,
			ActiveInUnit:    true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := s.repo.CreateProfessional(txCtx, &prof, requestedUnitID); err != nil {
			return err
		}

		auditEvent := domain.SchedulingAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			ActorUserID:    authenticatedUserID,
			Action:         "scheduling.professional.created",
			EntityType:     "professional",
			EntityID:       profID,
			Revision:       1,
			ChangedFields:  []string{"display_name", "status", "revision", "unit_assignment_active", "linked_account_present"},
			BeforeData:     map[string]any{},
			AfterData: map[string]any{
				"status":                 string(domain.ProfessionalStatusActive),
				"revision":               1,
				"unit_assignment_active": true,
				"linked_account_present": prof.LinkedUserID != nil,
				"anonymized":             false,
			},
			CreatedAt: now,
		}
		if err := s.repo.CreateAuditEvent(txCtx, &auditEvent); err != nil {
			return err
		}

		respMap, err := structToMap(prof)
		if err != nil {
			return err
		}

		cmdRequest := domain.SchedulingCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			Command:        "professional_create",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ProfessionalID: profID,
			ResponseData:   respMap,
			CreatedAt:      now,
		}
		if err := s.repo.CreateCommand(txCtx, &cmdRequest); err != nil {
			return err
		}

		result = &prof
		isNew = true
		return nil
	})

	if txErr != nil {
		return nil, false, txErr
	}

	return result, isNew, nil
}

func (s *Service) Update(ctx context.Context, authenticatedUserID, requestedUnitID, professionalID domain.ID, idempotencyKey string, input UpdateInput) (*domain.Professional, bool, error) {
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	if input.ExpectedRevision <= 0 {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	if input.DisplayName == nil && !input.ClearLinkedUser && input.LinkedUserID == nil {
		return nil, false, ErrSchedulingProfessionalInvalid
	}

	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, false, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, false, ErrSchedulingUnitNotFound
	}

	var normalizedName, normalizedKey string
	if input.DisplayName != nil {
		var valid bool
		normalizedName, normalizedKey, valid = domain.NormalizeProfessionalName(*input.DisplayName)
		if !valid {
			return nil, false, ErrSchedulingProfessionalInvalid
		}
	}

	fingerprint := computeFingerprint("professional_update", map[string]any{
		"professional_id":   professionalID,
		"expected_revision": input.ExpectedRevision,
		"display_name":      input.DisplayName,
		"linked_user_id":    input.LinkedUserID,
		"clear_linked_user": input.ClearLinkedUser,
	})

	var result *domain.Professional
	var isNew bool

	txErr := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.LockCommand(txCtx, unitCtx.OrganizationID, "professional_update", idempotencyKey); err != nil {
			return err
		}

		existingCmd, err := s.repo.FindCommand(txCtx, unitCtx.OrganizationID, "professional_update", idempotencyKey)
		if err != nil {
			return err
		}
		if existingCmd != nil {
			if existingCmd.Fingerprint != fingerprint {
				return ErrSchedulingIdempotencyKeyReused
			}
			data, err := json.Marshal(existingCmd.ResponseData)
			if err != nil {
				return err
			}
			var cached domain.Professional
			if err := json.Unmarshal(data, &cached); err != nil {
				return err
			}
			result = &cached
			isNew = false
			return nil
		}

		current, err := s.repo.GetByIDForUpdate(txCtx, unitCtx.OrganizationID, professionalID)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrSchedulingProfessionalNotFound
		}

		if current.AnonymizedAt != nil {
			return ErrSchedulingProfessionalAnonymized
		}
		if current.Status == domain.ProfessionalStatusArchived {
			return ErrSchedulingProfessionalArchived
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}

		beforeData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": current.ActiveInUnit,
			"linked_account_present": current.LinkedUserID != nil,
			"anonymized":             false,
		}

		var changedFields []string

		if input.DisplayName != nil {
			if current.DisplayName != normalizedName {
				current.DisplayName = normalizedName
				current.DisplayNameKey = normalizedKey
				changedFields = append(changedFields, "display_name")
			}
		}

		if input.ClearLinkedUser {
			if current.LinkedUserID != nil {
				current.LinkedUserID = nil
				changedFields = append(changedFields, "linked_account_present")
			}
		} else if input.LinkedUserID != nil && *input.LinkedUserID != "" {
			if current.LinkedUserID == nil || *current.LinkedUserID != *input.LinkedUserID {
				if err := s.repo.LockAccount(txCtx, unitCtx.OrganizationID, *input.LinkedUserID); err != nil {
					return err
				}
				account, err := s.accountResolver.ResolveProfessionalAccount(txCtx, unitCtx.OrganizationID, requestedUnitID, *input.LinkedUserID)
				if err != nil || account == nil || !account.Active || !account.AuthorizedInUnit || (account.Role != "OWNER" && account.Role != "COACH") {
					return ErrSchedulingAccountUnavailable
				}
				existingProf, err := s.repo.FindByLinkedUser(txCtx, unitCtx.OrganizationID, *input.LinkedUserID)
				if err != nil && !errors.Is(err, ErrSchedulingProfessionalNotFound) {
					return err
				}
				if existingProf != nil && existingProf.ID != current.ID {
					return ErrSchedulingAccountConflict
				}
				current.LinkedUserID = input.LinkedUserID
				changedFields = append(changedFields, "linked_account_present")
			}
		}

		now := s.now()
		current.Revision++
		current.UpdatedByUserID = authenticatedUserID
		current.UpdatedAt = now

		if err := s.repo.UpdateProfessional(txCtx, current); err != nil {
			return err
		}

		afterData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": current.ActiveInUnit,
			"linked_account_present": current.LinkedUserID != nil,
			"anonymized":             false,
		}

		auditEvent := domain.SchedulingAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			ActorUserID:    authenticatedUserID,
			Action:         "scheduling.professional.updated",
			EntityType:     "professional",
			EntityID:       current.ID,
			Revision:       current.Revision,
			ChangedFields:  changedFields,
			BeforeData:     beforeData,
			AfterData:      afterData,
			CreatedAt:      now,
		}
		if err := s.repo.CreateAuditEvent(txCtx, &auditEvent); err != nil {
			return err
		}

		respMap, err := structToMap(*current)
		if err != nil {
			return err
		}

		cmdRequest := domain.SchedulingCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			Command:        "professional_update",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ProfessionalID: current.ID,
			ResponseData:   respMap,
			CreatedAt:      now,
		}
		if err := s.repo.CreateCommand(txCtx, &cmdRequest); err != nil {
			return err
		}

		result = current
		isNew = true
		return nil
	})

	if txErr != nil {
		return nil, false, txErr
	}

	return result, isNew, nil
}

func (s *Service) Archive(ctx context.Context, authenticatedUserID, requestedUnitID, professionalID domain.ID, idempotencyKey string, input ArchiveInput) (*domain.Professional, bool, error) {
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	if input.ExpectedRevision <= 0 {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	reason, valid := domain.NormalizeSchedulingReason(input.Reason, 3, 240)
	if !valid {
		return nil, false, ErrSchedulingProfessionalInvalid
	}

	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, false, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, false, ErrSchedulingUnitNotFound
	}

	fingerprint := computeFingerprint("professional_archive", map[string]any{
		"professional_id":   professionalID,
		"expected_revision": input.ExpectedRevision,
		"reason":            reason,
	})

	var result *domain.Professional
	var isNew bool

	txErr := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.LockCommand(txCtx, unitCtx.OrganizationID, "professional_archive", idempotencyKey); err != nil {
			return err
		}

		existingCmd, err := s.repo.FindCommand(txCtx, unitCtx.OrganizationID, "professional_archive", idempotencyKey)
		if err != nil {
			return err
		}
		if existingCmd != nil {
			if existingCmd.Fingerprint != fingerprint {
				return ErrSchedulingIdempotencyKeyReused
			}
			data, err := json.Marshal(existingCmd.ResponseData)
			if err != nil {
				return err
			}
			var cached domain.Professional
			if err := json.Unmarshal(data, &cached); err != nil {
				return err
			}
			result = &cached
			isNew = false
			return nil
		}

		current, err := s.repo.GetByIDForUpdate(txCtx, unitCtx.OrganizationID, professionalID)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrSchedulingProfessionalNotFound
		}

		if current.AnonymizedAt != nil {
			return ErrSchedulingProfessionalAnonymized
		}
		if current.Status == domain.ProfessionalStatusArchived {
			return ErrSchedulingProfessionalArchived
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}

		beforeData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": current.ActiveInUnit,
			"linked_account_present": current.LinkedUserID != nil,
			"anonymized":             false,
		}

		now := s.now()
		current.Status = domain.ProfessionalStatusArchived
		current.ArchivedAt = &now
		current.ArchivedByUserID = &authenticatedUserID
		current.ActiveInUnit = false
		current.Revision++
		current.UpdatedByUserID = authenticatedUserID
		current.UpdatedAt = now

		if err := s.repo.UpdateProfessional(txCtx, current); err != nil {
			return err
		}
		if err := s.repo.SetUnitActive(txCtx, unitCtx.OrganizationID, requestedUnitID, professionalID, false); err != nil {
			return err
		}

		afterData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": false,
			"linked_account_present": current.LinkedUserID != nil,
			"anonymized":             false,
		}

		auditEvent := domain.SchedulingAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			ActorUserID:    authenticatedUserID,
			Action:         "scheduling.professional.archived",
			EntityType:     "professional",
			EntityID:       current.ID,
			Revision:       current.Revision,
			ChangedFields:  []string{"status", "revision", "unit_assignment_active", "archived_at", "archived_by_user_id"},
			BeforeData:     beforeData,
			AfterData:      afterData,
			Reason:         reason,
			CreatedAt:      now,
		}
		if err := s.repo.CreateAuditEvent(txCtx, &auditEvent); err != nil {
			return err
		}

		respMap, err := structToMap(*current)
		if err != nil {
			return err
		}

		cmdRequest := domain.SchedulingCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			Command:        "professional_archive",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ProfessionalID: current.ID,
			ResponseData:   respMap,
			CreatedAt:      now,
		}
		if err := s.repo.CreateCommand(txCtx, &cmdRequest); err != nil {
			return err
		}

		result = current
		isNew = true
		return nil
	})

	if txErr != nil {
		return nil, false, txErr
	}

	return result, isNew, nil
}

func (s *Service) Reactivate(ctx context.Context, authenticatedUserID, requestedUnitID, professionalID domain.ID, idempotencyKey string, input ReactivateInput) (*domain.Professional, bool, error) {
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	if input.ExpectedRevision <= 0 {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	reason, valid := domain.NormalizeSchedulingReason(input.Reason, 3, 240)
	if !valid {
		return nil, false, ErrSchedulingProfessionalInvalid
	}

	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, false, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, false, ErrSchedulingUnitNotFound
	}

	fingerprint := computeFingerprint("professional_reactivate", map[string]any{
		"professional_id":   professionalID,
		"expected_revision": input.ExpectedRevision,
		"reason":            reason,
	})

	var result *domain.Professional
	var isNew bool

	txErr := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.LockCommand(txCtx, unitCtx.OrganizationID, "professional_reactivate", idempotencyKey); err != nil {
			return err
		}

		existingCmd, err := s.repo.FindCommand(txCtx, unitCtx.OrganizationID, "professional_reactivate", idempotencyKey)
		if err != nil {
			return err
		}
		if existingCmd != nil {
			if existingCmd.Fingerprint != fingerprint {
				return ErrSchedulingIdempotencyKeyReused
			}
			data, err := json.Marshal(existingCmd.ResponseData)
			if err != nil {
				return err
			}
			var cached domain.Professional
			if err := json.Unmarshal(data, &cached); err != nil {
				return err
			}
			result = &cached
			isNew = false
			return nil
		}

		current, err := s.repo.GetByIDForUpdate(txCtx, unitCtx.OrganizationID, professionalID)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrSchedulingProfessionalNotFound
		}

		if current.AnonymizedAt != nil {
			return ErrSchedulingProfessionalAnonymized
		}
		if current.Status == domain.ProfessionalStatusActive {
			return ErrSchedulingProfessionalInvalid
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}

		if current.LinkedUserID != nil && *current.LinkedUserID != "" {
			if err := s.repo.LockAccount(txCtx, unitCtx.OrganizationID, *current.LinkedUserID); err != nil {
				return err
			}
			account, err := s.accountResolver.ResolveProfessionalAccount(txCtx, unitCtx.OrganizationID, requestedUnitID, *current.LinkedUserID)
			if err != nil || account == nil || !account.Active || !account.AuthorizedInUnit || (account.Role != "OWNER" && account.Role != "COACH") {
				return ErrSchedulingAccountUnavailable
			}
			existingProf, err := s.repo.FindByLinkedUser(txCtx, unitCtx.OrganizationID, *current.LinkedUserID)
			if err != nil && !errors.Is(err, ErrSchedulingProfessionalNotFound) {
				return err
			}
			if existingProf != nil && existingProf.ID != current.ID {
				return ErrSchedulingAccountConflict
			}
		}

		beforeData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": current.ActiveInUnit,
			"linked_account_present": current.LinkedUserID != nil,
			"anonymized":             false,
		}

		now := s.now()
		current.Status = domain.ProfessionalStatusActive
		current.ArchivedAt = nil
		current.ArchivedByUserID = nil
		current.ActiveInUnit = true
		current.Revision++
		current.UpdatedByUserID = authenticatedUserID
		current.UpdatedAt = now

		if err := s.repo.UpdateProfessional(txCtx, current); err != nil {
			return err
		}
		if err := s.repo.SetUnitActive(txCtx, unitCtx.OrganizationID, requestedUnitID, professionalID, true); err != nil {
			return err
		}

		afterData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": true,
			"linked_account_present": current.LinkedUserID != nil,
			"anonymized":             false,
		}

		auditEvent := domain.SchedulingAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			ActorUserID:    authenticatedUserID,
			Action:         "scheduling.professional.reactivated",
			EntityType:     "professional",
			EntityID:       current.ID,
			Revision:       current.Revision,
			ChangedFields:  []string{"status", "revision", "unit_assignment_active", "archived_at", "archived_by_user_id"},
			BeforeData:     beforeData,
			AfterData:      afterData,
			Reason:         reason,
			CreatedAt:      now,
		}
		if err := s.repo.CreateAuditEvent(txCtx, &auditEvent); err != nil {
			return err
		}

		respMap, err := structToMap(*current)
		if err != nil {
			return err
		}

		cmdRequest := domain.SchedulingCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			Command:        "professional_reactivate",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ProfessionalID: current.ID,
			ResponseData:   respMap,
			CreatedAt:      now,
		}
		if err := s.repo.CreateCommand(txCtx, &cmdRequest); err != nil {
			return err
		}

		result = current
		isNew = true
		return nil
	})

	if txErr != nil {
		return nil, false, txErr
	}

	return result, isNew, nil
}

func (s *Service) Anonymize(ctx context.Context, authenticatedUserID, requestedUnitID, professionalID domain.ID, idempotencyKey string, input AnonymizeInput) (*domain.Professional, bool, error) {
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	if input.ExpectedRevision <= 0 {
		return nil, false, ErrSchedulingProfessionalInvalid
	}
	reason, valid := domain.NormalizeSchedulingReason(input.Reason, 5, 240)
	if !valid {
		return nil, false, ErrSchedulingProfessionalInvalid
	}

	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, false, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, false, ErrSchedulingUnitNotFound
	}

	fingerprint := computeFingerprint("professional_anonymize", map[string]any{
		"professional_id":   professionalID,
		"expected_revision": input.ExpectedRevision,
		"reason":            reason,
	})

	var result *domain.Professional
	var isNew bool

	txErr := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.LockCommand(txCtx, unitCtx.OrganizationID, "professional_anonymize", idempotencyKey); err != nil {
			return err
		}

		existingCmd, err := s.repo.FindCommand(txCtx, unitCtx.OrganizationID, "professional_anonymize", idempotencyKey)
		if err != nil {
			return err
		}
		if existingCmd != nil {
			if existingCmd.Fingerprint != fingerprint {
				return ErrSchedulingIdempotencyKeyReused
			}
			data, err := json.Marshal(existingCmd.ResponseData)
			if err != nil {
				return err
			}
			var cached domain.Professional
			if err := json.Unmarshal(data, &cached); err != nil {
				return err
			}
			result = &cached
			isNew = false
			return nil
		}

		current, err := s.repo.GetByIDForUpdate(txCtx, unitCtx.OrganizationID, professionalID)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrSchedulingProfessionalNotFound
		}

		if current.AnonymizedAt != nil {
			return ErrSchedulingProfessionalAnonymized
		}
		if current.Status != domain.ProfessionalStatusArchived {
			return ErrSchedulingAnonymizationBlocked
		}
		if current.LinkedUserID != nil {
			return ErrSchedulingAnonymizationBlocked
		}
		if current.Revision != input.ExpectedRevision {
			return ErrSchedulingRevisionConflict
		}

		beforeData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": current.ActiveInUnit,
			"linked_account_present": false,
			"anonymized":             false,
		}

		now := s.now()
		current.DisplayName = domain.AnonymizedProfessionalName
		current.DisplayNameKey = domain.AnonymizedProfessionalKey
		current.AnonymizedAt = &now
		current.Revision++
		current.UpdatedByUserID = authenticatedUserID
		current.UpdatedAt = now

		if err := s.repo.UpdateProfessional(txCtx, current); err != nil {
			return err
		}

		afterData := map[string]any{
			"status":                 string(current.Status),
			"revision":               current.Revision,
			"unit_assignment_active": false,
			"linked_account_present": false,
			"anonymized":             true,
		}

		auditEvent := domain.SchedulingAuditEvent{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			ActorUserID:    authenticatedUserID,
			Action:         "scheduling.professional.anonymized",
			EntityType:     "professional",
			EntityID:       current.ID,
			Revision:       current.Revision,
			ChangedFields:  []string{"display_name", "display_name_key", "revision", "anonymized_at"},
			BeforeData:     beforeData,
			AfterData:      afterData,
			Reason:         reason,
			CreatedAt:      now,
		}
		if err := s.repo.CreateAuditEvent(txCtx, &auditEvent); err != nil {
			return err
		}

		respMap, err := structToMap(*current)
		if err != nil {
			return err
		}

		cmdRequest := domain.SchedulingCommandRequest{
			ID:             domain.ID(uuid.NewString()),
			OrganizationID: unitCtx.OrganizationID,
			UnitID:         requestedUnitID,
			Command:        "professional_anonymize",
			IdempotencyKey: idempotencyKey,
			Fingerprint:    fingerprint,
			ProfessionalID: current.ID,
			ResponseData:   respMap,
			CreatedAt:      now,
		}
		if err := s.repo.CreateCommand(txCtx, &cmdRequest); err != nil {
			return err
		}

		result = current
		isNew = true
		return nil
	})

	if txErr != nil {
		return nil, false, txErr
	}

	return result, isNew, nil
}

func (s *Service) ExportPrivacy(ctx context.Context, authenticatedUserID, requestedUnitID, professionalID domain.ID) (*domain.ProfessionalPrivacyExport, error) {
	unitCtx, err := s.contextResolver.ResolveSchedulingUnitContext(ctx, authenticatedUserID, requestedUnitID)
	if err != nil || !unitCtx.Authorized {
		return nil, ErrSchedulingUnitNotFound
	}
	if unitCtx.Role != "OWNER" {
		return nil, ErrSchedulingUnitNotFound
	}

	prof, err := s.repo.GetByID(ctx, unitCtx.OrganizationID, requestedUnitID, professionalID)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, ErrSchedulingProfessionalNotFound
	}

	events, err := s.repo.ListAuditEventsForProfessional(ctx, unitCtx.OrganizationID, requestedUnitID, professionalID)
	if err != nil {
		return nil, err
	}

	return &domain.ProfessionalPrivacyExport{
		Professional: *prof,
		Events:       events,
		ExportedAt:   s.now(),
	}, nil
}

func (s *Service) ResolveProfessionalForUser(ctx context.Context, organizationID, unitID, userID domain.ID) (*domain.ID, error) {
	prof, err := s.repo.FindByLinkedUser(ctx, organizationID, userID)
	if err != nil {
		if errors.Is(err, ErrSchedulingProfessionalNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if prof == nil || !prof.ActiveInUnit || prof.Status != domain.ProfessionalStatusActive {
		return nil, nil
	}
	return &prof.ID, nil
}

func computeFingerprint(command string, payload any) string {
	bytes, _ := json.Marshal(payload)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", command, string(bytes))))
	return hex.EncodeToString(sum[:])
}

func structToMap(data any) (map[string]any, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var res map[string]any
	if err := json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return res, nil
}
