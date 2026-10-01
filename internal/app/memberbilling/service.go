package memberbilling

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	portservices "boxengage/backend/internal/ports/services"
	"github.com/google/uuid"
)

var (
	ErrNotFound          = repositories.ErrMemberBillingNotFound
	ErrConflict          = repositories.ErrMemberBillingConflict
	ErrIdempotencyReused = repositories.ErrMemberBillingIdempotencyReused
	ErrInvalidRequest    = errors.New("invalid member billing request")
	ErrInvalidTransition = errors.New("invalid member billing transition")
)

type StartInput struct {
	PaymentMethod string `json:"payment_method"`
	DueOn         string `json:"due_on"`
}
type ManualPaymentInput struct {
	Method      string `json:"method"`
	PaidAt      string `json:"paid_at"`
	Reason      string `json:"reason"`
	AmountCents int    `json:"amount_cents"`
}
type ReconcileInput struct {
	ProviderEventID string `json:"provider_event_id"`
	Status          string `json:"status"`
	OccurredAt      string `json:"occurred_at"`
	AmountCents     int    `json:"amount_cents"`
}
type RefundInput struct {
	AmountCents int    `json:"amount_cents"`
	Reason      string `json:"reason"`
	OccurredAt  string `json:"occurred_at"`
}

type Service struct {
	repository   repositories.MemberBillingRepository
	enrollments  enrollmentReader
	boxes        repositories.BoxRepository
	gateway      portservices.MemberPaymentGateway
	transactions repositories.TransactionManager
	now          func() time.Time
}

type enrollmentReader interface {
	Get(context.Context, domain.ID, domain.ID) (*domain.Enrollment, error)
}

func NewService(repository repositories.MemberBillingRepository, enrollments enrollmentReader, boxes repositories.BoxRepository, gateway portservices.MemberPaymentGateway, transactions repositories.TransactionManager) *Service {
	return &Service{repository: repository, enrollments: enrollments, boxes: boxes, gateway: gateway, transactions: transactions, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Snapshot(ctx context.Context, unitID, enrollmentID domain.ID) (*domain.MemberBillingSnapshot, error) {
	enrollment, organizationID, err := s.enrollment(ctx, unitID, enrollmentID)
	if err != nil {
		return nil, err
	}
	contract, err := s.repository.GetContractByEnrollment(ctx, organizationID, unitID, enrollment.ID)
	if errors.Is(err, ErrNotFound) {
		return &domain.MemberBillingSnapshot{Charges: []domain.MemberCharge{}, FinancialStatus: "not_configured"}, nil
	}
	if err != nil {
		return nil, err
	}
	charges, err := s.repository.ListCharges(ctx, organizationID, unitID, contract.ID)
	if err != nil {
		return nil, err
	}
	snapshot := summarize(contract, charges, s.now())
	return &snapshot, nil
}

func (s *Service) Start(ctx context.Context, unitID, enrollmentID, actorID domain.ID, key string, input StartInput) (*domain.MemberBillingSnapshot, bool, error) {
	if !validKey(key) || !validMethod(input.PaymentMethod, false) {
		return nil, false, ErrInvalidRequest
	}
	dueOn, err := time.Parse("2006-01-02", input.DueOn)
	if err != nil {
		return nil, false, ErrInvalidRequest
	}
	enrollment, organizationID, err := s.enrollment(ctx, unitID, enrollmentID)
	if err != nil {
		return nil, false, err
	}
	if enrollment.TotalPriceCents <= 0 {
		return nil, false, ErrInvalidRequest
	}
	fingerprint := hash(struct {
		EnrollmentID  domain.ID
		Method, DueOn string
	}{enrollmentID, input.PaymentMethod, input.DueOn})
	var result *domain.MemberBillingSnapshot
	replayed := false
	err = s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repository.LockCommand(tx, organizationID, "start", key); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(tx, organizationID, "start", key)
		if err != nil {
			return err
		}
		if previous != nil {
			if previous.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			replayed = true
			result, err = s.Snapshot(tx, unitID, enrollmentID)
			return err
		}
		if _, err = s.repository.GetContractByEnrollment(tx, organizationID, unitID, enrollmentID); err == nil {
			return ErrConflict
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		now := s.now()
		next := nextDue(dueOn, enrollment.Terms.BillingCycle)
		contract := &domain.MemberBillingContract{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, PersonID: enrollment.PersonID, EnrollmentID: enrollmentID, BillingCycle: enrollment.Terms.BillingCycle, AmountCents: enrollment.TotalPriceCents, Currency: "BRL", Status: "active", PaymentMethod: input.PaymentMethod, StartedOn: dueOn, NextDueOn: next, CreatedByUserID: actorID, CreatedAt: now, UpdatedAt: now}
		if err = s.repository.CreateContract(tx, contract); err != nil {
			return err
		}
		charge, err := s.newCharge(tx, contract, 1, dueOn)
		if err != nil {
			return err
		}
		if err = s.repository.CreateCommand(tx, &domain.MemberBillingCommand{OrganizationID: organizationID, Command: "start", IdempotencyKey: key, Fingerprint: fingerprint, ResultID: contract.ID, CreatedAt: now}); err != nil {
			return err
		}
		resultValue := summarize(contract, []domain.MemberCharge{*charge}, now)
		result = &resultValue
		return nil
	})
	return result, replayed, err
}

func (s *Service) MaterializeNext(ctx context.Context, unitID, contractID, actorID domain.ID, key string) (*domain.MemberCharge, bool, error) {
	if !validKey(key) {
		return nil, false, ErrInvalidRequest
	}
	organizationID, err := s.organization(ctx, unitID)
	if err != nil {
		return nil, false, ErrNotFound
	}
	fingerprint := hash(struct{ ContractID domain.ID }{contractID})
	var result *domain.MemberCharge
	replayed := false
	err = s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repository.LockCommand(tx, organizationID, "materialize", key); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(tx, organizationID, "materialize", key)
		if err != nil {
			return err
		}
		if previous != nil {
			if previous.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			replayed = true
			result, err = s.repository.GetCharge(tx, organizationID, unitID, previous.ResultID)
			return err
		}
		if err := s.repository.LockContract(tx, organizationID, contractID); err != nil {
			return err
		}
		contract, err := s.repository.GetContract(tx, organizationID, unitID, contractID)
		if err != nil {
			return err
		}
		if contract.Status != "active" || contract.NextDueOn == nil || contract.BillingCycle == domain.MembershipBillingOneTime {
			return ErrInvalidTransition
		}
		charges, err := s.repository.ListCharges(tx, organizationID, unitID, contractID)
		if err != nil {
			return err
		}
		result, err = s.newCharge(tx, contract, len(charges)+1, *contract.NextDueOn)
		if err != nil {
			return err
		}
		next := nextDue(*contract.NextDueOn, contract.BillingCycle)
		if err = s.repository.UpdateNextDue(tx, organizationID, contractID, next, s.now()); err != nil {
			return err
		}
		return s.repository.CreateCommand(tx, &domain.MemberBillingCommand{OrganizationID: organizationID, Command: "materialize", IdempotencyKey: key, Fingerprint: fingerprint, ResultID: result.ID, CreatedAt: s.now()})
	})
	_ = actorID
	return result, replayed, err
}

func (s *Service) RecordManual(ctx context.Context, unitID, chargeID, actorID domain.ID, key string, input ManualPaymentInput) (*domain.MemberCharge, bool, error) {
	return s.applyPayment(ctx, unitID, chargeID, actorID, key, "manual", input.AmountCents, input.Method, input.PaidAt, "", input.Reason)
}

func (s *Service) Reconcile(ctx context.Context, unitID, chargeID, actorID domain.ID, key string, input ReconcileInput) (*domain.MemberCharge, bool, error) {
	if strings.TrimSpace(input.ProviderEventID) == "" || input.Status != "paid" {
		return nil, false, ErrInvalidRequest
	}
	return s.applyPayment(ctx, unitID, chargeID, actorID, key, "provider_settlement", input.AmountCents, "provider", input.OccurredAt, input.ProviderEventID, "Conciliação do provedor")
}

func (s *Service) Refund(ctx context.Context, unitID, chargeID, actorID domain.ID, key string, input RefundInput) (*domain.MemberCharge, bool, error) {
	return s.applyPayment(ctx, unitID, chargeID, actorID, key, "refund", input.AmountCents, "refund", input.OccurredAt, "", input.Reason)
}

func (s *Service) applyPayment(ctx context.Context, unitID, chargeID, actorID domain.ID, key, kind string, amount int, method, occurredRaw, providerEvent, reason string) (*domain.MemberCharge, bool, error) {
	if !validKey(key) || amount <= 0 || (kind == "manual" && (!validMethod(method, true) || len(strings.TrimSpace(reason)) < 3)) {
		return nil, false, ErrInvalidRequest
	}
	occurred, err := time.Parse(time.RFC3339, occurredRaw)
	if err != nil {
		return nil, false, ErrInvalidRequest
	}
	organizationID, err := s.organization(ctx, unitID)
	if err != nil {
		return nil, false, ErrNotFound
	}
	fingerprint := hash(struct {
		ChargeID                        domain.ID
		Kind                            string
		Amount                          int
		Method, Occurred, Event, Reason string
	}{chargeID, kind, amount, method, occurredRaw, providerEvent, strings.TrimSpace(reason)})
	command := kind
	var result *domain.MemberCharge
	replayed := false
	err = s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repository.LockCommand(tx, organizationID, command, key); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(tx, organizationID, command, key)
		if err != nil {
			return err
		}
		if previous != nil {
			if previous.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			replayed = true
			result, err = s.repository.GetCharge(tx, organizationID, unitID, previous.ResultID)
			return err
		}
		charge, err := s.repository.GetChargeForUpdate(tx, organizationID, unitID, chargeID)
		if err != nil {
			return err
		}
		if charge.Status == "canceled" {
			return ErrInvalidTransition
		}
		if err := applyPaymentProjection(charge, kind, amount, occurred); err != nil {
			return err
		}
		charge.UpdatedAt = s.now()
		if err = s.repository.UpdateCharge(tx, charge); err != nil {
			return err
		}
		actor := actorID
		payment := &domain.MemberPayment{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, ChargeID: charge.ID, Kind: kind, Method: method, AmountCents: amount, OccurredAt: occurred.UTC(), ActorUserID: &actor, ProviderEventID: providerEvent, Reason: strings.TrimSpace(reason), CreatedAt: s.now()}
		if err = s.repository.CreatePayment(tx, payment); err != nil {
			return err
		}
		charge.Payments = append(charge.Payments, *payment)
		if err = s.repository.CreateCommand(tx, &domain.MemberBillingCommand{OrganizationID: organizationID, Command: command, IdempotencyKey: key, Fingerprint: fingerprint, ResultID: charge.ID, CreatedAt: s.now()}); err != nil {
			return err
		}
		result = charge
		return nil
	})
	return result, replayed, err
}

func applyPaymentProjection(charge *domain.MemberCharge, kind string, amount int, occurred time.Time) error {
	if amount <= 0 {
		return ErrInvalidRequest
	}
	if kind == "refund" {
		if amount > charge.PaidCents-charge.RefundedCents {
			return ErrInvalidTransition
		}
		charge.RefundedCents += amount
		if charge.RefundedCents == charge.PaidCents {
			charge.Status = "refunded"
		} else if charge.PaidCents == charge.AmountCents {
			charge.Status = "paid"
		} else {
			charge.Status = "partially_paid"
		}
		return nil
	}
	if amount > charge.AmountCents-charge.PaidCents {
		return ErrInvalidTransition
	}
	charge.PaidCents += amount
	if charge.PaidCents == charge.AmountCents {
		charge.Status = "paid"
		value := occurred.UTC()
		charge.PaidAt = &value
	} else {
		charge.Status = "partially_paid"
	}
	return nil
}

func (s *Service) newCharge(ctx context.Context, contract *domain.MemberBillingContract, sequence int, dueOn time.Time) (*domain.MemberCharge, error) {
	charge := &domain.MemberCharge{ID: domain.ID(uuid.NewString()), OrganizationID: contract.OrganizationID, UnitID: contract.UnitID, PersonID: contract.PersonID, EnrollmentID: contract.EnrollmentID, ContractID: contract.ID, Sequence: sequence, DueOn: dueOn, AmountCents: contract.AmountCents, Currency: contract.Currency, Status: "pending", PaymentMethod: contract.PaymentMethod, CreatedAt: s.now(), UpdatedAt: s.now()}
	if contract.PaymentMethod == "pix" || contract.PaymentMethod == "boleto" || contract.PaymentMethod == "card" {
		checkout, err := s.gateway.CreateCheckout(ctx, portservices.MemberPaymentRequest{ChargeID: charge.ID, AmountCents: charge.AmountCents, Currency: charge.Currency, Method: charge.PaymentMethod, DueOn: dueOn})
		if err != nil {
			return nil, err
		}
		charge.Provider, charge.ProviderReference, charge.PaymentInstructions = checkout.Provider, checkout.Reference, checkout.Instructions
	}
	if err := s.repository.CreateCharge(ctx, charge); err != nil {
		return nil, err
	}
	return charge, nil
}

func (s *Service) enrollment(ctx context.Context, unitID, enrollmentID domain.ID) (*domain.Enrollment, domain.ID, error) {
	organizationID, err := s.organization(ctx, unitID)
	if err != nil {
		return nil, "", ErrNotFound
	}
	item, err := s.enrollments.Get(ctx, unitID, enrollmentID)
	if err != nil {
		return nil, "", ErrNotFound
	}
	return item, organizationID, nil
}
func (s *Service) organization(ctx context.Context, unitID domain.ID) (domain.ID, error) {
	box, err := s.boxes.FindByID(ctx, unitID)
	if err != nil || box == nil || box.OrganizationID == "" {
		return "", ErrNotFound
	}
	return box.OrganizationID, nil
}
func validKey(value string) bool {
	if len(value) < 8 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:-", r)) {
			return false
		}
	}
	return true
}
func validMethod(value string, manual bool) bool {
	allowed := map[string]bool{"pix": true, "boleto": true, "card": true}
	if manual {
		allowed = map[string]bool{"pix": true, "cash": true, "card_machine": true}
	}
	return allowed[value]
}
func hash(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func nextDue(value time.Time, cycle domain.MembershipBillingCycle) *time.Time {
	months := 0
	switch cycle {
	case domain.MembershipBillingMonthly:
		months = 1
	case domain.MembershipBillingQuarterly:
		months = 3
	case domain.MembershipBillingSemiannual:
		months = 6
	case domain.MembershipBillingAnnual:
		months = 12
	default:
		return nil
	}
	targetMonth := time.Date(value.Year(), value.Month()+time.Month(months), 1, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location())
	lastDay := time.Date(targetMonth.Year(), targetMonth.Month()+1, 0, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location()).Day()
	day := value.Day()
	if day > lastDay {
		day = lastDay
	}
	next := time.Date(targetMonth.Year(), targetMonth.Month(), day, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), value.Location())
	return &next
}
func summarize(contract *domain.MemberBillingContract, charges []domain.MemberCharge, now time.Time) domain.MemberBillingSnapshot {
	copiedCharges := append([]domain.MemberCharge(nil), charges...)
	value := domain.MemberBillingSnapshot{Contract: contract, Charges: copiedCharges, FinancialStatus: "paid"}
	for i := range value.Charges {
		c := &value.Charges[i]
		if c.Status == "pending" && c.DueOn.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)) {
			c.Status = "overdue"
		}
		value.PaidAmountCents += c.PaidCents
		value.RefundedAmountCents += c.RefundedCents
		value.OpenAmountCents += c.AmountCents - c.PaidCents
		if c.Status == "overdue" {
			value.FinancialStatus = "overdue"
		} else if value.FinancialStatus != "overdue" && c.Status != "paid" && c.Status != "refunded" {
			value.FinancialStatus = "open"
		}
	}
	return value
}
