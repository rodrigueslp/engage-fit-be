package repositories

import (
	"context"
	"errors"

	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

type SchedulingAccessGormAdapter struct {
	db *gorm.DB
}

func NewSchedulingAccessGormAdapter(db *gorm.DB) SchedulingAccessGormAdapter {
	return SchedulingAccessGormAdapter{db: db}
}

func (a SchedulingAccessGormAdapter) ResolveSchedulingUnitContext(ctx context.Context, authenticatedUserID, unitID domain.ID) (*domain.SchedulingUnitContext, error) {
	var row struct {
		UserID             string
		UserBoxID          *string
		Role               string
		UserActive         bool
		UnitID             string
		OrganizationID     *string
		UnitName           string
		Timezone           string
		Authorized         bool
		UnitStatus         string
		OrganizationStatus string
	}

	err := databaseForContext(a.db, ctx).Table("users u").
		Select(`u.id AS user_id, u.box_id AS user_box_id, u.role, u.active AS user_active,
			b.id AS unit_id, b.organization_id, b.name AS unit_name, b.timezone, b.status AS unit_status,
			o.status AS organization_status,
			(u.box_id = b.id OR EXISTS (
				SELECT 1 FROM organization_user_memberships oum
				JOIN user_unit_access uua ON uua.organization_id = oum.organization_id AND uua.user_id = oum.user_id
				WHERE oum.organization_id = b.organization_id AND oum.user_id = u.id AND oum.active = true
				AND uua.unit_id = b.id AND uua.active = true
			)) AS authorized`).
		Joins("JOIN boxes b ON b.id = ?", stringID(unitID)).
		Joins("JOIN organizations o ON o.id = b.organization_id").
		Where("u.id = ?", stringID(authenticatedUserID)).
		Take(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}
	if err != nil {
		return nil, err
	}

	if !row.UserActive || !row.Authorized || row.OrganizationID == nil || *row.OrganizationID == "" || row.UnitStatus != "active" || row.OrganizationStatus != "active" {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}

	if row.Role != string(domain.UserRoleOwner) && row.Role != string(domain.UserRoleCoach) {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}

	return &domain.SchedulingUnitContext{
		OrganizationID: domainID(*row.OrganizationID),
		UnitID:         unitID,
		UnitName:       row.UnitName,
		Timezone:       row.Timezone,
		Role:           row.Role,
		Authorized:     true,
	}, nil
}

func (a SchedulingAccessGormAdapter) ListProfessionalAccountCandidates(ctx context.Context, organizationID, unitID domain.ID) ([]domain.ProfessionalAccountCandidate, error) {
	var rows []struct {
		ID               string
		Name             string
		Role             string
		Active           bool
		AuthorizedInUnit bool
	}

	err := databaseForContext(a.db, ctx).Table("users u").
		Select("u.id, u.name, u.role, u.active, (u.box_id = b.id) AS authorized_in_unit").
		Joins("JOIN boxes b ON b.id = u.box_id AND b.organization_id = ?", stringID(organizationID)).
		Where("u.box_id = ? AND u.role IN (?, ?) AND u.active = true", stringID(unitID), string(domain.UserRoleOwner), string(domain.UserRoleCoach)).
		Order("u.name ASC, u.id ASC").
		Find(&rows).Error

	if err != nil {
		return nil, err
	}

	candidates := make([]domain.ProfessionalAccountCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, domain.ProfessionalAccountCandidate{
			UserID:           domainID(row.ID),
			DisplayName:      row.Name,
			Role:             row.Role,
			Active:           row.Active,
			AuthorizedInUnit: row.AuthorizedInUnit,
		})
	}
	return candidates, nil
}

func (a SchedulingAccessGormAdapter) ResolveProfessionalAccount(ctx context.Context, organizationID, unitID, userID domain.ID) (*domain.ProfessionalAccountCandidate, error) {
	var row struct {
		ID               string
		Name             string
		Role             string
		Active           bool
		AuthorizedInUnit bool
	}

	err := databaseForContext(a.db, ctx).Table("users u").
		Select("u.id, u.name, u.role, u.active, (u.box_id = ?) AS authorized_in_unit", stringID(unitID)).
		Joins("JOIN boxes b ON b.id = u.box_id AND b.organization_id = ?", stringID(organizationID)).
		Where("u.id = ?", stringID(userID)).
		Take(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &domain.ProfessionalAccountCandidate{
		UserID:           domainID(row.ID),
		DisplayName:      row.Name,
		Role:             row.Role,
		Active:           row.Active,
		AuthorizedInUnit: row.AuthorizedInUnit,
	}, nil
}
