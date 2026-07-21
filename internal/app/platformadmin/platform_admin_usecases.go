package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

type EnsureAdminUseCase struct {
	users     repositories.UserRepository
	passwords services.PasswordService
}

func NewEnsureAdminUseCase(users repositories.UserRepository, passwords services.PasswordService) EnsureAdminUseCase {
	return EnsureAdminUseCase{users: users, passwords: passwords}
}

func (uc EnsureAdminUseCase) Execute(ctx context.Context, name, email, password string) error {
	if email == "" && password == "" {
		return nil
	}
	if email == "" || password == "" {
		return errors.New("PLATFORM_ADMIN_EMAIL e PLATFORM_ADMIN_PASSWORD devem ser configurados juntos")
	}
	existing, err := uc.users.FindByEmail(ctx, email)
	if err == nil {
		if existing.Role != domain.UserRolePlatformAdmin {
			return errors.New("PLATFORM_ADMIN_EMAIL já pertence a um usuário que não é administrador da plataforma")
		}
		if compareErr := uc.passwords.Compare(ctx, existing.PasswordHash, password); compareErr == nil && existing.Name == name {
			return nil
		}
		hash, hashErr := uc.passwords.Hash(ctx, password)
		if hashErr != nil {
			return hashErr
		}
		return uc.users.UpdatePlatformAdminCredentials(ctx, existing.ID, name, hash)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	hash, err := uc.passwords.Hash(ctx, password)
	if err != nil {
		return err
	}
	now := time.Now()
	return uc.users.Save(ctx, &domain.User{Name: name, Email: email, PasswordHash: hash, AuthVersion: 1, Role: domain.UserRolePlatformAdmin, CreatedAt: now, UpdatedAt: now})
}

type MessagingAdminUseCases struct {
	boxes      repositories.BoxRepository
	settings   services.WhatsappSettingsResolver
	governance repositories.MessagingGovernanceRepository
}

func NewMessagingAdminUseCases(boxes repositories.BoxRepository, settings services.WhatsappSettingsResolver, governance repositories.MessagingGovernanceRepository) MessagingAdminUseCases {
	return MessagingAdminUseCases{boxes: boxes, settings: settings, governance: governance}
}

func (uc MessagingAdminUseCases) ListBoxes(ctx context.Context) ([]domain.MessagingBoxOverview, error) {
	boxes, err := uc.boxes.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.MessagingBoxOverview, 0, len(boxes))
	for _, box := range boxes {
		policy, err := uc.governance.GetBoxPolicy(ctx, box.ID)
		if err != nil {
			return nil, err
		}
		usage, err := uc.governance.GetBoxUsage(ctx, box.ID, time.Now())
		if err != nil {
			return nil, err
		}
		settings, err := uc.settings.ResolveMetadata(ctx, box.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.MessagingBoxOverview{Box: box, ConnectionMode: settings.ConnectionMode, Policy: *policy, Usage: *usage})
	}
	return result, nil
}

func (uc MessagingAdminUseCases) BoxPolicy(ctx context.Context, boxID domain.ID) (*domain.MessagingPolicy, *domain.MessagingUsage, error) {
	policy, err := uc.governance.GetBoxPolicy(ctx, boxID)
	if err != nil {
		return nil, nil, err
	}
	usage, err := uc.governance.GetBoxUsage(ctx, boxID, time.Now())
	return policy, usage, err
}

func (uc MessagingAdminUseCases) PlatformPolicy(ctx context.Context) (*domain.MessagingPolicy, *domain.MessagingUsage, error) {
	policy, err := uc.governance.GetPlatformPolicy(ctx)
	if err != nil {
		return nil, nil, err
	}
	usage, err := uc.governance.GetPlatformUsage(ctx, time.Now())
	return policy, usage, err
}

type UpdatePolicyInput struct {
	Policy      domain.MessagingPolicy
	AdminUserID domain.ID
	Reason      string
	IPAddress   string
}

func (uc MessagingAdminUseCases) UpdatePolicy(ctx context.Context, input UpdatePolicyInput) error {
	var before *domain.MessagingPolicy
	var err error
	if input.Policy.Scope == domain.MessagingPolicyScopePlatform {
		before, err = uc.governance.GetPlatformPolicy(ctx)
	} else {
		before, err = uc.governance.GetBoxPolicy(ctx, input.Policy.BoxID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if before != nil {
		input.Policy.ID = before.ID
		input.Policy.CreatedAt = before.CreatedAt
	}
	if err := uc.governance.UpsertPolicy(ctx, &input.Policy); err != nil {
		return err
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(input.Policy)
	targetID := string(input.Policy.BoxID)
	if input.Policy.Scope == domain.MessagingPolicyScopePlatform {
		targetID = "platform"
	}
	return uc.governance.SaveAuditLog(ctx, &domain.AdminAuditLog{AdminUserID: input.AdminUserID, Action: "messaging_policy.updated", TargetType: "messaging_policy", TargetID: targetID, BeforeData: beforeJSON, AfterData: afterJSON, Reason: input.Reason, IPAddress: input.IPAddress})
}

type GetTenantMessagingUsageUseCase struct {
	governance repositories.MessagingGovernanceRepository
}

func NewGetTenantMessagingUsageUseCase(governance repositories.MessagingGovernanceRepository) GetTenantMessagingUsageUseCase {
	return GetTenantMessagingUsageUseCase{governance: governance}
}

func (uc GetTenantMessagingUsageUseCase) Execute(ctx context.Context, boxID domain.ID) (*domain.MessagingPolicy, *domain.MessagingUsage, error) {
	policy, err := uc.governance.GetBoxPolicy(ctx, boxID)
	if err != nil {
		return nil, nil, err
	}
	usage, err := uc.governance.GetBoxUsage(ctx, boxID, time.Now())
	return policy, usage, err
}
