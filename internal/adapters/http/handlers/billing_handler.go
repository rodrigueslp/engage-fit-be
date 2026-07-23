package handlers

import (
	"errors"
	"io"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	billingapp "boxengage/backend/internal/app/billing"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BillingHandler struct {
	service *billingapp.Service
}

func NewBillingHandler(service *billingapp.Service) BillingHandler {
	return BillingHandler{service: service}
}

func (h BillingHandler) Webhook(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondBadRequest(c)
		return
	}
	err = h.service.ProcessWebhook(c.Request.Context(), c.GetHeader("asaas-access-token"), raw)
	switch {
	case err == nil:
		c.Status(http.StatusOK)
	case errors.Is(err, billingapp.ErrBillingWebhookUnauthorized):
		respondPublicError(c, http.StatusUnauthorized, "billing_webhook_unauthorized", "webhook não autorizado")
	case errors.Is(err, billingapp.ErrBillingWebhookInvalid):
		respondPublicError(c, http.StatusBadRequest, "billing_webhook_invalid", "evento inválido")
	case errors.Is(err, billingapp.ErrBillingDisabled):
		respondPublicError(c, http.StatusNotFound, "capability_disabled", "capability disabled")
	default:
		respondError(c, err)
	}
}

func (h BillingHandler) ListPlans(c *gin.Context) {
	items, err := h.service.ListPlans(c.Request.Context(), true)
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.BillingPlanResponse, 0, len(items))
	for _, item := range items {
		response = append(response, billingPlanResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h BillingHandler) CreatePlan(c *gin.Context) { h.savePlan(c, "") }
func (h BillingHandler) UpdatePlan(c *gin.Context) { h.savePlan(c, domain.ID(c.Param("id"))) }

func (h BillingHandler) savePlan(c *gin.Context, id domain.ID) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.BillingPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.service.SavePlan(c.Request.Context(), billingapp.SavePlanInput{
		ID: id, Code: request.Code, Version: request.Version, Name: request.Name, Description: request.Description,
		MonthlyPriceCents: request.MonthlyPriceCents, MonthlyMessageLimit: request.MonthlyMessageLimit,
		DailyMessageLimit: request.DailyMessageLimit, PerDispatchLimit: request.PerDispatchLimit,
		WarningPercent: request.WarningPercent, GracePeriodDays: request.GracePeriodDays, Active: request.Active,
		AdminUserID: adminID, Reason: request.Reason, IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondBillingError(c, err)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	c.JSON(status, billingPlanResponse(*item))
}

func (h BillingHandler) Summary(c *gin.Context) {
	item, err := h.service.Summary(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.BillingSummaryResponse{
		MonthlyRecurringRevenueCents: item.MonthlyRecurringRevenueCents,
		ActiveSubscriptions:          item.ActiveSubscriptions, PastDueSubscriptions: item.PastDueSubscriptions,
		SuspendedSubscriptions: item.SuspendedSubscriptions, CanceledSubscriptions: item.CanceledSubscriptions,
		PendingAmountCents: item.PendingAmountCents, ReceivedThisMonthCents: item.ReceivedThisMonthCents,
	})
}

func (h BillingHandler) ListBoxes(c *gin.Context) {
	items, err := h.service.ListOverviews(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response := make([]dto.BillingOverviewResponse, 0, len(items))
	for _, item := range items {
		response = append(response, billingOverviewResponse(item, nil))
	}
	c.JSON(http.StatusOK, response)
}

func (h BillingHandler) GetAdminBox(c *gin.Context) {
	item, invoices, err := h.service.BoxBilling(c.Request.Context(), domain.ID(c.Param("id")))
	if err != nil {
		respondBillingError(c, err)
		return
	}
	c.JSON(http.StatusOK, billingOverviewResponse(*item, invoices))
}

func (h BillingHandler) GetOwnerBilling(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	item, invoices, err := h.service.BoxBilling(c.Request.Context(), boxID)
	if err != nil {
		respondBillingError(c, err)
		return
	}
	response := billingOverviewResponse(*item, invoices)
	if response.Customer != nil {
		response.Customer.ProviderCustomerID = ""
	}
	if response.Subscription != nil {
		response.Subscription.ProviderSubscriptionID = ""
	}
	for index := range response.Invoices {
		response.Invoices[index].ProviderPaymentID = ""
	}
	c.JSON(http.StatusOK, response)
}

func (h BillingHandler) SaveCustomer(c *gin.Context) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.BillingCustomerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	item, err := h.service.SaveCustomer(c.Request.Context(), billingapp.SaveCustomerInput{
		BoxID: domain.ID(c.Param("id")), LegalName: request.LegalName, CPFCNPJ: request.CPFCNPJ,
		Email: request.Email, Phone: request.Phone, PostalCode: request.PostalCode, Address: request.Address,
		AddressNumber: request.AddressNumber, Complement: request.Complement, Province: request.Province,
		City: request.City, State: request.State, NotificationDisabled: request.NotificationDisabled,
		AdminUserID: adminID, Reason: request.Reason, IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondBillingError(c, err)
		return
	}
	response := billingCustomerResponse(*item)
	c.JSON(http.StatusOK, response)
}

func (h BillingHandler) CreateSubscription(c *gin.Context) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.CreateBillingSubscriptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	nextDueDate, err := time.Parse("2006-01-02", request.NextDueDate)
	if err != nil {
		respondBadRequest(c)
		return
	}
	var endDate *time.Time
	if request.EndDate != "" {
		parsed, parseErr := time.Parse("2006-01-02", request.EndDate)
		if parseErr != nil {
			respondBadRequest(c)
			return
		}
		endDate = &parsed
	}
	item, err := h.service.CreateSubscription(c.Request.Context(), billingapp.CreateSubscriptionInput{
		BoxID: domain.ID(c.Param("id")), PlanID: domain.ID(request.PlanID),
		BillingType: domain.BillingType(request.BillingType), NextDueDate: nextDueDate, EndDate: endDate,
		AdminUserID: adminID, Reason: request.Reason, IPAddress: c.ClientIP(),
		RequestKey: c.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		respondBillingError(c, err)
		return
	}
	c.JSON(http.StatusCreated, billingSubscriptionResponse(*item))
}

func (h BillingHandler) CancelSubscription(c *gin.Context) {
	h.subscriptionAction(c, func(boxID, adminID domain.ID, reason, ip string) error {
		return h.service.CancelSubscription(c.Request.Context(), boxID, adminID, reason, ip)
	})
}

func (h BillingHandler) subscriptionAction(c *gin.Context, action func(domain.ID, domain.ID, string, string) error) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.BillingActionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	if err := action(domain.ID(c.Param("id")), adminID, request.Reason, c.ClientIP()); err != nil {
		respondBillingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h BillingHandler) GrantGrace(c *gin.Context) {
	adminID, ok := adminUserID(c)
	if !ok {
		return
	}
	var request dto.GrantBillingGraceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c)
		return
	}
	until, err := time.Parse("2006-01-02", request.Until)
	if err != nil {
		respondBadRequest(c)
		return
	}
	if err := h.service.GrantGrace(c.Request.Context(), domain.ID(c.Param("id")), adminID, until, request.Reason, c.ClientIP()); err != nil {
		respondBillingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h BillingHandler) Reconcile(c *gin.Context) {
	if err := h.service.Reconcile(c.Request.Context()); err != nil {
		respondBillingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondBillingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, billingapp.ErrBillingDisabled):
		respondPublicError(c, http.StatusNotFound, "capability_disabled", "capability disabled")
	case errors.Is(err, billingapp.ErrBillingPlanInvalid), errors.Is(err, billingapp.ErrBillingCustomerInvalid),
		errors.Is(err, billingapp.ErrBillingTypeInvalid), errors.Is(err, billingapp.ErrBillingIdempotencyInvalid):
		respondPublicError(c, http.StatusBadRequest, "billing_invalid", err.Error())
	case errors.Is(err, billingapp.ErrBillingSubscriptionExists):
		respondPublicError(c, http.StatusConflict, "billing_subscription_exists", err.Error())
	case errors.Is(err, billingapp.ErrBillingPlanInUse):
		respondPublicError(c, http.StatusConflict, "billing_plan_in_use", err.Error())
	case errors.Is(err, billingapp.ErrBillingSubscriptionAbsent), errors.Is(err, gorm.ErrRecordNotFound):
		respondPublicError(c, http.StatusNotFound, "billing_not_found", "dados financeiros não encontrados")
	default:
		respondError(c, err)
	}
}

func billingPlanResponse(item domain.BillingPlan) dto.BillingPlanResponse {
	return dto.BillingPlanResponse{
		ID: string(item.ID), Code: item.Code, Version: item.Version, Name: item.Name, Description: item.Description,
		MonthlyPriceCents: item.MonthlyPriceCents, Currency: item.Currency,
		MonthlyMessageLimit: item.MonthlyMessageLimit, DailyMessageLimit: item.DailyMessageLimit,
		PerDispatchLimit: item.PerDispatchLimit, WarningPercent: item.WarningPercent,
		GracePeriodDays: item.GracePeriodDays, Active: item.Active,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func billingCustomerResponse(item domain.BillingCustomer) dto.BillingCustomerResponse {
	return dto.BillingCustomerResponse{
		ID: string(item.ID), BoxID: string(item.BoxID), Provider: item.Provider,
		ProviderCustomerID: item.ProviderCustomerID, LegalName: item.LegalName, CPFCNPJ: item.CPFCNPJ,
		Email: item.Email, Phone: item.Phone, PostalCode: item.PostalCode, Address: item.Address,
		AddressNumber: item.AddressNumber, Complement: item.Complement, Province: item.Province,
		City: item.City, State: item.State, NotificationDisabled: item.NotificationDisabled,
	}
}

func billingSubscriptionResponse(item domain.BillingSubscription) dto.BillingSubscriptionResponse {
	response := dto.BillingSubscriptionResponse{
		ID: string(item.ID), BoxID: string(item.BoxID), PlanID: string(item.PlanID), Provider: item.Provider,
		ProviderSubscriptionID: item.ProviderSubscriptionID, Status: string(item.Status),
		BillingType: string(item.BillingType), NextDueDate: item.NextDueDate.Format("2006-01-02"),
		StartedAt: item.StartedAt.UTC().Format(time.RFC3339), CancelAtPeriodEnd: item.CancelAtPeriodEnd,
	}
	if item.CurrentPeriodStart != nil {
		response.CurrentPeriodStart = item.CurrentPeriodStart.Format("2006-01-02")
	}
	if item.CurrentPeriodEnd != nil {
		response.CurrentPeriodEnd = item.CurrentPeriodEnd.Format("2006-01-02")
	}
	if item.GraceUntil != nil {
		response.GraceUntil = item.GraceUntil.Format("2006-01-02")
	}
	if item.CanceledAt != nil {
		response.CanceledAt = item.CanceledAt.UTC().Format(time.RFC3339)
	}
	if item.LastReconciledAt != nil {
		response.LastReconciledAt = item.LastReconciledAt.UTC().Format(time.RFC3339)
	}
	return response
}

func billingInvoiceResponse(item domain.BillingInvoice) dto.BillingInvoiceResponse {
	response := dto.BillingInvoiceResponse{
		ID: string(item.ID), Status: item.Status, BillingType: string(item.BillingType),
		ValueCents: item.ValueCents, NetValueCents: item.NetValueCents, DueDate: item.DueDate.Format("2006-01-02"),
		InvoiceURL: item.InvoiceURL, BankSlipURL: item.BankSlipURL, Description: item.Description,
		ProviderPaymentID: item.ProviderPaymentID,
	}
	if item.ConfirmedAt != nil {
		response.ConfirmedAt = item.ConfirmedAt.UTC().Format(time.RFC3339)
	}
	if item.ReceivedAt != nil {
		response.ReceivedAt = item.ReceivedAt.UTC().Format(time.RFC3339)
	}
	return response
}

func billingOverviewResponse(item domain.BillingOverview, invoices []domain.BillingInvoice) dto.BillingOverviewResponse {
	response := dto.BillingOverviewResponse{
		BoxID: string(item.Box.ID), BoxName: item.Box.Name, BoxStatus: string(item.Box.EffectiveStatus()),
		BillingAccessBlocked: item.Box.BillingAccessBlocked, BillingAccessReason: item.Box.BillingAccessReason,
	}
	if item.Customer != nil {
		value := billingCustomerResponse(*item.Customer)
		response.Customer = &value
	}
	if item.Subscription != nil {
		value := billingSubscriptionResponse(*item.Subscription)
		response.Subscription = &value
	}
	if item.Plan != nil {
		value := billingPlanResponse(*item.Plan)
		response.Plan = &value
	}
	if item.LatestInvoice != nil {
		value := billingInvoiceResponse(*item.LatestInvoice)
		response.LatestInvoice = &value
	}
	response.Invoices = make([]dto.BillingInvoiceResponse, 0, len(invoices))
	for _, invoice := range invoices {
		response.Invoices = append(response.Invoices, billingInvoiceResponse(invoice))
	}
	return response
}
