package billing

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
	"gorm.io/gorm"
)

var (
	ErrBillingDisabled            = errors.New("billing não está habilitado")
	ErrBillingPlanInvalid         = errors.New("plano financeiro inválido")
	ErrBillingPlanInUse           = errors.New("plano contratado não permite alterar preço ou franquias; crie uma nova versão")
	ErrBillingCustomerInvalid     = errors.New("dados de cobrança inválidos")
	ErrBillingSubscriptionExists  = errors.New("a academia já possui uma assinatura atual")
	ErrBillingSubscriptionAbsent  = errors.New("a academia não possui assinatura")
	ErrBillingTypeInvalid         = errors.New("forma de pagamento inválida")
	ErrBillingIdempotencyInvalid  = errors.New("chave de idempotência inválida")
	ErrBillingWebhookUnauthorized = errors.New("webhook financeiro não autorizado")
	ErrBillingWebhookInvalid      = errors.New("evento financeiro inválido")
)

var digitsOnly = regexp.MustCompile(`\D`)

type Service struct {
	repository repositories.BillingRepository
	gateway    services.BillingGateway
	audit      interface {
		SaveAuditLog(context.Context, *domain.AdminAuditLog) error
	}
	enabled      bool
	webhookToken string
	now          func() time.Time
}

func NewService(repository repositories.BillingRepository, gateway services.BillingGateway, audit interface {
	SaveAuditLog(context.Context, *domain.AdminAuditLog) error
}, enabled bool, webhookToken string) *Service {
	return &Service{repository: repository, gateway: gateway, audit: audit, enabled: enabled, webhookToken: webhookToken, now: time.Now}
}

type SavePlanInput struct {
	ID                  domain.ID
	Code                string
	Version             int
	Name                string
	Description         string
	MonthlyPriceCents   int64
	MonthlyMessageLimit int
	DailyMessageLimit   int
	PerDispatchLimit    int
	WarningPercent      int
	GracePeriodDays     int
	Active              bool
	AdminUserID         domain.ID
	Reason              string
	IPAddress           string
}

type SaveCustomerInput struct {
	BoxID                domain.ID
	LegalName            string
	CPFCNPJ              string
	Email                string
	Phone                string
	PostalCode           string
	Address              string
	AddressNumber        string
	Complement           string
	Province             string
	City                 string
	State                string
	NotificationDisabled bool
	AdminUserID          domain.ID
	Reason               string
	IPAddress            string
}

type CreateSubscriptionInput struct {
	BoxID       domain.ID
	PlanID      domain.ID
	BillingType domain.BillingType
	NextDueDate time.Time
	EndDate     *time.Time
	AdminUserID domain.ID
	Reason      string
	IPAddress   string
	RequestKey  string
}

func (s *Service) ListPlans(ctx context.Context, includeInactive bool) ([]domain.BillingPlan, error) {
	return s.repository.ListPlans(ctx, includeInactive)
}

func (s *Service) SavePlan(ctx context.Context, input SavePlanInput) (*domain.BillingPlan, error) {
	if err := validatePlanInput(input); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Reason) == "" {
		return nil, ErrBillingPlanInvalid
	}
	now := s.now()
	plan := domain.BillingPlan{
		ID: input.ID, Code: strings.ToLower(strings.TrimSpace(input.Code)), Version: input.Version,
		Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description),
		MonthlyPriceCents: input.MonthlyPriceCents, Currency: "BRL",
		MonthlyMessageLimit: input.MonthlyMessageLimit, DailyMessageLimit: input.DailyMessageLimit,
		PerDispatchLimit: input.PerDispatchLimit, WarningPercent: input.WarningPercent,
		GracePeriodDays: input.GracePeriodDays, Active: input.Active, UpdatedAt: now,
	}
	action := "billing_plan.created"
	if plan.ID == "" {
		plan.CreatedAt = now
		if err := s.repository.SavePlan(ctx, &plan); err != nil {
			return nil, err
		}
	} else {
		existing, err := s.repository.FindPlanByID(ctx, plan.ID)
		if err != nil {
			return nil, err
		}
		plan.Code = existing.Code
		plan.Version = existing.Version
		plan.CreatedAt = existing.CreatedAt
		inUse, err := s.repository.PlanHasSubscriptions(ctx, plan.ID)
		if err != nil {
			return nil, err
		}
		if inUse && planCommercialTermsChanged(*existing, plan) {
			return nil, ErrBillingPlanInUse
		}
		action = "billing_plan.updated"
		if err := s.repository.UpdatePlan(ctx, plan); err != nil {
			return nil, err
		}
	}
	if err := s.saveAudit(ctx, input.AdminUserID, action, "billing_plan", string(plan.ID), input.Reason, input.IPAddress, plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func planCommercialTermsChanged(before, after domain.BillingPlan) bool {
	return before.MonthlyPriceCents != after.MonthlyPriceCents ||
		before.MonthlyMessageLimit != after.MonthlyMessageLimit ||
		before.DailyMessageLimit != after.DailyMessageLimit ||
		before.PerDispatchLimit != after.PerDispatchLimit ||
		before.WarningPercent != after.WarningPercent ||
		before.GracePeriodDays != after.GracePeriodDays
}

func (s *Service) SaveCustomer(ctx context.Context, input SaveCustomerInput) (*domain.BillingCustomer, error) {
	if !s.enabled {
		return nil, ErrBillingDisabled
	}
	if err := validateCustomerInput(input); err != nil {
		return nil, err
	}
	providerInput := services.CreateBillingCustomerInput{
		Name: strings.TrimSpace(input.LegalName), CPFCNPJ: digitsOnly.ReplaceAllString(input.CPFCNPJ, ""),
		Email: strings.TrimSpace(strings.ToLower(input.Email)), Phone: digitsOnly.ReplaceAllString(input.Phone, ""),
		PostalCode: digitsOnly.ReplaceAllString(input.PostalCode, ""), Address: strings.TrimSpace(input.Address),
		AddressNumber: strings.TrimSpace(input.AddressNumber), Complement: strings.TrimSpace(input.Complement),
		Province: strings.TrimSpace(input.Province), ExternalReference: string(input.BoxID),
		NotificationDisabled: input.NotificationDisabled,
	}
	customer, err := s.repository.FindCustomerByBoxID(ctx, input.BoxID)
	action := "billing_customer.updated"
	if errors.Is(err, gorm.ErrRecordNotFound) {
		customer = &domain.BillingCustomer{BoxID: input.BoxID, Provider: "asaas"}
		action = "billing_customer.created"
	} else if err != nil {
		return nil, err
	}
	if customer.ProviderCustomerID == "" {
		found, findErr := s.gateway.FindCustomerByExternalReference(ctx, string(input.BoxID))
		if findErr != nil {
			return nil, findErr
		}
		if found == nil {
			found, findErr = s.gateway.CreateCustomer(ctx, providerInput)
			if findErr != nil {
				return nil, findErr
			}
		}
		customer.ProviderCustomerID = found.ID
	} else if err := s.gateway.UpdateCustomer(ctx, customer.ProviderCustomerID, providerInput); err != nil {
		return nil, err
	}
	customer.LegalName = providerInput.Name
	customer.CPFCNPJ = providerInput.CPFCNPJ
	customer.Email = providerInput.Email
	customer.Phone = providerInput.Phone
	customer.PostalCode = providerInput.PostalCode
	customer.Address = providerInput.Address
	customer.AddressNumber = providerInput.AddressNumber
	customer.Complement = providerInput.Complement
	customer.Province = providerInput.Province
	customer.City = strings.TrimSpace(input.City)
	customer.State = strings.ToUpper(strings.TrimSpace(input.State))
	customer.NotificationDisabled = input.NotificationDisabled
	if customer.ID == "" {
		err = s.repository.SaveCustomer(ctx, customer)
	} else {
		err = s.repository.UpdateCustomer(ctx, *customer)
	}
	if err != nil {
		return nil, err
	}
	if err := s.saveAudit(ctx, input.AdminUserID, action, "billing_customer", string(input.BoxID), input.Reason, input.IPAddress, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *Service) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*domain.BillingSubscription, error) {
	if !s.enabled {
		return nil, ErrBillingDisabled
	}
	if !validBillingType(input.BillingType) || input.NextDueDate.IsZero() || strings.TrimSpace(input.Reason) == "" {
		return nil, ErrBillingTypeInvalid
	}
	requestKey := strings.TrimSpace(input.RequestKey)
	if len(requestKey) < 8 || len(requestKey) > 128 {
		return nil, ErrBillingIdempotencyInvalid
	}
	externalReference := fmt.Sprintf("subscription:%s:%s", input.BoxID, requestKey)
	if current, err := s.repository.FindCurrentSubscriptionByBoxID(ctx, input.BoxID); err == nil {
		if current.ExternalReference == externalReference {
			plan, planErr := s.repository.FindPlanByID(ctx, current.PlanID)
			if planErr != nil {
				return nil, planErr
			}
			if applyErr := s.applySubscriptionEntitlements(ctx, *current, *plan); applyErr != nil {
				return nil, applyErr
			}
			return current, nil
		}
		return nil, ErrBillingSubscriptionExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	customer, err := s.repository.FindCustomerByBoxID(ctx, input.BoxID)
	if err != nil || customer.ProviderCustomerID == "" {
		return nil, ErrBillingCustomerInvalid
	}
	plan, err := s.repository.FindPlanByID(ctx, input.PlanID)
	if err != nil {
		return nil, err
	}
	if !plan.Active {
		return nil, ErrBillingPlanInvalid
	}
	now := s.now()
	providerSubscription, err := s.gateway.FindSubscriptionByExternalReference(ctx, externalReference)
	if err != nil {
		return nil, err
	}
	if providerSubscription == nil {
		providerSubscription, err = s.gateway.CreateSubscription(ctx, services.CreateBillingSubscriptionInput{
			CustomerID: customer.ProviderCustomerID, BillingType: input.BillingType,
			NextDueDate: input.NextDueDate, ValueCents: plan.MonthlyPriceCents,
			Description: "EngageFit — " + plan.Name, ExternalReference: externalReference, EndDate: input.EndDate,
		})
		if err != nil {
			return nil, err
		}
	}
	status := domain.BillingStatusPending
	if input.NextDueDate.After(now) {
		status = domain.BillingStatusTrialing
	}
	subscription := &domain.BillingSubscription{
		BoxID: input.BoxID, BillingCustomerID: customer.ID, PlanID: plan.ID, Provider: "asaas",
		ProviderSubscriptionID: providerSubscription.ID, Status: status, BillingType: input.BillingType,
		NextDueDate: input.NextDueDate, StartedAt: now, ExternalReference: externalReference,
	}
	if err := s.repository.SaveSubscription(ctx, subscription); err != nil {
		return nil, err
	}
	if err := s.applySubscriptionEntitlements(ctx, *subscription, *plan); err != nil {
		return nil, err
	}
	if err := s.saveAudit(ctx, input.AdminUserID, "billing_subscription.created", "billing_subscription", string(subscription.ID), input.Reason, input.IPAddress, subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

func (s *Service) applySubscriptionEntitlements(ctx context.Context, subscription domain.BillingSubscription, plan domain.BillingPlan) error {
	if err := s.repository.ApplyPlanMessagingPolicy(ctx, subscription.BoxID, plan); err != nil {
		return err
	}
	now := s.now()
	blocked := !subscription.AllowsAccess(now)
	reason := ""
	if blocked {
		switch subscription.Status {
		case domain.BillingStatusPending:
			reason = "assinatura aguardando primeiro pagamento"
		case domain.BillingStatusSuspended, domain.BillingStatusPastDue:
			reason = "pagamento vencido após período de tolerância"
		default:
			reason = "assinatura sem acesso financeiro"
		}
	}
	return s.repository.SetBillingAccess(ctx, subscription.BoxID, blocked, reason, now)
}

func (s *Service) CancelSubscription(ctx context.Context, boxID, adminID domain.ID, reason, ipAddress string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrBillingSubscriptionAbsent
	}
	subscription, err := s.repository.FindCurrentSubscriptionByBoxID(ctx, boxID)
	if err != nil {
		return ErrBillingSubscriptionAbsent
	}
	if subscription.ProviderSubscriptionID != "" {
		if err := s.gateway.CancelSubscription(ctx, subscription.ProviderSubscriptionID); err != nil {
			return err
		}
	}
	now := s.now()
	subscription.Status = domain.BillingStatusCanceled
	subscription.CanceledAt = &now
	subscription.CancelAtPeriodEnd = false
	if err := s.repository.UpdateSubscription(ctx, *subscription); err != nil {
		return err
	}
	if err := s.repository.SetBillingAccess(ctx, boxID, true, "assinatura cancelada", now); err != nil {
		return err
	}
	return s.saveAudit(ctx, adminID, "billing_subscription.canceled", "billing_subscription", string(subscription.ID), reason, ipAddress, subscription)
}

func (s *Service) GrantGrace(ctx context.Context, boxID, adminID domain.ID, until time.Time, reason, ipAddress string) error {
	if until.Before(s.now()) || strings.TrimSpace(reason) == "" {
		return ErrBillingSubscriptionAbsent
	}
	subscription, err := s.repository.FindCurrentSubscriptionByBoxID(ctx, boxID)
	if err != nil {
		return ErrBillingSubscriptionAbsent
	}
	subscription.Status = domain.BillingStatusPastDue
	subscription.GraceUntil = datePtr(until)
	if err := s.repository.UpdateSubscription(ctx, *subscription); err != nil {
		return err
	}
	if err := s.repository.SetBillingAccess(ctx, boxID, false, "", s.now()); err != nil {
		return err
	}
	return s.saveAudit(ctx, adminID, "billing_subscription.grace_granted", "billing_subscription", string(subscription.ID), reason, ipAddress, subscription)
}

func (s *Service) ListOverviews(ctx context.Context) ([]domain.BillingOverview, error) {
	return s.repository.ListOverviews(ctx)
}

func (s *Service) Summary(ctx context.Context) (*domain.BillingSummary, error) {
	return s.repository.Summary(ctx, s.now())
}

func (s *Service) BoxBilling(ctx context.Context, boxID domain.ID) (*domain.BillingOverview, []domain.BillingInvoice, error) {
	overviews, err := s.repository.ListOverviews(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range overviews {
		if item.Box.ID == boxID {
			invoices, err := s.repository.ListInvoicesByBoxID(ctx, boxID)
			return &item, invoices, err
		}
	}
	return nil, nil, gorm.ErrRecordNotFound
}

func (s *Service) ProcessWebhook(ctx context.Context, token string, raw []byte) error {
	if !s.enabled {
		return ErrBillingDisabled
	}
	if len(token) != len(s.webhookToken) || subtle.ConstantTimeCompare([]byte(token), []byte(s.webhookToken)) != 1 {
		return ErrBillingWebhookUnauthorized
	}
	var envelope struct {
		ID      string `json:"id"`
		Event   string `json:"event"`
		Payment struct {
			ID           string `json:"id"`
			Subscription string `json:"subscription"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.ID == "" || envelope.Event == "" {
		return ErrBillingWebhookInvalid
	}
	event := &domain.BillingWebhookEvent{
		Provider: "asaas", ProviderEventID: envelope.ID, EventType: envelope.Event,
		ProviderPaymentID: envelope.Payment.ID, Payload: raw, Status: "pending", ReceivedAt: s.now(),
	}
	created, err := s.repository.SaveWebhookEvent(ctx, event)
	if err != nil || !created {
		return err
	}
	if envelope.Payment.ID == "" {
		_ = s.repository.MarkWebhookEventProcessed(ctx, event.ID, s.now())
		return nil
	}
	payment, err := s.gateway.GetPayment(ctx, envelope.Payment.ID)
	if err != nil {
		_ = s.repository.MarkWebhookEventFailed(ctx, event.ID, "provider_payment_lookup_failed")
		return err
	}
	if err := s.applyPayment(ctx, paymentEventForStatus(payment.Status), *payment); err != nil {
		_ = s.repository.MarkWebhookEventFailed(ctx, event.ID, "payment_projection_failed")
		return err
	}
	return s.repository.MarkWebhookEventProcessed(ctx, event.ID, s.now())
}

func (s *Service) Reconcile(ctx context.Context) error {
	if !s.enabled {
		return ErrBillingDisabled
	}
	subscriptions, err := s.repository.ListSubscriptions(ctx)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		if subscription.Status == domain.BillingStatusCanceled || subscription.ProviderSubscriptionID == "" {
			continue
		}
		payments, err := s.gateway.ListSubscriptionPayments(ctx, subscription.ProviderSubscriptionID)
		if err != nil {
			return err
		}
		sort.SliceStable(payments, func(i, j int) bool {
			return payments[i].DueDate.Before(payments[j].DueDate)
		})
		for _, payment := range payments {
			if err := s.applyPayment(ctx, paymentEventForStatus(payment.Status), payment); err != nil {
				return err
			}
		}
		refreshed, err := s.repository.FindSubscriptionByProviderID(ctx, subscription.ProviderSubscriptionID)
		if err != nil {
			return err
		}
		now := s.now()
		refreshed.LastReconciledAt = &now
		if err := s.repository.UpdateSubscription(ctx, *refreshed); err != nil {
			return err
		}
	}
	return s.EnforceOverdue(ctx)
}

func (s *Service) EnforceOverdue(ctx context.Context) error {
	now := s.now()
	subscriptions, err := s.repository.ListSubscriptionsDueForEnforcement(ctx, now)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		subscription.Status = domain.BillingStatusSuspended
		if err := s.repository.UpdateSubscription(ctx, subscription); err != nil {
			return err
		}
		if err := s.repository.SetBillingAccess(ctx, subscription.BoxID, true, "pagamento vencido após período de tolerância", now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyPayment(ctx context.Context, eventType string, payment services.BillingProviderPayment) error {
	if payment.SubscriptionID == "" {
		return nil
	}
	subscription, err := s.repository.FindSubscriptionByProviderID(ctx, payment.SubscriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	invoice := providerPaymentToDomain(*subscription, payment)
	if err := s.repository.UpsertInvoice(ctx, &invoice); err != nil {
		return err
	}
	plan, err := s.repository.FindPlanByID(ctx, subscription.PlanID)
	if err != nil {
		return err
	}
	now := s.now()
	switch eventType {
	case "PAYMENT_CONFIRMED", "PAYMENT_RECEIVED":
		subscription.Status = domain.BillingStatusActive
		subscription.GraceUntil = nil
		subscription.CurrentPeriodStart = datePtr(payment.DueDate)
		periodEnd := payment.DueDate.AddDate(0, 1, 0).AddDate(0, 0, -1)
		subscription.CurrentPeriodEnd = datePtr(periodEnd)
		subscription.NextDueDate = payment.DueDate.AddDate(0, 1, 0)
		if err := s.repository.SetBillingAccess(ctx, subscription.BoxID, false, "", now); err != nil {
			return err
		}
	case "PAYMENT_OVERDUE", "PAYMENT_REPROVED_BY_RISK_ANALYSIS":
		subscription.Status = domain.BillingStatusPastDue
		graceUntil := payment.DueDate.AddDate(0, 0, plan.GracePeriodDays)
		subscription.GraceUntil = datePtr(graceUntil)
		if now.After(graceUntil.Add(24*time.Hour - time.Nanosecond)) {
			subscription.Status = domain.BillingStatusSuspended
			if err := s.repository.SetBillingAccess(ctx, subscription.BoxID, true, "pagamento vencido após período de tolerância", now); err != nil {
				return err
			}
		}
	case "PAYMENT_REFUNDED", "PAYMENT_PARTIALLY_REFUNDED", "PAYMENT_CHARGEBACK_REQUESTED":
		subscription.Status = domain.BillingStatusSuspended
		if err := s.repository.SetBillingAccess(ctx, subscription.BoxID, true, "pagamento estornado ou em chargeback", now); err != nil {
			return err
		}
	case "PAYMENT_CREATED", "PAYMENT_UPDATED", "PAYMENT_AWAITING_RISK_ANALYSIS":
		if subscription.Status == domain.BillingStatusPending && payment.DueDate.After(now) {
			subscription.Status = domain.BillingStatusTrialing
			if err := s.repository.SetBillingAccess(ctx, subscription.BoxID, false, "", now); err != nil {
				return err
			}
		}
	}
	return s.repository.UpdateSubscription(ctx, *subscription)
}

func providerPaymentToDomain(subscription domain.BillingSubscription, payment services.BillingProviderPayment) domain.BillingInvoice {
	return domain.BillingInvoice{
		BoxID: subscription.BoxID, SubscriptionID: subscription.ID, Provider: "asaas",
		ProviderPaymentID: payment.ID, Status: payment.Status, BillingType: payment.BillingType,
		ValueCents: payment.ValueCents, NetValueCents: payment.NetValueCents, DueDate: payment.DueDate,
		OriginalDueDate: payment.OriginalDueDate, ConfirmedAt: payment.ConfirmedAt, ReceivedAt: payment.ReceivedAt,
		InvoiceURL: payment.InvoiceURL, BankSlipURL: payment.BankSlipURL,
		ExternalReference: payment.ExternalReference, Description: payment.Description,
	}
}

func paymentEventForStatus(status string) string {
	switch status {
	case "CONFIRMED":
		return "PAYMENT_CONFIRMED"
	case "RECEIVED", "RECEIVED_IN_CASH":
		return "PAYMENT_RECEIVED"
	case "OVERDUE":
		return "PAYMENT_OVERDUE"
	case "REPROVED_BY_RISK_ANALYSIS":
		return "PAYMENT_REPROVED_BY_RISK_ANALYSIS"
	case "REFUNDED":
		return "PAYMENT_REFUNDED"
	case "PARTIALLY_REFUNDED":
		return "PAYMENT_PARTIALLY_REFUNDED"
	case "CHARGEBACK_REQUESTED", "CHARGEBACK_DISPUTE", "AWAITING_CHARGEBACK_REVERSAL":
		return "PAYMENT_CHARGEBACK_REQUESTED"
	case "AWAITING_RISK_ANALYSIS":
		return "PAYMENT_AWAITING_RISK_ANALYSIS"
	default:
		return "PAYMENT_UPDATED"
	}
}

func validatePlanInput(input SavePlanInput) error {
	if strings.TrimSpace(input.Name) == "" || input.MonthlyPriceCents < 0 ||
		input.MonthlyMessageLimit < 0 || input.DailyMessageLimit < 0 || input.PerDispatchLimit < 0 ||
		input.DailyMessageLimit > input.MonthlyMessageLimit || input.PerDispatchLimit > input.DailyMessageLimit ||
		input.WarningPercent < 1 || input.WarningPercent > 100 || input.GracePeriodDays < 0 || input.GracePeriodDays > 90 {
		return ErrBillingPlanInvalid
	}
	if input.ID == "" && (strings.TrimSpace(input.Code) == "" || input.Version < 1) {
		return ErrBillingPlanInvalid
	}
	return nil
}

func validateCustomerInput(input SaveCustomerInput) error {
	if input.BoxID == "" || strings.TrimSpace(input.LegalName) == "" || !strings.Contains(input.Email, "@") {
		return ErrBillingCustomerInvalid
	}
	document := digitsOnly.ReplaceAllString(input.CPFCNPJ, "")
	if document != "" && len(document) != 11 && len(document) != 14 {
		return ErrBillingCustomerInvalid
	}
	state := strings.TrimSpace(input.State)
	if state != "" && len(state) != 2 {
		return ErrBillingCustomerInvalid
	}
	if strings.TrimSpace(input.Reason) == "" {
		return ErrBillingCustomerInvalid
	}
	return nil
}

func validBillingType(value domain.BillingType) bool {
	return value == domain.BillingTypeUndefined || value == domain.BillingTypeBoleto ||
		value == domain.BillingTypeCreditCard || value == domain.BillingTypePix
}

func datePtr(value time.Time) *time.Time {
	date := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	return &date
}

func (s *Service) saveAudit(ctx context.Context, adminID domain.ID, action, targetType, targetID, reason, ipAddress string, after any) error {
	if s.audit == nil || adminID == "" {
		return nil
	}
	afterJSON, _ := json.Marshal(after)
	return s.audit.SaveAuditLog(ctx, &domain.AdminAuditLog{
		AdminUserID: adminID, Action: action, TargetType: targetType, TargetID: targetID,
		AfterData: afterJSON, Reason: strings.TrimSpace(reason), IPAddress: ipAddress, CreatedAt: s.now(),
	})
}
