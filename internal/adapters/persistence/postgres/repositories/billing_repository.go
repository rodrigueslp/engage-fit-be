package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r BillingGormRepository) ListPlans(ctx context.Context, includeInactive bool) ([]domain.BillingPlan, error) {
	var rows []models.BillingPlanModel
	query := r.db.WithContext(ctx).Order("code ASC, version DESC")
	if !includeInactive {
		query = query.Where("active = TRUE")
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.BillingPlan, 0, len(rows))
	for _, row := range rows {
		result = append(result, billingPlanToDomain(row))
	}
	return result, nil
}

func (r BillingGormRepository) FindPlanByID(ctx context.Context, id domain.ID) (*domain.BillingPlan, error) {
	var row models.BillingPlanModel
	if err := r.db.WithContext(ctx).Where("id = ?", stringID(id)).First(&row).Error; err != nil {
		return nil, err
	}
	item := billingPlanToDomain(row)
	return &item, nil
}

func (r BillingGormRepository) FindPlanByCode(ctx context.Context, code string) (*domain.BillingPlan, error) {
	var row models.BillingPlanModel
	if err := r.db.WithContext(ctx).Where("code = ? AND active = TRUE", code).Order("version DESC").First(&row).Error; err != nil {
		return nil, err
	}
	item := billingPlanToDomain(row)
	return &item, nil
}

func (r BillingGormRepository) SavePlan(ctx context.Context, plan *domain.BillingPlan) error {
	if err := ensureID(&plan.ID); err != nil {
		return err
	}
	now := time.Now()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	return r.db.WithContext(ctx).Create(&models.BillingPlanModel{
		ID: stringID(plan.ID), Code: plan.Code, Version: plan.Version, Name: plan.Name, Description: plan.Description,
		MonthlyPriceCents: plan.MonthlyPriceCents, Currency: plan.Currency, MonthlyMessageLimit: plan.MonthlyMessageLimit,
		DailyMessageLimit: plan.DailyMessageLimit, PerDispatchLimit: plan.PerDispatchLimit,
		WarningPercent: plan.WarningPercent, GracePeriodDays: plan.GracePeriodDays, Active: plan.Active,
		CreatedAt: plan.CreatedAt, UpdatedAt: plan.UpdatedAt,
	}).Error
}

func (r BillingGormRepository) UpdatePlan(ctx context.Context, plan domain.BillingPlan) error {
	plan.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Model(&models.BillingPlanModel{}).Where("id = ?", stringID(plan.ID)).Updates(map[string]any{
		"name": plan.Name, "description": plan.Description, "monthly_price_cents": plan.MonthlyPriceCents,
		"monthly_message_limit": plan.MonthlyMessageLimit, "daily_message_limit": plan.DailyMessageLimit,
		"per_dispatch_limit": plan.PerDispatchLimit, "warning_percent": plan.WarningPercent,
		"grace_period_days": plan.GracePeriodDays, "active": plan.Active, "updated_at": plan.UpdatedAt,
	}).Error
}

func (r BillingGormRepository) PlanHasSubscriptions(ctx context.Context, id domain.ID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.BillingSubscriptionModel{}).
		Where("plan_id = ?", stringID(id)).Count(&count).Error
	return count > 0, err
}

func (r BillingGormRepository) FindCustomerByBoxID(ctx context.Context, boxID domain.ID) (*domain.BillingCustomer, error) {
	var row models.BillingCustomerModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).First(&row).Error; err != nil {
		return nil, err
	}
	item := billingCustomerToDomain(row)
	return &item, nil
}

func (r BillingGormRepository) SaveCustomer(ctx context.Context, customer *domain.BillingCustomer) error {
	if err := ensureID(&customer.ID); err != nil {
		return err
	}
	now := time.Now()
	customer.CreatedAt = now
	customer.UpdatedAt = now
	return r.db.WithContext(ctx).Create(billingCustomerToModel(*customer)).Error
}

func (r BillingGormRepository) UpdateCustomer(ctx context.Context, customer domain.BillingCustomer) error {
	customer.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(billingCustomerToModel(customer)).Error
}

func (r BillingGormRepository) FindCurrentSubscriptionByBoxID(ctx context.Context, boxID domain.ID) (*domain.BillingSubscription, error) {
	var row models.BillingSubscriptionModel
	if err := r.db.WithContext(ctx).Where("box_id = ? AND status <> 'canceled'", stringID(boxID)).Order("created_at DESC").First(&row).Error; err != nil {
		return nil, err
	}
	item := billingSubscriptionToDomain(row)
	return &item, nil
}

func (r BillingGormRepository) FindSubscriptionByProviderID(ctx context.Context, providerID string) (*domain.BillingSubscription, error) {
	var row models.BillingSubscriptionModel
	if err := r.db.WithContext(ctx).Where("provider_subscription_id = ?", providerID).First(&row).Error; err != nil {
		return nil, err
	}
	item := billingSubscriptionToDomain(row)
	return &item, nil
}

func (r BillingGormRepository) ListSubscriptions(ctx context.Context) ([]domain.BillingSubscription, error) {
	var rows []models.BillingSubscriptionModel
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.BillingSubscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, billingSubscriptionToDomain(row))
	}
	return result, nil
}

func (r BillingGormRepository) ListSubscriptionsDueForEnforcement(ctx context.Context, now time.Time) ([]domain.BillingSubscription, error) {
	var rows []models.BillingSubscriptionModel
	if err := r.db.WithContext(ctx).
		Where("status = 'past_due' AND grace_until IS NOT NULL AND grace_until < ?", now.Format("2006-01-02")).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.BillingSubscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, billingSubscriptionToDomain(row))
	}
	return result, nil
}

func (r BillingGormRepository) SaveSubscription(ctx context.Context, subscription *domain.BillingSubscription) error {
	if err := ensureID(&subscription.ID); err != nil {
		return err
	}
	now := time.Now()
	if subscription.StartedAt.IsZero() {
		subscription.StartedAt = now
	}
	subscription.CreatedAt = now
	subscription.UpdatedAt = now
	return r.db.WithContext(ctx).Create(billingSubscriptionToModel(*subscription)).Error
}

func (r BillingGormRepository) UpdateSubscription(ctx context.Context, subscription domain.BillingSubscription) error {
	subscription.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(billingSubscriptionToModel(subscription)).Error
}

func (r BillingGormRepository) UpsertInvoice(ctx context.Context, invoice *domain.BillingInvoice) error {
	if invoice.ID == "" {
		if err := ensureID(&invoice.ID); err != nil {
			return err
		}
	}
	now := time.Now()
	if invoice.CreatedAt.IsZero() {
		invoice.CreatedAt = now
	}
	invoice.UpdatedAt = now
	row := billingInvoiceToModel(*invoice)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider_payment_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "billing_type", "value_cents", "net_value_cents", "due_date", "original_due_date",
			"confirmed_at", "received_at", "invoice_url", "bank_slip_url", "external_reference",
			"description", "updated_at",
		}),
	}).Create(&row).Error
}

func (r BillingGormRepository) FindInvoiceByProviderPaymentID(ctx context.Context, providerPaymentID string) (*domain.BillingInvoice, error) {
	var row models.BillingInvoiceModel
	if err := r.db.WithContext(ctx).Where("provider_payment_id = ?", providerPaymentID).First(&row).Error; err != nil {
		return nil, err
	}
	item := billingInvoiceToDomain(row)
	return &item, nil
}

func (r BillingGormRepository) ListInvoicesByBoxID(ctx context.Context, boxID domain.ID) ([]domain.BillingInvoice, error) {
	var rows []models.BillingInvoiceModel
	if err := r.db.WithContext(ctx).Where("box_id = ?", stringID(boxID)).Order("due_date DESC").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.BillingInvoice, 0, len(rows))
	for _, row := range rows {
		result = append(result, billingInvoiceToDomain(row))
	}
	return result, nil
}

func (r BillingGormRepository) SaveWebhookEvent(ctx context.Context, event *domain.BillingWebhookEvent) (bool, error) {
	if err := ensureID(&event.ID); err != nil {
		return false, err
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now()
	}
	row := models.BillingWebhookEventModel{
		ID: stringID(event.ID), Provider: event.Provider, ProviderEventID: event.ProviderEventID,
		EventType: event.EventType, ProviderPaymentID: event.ProviderPaymentID, Payload: event.Payload,
		Status: event.Status, Attempts: event.Attempts, ErrorMessage: event.ErrorMessage,
		ReceivedAt: event.ReceivedAt, ProcessedAt: event.ProcessedAt,
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "provider_event_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"id": row.ID, "payload": row.Payload, "status": "pending", "received_at": row.ReceivedAt,
			"processed_at": nil, "error_message": "",
		}),
		Where: clause.Where{Exprs: []clause.Expression{clause.Eq{Column: "billing_webhook_events.status", Value: "failed"}}},
	}).Create(&row)
	return result.RowsAffected == 1, result.Error
}

func (r BillingGormRepository) MarkWebhookEventProcessed(ctx context.Context, id domain.ID, processedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.BillingWebhookEventModel{}).Where("id = ?", stringID(id)).
		Updates(map[string]any{"status": "processed", "processed_at": processedAt, "attempts": gorm.Expr("attempts + 1"), "error_message": ""}).Error
}

func (r BillingGormRepository) MarkWebhookEventFailed(ctx context.Context, id domain.ID, message string) error {
	return r.db.WithContext(ctx).Model(&models.BillingWebhookEventModel{}).Where("id = ?", stringID(id)).
		Updates(map[string]any{"status": "failed", "attempts": gorm.Expr("attempts + 1"), "error_message": message}).Error
}

func (r BillingGormRepository) SetBillingAccess(ctx context.Context, boxID domain.ID, blocked bool, reason string, changedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current bool
		if err := tx.Raw("SELECT billing_access_blocked FROM boxes WHERE id = ? FOR UPDATE", stringID(boxID)).Scan(&current).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.BoxModel{}).Where("id = ?", stringID(boxID)).Updates(map[string]any{
			"billing_access_blocked": blocked, "billing_access_reason": reason,
			"billing_access_changed_at": changedAt, "updated_at": changedAt,
		}).Error; err != nil {
			return err
		}
		if blocked && !current {
			return tx.Model(&models.UserModel{}).Where("box_id = ?", stringID(boxID)).
				UpdateColumn("auth_version", gorm.Expr("auth_version + 1")).Error
		}
		return nil
	})
}

func (r BillingGormRepository) ApplyPlanMessagingPolicy(ctx context.Context, boxID domain.ID, plan domain.BillingPlan) error {
	id, err := newID()
	if err != nil {
		return err
	}
	const estimatedCostMicros = int64(100000)
	return r.db.WithContext(ctx).Exec(`INSERT INTO messaging_policies (
		id, scope, box_id, daily_message_limit, monthly_message_limit, per_dispatch_limit,
		estimated_cost_micros_per_message, daily_cost_limit_micros, monthly_cost_limit_micros,
		currency, warning_percent, timezone, blocked, created_at, updated_at
	) VALUES (?, 'box', ?, ?, ?, ?, ?, ?, ?, 'USD', ?, 'America/Sao_Paulo', FALSE, NOW(), NOW())
	ON CONFLICT (box_id) WHERE scope = 'box' DO UPDATE SET
		daily_message_limit = EXCLUDED.daily_message_limit,
		monthly_message_limit = EXCLUDED.monthly_message_limit,
		per_dispatch_limit = EXCLUDED.per_dispatch_limit,
		estimated_cost_micros_per_message = EXCLUDED.estimated_cost_micros_per_message,
		daily_cost_limit_micros = EXCLUDED.daily_cost_limit_micros,
		monthly_cost_limit_micros = EXCLUDED.monthly_cost_limit_micros,
		warning_percent = EXCLUDED.warning_percent,
		updated_at = NOW()`,
		stringID(id), stringID(boxID), plan.DailyMessageLimit, plan.MonthlyMessageLimit,
		plan.PerDispatchLimit, estimatedCostMicros,
		int64(plan.DailyMessageLimit)*estimatedCostMicros,
		int64(plan.MonthlyMessageLimit)*estimatedCostMicros,
		plan.WarningPercent).Error
}

func (r BillingGormRepository) ListOverviews(ctx context.Context) ([]domain.BillingOverview, error) {
	var boxRows []models.BoxModel
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&boxRows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.BillingOverview, 0, len(boxRows))
	for _, boxRow := range boxRows {
		item := domain.BillingOverview{Box: boxToDomain(boxRow)}
		customer, err := r.FindCustomerByBoxID(ctx, item.Box.ID)
		if err == nil {
			item.Customer = customer
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		subscription, err := r.FindCurrentSubscriptionByBoxID(ctx, item.Box.ID)
		if err == nil {
			item.Subscription = subscription
			plan, planErr := r.FindPlanByID(ctx, subscription.PlanID)
			if planErr != nil {
				return nil, planErr
			}
			item.Plan = plan
			var invoiceRow models.BillingInvoiceModel
			invoiceErr := r.db.WithContext(ctx).Where("subscription_id = ?", stringID(subscription.ID)).Order("due_date DESC").First(&invoiceRow).Error
			if invoiceErr == nil {
				invoice := billingInvoiceToDomain(invoiceRow)
				item.LatestInvoice = &invoice
			} else if !errors.Is(invoiceErr, gorm.ErrRecordNotFound) {
				return nil, invoiceErr
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r BillingGormRepository) Summary(ctx context.Context, now time.Time) (*domain.BillingSummary, error) {
	var result domain.BillingSummary
	if err := r.db.WithContext(ctx).Raw(`SELECT
		COALESCE(SUM(CASE WHEN s.status IN ('active','past_due') THEN p.monthly_price_cents ELSE 0 END), 0) AS monthly_recurring_revenue_cents,
		COUNT(*) FILTER (WHERE s.status = 'active') AS active_subscriptions,
		COUNT(*) FILTER (WHERE s.status = 'past_due') AS past_due_subscriptions,
		COUNT(*) FILTER (WHERE s.status = 'suspended') AS suspended_subscriptions,
		COUNT(*) FILTER (WHERE s.status = 'canceled') AS canceled_subscriptions
		FROM billing_subscriptions s JOIN billing_plans p ON p.id = s.plan_id`).Scan(&result).Error; err != nil {
		return nil, err
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonth := monthStart.AddDate(0, 1, 0)
	var amounts struct {
		Pending  int64
		Received int64
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT
		COALESCE(SUM(value_cents) FILTER (WHERE status IN ('PENDING','OVERDUE','AWAITING_RISK_ANALYSIS')), 0) AS pending,
		COALESCE(SUM(value_cents) FILTER (
			WHERE status IN ('CONFIRMED','RECEIVED')
			AND COALESCE(received_at, confirmed_at) >= ?
			AND COALESCE(received_at, confirmed_at) < ?
		), 0) AS received
		FROM billing_invoices`, monthStart, nextMonth).Scan(&amounts).Error; err != nil {
		return nil, err
	}
	result.PendingAmountCents = amounts.Pending
	result.ReceivedThisMonthCents = amounts.Received
	return &result, nil
}

func billingPlanToDomain(row models.BillingPlanModel) domain.BillingPlan {
	return domain.BillingPlan{
		ID: domainID(row.ID), Code: row.Code, Version: row.Version, Name: row.Name, Description: row.Description,
		MonthlyPriceCents: row.MonthlyPriceCents, Currency: row.Currency,
		MonthlyMessageLimit: row.MonthlyMessageLimit, DailyMessageLimit: row.DailyMessageLimit,
		PerDispatchLimit: row.PerDispatchLimit, WarningPercent: row.WarningPercent,
		GracePeriodDays: row.GracePeriodDays, Active: row.Active, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func billingCustomerToDomain(row models.BillingCustomerModel) domain.BillingCustomer {
	return domain.BillingCustomer{
		ID: domainID(row.ID), BoxID: domainID(row.BoxID), Provider: row.Provider,
		ProviderCustomerID: row.ProviderCustomerID, LegalName: row.LegalName, CPFCNPJ: row.CPFCNPJ,
		Email: row.Email, Phone: row.Phone, PostalCode: row.PostalCode, Address: row.Address,
		AddressNumber: row.AddressNumber, Complement: row.Complement, Province: row.Province,
		City: row.City, State: row.State, NotificationDisabled: row.NotificationDisabled,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func billingCustomerToModel(item domain.BillingCustomer) models.BillingCustomerModel {
	return models.BillingCustomerModel{
		ID: stringID(item.ID), BoxID: stringID(item.BoxID), Provider: item.Provider,
		ProviderCustomerID: item.ProviderCustomerID, LegalName: item.LegalName, CPFCNPJ: item.CPFCNPJ,
		Email: item.Email, Phone: item.Phone, PostalCode: item.PostalCode, Address: item.Address,
		AddressNumber: item.AddressNumber, Complement: item.Complement, Province: item.Province,
		City: item.City, State: item.State, NotificationDisabled: item.NotificationDisabled,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func billingSubscriptionToDomain(row models.BillingSubscriptionModel) domain.BillingSubscription {
	return domain.BillingSubscription{
		ID: domainID(row.ID), BoxID: domainID(row.BoxID), BillingCustomerID: domainID(row.BillingCustomerID),
		PlanID: domainID(row.PlanID), Provider: row.Provider, ProviderSubscriptionID: row.ProviderSubscriptionID,
		Status: domain.BillingSubscriptionStatus(row.Status), BillingType: domain.BillingType(row.BillingType),
		NextDueDate: row.NextDueDate, CurrentPeriodStart: row.CurrentPeriodStart, CurrentPeriodEnd: row.CurrentPeriodEnd,
		GraceUntil: row.GraceUntil, StartedAt: row.StartedAt, CanceledAt: row.CanceledAt,
		CancelAtPeriodEnd: row.CancelAtPeriodEnd, ExternalReference: row.ExternalReference,
		LastReconciledAt: row.LastReconciledAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func billingSubscriptionToModel(item domain.BillingSubscription) models.BillingSubscriptionModel {
	return models.BillingSubscriptionModel{
		ID: stringID(item.ID), BoxID: stringID(item.BoxID), BillingCustomerID: stringID(item.BillingCustomerID),
		PlanID: stringID(item.PlanID), Provider: item.Provider, ProviderSubscriptionID: item.ProviderSubscriptionID,
		Status: string(item.Status), BillingType: string(item.BillingType), NextDueDate: item.NextDueDate,
		CurrentPeriodStart: item.CurrentPeriodStart, CurrentPeriodEnd: item.CurrentPeriodEnd,
		GraceUntil: item.GraceUntil, StartedAt: item.StartedAt, CanceledAt: item.CanceledAt,
		CancelAtPeriodEnd: item.CancelAtPeriodEnd, ExternalReference: item.ExternalReference,
		LastReconciledAt: item.LastReconciledAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func billingInvoiceToDomain(row models.BillingInvoiceModel) domain.BillingInvoice {
	return domain.BillingInvoice{
		ID: domainID(row.ID), BoxID: domainID(row.BoxID), SubscriptionID: domainID(row.SubscriptionID),
		Provider: row.Provider, ProviderPaymentID: row.ProviderPaymentID, Status: row.Status,
		BillingType: domain.BillingType(row.BillingType), ValueCents: row.ValueCents,
		NetValueCents: row.NetValueCents, DueDate: row.DueDate, OriginalDueDate: row.OriginalDueDate,
		ConfirmedAt: row.ConfirmedAt, ReceivedAt: row.ReceivedAt, InvoiceURL: row.InvoiceURL,
		BankSlipURL: row.BankSlipURL, ExternalReference: row.ExternalReference, Description: row.Description,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func billingInvoiceToModel(item domain.BillingInvoice) models.BillingInvoiceModel {
	return models.BillingInvoiceModel{
		ID: stringID(item.ID), BoxID: stringID(item.BoxID), SubscriptionID: stringID(item.SubscriptionID),
		Provider: item.Provider, ProviderPaymentID: item.ProviderPaymentID, Status: item.Status,
		BillingType: string(item.BillingType), ValueCents: item.ValueCents, NetValueCents: item.NetValueCents,
		DueDate: item.DueDate, OriginalDueDate: item.OriginalDueDate, ConfirmedAt: item.ConfirmedAt,
		ReceivedAt: item.ReceivedAt, InvoiceURL: item.InvoiceURL, BankSlipURL: item.BankSlipURL,
		ExternalReference: item.ExternalReference, Description: item.Description,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
