package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type memberBillingContractModel struct {
	ID, OrganizationID, UnitID, PersonID, EnrollmentID, BillingCycle, Currency, Status, PaymentMethod, CreatedByUserID string
	AmountCents                                                                                                        int
	StartedOn                                                                                                          time.Time
	NextDueOn                                                                                                          *time.Time
	CreatedAt, UpdatedAt                                                                                               time.Time
}

func (memberBillingContractModel) TableName() string { return "member_billing_contracts" }

type memberChargeModel struct {
	ID, OrganizationID, UnitID, PersonID, EnrollmentID, ContractID                    string
	Sequence                                                                          int
	DueOn                                                                             time.Time
	AmountCents, PaidCents, RefundedCents                                             int
	Currency, Status, PaymentMethod, Provider, ProviderReference, PaymentInstructions string
	PaidAt, CanceledAt                                                                *time.Time
	CreatedAt, UpdatedAt                                                              time.Time
}

func (memberChargeModel) TableName() string { return "member_charges" }

type memberPaymentModel struct {
	ID, OrganizationID, ChargeID, Kind, Method string
	AmountCents                                int
	OccurredAt                                 time.Time
	ActorUserID                                *string
	ProviderEventID, Reason                    string
	CreatedAt                                  time.Time
}

func (memberPaymentModel) TableName() string { return "member_payments" }

type memberBillingCommandModel struct {
	ID, OrganizationID, Command, IdempotencyKey, RequestFingerprint, ResultID string
	CreatedAt                                                                 time.Time
}

func (memberBillingCommandModel) TableName() string { return "member_billing_command_requests" }

func (r MemberBillingGormRepository) FindCommand(ctx context.Context, organizationID domain.ID, command, key string) (*domain.MemberBillingCommand, error) {
	var m memberBillingCommandModel
	err := databaseForContext(r.db, ctx).Where("organization_id = ? AND command = ? AND idempotency_key = ?", organizationID, command, key).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.MemberBillingCommand{OrganizationID: domain.ID(m.OrganizationID), Command: m.Command, IdempotencyKey: m.IdempotencyKey, Fingerprint: m.RequestFingerprint, ResultID: domain.ID(m.ResultID), CreatedAt: m.CreatedAt}, nil
}
func (r MemberBillingGormRepository) LockCommand(ctx context.Context, organizationID domain.ID, command, key string) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", string(organizationID)+":memberbilling:"+command+":"+key).Error
}
func (r MemberBillingGormRepository) LockContract(ctx context.Context, organizationID, contractID domain.ID) error {
	return databaseForContext(r.db, ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", string(organizationID)+":memberbilling:contract:"+string(contractID)).Error
}
func (r MemberBillingGormRepository) CreateCommand(ctx context.Context, value *domain.MemberBillingCommand) error {
	m := memberBillingCommandModel{ID: uuid.NewString(), OrganizationID: string(value.OrganizationID), Command: value.Command, IdempotencyKey: value.IdempotencyKey, RequestFingerprint: value.Fingerprint, ResultID: string(value.ResultID), CreatedAt: value.CreatedAt}
	if err := databaseForContext(r.db, ctx).Create(&m).Error; err != nil {
		return mapMemberBillingError(err)
	}
	return nil
}
func (r MemberBillingGormRepository) GetContractByEnrollment(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.MemberBillingContract, error) {
	var m memberBillingContractModel
	err := databaseForContext(r.db, ctx).Where("organization_id=? AND unit_id=? AND enrollment_id=?", organizationID, unitID, enrollmentID).First(&m).Error
	return contractDomain(m, err)
}
func (r MemberBillingGormRepository) GetContract(ctx context.Context, organizationID, unitID, id domain.ID) (*domain.MemberBillingContract, error) {
	var m memberBillingContractModel
	err := databaseForContext(r.db, ctx).Where("organization_id=? AND unit_id=? AND id=?", organizationID, unitID, id).First(&m).Error
	return contractDomain(m, err)
}
func contractDomain(m memberBillingContractModel, err error) (*domain.MemberBillingContract, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, portrepo.ErrMemberBillingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.MemberBillingContract{ID: domain.ID(m.ID), OrganizationID: domain.ID(m.OrganizationID), UnitID: domain.ID(m.UnitID), PersonID: domain.ID(m.PersonID), EnrollmentID: domain.ID(m.EnrollmentID), BillingCycle: domain.MembershipBillingCycle(m.BillingCycle), AmountCents: m.AmountCents, Currency: m.Currency, Status: m.Status, PaymentMethod: m.PaymentMethod, StartedOn: m.StartedOn, NextDueOn: m.NextDueOn, CreatedByUserID: domain.ID(m.CreatedByUserID), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}, nil
}
func (r MemberBillingGormRepository) CreateContract(ctx context.Context, v *domain.MemberBillingContract) error {
	m := memberBillingContractModel{ID: string(v.ID), OrganizationID: string(v.OrganizationID), UnitID: string(v.UnitID), PersonID: string(v.PersonID), EnrollmentID: string(v.EnrollmentID), BillingCycle: string(v.BillingCycle), AmountCents: v.AmountCents, Currency: v.Currency, Status: v.Status, PaymentMethod: v.PaymentMethod, StartedOn: v.StartedOn, NextDueOn: v.NextDueOn, CreatedByUserID: string(v.CreatedByUserID), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
	return mapMemberBillingError(databaseForContext(r.db, ctx).Create(&m).Error)
}
func (r MemberBillingGormRepository) UpdateNextDue(ctx context.Context, organizationID, id domain.ID, next *time.Time, updated time.Time) error {
	return databaseForContext(r.db, ctx).Model(&memberBillingContractModel{}).Where("organization_id=? AND id=?", organizationID, id).Updates(map[string]any{"next_due_on": next, "updated_at": updated}).Error
}
func (r MemberBillingGormRepository) ListCharges(ctx context.Context, organizationID, unitID, contractID domain.ID) ([]domain.MemberCharge, error) {
	var rows []memberChargeModel
	if err := databaseForContext(r.db, ctx).Where("organization_id=? AND unit_id=? AND contract_id=?", organizationID, unitID, contractID).Order("sequence").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MemberCharge, 0, len(rows))
	for _, m := range rows {
		v := chargeDomain(m)
		payments, err := r.listPayments(ctx, organizationID, v.ID)
		if err != nil {
			return nil, err
		}
		v.Payments = payments
		result = append(result, v)
	}
	return result, nil
}
func (r MemberBillingGormRepository) GetCharge(ctx context.Context, organizationID, unitID, id domain.ID) (*domain.MemberCharge, error) {
	return r.getCharge(ctx, organizationID, unitID, id, false)
}
func (r MemberBillingGormRepository) GetChargeForUpdate(ctx context.Context, organizationID, unitID, id domain.ID) (*domain.MemberCharge, error) {
	return r.getCharge(ctx, organizationID, unitID, id, true)
}
func (r MemberBillingGormRepository) getCharge(ctx context.Context, organizationID, unitID, id domain.ID, forUpdate bool) (*domain.MemberCharge, error) {
	var m memberChargeModel
	query := databaseForContext(r.db, ctx)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Where("organization_id=? AND unit_id=? AND id=?", organizationID, unitID, id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, portrepo.ErrMemberBillingNotFound
	}
	if err != nil {
		return nil, err
	}
	v := chargeDomain(m)
	v.Payments, err = r.listPayments(ctx, organizationID, id)
	return &v, err
}
func chargeDomain(m memberChargeModel) domain.MemberCharge {
	instructions := map[string]any{}
	_ = json.Unmarshal([]byte(m.PaymentInstructions), &instructions)
	return domain.MemberCharge{ID: domain.ID(m.ID), OrganizationID: domain.ID(m.OrganizationID), UnitID: domain.ID(m.UnitID), PersonID: domain.ID(m.PersonID), EnrollmentID: domain.ID(m.EnrollmentID), ContractID: domain.ID(m.ContractID), Sequence: m.Sequence, DueOn: m.DueOn, AmountCents: m.AmountCents, PaidCents: m.PaidCents, RefundedCents: m.RefundedCents, Currency: m.Currency, Status: m.Status, PaymentMethod: m.PaymentMethod, Provider: m.Provider, ProviderReference: m.ProviderReference, PaymentInstructions: instructions, PaidAt: m.PaidAt, CanceledAt: m.CanceledAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func (r MemberBillingGormRepository) CreateCharge(ctx context.Context, v *domain.MemberCharge) error {
	raw, _ := json.Marshal(v.PaymentInstructions)
	m := memberChargeModel{ID: string(v.ID), OrganizationID: string(v.OrganizationID), UnitID: string(v.UnitID), PersonID: string(v.PersonID), EnrollmentID: string(v.EnrollmentID), ContractID: string(v.ContractID), Sequence: v.Sequence, DueOn: v.DueOn, AmountCents: v.AmountCents, PaidCents: v.PaidCents, RefundedCents: v.RefundedCents, Currency: v.Currency, Status: v.Status, PaymentMethod: v.PaymentMethod, Provider: v.Provider, ProviderReference: v.ProviderReference, PaymentInstructions: string(raw), CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
	return mapMemberBillingError(databaseForContext(r.db, ctx).Create(&m).Error)
}
func (r MemberBillingGormRepository) UpdateCharge(ctx context.Context, v *domain.MemberCharge) error {
	return databaseForContext(r.db, ctx).Model(&memberChargeModel{}).Where("organization_id=? AND unit_id=? AND id=?", v.OrganizationID, v.UnitID, v.ID).Updates(map[string]any{"paid_cents": v.PaidCents, "refunded_cents": v.RefundedCents, "status": v.Status, "paid_at": v.PaidAt, "canceled_at": v.CanceledAt, "updated_at": v.UpdatedAt}).Error
}
func (r MemberBillingGormRepository) CreatePayment(ctx context.Context, v *domain.MemberPayment) error {
	var actor *string
	if v.ActorUserID != nil {
		x := string(*v.ActorUserID)
		actor = &x
	}
	m := memberPaymentModel{ID: string(v.ID), OrganizationID: string(v.OrganizationID), ChargeID: string(v.ChargeID), Kind: v.Kind, Method: v.Method, AmountCents: v.AmountCents, OccurredAt: v.OccurredAt, ActorUserID: actor, ProviderEventID: v.ProviderEventID, Reason: v.Reason, CreatedAt: v.CreatedAt}
	return mapMemberBillingError(databaseForContext(r.db, ctx).Create(&m).Error)
}
func (r MemberBillingGormRepository) listPayments(ctx context.Context, organizationID, chargeID domain.ID) ([]domain.MemberPayment, error) {
	var rows []memberPaymentModel
	if err := databaseForContext(r.db, ctx).Where("organization_id=? AND charge_id=?", organizationID, chargeID).Order("occurred_at,id").Find(&rows).Error; err != nil {
		return nil, err
	}
	values := make([]domain.MemberPayment, 0, len(rows))
	for _, m := range rows {
		var actor *domain.ID
		if m.ActorUserID != nil {
			x := domain.ID(*m.ActorUserID)
			actor = &x
		}
		values = append(values, domain.MemberPayment{ID: domain.ID(m.ID), OrganizationID: domain.ID(m.OrganizationID), ChargeID: domain.ID(m.ChargeID), Kind: m.Kind, Method: m.Method, AmountCents: m.AmountCents, OccurredAt: m.OccurredAt, ActorUserID: actor, ProviderEventID: m.ProviderEventID, Reason: m.Reason, CreatedAt: m.CreatedAt})
	}
	return values, nil
}
func (r MemberBillingGormRepository) ListPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.MemberBillingSnapshot, error) {
	var contracts []memberBillingContractModel
	if err := databaseForContext(r.db, ctx).Where("unit_id=? AND person_id=?", unitID, personID).Order("created_at").Find(&contracts).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MemberBillingSnapshot, 0, len(contracts))
	for _, m := range contracts {
		contract, _ := contractDomain(m, nil)
		charges, err := r.ListCharges(ctx, contract.OrganizationID, unitID, contract.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.MemberBillingSnapshot{Contract: contract, Charges: charges})
	}
	return out, nil
}
func mapMemberBillingError(err error) error {
	if err == nil {
		return nil
	}
	if stringsContains(err.Error(), "duplicate key") {
		return portrepo.ErrMemberBillingConflict
	}
	return err
}
func stringsContains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
