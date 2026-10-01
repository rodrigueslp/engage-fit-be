package organizations

import (
	"context"
	"errors"
	"slices"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

var (
	ErrInvalidUnit             = errors.New("invalid organization unit")
	ErrUnitNotFound            = repositories.ErrOrganizationUnitNotFound
	ErrUnitNameConflict        = repositories.ErrOrganizationUnitNameConflict
	ErrUnitAccessNotFound      = repositories.ErrOrganizationAccessNotFound
	ErrMemberNotFound          = repositories.ErrOrganizationMemberNotFound
	ErrLastOperationalUnit     = errors.New("organization must retain an operational unit")
	ErrCurrentUnitArchive      = errors.New("current unit cannot be archived")
	ErrInvalidMemberUnitAccess = errors.New("invalid member unit access")
)

type UnitChanges struct {
	Name     *string
	Timezone *string
	Status   *domain.BoxStatus
	Reason   *string
}

type Service struct {
	repository repositories.OrganizationRepository
	now        func() time.Time
}

func NewService(repository repositories.OrganizationRepository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s Service) ResolveAccess(ctx context.Context, userID, unitID domain.ID) (*domain.UnitAccessContext, error) {
	return s.repository.ResolveUserUnitAccess(ctx, userID, unitID)
}

func (s Service) ListUnits(ctx context.Context, userID, currentUnitID domain.ID) ([]domain.Box, error) {
	current, err := s.repository.ResolveUserUnitAccess(ctx, userID, currentUnitID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListAccessibleUnits(ctx, userID, current.Organization.ID)
}

func (s Service) CreateUnit(ctx context.Context, userID, currentUnitID domain.ID, name, timezone string) (*domain.Box, error) {
	current, err := s.repository.ResolveUserUnitAccess(ctx, userID, currentUnitID)
	if err != nil {
		return nil, err
	}
	if current.Role != domain.UserRoleOwner || !current.Organization.IsActive() {
		return nil, ErrUnitAccessNotFound
	}
	name, nameKey, ok := domain.NormalizeUnitName(name)
	if !ok {
		return nil, ErrInvalidUnit
	}
	if timezone == "" {
		timezone = current.Unit.Timezone
	}
	if !validTimezone(timezone) {
		return nil, ErrInvalidUnit
	}
	now := s.now().UTC()
	unit := domain.Box{
		OrganizationID: current.Organization.ID,
		Name:           name, NameKey: nameKey, Timezone: timezone, Status: domain.BoxStatusActive,
		RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now,
	}
	event := domain.OrganizationAuditEvent{OrganizationID: current.Organization.ID, ActorUserID: userID, Action: "unit.created", EntityType: "unit", AfterData: unitAuditData(unit), CreatedAt: now}
	if err := s.repository.CreateUnit(ctx, &unit, userID, userID, event); err != nil {
		return nil, err
	}
	return &unit, nil
}

func (s Service) CreateUnitAsPlatformAdmin(ctx context.Context, adminUserID, organizationID domain.ID, name, timezone string) (*domain.Box, error) {
	organization, err := s.repository.FindOrganization(ctx, organizationID)
	if err != nil || !organization.IsActive() {
		return nil, ErrUnitAccessNotFound
	}
	owner, err := s.repository.FindOrganizationOwner(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	name, nameKey, ok := domain.NormalizeUnitName(name)
	if !ok {
		return nil, ErrInvalidUnit
	}
	if timezone == "" {
		timezone = domain.DefaultUnitTimezone
	}
	if !validTimezone(timezone) {
		return nil, ErrInvalidUnit
	}
	now := s.now().UTC()
	unit := domain.Box{OrganizationID: organizationID, Name: name, NameKey: nameKey, Timezone: timezone, Status: domain.BoxStatusActive, RiskInactiveDays: 7, RiskMessageCooldownDays: 14, CreatedAt: now, UpdatedAt: now}
	event := domain.OrganizationAuditEvent{OrganizationID: organizationID, ActorUserID: adminUserID, Action: "unit.created", EntityType: "unit", AfterData: unitAuditData(unit), CreatedAt: now}
	if err := s.repository.CreateUnit(ctx, &unit, owner.ID, adminUserID, event); err != nil {
		return nil, err
	}
	return &unit, nil
}

func (s Service) UpdateUnit(ctx context.Context, userID, currentUnitID, unitID domain.ID, changes UnitChanges) (*domain.Box, error) {
	if changes.Name == nil && changes.Timezone == nil && changes.Status == nil {
		return nil, ErrInvalidUnit
	}
	current, err := s.repository.ResolveUserUnitAccess(ctx, userID, currentUnitID)
	if err != nil || current.Role != domain.UserRoleOwner {
		return nil, ErrUnitAccessNotFound
	}
	target, err := s.repository.ResolveUserUnitAccess(ctx, userID, unitID)
	if err != nil || target.Organization.ID != current.Organization.ID || target.Role != domain.UserRoleOwner {
		return nil, ErrUnitAccessNotFound
	}
	unit := &target.Unit
	if unit.OrganizationID != current.Organization.ID {
		return nil, ErrUnitAccessNotFound
	}
	before := *unit
	if changes.Name != nil {
		name, key, ok := domain.NormalizeUnitName(*changes.Name)
		if !ok {
			return nil, ErrInvalidUnit
		}
		unit.Name, unit.NameKey = name, key
	}
	if changes.Timezone != nil {
		if !validTimezone(*changes.Timezone) {
			return nil, ErrInvalidUnit
		}
		unit.Timezone = *changes.Timezone
	}
	if changes.Status != nil {
		if *changes.Status != domain.BoxStatusActive && *changes.Status != domain.BoxStatusArchived {
			return nil, ErrInvalidUnit
		}
		if *changes.Status == domain.BoxStatusArchived {
			if unit.ID == currentUnitID {
				return nil, ErrCurrentUnitArchive
			}
			count, countErr := s.repository.CountNonArchivedUnits(ctx, current.Organization.ID)
			if countErr != nil {
				return nil, countErr
			}
			if count <= 1 {
				return nil, ErrLastOperationalUnit
			}
		}
		unit.Status = *changes.Status
		now := s.now().UTC()
		unit.StatusChangedAt = &now
		unit.StatusChangedBy = userID
		if changes.Reason != nil {
			unit.StatusReason = *changes.Reason
		}
	}
	unit.UpdatedAt = s.now().UTC()
	action := "unit.updated"
	if before.EffectiveStatus() != unit.EffectiveStatus() {
		action = "unit." + string(unit.EffectiveStatus())
	}
	event := domain.OrganizationAuditEvent{OrganizationID: current.Organization.ID, UnitID: unit.ID, ActorUserID: userID, Action: action, EntityType: "unit", EntityID: unit.ID, BeforeData: unitAuditData(before), AfterData: unitAuditData(*unit), CreatedAt: unit.UpdatedAt}
	if err := s.repository.UpdateUnit(ctx, *unit, event); err != nil {
		return nil, err
	}
	return unit, nil
}

func (s Service) ListMemberUnitIDs(ctx context.Context, actorUserID, currentUnitID, memberUserID domain.ID) ([]domain.ID, error) {
	current, err := s.repository.ResolveUserUnitAccess(ctx, actorUserID, currentUnitID)
	if err != nil || current.Role != domain.UserRoleOwner {
		return nil, ErrUnitAccessNotFound
	}
	if _, err := s.repository.FindOrganizationMember(ctx, current.Organization.ID, memberUserID); err != nil {
		return nil, err
	}
	memberUnitIDs, err := s.repository.ListMemberUnitIDs(ctx, current.Organization.ID, memberUserID)
	if err != nil {
		return nil, err
	}
	actorUnits, err := s.repository.ListAccessibleUnits(ctx, actorUserID, current.Organization.ID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[domain.ID]struct{}, len(actorUnits))
	for _, unit := range actorUnits {
		allowed[unit.ID] = struct{}{}
	}
	result := make([]domain.ID, 0, len(memberUnitIDs))
	for _, unitID := range memberUnitIDs {
		if _, ok := allowed[unitID]; ok {
			result = append(result, unitID)
		}
	}
	return result, nil
}

func (s Service) ReplaceMemberUnitAccess(ctx context.Context, actorUserID, currentUnitID, memberUserID domain.ID, unitIDs []domain.ID) ([]domain.ID, error) {
	current, err := s.repository.ResolveUserUnitAccess(ctx, actorUserID, currentUnitID)
	if err != nil || current.Role != domain.UserRoleOwner {
		return nil, ErrUnitAccessNotFound
	}
	member, err := s.repository.FindOrganizationMember(ctx, current.Organization.ID, memberUserID)
	if err != nil {
		return nil, err
	}
	if member.Role != domain.UserRoleCoach || len(unitIDs) == 0 {
		return nil, ErrInvalidMemberUnitAccess
	}
	unitIDs = uniqueIDs(unitIDs)
	for _, unitID := range unitIDs {
		access, accessErr := s.repository.ResolveUserUnitAccess(ctx, actorUserID, unitID)
		if accessErr != nil || access.Organization.ID != current.Organization.ID || access.Role != domain.UserRoleOwner || access.Unit.EffectiveStatus() == domain.BoxStatusArchived {
			return nil, ErrInvalidMemberUnitAccess
		}
	}
	before, err := s.repository.ListMemberUnitIDs(ctx, current.Organization.ID, memberUserID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	event := domain.OrganizationAuditEvent{OrganizationID: current.Organization.ID, UnitID: currentUnitID, ActorUserID: actorUserID, Action: "user_unit_access.replaced", EntityType: "user_unit_access", EntityID: memberUserID, BeforeData: map[string]any{"unit_ids": before}, AfterData: map[string]any{"unit_ids": unitIDs}, CreatedAt: now}
	if err := s.repository.ReplaceMemberUnitAccess(ctx, current.Organization.ID, memberUserID, unitIDs, actorUserID, event); err != nil {
		return nil, err
	}
	return unitIDs, nil
}

func validTimezone(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	_, err := time.LoadLocation(value)
	return err == nil
}

func uniqueIDs(values []domain.ID) []domain.ID {
	result := make([]domain.ID, 0, len(values))
	for _, value := range values {
		if value != "" && !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}

func unitAuditData(unit domain.Box) map[string]any {
	return map[string]any{"name": unit.Name, "timezone": unit.Timezone, "status": unit.EffectiveStatus()}
}
