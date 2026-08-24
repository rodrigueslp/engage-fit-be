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
		UserID         string
		UserBoxID      *string
		Role           string
		UserActive     bool
		UnitID         string
		OrganizationID *string
		UnitName       string
	}

	err := databaseForContext(a.db, ctx).Table("users u").
		Select("u.id AS user_id, u.box_id AS user_box_id, u.role, u.active AS user_active, b.id AS unit_id, b.organization_id, b.name AS unit_name").
		Joins("JOIN boxes b ON b.id = ?", stringID(unitID)).
		Where("u.id = ?", stringID(authenticatedUserID)).
		Take(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}
	if err != nil {
		return nil, err
	}

	if !row.UserActive || row.UserBoxID == nil || *row.UserBoxID != stringID(unitID) || row.OrganizationID == nil || *row.OrganizationID == "" {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}

	if row.Role != string(domain.UserRoleOwner) && row.Role != string(domain.UserRoleCoach) {
		return &domain.SchedulingUnitContext{Authorized: false}, nil
	}

	return &domain.SchedulingUnitContext{
		OrganizationID: domainID(*row.OrganizationID),
		UnitID:         unitID,
		UnitName:       row.UnitName,
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
