package handlers

import (
	"net/http"
	"strings"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/platformadmin"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type MessagingGovernanceHandler struct {
	admin  platformadmin.MessagingAdminUseCases
	tenant platformadmin.GetTenantMessagingUsageUseCase
}

func NewMessagingGovernanceHandler(admin platformadmin.MessagingAdminUseCases, tenant platformadmin.GetTenantMessagingUsageUseCase) MessagingGovernanceHandler {
	return MessagingGovernanceHandler{admin: admin, tenant: tenant}
}

func (h MessagingGovernanceHandler) ListBoxes(c *gin.Context) {
	items, err := h.admin.ListBoxes(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.MessagingBoxOverviewResponse, 0, len(items))
	for _, item := range items {
		response = append(response, dto.MessagingBoxOverviewResponse{BoxID: string(item.Box.ID), BoxName: item.Box.Name, ConnectionMode: string(item.ConnectionMode), Policy: policyResponse(item.Policy), Usage: usageResponse(item.Usage)})
	}
	c.JSON(http.StatusOK, response)
}

func (h MessagingGovernanceHandler) GetBoxPolicy(c *gin.Context) {
	policy, usage, err := h.admin.BoxPolicy(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessagingPolicyWithUsageResponse{Policy: policyResponse(*policy), Usage: usageResponse(*usage)})
}

func (h MessagingGovernanceHandler) UpdateBoxPolicy(c *gin.Context) {
	h.updatePolicy(c, domain.MessagingPolicyScopeBox, domain.ID(c.Param("id")))
}

func (h MessagingGovernanceHandler) GetPlatformPolicy(c *gin.Context) {
	policy, usage, err := h.admin.PlatformPolicy(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessagingPolicyWithUsageResponse{Policy: policyResponse(*policy), Usage: usageResponse(*usage)})
}

func (h MessagingGovernanceHandler) UpdatePlatformPolicy(c *gin.Context) {
	h.updatePolicy(c, domain.MessagingPolicyScopePlatform, "")
}

func (h MessagingGovernanceHandler) TenantUsage(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	policy, usage, err := h.tenant.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessagingPolicyWithUsageResponse{Policy: policyResponse(*policy), Usage: usageResponse(*usage)})
}

func (h MessagingGovernanceHandler) updatePolicy(c *gin.Context, scope domain.MessagingPolicyScope, boxID domain.ID) {
	var request dto.UpdateMessagingPolicyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	adminID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	policy := domain.MessagingPolicy{Scope: scope, BoxID: boxID, DailyMessageLimit: request.DailyMessageLimit,
		MonthlyMessageLimit: request.MonthlyMessageLimit, PerDispatchLimit: request.PerDispatchLimit,
		EstimatedCostMicrosPerMessage: request.EstimatedCostMicrosPerMessage, DailyCostLimitMicros: request.DailyCostLimitMicros,
		MonthlyCostLimitMicros: request.MonthlyCostLimitMicros, Currency: strings.ToUpper(strings.TrimSpace(request.Currency)),
		WarningPercent: request.WarningPercent, Timezone: strings.TrimSpace(request.Timezone), Blocked: request.Blocked}
	if err := h.admin.UpdatePolicy(c.Request.Context(), platformadmin.UpdatePolicyInput{Policy: policy, AdminUserID: adminID, Reason: strings.TrimSpace(request.Reason), IPAddress: c.ClientIP()}); err != nil {
		respondPublicError(c, http.StatusBadRequest, "messaging_policy_invalid", err.Error())
		return
	}
	var updated *domain.MessagingPolicy
	var usage *domain.MessagingUsage
	if scope == domain.MessagingPolicyScopePlatform {
		updated, usage, err = h.admin.PlatformPolicy(c.Request.Context())
	} else {
		updated, usage, err = h.admin.BoxPolicy(c.Request.Context(), boxID)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessagingPolicyWithUsageResponse{Policy: policyResponse(*updated), Usage: usageResponse(*usage)})
}

func policyResponse(policy domain.MessagingPolicy) dto.MessagingPolicyResponse {
	response := dto.MessagingPolicyResponse{ID: string(policy.ID), Scope: string(policy.Scope), BoxID: string(policy.BoxID), DailyMessageLimit: policy.DailyMessageLimit,
		MonthlyMessageLimit: policy.MonthlyMessageLimit, PerDispatchLimit: policy.PerDispatchLimit,
		EstimatedCostMicrosPerMessage: policy.EstimatedCostMicrosPerMessage, DailyCostLimitMicros: policy.DailyCostLimitMicros,
		MonthlyCostLimitMicros: policy.MonthlyCostLimitMicros, Currency: policy.Currency, WarningPercent: policy.WarningPercent,
		Timezone: policy.Timezone, Blocked: policy.Blocked}
	if !policy.UpdatedAt.IsZero() {
		response.UpdatedAt = policy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}

func usageResponse(usage domain.MessagingUsage) dto.MessagingUsageResponse {
	return dto.MessagingUsageResponse{DailyAccepted: usage.DailyAccepted, DailyReserved: usage.DailyReserved,
		MonthlyAccepted: usage.MonthlyAccepted, MonthlyReserved: usage.MonthlyReserved,
		DailyEstimatedCostMicros: usage.DailyEstimatedCostMicros, DailyReservedCostMicros: usage.DailyReservedCostMicros,
		MonthlyEstimatedCostMicros: usage.MonthlyEstimatedCostMicros, MonthlyReservedCostMicros: usage.MonthlyReservedCostMicros}
}
