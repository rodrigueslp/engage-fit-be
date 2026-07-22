package platformadmin

import (
	"context"
	"errors"
	"testing"

	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
)

type lifecycleRepository struct {
	boxes map[domain.ID]domain.Box
	users map[domain.ID]domain.User
}

func newLifecycleRepository() *lifecycleRepository {
	return &lifecycleRepository{boxes: map[domain.ID]domain.Box{}, users: map[domain.ID]domain.User{}}
}

func (r *lifecycleRepository) FindByID(_ context.Context, id domain.ID) (*domain.Box, error) {
	box, ok := r.boxes[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &box, nil
}
func (r *lifecycleRepository) ListAll(context.Context) ([]domain.Box, error) {
	result := make([]domain.Box, 0, len(r.boxes))
	for _, box := range r.boxes {
		result = append(result, box)
	}
	return result, nil
}
func (r *lifecycleRepository) Save(_ context.Context, box *domain.Box) error {
	if box.ID == "" {
		box.ID = "box-created"
	}
	r.boxes[box.ID] = *box
	return nil
}
func (r *lifecycleRepository) SaveWithOwner(_ context.Context, box *domain.Box, owner *domain.User) error {
	if box.ID == "" {
		box.ID = "box-created"
	}
	if owner.ID == "" {
		owner.ID = "owner-created"
	}
	owner.BoxID = box.ID
	r.boxes[box.ID] = *box
	r.users[owner.ID] = *owner
	return nil
}
func (r *lifecycleRepository) Update(_ context.Context, box domain.Box) error {
	if _, ok := r.boxes[box.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	r.boxes[box.ID] = box
	return nil
}

type lifecycleUsers struct{ repository *lifecycleRepository }

func (r lifecycleUsers) FindByID(_ context.Context, id domain.ID) (*domain.User, error) {
	user, ok := r.repository.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}
func (r lifecycleUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range r.repository.users {
		if user.Email == email {
			copy := user
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r lifecycleUsers) FindOwnerByBoxID(_ context.Context, boxID domain.ID) (*domain.User, error) {
	for _, user := range r.repository.users {
		if user.BoxID == boxID && user.Role == domain.UserRoleOwner {
			copy := user
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r lifecycleUsers) UpdatePassword(context.Context, domain.ID, string) error { return nil }
func (r lifecycleUsers) BumpAuthVersion(_ context.Context, id domain.ID) error {
	user, ok := r.repository.users[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.AuthVersion++
	r.repository.users[id] = user
	return nil
}
func (r lifecycleUsers) UpdatePlatformAdminCredentials(context.Context, domain.ID, string, string) error {
	return nil
}
func (r lifecycleUsers) Save(_ context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "user-created"
	}
	r.repository.users[user.ID] = *user
	return nil
}

type auditRepository struct{ logs []domain.AdminAuditLog }

func (r *auditRepository) SaveAuditLog(_ context.Context, log *domain.AdminAuditLog) error {
	r.logs = append(r.logs, *log)
	return nil
}

func TestBoxAdminLifecycle(t *testing.T) {
	repository := newLifecycleRepository()
	users := lifecycleUsers{repository: repository}
	audit := &auditRepository{}
	create := boxes.NewCreateBoxUseCase(repository, users, security.NewPasswordService())
	useCases := NewBoxAdminUseCases(repository, users, create, audit)

	created, err := useCases.Create(context.Background(), CreateAdminBoxInput{
		BoxName: "CrossFit Alados", OwnerName: "Owner", OwnerEmail: "owner@alados.test",
		Password: "initial-password-123", Reason: "contrato aprovado", AdminUserID: "admin-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Box.EffectiveStatus() != domain.BoxStatusActive || len(audit.logs) != 1 || audit.logs[0].Action != "box.created" {
		t.Fatalf("unexpected creation result: %+v logs=%+v", created, audit.logs)
	}

	suspended, err := useCases.ChangeStatus(context.Background(), ChangeBoxStatusInput{
		BoxID: created.Box.ID, Status: domain.BoxStatusSuspended, Reason: "inadimplencia", AdminUserID: "admin-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if suspended.Box.Status != domain.BoxStatusSuspended || repository.users[created.Owner.ID].AuthVersion != 2 {
		t.Fatalf("suspension did not block owner: %+v", suspended)
	}

	if _, err := useCases.ChangeStatus(context.Background(), ChangeBoxStatusInput{BoxID: created.Box.ID, Status: domain.BoxStatusActive, Reason: "regularizado", AdminUserID: "admin-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := useCases.ChangeStatus(context.Background(), ChangeBoxStatusInput{BoxID: created.Box.ID, Status: domain.BoxStatusArchived, Reason: "encerramento", AdminUserID: "admin-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := useCases.ChangeStatus(context.Background(), ChangeBoxStatusInput{BoxID: created.Box.ID, Status: domain.BoxStatusActive, Reason: "tentativa", AdminUserID: "admin-1"}); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected archived box to be terminal, got %v", err)
	}
}

func TestBoxAdminCreateRequiresStrongPasswordAndReason(t *testing.T) {
	repository := newLifecycleRepository()
	users := lifecycleUsers{repository: repository}
	useCases := NewBoxAdminUseCases(repository, users, boxes.NewCreateBoxUseCase(repository, users, security.NewPasswordService()), &auditRepository{})
	input := CreateAdminBoxInput{BoxName: "Box", OwnerName: "Owner", OwnerEmail: "owner@test", Password: "short", Reason: "cadastro", AdminUserID: "admin"}
	if _, err := useCases.Create(context.Background(), input); err == nil {
		t.Fatal("expected weak password rejection")
	}
	input.Password = "strong-password-123"
	input.Reason = ""
	if _, err := useCases.Create(context.Background(), input); !errors.Is(err, ErrLifecycleReasonRequired) {
		t.Fatalf("expected reason error, got %v", err)
	}
}
