package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"gorm.io/gorm"
)

const (
	defaultTimezone = "America/Sao_Paulo"
	defaultCurrency = "USD"
)

type policyRow struct {
	ID                            string
	Scope                         string
	BoxID                         *string
	DailyMessageLimit             int
	MonthlyMessageLimit           int
	PerDispatchLimit              int
	EstimatedCostMicrosPerMessage int64
	DailyCostLimitMicros          int64
	MonthlyCostLimitMicros        int64
	Currency                      string
	WarningPercent                int
	Timezone                      string
	Blocked                       bool
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

type usageRow struct {
	ReservedCount       int
	AcceptedCount       int
	FailedCount         int
	ReservedCostMicros  int64
	EstimatedCostMicros int64
}

type dispatchRow struct {
	ID                  string
	BoxID               string
	ConnectionMode      string
	ReservedMessages    int
	EstimatedCostMicros int64
	Status              string
	CreatedAt           time.Time
}

func (r MessagingGovernanceGormRepository) Reserve(ctx context.Context, request portrepo.MessagingReservation) (*domain.MessageDispatch, error) {
	if request.Recipients <= 0 {
		return nil, errors.New("o disparo não possui destinatários")
	}
	dispatchID, err := newID()
	if err != nil {
		return nil, err
	}

	var result domain.MessageDispatch
	var blockReason string
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		boxPolicy, err := ensurePolicy(tx, domain.MessagingPolicyScopeBox, request.BoxID)
		if err != nil {
			return err
		}
		policies := []policyRow{boxPolicy}
		if request.ConnectionMode == domain.WhatsappConnectionPlatform {
			platformPolicy, err := ensurePolicy(tx, domain.MessagingPolicyScopePlatform, "")
			if err != nil {
				return err
			}
			policies = append(policies, platformPolicy)
		}

		estimatedCost := int64(request.Recipients) * boxPolicy.EstimatedCostMicrosPerMessage
		for _, policy := range policies {
			daily, monthly, err := lockCurrentBuckets(tx, policy, time.Now())
			if err != nil {
				return err
			}
			policyRequestCost := int64(request.Recipients) * policy.EstimatedCostMicrosPerMessage
			if reason := validateReservation(policy, daily, monthly, request.Recipients, policyRequestCost); reason != "" {
				blockReason = reason
				break
			}
		}

		status := "reserved"
		reserved := request.Recipients
		if blockReason != "" {
			status = "blocked"
			reserved = 0
		}
		var userID any
		if request.RequestedByUserID != "" {
			userID = string(request.RequestedByUserID)
		}
		var sourceID any
		if request.SourceID != "" {
			sourceID = string(request.SourceID)
		}
		if err := tx.Exec(`INSERT INTO message_dispatches
			(id, box_id, requested_by_user_id, source_type, source_id, connection_mode,
			recipients_total, reserved_messages, estimated_cost_micros, currency, status, block_reason)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			string(dispatchID), string(request.BoxID), userID, request.SourceType, sourceID,
			string(request.ConnectionMode), request.Recipients, reserved, estimatedCost,
			boxPolicy.Currency, status, blockReason).Error; err != nil {
			return err
		}

		if blockReason == "" {
			for _, policy := range policies {
				cost := int64(request.Recipients) * policy.EstimatedCostMicrosPerMessage
				if err := incrementReservation(tx, policy, time.Now(), request.Recipients, cost); err != nil {
					return err
				}
			}
		}
		result = domain.MessageDispatch{ID: dispatchID, BoxID: request.BoxID, RequestedByUserID: request.RequestedByUserID, SourceType: request.SourceType, SourceID: request.SourceID, ConnectionMode: request.ConnectionMode, RecipientsTotal: request.Recipients, ReservedMessages: reserved, EstimatedCostMicros: estimatedCost, Currency: boxPolicy.Currency, Status: status, BlockReason: blockReason, CreatedAt: time.Now()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if blockReason != "" {
		return &result, domain.MessagingLimitError{Reason: blockReason}
	}
	return &result, nil
}

func (r MessagingGovernanceGormRepository) Complete(ctx context.Context, dispatchID domain.ID, accepted, failed int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dispatch dispatchRow
		result := tx.Raw("SELECT id, box_id, connection_mode, reserved_messages, estimated_cost_micros, status, created_at FROM message_dispatches WHERE id = ? FOR UPDATE", string(dispatchID)).Scan(&dispatch)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if dispatch.Status != "reserved" {
			return nil
		}
		if accepted < 0 || failed < 0 || accepted+failed > dispatch.ReservedMessages {
			return errors.New("resultado de disparo inválido")
		}
		boxPolicy, err := ensurePolicy(tx, domain.MessagingPolicyScopeBox, domain.ID(dispatch.BoxID))
		if err != nil {
			return err
		}
		policies := []policyRow{boxPolicy}
		if dispatch.ConnectionMode == string(domain.WhatsappConnectionPlatform) {
			platformPolicy, err := ensurePolicy(tx, domain.MessagingPolicyScopePlatform, "")
			if err != nil {
				return err
			}
			policies = append(policies, platformPolicy)
		}
		for _, policy := range policies {
			reservedCost := int64(dispatch.ReservedMessages) * policy.EstimatedCostMicrosPerMessage
			acceptedCost := int64(accepted) * policy.EstimatedCostMicrosPerMessage
			if err := completeBuckets(tx, policy, dispatch.CreatedAt, dispatch.ReservedMessages, accepted, failed, reservedCost, acceptedCost); err != nil {
				return err
			}
		}
		status := "completed"
		if accepted == 0 && failed > 0 {
			status = "failed"
		}
		return tx.Exec(`UPDATE message_dispatches SET reserved_messages = 0, accepted_messages = ?, failed_messages = ?,
			status = ?, completed_at = NOW() WHERE id = ?`, accepted, failed, status, dispatch.ID).Error
	})
}

func (r MessagingGovernanceGormRepository) GetBoxPolicy(ctx context.Context, boxID domain.ID) (*domain.MessagingPolicy, error) {
	var row policyRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = ensurePolicy(tx, domain.MessagingPolicyScopeBox, boxID)
		return err
	})
	if err != nil {
		return nil, err
	}
	policy := policyToDomain(row)
	return &policy, nil
}

func (r MessagingGovernanceGormRepository) GetPlatformPolicy(ctx context.Context) (*domain.MessagingPolicy, error) {
	var row policyRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = ensurePolicy(tx, domain.MessagingPolicyScopePlatform, "")
		return err
	})
	if err != nil {
		return nil, err
	}
	policy := policyToDomain(row)
	return &policy, nil
}

func (r MessagingGovernanceGormRepository) UpsertPolicy(ctx context.Context, policy *domain.MessagingPolicy) error {
	if err := validatePolicy(*policy); err != nil {
		return err
	}
	if policy.ID == "" {
		generated, err := newID()
		if err != nil {
			return err
		}
		policy.ID = generated
	}
	now := time.Now()
	policy.UpdatedAt = now
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	var boxID any
	if policy.BoxID != "" {
		boxID = string(policy.BoxID)
	}
	if policy.Scope == domain.MessagingPolicyScopeBox {
		return r.db.WithContext(ctx).Exec(`INSERT INTO messaging_policies
			(id, scope, box_id, daily_message_limit, monthly_message_limit, per_dispatch_limit,
			estimated_cost_micros_per_message, daily_cost_limit_micros, monthly_cost_limit_micros,
			currency, warning_percent, timezone, blocked, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (box_id) WHERE scope = 'box' DO UPDATE SET
			daily_message_limit = EXCLUDED.daily_message_limit, monthly_message_limit = EXCLUDED.monthly_message_limit,
			per_dispatch_limit = EXCLUDED.per_dispatch_limit, estimated_cost_micros_per_message = EXCLUDED.estimated_cost_micros_per_message,
			daily_cost_limit_micros = EXCLUDED.daily_cost_limit_micros, monthly_cost_limit_micros = EXCLUDED.monthly_cost_limit_micros,
			currency = EXCLUDED.currency, warning_percent = EXCLUDED.warning_percent, timezone = EXCLUDED.timezone,
			blocked = EXCLUDED.blocked, updated_at = EXCLUDED.updated_at`, string(policy.ID), string(policy.Scope), boxID,
			policy.DailyMessageLimit, policy.MonthlyMessageLimit, policy.PerDispatchLimit,
			policy.EstimatedCostMicrosPerMessage, policy.DailyCostLimitMicros, policy.MonthlyCostLimitMicros,
			policy.Currency, policy.WarningPercent, policy.Timezone, policy.Blocked, policy.CreatedAt, policy.UpdatedAt).Error
	}
	return r.db.WithContext(ctx).Exec(`INSERT INTO messaging_policies
		(id, scope, box_id, daily_message_limit, monthly_message_limit, per_dispatch_limit,
		estimated_cost_micros_per_message, daily_cost_limit_micros, monthly_cost_limit_micros,
		currency, warning_percent, timezone, blocked, created_at, updated_at)
		VALUES (?, 'platform', NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (scope) WHERE scope = 'platform' DO UPDATE SET
		daily_message_limit = EXCLUDED.daily_message_limit, monthly_message_limit = EXCLUDED.monthly_message_limit,
		per_dispatch_limit = EXCLUDED.per_dispatch_limit, estimated_cost_micros_per_message = EXCLUDED.estimated_cost_micros_per_message,
		daily_cost_limit_micros = EXCLUDED.daily_cost_limit_micros, monthly_cost_limit_micros = EXCLUDED.monthly_cost_limit_micros,
		currency = EXCLUDED.currency, warning_percent = EXCLUDED.warning_percent, timezone = EXCLUDED.timezone,
		blocked = EXCLUDED.blocked, updated_at = EXCLUDED.updated_at`, string(policy.ID), policy.DailyMessageLimit,
		policy.MonthlyMessageLimit, policy.PerDispatchLimit, policy.EstimatedCostMicrosPerMessage,
		policy.DailyCostLimitMicros, policy.MonthlyCostLimitMicros, policy.Currency, policy.WarningPercent,
		policy.Timezone, policy.Blocked, policy.CreatedAt, policy.UpdatedAt).Error
}

func (r MessagingGovernanceGormRepository) GetBoxUsage(ctx context.Context, boxID domain.ID, atTime time.Time) (*domain.MessagingUsage, error) {
	policy, err := r.GetBoxPolicy(ctx, boxID)
	if err != nil {
		return nil, err
	}
	return r.getUsage(ctx, *policy, atTime)
}

func (r MessagingGovernanceGormRepository) GetPlatformUsage(ctx context.Context, atTime time.Time) (*domain.MessagingUsage, error) {
	policy, err := r.GetPlatformPolicy(ctx)
	if err != nil {
		return nil, err
	}
	return r.getUsage(ctx, *policy, atTime)
}

func (r MessagingGovernanceGormRepository) getUsage(ctx context.Context, policy domain.MessagingPolicy, atTime time.Time) (*domain.MessagingUsage, error) {
	dailyStart, monthlyStart := periodStarts(atTime, policy.Timezone)
	daily, err := readUsage(r.db.WithContext(ctx), policy.Scope, policy.BoxID, "daily", dailyStart)
	if err != nil {
		return nil, err
	}
	monthly, err := readUsage(r.db.WithContext(ctx), policy.Scope, policy.BoxID, "monthly", monthlyStart)
	if err != nil {
		return nil, err
	}
	return &domain.MessagingUsage{DailyAccepted: daily.AcceptedCount, DailyReserved: daily.ReservedCount,
		MonthlyAccepted: monthly.AcceptedCount, MonthlyReserved: monthly.ReservedCount,
		DailyEstimatedCostMicros: daily.EstimatedCostMicros, DailyReservedCostMicros: daily.ReservedCostMicros,
		MonthlyEstimatedCostMicros: monthly.EstimatedCostMicros, MonthlyReservedCostMicros: monthly.ReservedCostMicros}, nil
}

func (r MessagingGovernanceGormRepository) SaveAuditLog(ctx context.Context, log *domain.AdminAuditLog) error {
	if err := ensureID(&log.ID); err != nil {
		return err
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	jsonValue := func(data []byte) any {
		if len(data) == 0 || !json.Valid(data) {
			return nil
		}
		return string(data)
	}
	return r.db.WithContext(ctx).Exec(`INSERT INTO admin_audit_logs
		(id, admin_user_id, action, target_type, target_id, before_data, after_data, reason, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, CAST(? AS jsonb), CAST(? AS jsonb), ?, ?, ?)`, string(log.ID), string(log.AdminUserID),
		log.Action, log.TargetType, log.TargetID, jsonValue(log.BeforeData), jsonValue(log.AfterData), log.Reason, log.IPAddress, log.CreatedAt).Error
}

func ensurePolicy(tx *gorm.DB, scope domain.MessagingPolicyScope, boxID domain.ID) (policyRow, error) {
	where := "scope = 'platform'"
	args := []any{}
	if scope == domain.MessagingPolicyScopeBox {
		where = "scope = 'box' AND box_id = ?"
		args = append(args, string(boxID))
	}
	var row policyRow
	result := tx.Raw("SELECT * FROM messaging_policies WHERE "+where+" FOR UPDATE", args...).Scan(&row)
	if result.Error != nil {
		return row, result.Error
	}
	if result.RowsAffected > 0 {
		return row, nil
	}
	id, err := newID()
	if err != nil {
		return row, err
	}
	if scope == domain.MessagingPolicyScopeBox {
		err = tx.Exec(`INSERT INTO messaging_policies (id, scope, box_id, daily_message_limit, monthly_message_limit,
			per_dispatch_limit, estimated_cost_micros_per_message, daily_cost_limit_micros, monthly_cost_limit_micros)
			VALUES (?, 'box', ?, 100, 1000, 100, 100000, 10000000, 100000000) ON CONFLICT DO NOTHING`, string(id), string(boxID)).Error
	} else {
		err = tx.Exec(`INSERT INTO messaging_policies (id, scope, daily_message_limit, monthly_message_limit,
			per_dispatch_limit, estimated_cost_micros_per_message, daily_cost_limit_micros, monthly_cost_limit_micros)
			VALUES (?, 'platform', 1000, 10000, 250, 100000, 100000000, 1000000000) ON CONFLICT DO NOTHING`, string(id)).Error
	}
	if err != nil {
		return row, err
	}
	result = tx.Raw("SELECT * FROM messaging_policies WHERE "+where+" FOR UPDATE", args...).Scan(&row)
	return row, result.Error
}

func lockCurrentBuckets(tx *gorm.DB, policy policyRow, atTime time.Time) (usageRow, usageRow, error) {
	dailyStart, monthlyStart := periodStarts(atTime, policy.Timezone)
	daily, err := lockBucket(tx, policy, "daily", dailyStart)
	if err != nil {
		return daily, usageRow{}, err
	}
	monthly, err := lockBucket(tx, policy, "monthly", monthlyStart)
	return daily, monthly, err
}

func lockBucket(tx *gorm.DB, policy policyRow, periodType string, periodStart time.Time) (usageRow, error) {
	id, err := newID()
	if err != nil {
		return usageRow{}, err
	}
	var boxID any
	if policy.BoxID != nil {
		boxID = *policy.BoxID
	}
	if err := tx.Exec(`INSERT INTO messaging_usage_buckets (id, scope, box_id, period_type, period_start)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`, string(id), policy.Scope, boxID, periodType, periodStart).Error; err != nil {
		return usageRow{}, err
	}
	where, args := bucketWhere(policy, periodType, periodStart)
	var row usageRow
	result := tx.Raw("SELECT reserved_count, accepted_count, failed_count, reserved_cost_micros, estimated_cost_micros FROM messaging_usage_buckets WHERE "+where+" FOR UPDATE", args...).Scan(&row)
	return row, result.Error
}

func validateReservation(policy policyRow, daily, monthly usageRow, recipients int, estimatedCost int64) string {
	scopeLabel := "da academia"
	if policy.Scope == string(domain.MessagingPolicyScopePlatform) {
		scopeLabel = "global da conexão compartilhada"
	}
	if policy.Blocked {
		return "envios bloqueados pela política " + scopeLabel
	}
	if recipients > policy.PerDispatchLimit {
		return fmt.Sprintf("o disparo possui %d destinatários e excede o limite de %d por disparo %s", recipients, policy.PerDispatchLimit, scopeLabel)
	}
	if daily.AcceptedCount+daily.ReservedCount+recipients > policy.DailyMessageLimit {
		return fmt.Sprintf("o disparo excede o limite diário de %d mensagens %s", policy.DailyMessageLimit, scopeLabel)
	}
	if monthly.AcceptedCount+monthly.ReservedCount+recipients > policy.MonthlyMessageLimit {
		return fmt.Sprintf("o disparo excede o limite mensal de %d mensagens %s", policy.MonthlyMessageLimit, scopeLabel)
	}
	if daily.EstimatedCostMicros+daily.ReservedCostMicros+estimatedCost > policy.DailyCostLimitMicros {
		return "o disparo excede o orçamento diário de segurança " + scopeLabel
	}
	if monthly.EstimatedCostMicros+monthly.ReservedCostMicros+estimatedCost > policy.MonthlyCostLimitMicros {
		return "o disparo excede o orçamento mensal de segurança " + scopeLabel
	}
	return ""
}

func incrementReservation(tx *gorm.DB, policy policyRow, atTime time.Time, count int, cost int64) error {
	dailyStart, monthlyStart := periodStarts(atTime, policy.Timezone)
	for _, period := range []struct {
		typeName string
		start    time.Time
	}{{"daily", dailyStart}, {"monthly", monthlyStart}} {
		where, args := bucketWhere(policy, period.typeName, period.start)
		args = append([]any{count, cost}, args...)
		if err := tx.Exec("UPDATE messaging_usage_buckets SET reserved_count = reserved_count + ?, reserved_cost_micros = reserved_cost_micros + ?, updated_at = NOW() WHERE "+where, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func completeBuckets(tx *gorm.DB, policy policyRow, atTime time.Time, reserved, accepted, failed int, reservedCost, acceptedCost int64) error {
	dailyStart, monthlyStart := periodStarts(atTime, policy.Timezone)
	for _, period := range []struct {
		typeName string
		start    time.Time
	}{{"daily", dailyStart}, {"monthly", monthlyStart}} {
		where, args := bucketWhere(policy, period.typeName, period.start)
		args = append([]any{reserved, accepted, failed, reservedCost, acceptedCost}, args...)
		if err := tx.Exec(`UPDATE messaging_usage_buckets SET
			reserved_count = GREATEST(0, reserved_count - ?), accepted_count = accepted_count + ?,
			failed_count = failed_count + ?, reserved_cost_micros = GREATEST(0, reserved_cost_micros - ?),
			estimated_cost_micros = estimated_cost_micros + ?, updated_at = NOW() WHERE `+where, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func readUsage(db *gorm.DB, scope domain.MessagingPolicyScope, boxID domain.ID, periodType string, periodStart time.Time) (usageRow, error) {
	policy := policyRow{Scope: string(scope)}
	if boxID != "" {
		value := string(boxID)
		policy.BoxID = &value
	}
	where, args := bucketWhere(policy, periodType, periodStart)
	var row usageRow
	result := db.Raw("SELECT reserved_count, accepted_count, failed_count, reserved_cost_micros, estimated_cost_micros FROM messaging_usage_buckets WHERE "+where, args...).Scan(&row)
	return row, result.Error
}

func bucketWhere(policy policyRow, periodType string, periodStart time.Time) (string, []any) {
	if policy.Scope == string(domain.MessagingPolicyScopeBox) && policy.BoxID != nil {
		return "scope = 'box' AND box_id = ? AND period_type = ? AND period_start = ?", []any{*policy.BoxID, periodType, periodStart}
	}
	return "scope = 'platform' AND period_type = ? AND period_start = ?", []any{periodType, periodStart}
}

func periodStarts(atTime time.Time, timezone string) (time.Time, time.Time) {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		location, _ = time.LoadLocation(defaultTimezone)
	}
	local := atTime.In(location)
	daily := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	monthly := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	return daily, monthly
}

func policyToDomain(row policyRow) domain.MessagingPolicy {
	policy := domain.MessagingPolicy{ID: domain.ID(row.ID), Scope: domain.MessagingPolicyScope(row.Scope), DailyMessageLimit: row.DailyMessageLimit,
		MonthlyMessageLimit: row.MonthlyMessageLimit, PerDispatchLimit: row.PerDispatchLimit,
		EstimatedCostMicrosPerMessage: row.EstimatedCostMicrosPerMessage, DailyCostLimitMicros: row.DailyCostLimitMicros,
		MonthlyCostLimitMicros: row.MonthlyCostLimitMicros, Currency: row.Currency, WarningPercent: row.WarningPercent,
		Timezone: row.Timezone, Blocked: row.Blocked, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.BoxID != nil {
		policy.BoxID = domain.ID(*row.BoxID)
	}
	return policy
}

func validatePolicy(policy domain.MessagingPolicy) error {
	if policy.Scope != domain.MessagingPolicyScopeBox && policy.Scope != domain.MessagingPolicyScopePlatform {
		return errors.New("escopo de política inválido")
	}
	if policy.Scope == domain.MessagingPolicyScopeBox && policy.BoxID == "" {
		return errors.New("academia é obrigatória")
	}
	if policy.DailyMessageLimit < 0 || policy.MonthlyMessageLimit < 0 || policy.PerDispatchLimit < 0 ||
		policy.EstimatedCostMicrosPerMessage < 0 || policy.DailyCostLimitMicros < 0 || policy.MonthlyCostLimitMicros < 0 {
		return errors.New("limites não podem ser negativos")
	}
	if policy.DailyMessageLimit > policy.MonthlyMessageLimit && policy.MonthlyMessageLimit > 0 {
		return errors.New("limite diário não pode ser maior que o mensal")
	}
	if policy.PerDispatchLimit > policy.DailyMessageLimit && policy.DailyMessageLimit > 0 {
		return errors.New("limite por disparo não pode ser maior que o diário")
	}
	if policy.WarningPercent < 1 || policy.WarningPercent > 100 {
		return errors.New("percentual de alerta deve ficar entre 1 e 100")
	}
	if strings.TrimSpace(policy.Currency) == "" {
		policy.Currency = defaultCurrency
	}
	if _, err := time.LoadLocation(policy.Timezone); err != nil {
		return errors.New("timezone inválido")
	}
	return nil
}
