package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

var ErrResetReasonRequired = errors.New("o motivo da redefinicao e obrigatorio")

type ResetOwnerPasswordInput struct {
	BoxID       domain.ID
	AdminUserID domain.ID
	NewPassword string
	Reason      string
	IPAddress   string
}

type ResetOwnerPasswordUseCase struct {
	users      repositories.UserRepository
	passwords  services.PasswordService
	governance AdminAuditRepository
}

type AdminAuditRepository interface {
	SaveAuditLog(ctx context.Context, log *domain.AdminAuditLog) error
}

func NewResetOwnerPasswordUseCase(users repositories.UserRepository, passwords services.PasswordService, governance AdminAuditRepository) ResetOwnerPasswordUseCase {
	return ResetOwnerPasswordUseCase{users: users, passwords: passwords, governance: governance}
}

func (uc ResetOwnerPasswordUseCase) Execute(ctx context.Context, input ResetOwnerPasswordInput) error {
	if err := auth.ValidateNewPassword(input.NewPassword); err != nil {
		return err
	}
	if strings.TrimSpace(input.Reason) == "" {
		return ErrResetReasonRequired
	}
	owner, err := uc.users.FindOwnerByBoxID(ctx, input.BoxID)
	if err != nil {
		return err
	}
	hash, err := uc.passwords.Hash(ctx, input.NewPassword)
	if err != nil {
		return err
	}
	if err := uc.users.UpdatePassword(ctx, owner.ID, hash); err != nil {
		return err
	}
	after, _ := json.Marshal(map[string]any{"owner_user_id": owner.ID, "password_reset": true})
	return uc.governance.SaveAuditLog(ctx, &domain.AdminAuditLog{
		AdminUserID: input.AdminUserID,
		Action:      "owner.password.reset",
		TargetType:  "box",
		TargetID:    string(input.BoxID),
		AfterData:   after,
		Reason:      strings.TrimSpace(input.Reason),
		IPAddress:   input.IPAddress,
		CreatedAt:   time.Now(),
	})
}
