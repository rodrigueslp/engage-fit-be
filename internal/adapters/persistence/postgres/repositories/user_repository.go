package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
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

func (r UserGormRepository) FindOwnerByBoxID(ctx context.Context, boxID domain.ID) (*domain.User, error) {
	var model models.UserModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND role = ?", stringID(boxID), string(domain.UserRoleOwner)).First(&model).Error; err != nil {
		return nil, err
	}
	user := userToDomain(model)
	return &user, nil
}

func (r UserGormRepository) Save(ctx context.Context, user *domain.User) error {
	if err := ensureID(&user.ID); err != nil {
		return err
	}

	model := userToModel(*user)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r UserGormRepository) UpdatePlatformAdminCredentials(ctx context.Context, id domain.ID, name, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&models.UserModel{}).Where("id = ? AND role = ?", stringID(id), string(domain.UserRolePlatformAdmin)).Updates(map[string]any{"name": name, "password_hash": passwordHash, "auth_version": gorm.Expr("auth_version + 1"), "updated_at": time.Now()}).Error
}

func (r UserGormRepository) UpdatePassword(ctx context.Context, id domain.ID, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&models.UserModel{}).Where("id = ?", stringID(id)).Updates(map[string]any{"password_hash": passwordHash, "auth_version": gorm.Expr("auth_version + 1"), "updated_at": time.Now()}).Error
}

func (r UserGormRepository) BumpAuthVersion(ctx context.Context, id domain.ID) error {
	return r.db.WithContext(ctx).Model(&models.UserModel{}).Where("id = ?", stringID(id)).Updates(map[string]any{"auth_version": gorm.Expr("auth_version + 1"), "updated_at": time.Now()}).Error
}
