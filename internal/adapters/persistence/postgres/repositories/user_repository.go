package repositories

import (
	"context"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
)

func (r UserGormRepository) FindByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	var model models.UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", stringID(id)).First(&model).Error; err != nil {
		return nil, err
	}

	user := userToDomain(model)
	return &user, nil
}

func (r UserGormRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model models.UserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		return nil, err
	}

	user := userToDomain(model)
	return &user, nil
}

func (r UserGormRepository) Save(ctx context.Context, user *domain.User) error {
	if err := ensureID(&user.ID); err != nil {
		return err
	}

	model := models.UserModel{
		ID:           stringID(user.ID),
		BoxID:        stringID(user.BoxID),
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}
