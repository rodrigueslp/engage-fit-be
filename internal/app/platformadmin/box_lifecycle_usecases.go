package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

var (
	ErrBoxNameRequired         = errors.New("o nome da academia e obrigatorio")
	ErrLifecycleReasonRequired = errors.New("o motivo da alteracao e obrigatorio")
	ErrInvalidBoxStatus        = errors.New("status da academia invalido")
	ErrInvalidStatusTransition = errors.New("transicao de status da academia invalida")
)

type AdminBoxOverview struct {
	Box   domain.Box
	Owner domain.User
}

type CreateAdminBoxInput struct {
	BoxName     string
	OwnerName   string
	OwnerEmail  string
	Password    string
	Reason      string
	AdminUserID domain.ID
	IPAddress   string
}

type UpdateAdminBoxInput struct {
	BoxID       domain.ID
	Name        string
	Reason      string
	AdminUserID domain.ID
	IPAddress   string
}

type ChangeBoxStatusInput struct {
	BoxID       domain.ID
	Status      domain.BoxStatus
	Reason      string
	AdminUserID domain.ID
	IPAddress   string
}

type BoxAdminUseCases struct {
	boxes  repositories.BoxRepository
	users  repositories.UserRepository
	create boxes.CreateBoxUseCase
	audit  AdminAuditRepository
}

func NewBoxAdminUseCases(boxRepository repositories.BoxRepository, users repositories.UserRepository, create boxes.CreateBoxUseCase, audit AdminAuditRepository) BoxAdminUseCases {
	return BoxAdminUseCases{boxes: boxRepository, users: users, create: create, audit: audit}
}

func (uc BoxAdminUseCases) List(ctx context.Context) ([]AdminBoxOverview, error) {
	items, err := uc.boxes.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]AdminBoxOverview, 0, len(items))
	for _, box := range items {
		owner, err := uc.users.FindOwnerByBoxID(ctx, box.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, AdminBoxOverview{Box: box, Owner: *owner})
	}
	return result, nil
}

func (uc BoxAdminUseCases) Create(ctx context.Context, input CreateAdminBoxInput) (*AdminBoxOverview, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrLifecycleReasonRequired
	}
	if err := auth.ValidateNewPassword(input.Password); err != nil {
		return nil, err
	}
	created, err := uc.create.Execute(ctx, boxes.CreateBoxInput{
		Name: input.BoxName, OwnerName: input.OwnerName, OwnerEmail: input.OwnerEmail, Password: input.Password,
	})
	if err != nil {
		return nil, err
	}
	after, _ := json.Marshal(map[string]any{
		"box_id": created.Box.ID, "box_name": created.Box.Name, "status": created.Box.EffectiveStatus(),
		"owner_user_id": created.User.ID, "owner_name": created.User.Name, "owner_email": created.User.Email,
	})
	if err := uc.audit.SaveAuditLog(ctx, &domain.AdminAuditLog{
		AdminUserID: input.AdminUserID, Action: "box.created", TargetType: "box", TargetID: string(created.Box.ID),
		AfterData: after, Reason: reason, IPAddress: input.IPAddress, CreatedAt: time.Now(),
	}); err != nil {
		return nil, err
	}
	return &AdminBoxOverview{Box: created.Box, Owner: created.User}, nil
}

func (uc BoxAdminUseCases) Update(ctx context.Context, input UpdateAdminBoxInput) (*AdminBoxOverview, error) {
	name := strings.TrimSpace(input.Name)
	reason := strings.TrimSpace(input.Reason)
	if name == "" {
		return nil, ErrBoxNameRequired
	}
	if reason == "" {
		return nil, ErrLifecycleReasonRequired
	}
	box, err := uc.boxes.FindByID(ctx, input.BoxID)
	if err != nil {
		return nil, err
	}
	if box.EffectiveStatus() == domain.BoxStatusArchived {
		return nil, ErrInvalidStatusTransition
	}
	before, _ := json.Marshal(box)
	box.Name = name
	box.UpdatedAt = time.Now()
	if err := uc.boxes.Update(ctx, *box); err != nil {
		return nil, err
	}
	if err := uc.saveBoxAudit(ctx, input.AdminUserID, "box.updated", *box, before, reason, input.IPAddress); err != nil {
		return nil, err
	}
	owner, err := uc.users.FindOwnerByBoxID(ctx, box.ID)
	if err != nil {
		return nil, err
	}
	return &AdminBoxOverview{Box: *box, Owner: *owner}, nil
}

func (uc BoxAdminUseCases) ChangeStatus(ctx context.Context, input ChangeBoxStatusInput) (*AdminBoxOverview, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrLifecycleReasonRequired
	}
	if input.Status != domain.BoxStatusActive && input.Status != domain.BoxStatusSuspended && input.Status != domain.BoxStatusArchived {
		return nil, ErrInvalidBoxStatus
	}
	box, err := uc.boxes.FindByID(ctx, input.BoxID)
	if err != nil {
		return nil, err
	}
	current := box.EffectiveStatus()
	if !allowedStatusTransition(current, input.Status) {
		return nil, ErrInvalidStatusTransition
	}
	before, _ := json.Marshal(box)
	now := time.Now()
	box.Status = input.Status
	box.StatusReason = reason
	box.StatusChangedAt = &now
	box.StatusChangedBy = input.AdminUserID
	box.UpdatedAt = now
	if err := uc.boxes.Update(ctx, *box); err != nil {
		return nil, err
	}
	owner, err := uc.users.FindOwnerByBoxID(ctx, box.ID)
	if err != nil {
		return nil, err
	}
	if input.Status != domain.BoxStatusActive {
		if err := uc.users.BumpAuthVersion(ctx, owner.ID); err != nil {
			return nil, err
		}
		owner.AuthVersion++
	}
	action := "box." + string(input.Status)
	if err := uc.saveBoxAudit(ctx, input.AdminUserID, action, *box, before, reason, input.IPAddress); err != nil {
		return nil, err
	}
	return &AdminBoxOverview{Box: *box, Owner: *owner}, nil
}

func allowedStatusTransition(from, to domain.BoxStatus) bool {
	if from == to || from == domain.BoxStatusArchived {
		return false
	}
	switch to {
	case domain.BoxStatusActive:
		return from == domain.BoxStatusSuspended
	case domain.BoxStatusSuspended:
		return from == domain.BoxStatusActive
	case domain.BoxStatusArchived:
		return from == domain.BoxStatusActive || from == domain.BoxStatusSuspended
	default:
		return false
	}
}

func (uc BoxAdminUseCases) saveBoxAudit(ctx context.Context, adminID domain.ID, action string, box domain.Box, before []byte, reason, ipAddress string) error {
	after, _ := json.Marshal(box)
	return uc.audit.SaveAuditLog(ctx, &domain.AdminAuditLog{
		AdminUserID: adminID, Action: action, TargetType: "box", TargetID: string(box.ID), BeforeData: before,
		AfterData: after, Reason: reason, IPAddress: ipAddress, CreatedAt: time.Now(),
	})
}
