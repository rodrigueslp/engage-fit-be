package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

func (r TeamGormRepository) ListMembers(ctx context.Context, boxID domain.ID) ([]domain.User, error) {
	var rows []models.UserModel
	if err := r.db.WithContext(ctx).Table("users u").
		Joins("JOIN organization_user_memberships oum ON oum.user_id = u.id").
		Where("oum.organization_id = (SELECT organization_id FROM boxes WHERE id = ?) AND u.role IN ?", stringID(boxID), []string{string(domain.UserRoleOwner), string(domain.UserRoleCoach)}).
		Order("u.role DESC, u.name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		result = append(result, userToDomain(row))
	}
	return result, nil
}

func (r TeamGormRepository) FindMember(ctx context.Context, boxID, userID domain.ID) (*domain.User, error) {
	var row models.UserModel
	if err := r.db.WithContext(ctx).Table("users u").
		Joins("JOIN organization_user_memberships oum ON oum.user_id = u.id").
		Where("oum.organization_id = (SELECT organization_id FROM boxes WHERE id = ?) AND u.id = ? AND u.role IN ?", stringID(boxID), stringID(userID), []string{string(domain.UserRoleOwner), string(domain.UserRoleCoach)}).First(&row).Error; err != nil {
		return nil, err
	}
	item := userToDomain(row)
	return &item, nil
}

func (r TeamGormRepository) UpdateMember(ctx context.Context, boxID, userID domain.ID, name string, active bool) error {
	updates := map[string]any{"name": name, "active": active, "updated_at": time.Now().UTC()}
	if !active {
		updates["auth_version"] = gorm.Expr("auth_version + 1")
	}
	return r.db.WithContext(ctx).Model(&models.UserModel{}).
		Where("id = ? AND role = ? AND EXISTS (SELECT 1 FROM organization_user_memberships oum WHERE oum.user_id = users.id AND oum.organization_id = (SELECT organization_id FROM boxes WHERE id = ?))", stringID(userID), string(domain.UserRoleCoach), stringID(boxID)).
		Updates(updates).Error
}
