package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

func (r BoxGormRepository) FindByID(ctx context.Context, id domain.ID) (*domain.Box, error) {
	var model models.BoxModel
	if err := r.db.WithContext(ctx).Where("id = ?", stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	box := boxToDomain(model)
	return &box, nil
}

func (r BoxGormRepository) ListAll(ctx context.Context) ([]domain.Box, error) {
	var rows []models.BoxModel
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Box, 0, len(rows))
	for _, row := range rows {
		result = append(result, boxToDomain(row))
	}
	return result, nil
}

func (r BoxGormRepository) Save(ctx context.Context, box *domain.Box) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := prepareNewBox(tx, box); err != nil {
			return err
		}
		model := boxToModel(*box)
		return tx.Create(&model).Error
	})
}

func (r BoxGormRepository) SaveWithOwner(ctx context.Context, box *domain.Box, owner *domain.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := prepareNewBox(tx, box); err != nil {
			return err
		}
		owner.BoxID = box.ID
		if err := ensureID(&owner.ID); err != nil {
			return err
		}

		boxModel := boxToModel(*box)
		if err := tx.Create(&boxModel).Error; err != nil {
			return err
		}
		ownerModel := userToModel(*owner)
		return tx.Create(&ownerModel).Error
	})
}

func prepareNewBox(tx *gorm.DB, box *domain.Box) error {
	if err := ensureID(&box.ID); err != nil {
		return err
	}
	if box.OrganizationID != "" {
		return nil
	}

	box.OrganizationID = box.ID
	organization := domain.Organization{
		ID:              box.OrganizationID,
		Name:            box.Name,
		Status:          organizationStatusFromBox(box.EffectiveStatus()),
		StatusReason:    box.StatusReason,
		StatusChangedAt: box.StatusChangedAt,
		StatusChangedBy: box.StatusChangedBy,
		CreatedAt:       box.CreatedAt,
		UpdatedAt:       box.UpdatedAt,
	}
	model := organizationToModel(organization)
	return tx.Create(&model).Error
}

func organizationStatusFromBox(status domain.BoxStatus) domain.OrganizationStatus {
	switch status {
	case domain.BoxStatusSuspended:
		return domain.OrganizationStatusSuspended
	case domain.BoxStatusArchived:
		return domain.OrganizationStatusArchived
	default:
		return domain.OrganizationStatusActive
	}
}

func (r BoxGormRepository) Update(ctx context.Context, box domain.Box) error {
	model := boxToModel(box)
	return r.db.WithContext(ctx).Save(&model).Error
}
