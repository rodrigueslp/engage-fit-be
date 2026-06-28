package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/handlers"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/dashboard"
	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/app/messages"
	"boxengage/backend/internal/app/reports"
	"boxengage/backend/internal/app/rewards"
	"boxengage/backend/internal/app/students"
	"boxengage/backend/internal/app/whatsapp"
	"boxengage/backend/internal/ports/services"
)

type RouterDependencies struct {
	TokenService       services.TokenService
	LoginUseCase       auth.LoginUseCase
	CurrentUserUseCase auth.GetCurrentUserUseCase
	CreateBoxUseCase   boxes.CreateBoxUseCase
	GetBoxUseCase      boxes.GetBoxUseCase
	UpdateBoxUseCase   boxes.UpdateBoxUseCase

	ListStudentsUseCase            students.ListStudentsUseCase
	GetStudentUseCase              students.GetStudentUseCase
	ListStudentCheckinsUseCase     students.ListStudentCheckinsUseCase
	UpdateStudentRiskStatusUseCase students.UpdateStudentRiskStatusUseCase

	ListImportsUseCase    imports.ListImportsUseCase
	GetImportUseCase      imports.GetImportUseCase
	ImportCheckinsUseCase imports.ImportCheckinsUseCase

	ListCampaignsUseCase               campaigns.ListCampaignsUseCase
	CreateCampaignUseCase              campaigns.CreateCampaignUseCase
	GetCampaignUseCase                 campaigns.GetCampaignUseCase
	UpdateCampaignUseCase              campaigns.UpdateCampaignUseCase
	CloseCampaignUseCase               campaigns.CloseCampaignUseCase
	DeleteCampaignUseCase              campaigns.DeleteCampaignUseCase
	ListCampaignGoalsUseCase           campaigns.ListCampaignGoalsUseCase
	UpsertCampaignGoalUseCase          campaigns.UpsertCampaignGoalUseCase
	DeleteCampaignGoalUseCase          campaigns.DeleteCampaignGoalUseCase
	ListCampaignProgressUseCase        campaigns.ListCampaignProgressUseCase
	RecalculateCampaignProgressUseCase campaigns.RecalculateCampaignProgressUseCase

	ListRewardsUseCase                 rewards.ListRewardsUseCase
	CreateRewardUseCase                rewards.CreateRewardUseCase
	GetRewardUseCase                   rewards.GetRewardUseCase
	UpdateRewardUseCase                rewards.UpdateRewardUseCase
	DeleteRewardUseCase                rewards.DeleteRewardUseCase
	ListRewardDeliveriesUseCase        rewards.ListRewardDeliveriesUseCase
	ListPendingRewardDeliveriesUseCase rewards.ListPendingRewardDeliveriesUseCase
	MarkRewardDeliveredUseCase         rewards.MarkRewardDeliveredUseCase

	DashboardSummaryUseCase         dashboard.GetDashboardSummaryUseCase
	DashboardActiveCampaignsUseCase dashboard.ListActiveCampaignsUseCase
	DashboardNearGoalUseCase        dashboard.ListNearGoalStudentsUseCase
	DashboardAtRiskUseCase          dashboard.ListAtRiskStudentsUseCase

	GetWhatsappSettingsUseCase    whatsapp.GetSettingsUseCase
	UpdateWhatsappSettingsUseCase whatsapp.UpdateSettingsUseCase
	TestWhatsappSettingsUseCase   whatsapp.TestSettingsUseCase

	ListMessageTemplatesUseCase  messages.ListMessageTemplatesUseCase
	CreateMessageTemplateUseCase messages.CreateMessageTemplateUseCase
	GetMessageTemplateUseCase    messages.GetMessageTemplateUseCase
	UpdateMessageTemplateUseCase messages.UpdateMessageTemplateUseCase
	DeleteMessageTemplateUseCase messages.DeleteMessageTemplateUseCase
	ListMessageCampaignsUseCase  messages.ListMessageCampaignsUseCase
	CreateMessageCampaignUseCase messages.CreateMessageCampaignUseCase
	GetMessageCampaignUseCase    messages.GetMessageCampaignUseCase
	SendMessageCampaignUseCase   messages.SendMessageCampaignUseCase
	ListMessageRecipientsUseCase messages.ListMessageRecipientsUseCase

	EligibleStudentsReportUseCase reports.EligibleStudentsReportUseCase
	PendingRewardsReportUseCase   reports.PendingRewardsReportUseCase
	MonthlyFrequencyReportUseCase reports.MonthlyFrequencyReportUseCase
	ReportExporter                services.ReportExporter
}

func NewRouter(deps RouterDependencies) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")

	authHandler := handlers.NewAuthHandler(deps.LoginUseCase, deps.CurrentUserUseCase)
	api.POST("/auth/login", authHandler.Login)

	setupHandler := handlers.NewSetupHandler(deps.CreateBoxUseCase)
	api.POST("/setup/owner", setupHandler.CreateOwner)

	protected := api.Group("")
	protected.Use(middleware.Auth(deps.TokenService), middleware.Tenant())

	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/auth/me", authHandler.Me)

	boxesHandler := handlers.NewBoxesHandler(deps.GetBoxUseCase, deps.UpdateBoxUseCase)
	protected.GET("/box", boxesHandler.Get)
	protected.PUT("/box", boxesHandler.Update)

	dashboardHandler := handlers.NewDashboardHandler(deps.DashboardSummaryUseCase, deps.DashboardActiveCampaignsUseCase, deps.DashboardNearGoalUseCase, deps.DashboardAtRiskUseCase, deps.ListPendingRewardDeliveriesUseCase)
	protected.GET("/dashboard/summary", dashboardHandler.Summary)
	protected.GET("/dashboard/active-campaigns", dashboardHandler.ActiveCampaigns)
	protected.GET("/dashboard/near-goal-students", dashboardHandler.NearGoalStudents)
	protected.GET("/dashboard/at-risk-students", dashboardHandler.AtRiskStudents)
	protected.GET("/dashboard/pending-rewards", dashboardHandler.PendingRewards)

	studentsHandler := handlers.NewStudentsHandler(deps.ListStudentsUseCase, deps.GetStudentUseCase, deps.ListStudentCheckinsUseCase, deps.UpdateStudentRiskStatusUseCase)
	protected.GET("/students", studentsHandler.List)
	protected.GET("/students/:id", studentsHandler.Get)
	protected.PATCH("/students/:id/risk-status", studentsHandler.UpdateRiskStatus)
	protected.GET("/students/:id/checkins", studentsHandler.Checkins)
	protected.GET("/students/:id/campaign-progress", studentsHandler.CampaignProgress)

	importsHandler := handlers.NewImportsHandler(deps.ImportCheckinsUseCase, deps.ListImportsUseCase, deps.GetImportUseCase)
	protected.POST("/imports", importsHandler.Create)
	protected.GET("/imports", importsHandler.List)
	protected.GET("/imports/:id", importsHandler.Get)

	campaignsHandler := handlers.NewCampaignsHandler(deps.ListCampaignsUseCase, deps.CreateCampaignUseCase, deps.GetCampaignUseCase, deps.UpdateCampaignUseCase, deps.CloseCampaignUseCase, deps.DeleteCampaignUseCase, deps.ListCampaignGoalsUseCase, deps.UpsertCampaignGoalUseCase, deps.DeleteCampaignGoalUseCase, deps.ListCampaignProgressUseCase, deps.RecalculateCampaignProgressUseCase, deps.GetStudentUseCase)
	protected.GET("/campaigns", campaignsHandler.List)
	protected.POST("/campaigns", campaignsHandler.Create)
	protected.GET("/campaigns/:id", campaignsHandler.Get)
	protected.PUT("/campaigns/:id", campaignsHandler.Update)
	protected.PATCH("/campaigns/:id/close", campaignsHandler.Close)
	protected.DELETE("/campaigns/:id", campaignsHandler.Delete)
	protected.GET("/campaigns/:id/goals", campaignsHandler.ListGoals)
	protected.POST("/campaigns/:id/goals", campaignsHandler.CreateGoal)
	protected.PUT("/campaigns/:id/goals/:goalId", campaignsHandler.UpdateGoal)
	protected.DELETE("/campaigns/:id/goals/:goalId", campaignsHandler.DeleteGoal)
	protected.GET("/campaigns/:id/progress", campaignsHandler.Progress)
	protected.POST("/campaigns/:id/recalculate-progress", campaignsHandler.RecalculateProgress)
	protected.GET("/campaigns/:id/eligible-students", campaignsHandler.EligibleStudents)
	protected.GET("/campaigns/:id/near-goal-students", campaignsHandler.NearGoalStudents)

	rewardsHandler := handlers.NewRewardsHandler(deps.ListRewardsUseCase, deps.CreateRewardUseCase, deps.GetRewardUseCase, deps.UpdateRewardUseCase, deps.DeleteRewardUseCase, deps.ListRewardDeliveriesUseCase, deps.ListPendingRewardDeliveriesUseCase, deps.MarkRewardDeliveredUseCase)
	protected.GET("/campaigns/:id/rewards", rewardsHandler.ListByCampaign)
	protected.POST("/campaigns/:id/rewards", rewardsHandler.Create)
	protected.PUT("/rewards/:id", rewardsHandler.Update)
	protected.DELETE("/rewards/:id", rewardsHandler.Delete)
	protected.GET("/rewards/deliveries", rewardsHandler.Deliveries)
	protected.GET("/rewards/pending-deliveries", rewardsHandler.PendingDeliveries)
	protected.PATCH("/reward-deliveries/:id/deliver", rewardsHandler.MarkDelivered)

	whatsappHandler := handlers.NewWhatsappHandler(deps.GetWhatsappSettingsUseCase, deps.UpdateWhatsappSettingsUseCase, deps.TestWhatsappSettingsUseCase)
	protected.GET("/whatsapp/settings", whatsappHandler.GetSettings)
	protected.PUT("/whatsapp/settings", whatsappHandler.UpdateSettings)
	protected.POST("/whatsapp/settings/test", whatsappHandler.TestSettings)

	messagesHandler := handlers.NewMessagesHandler(deps.ListMessageTemplatesUseCase, deps.CreateMessageTemplateUseCase, deps.GetMessageTemplateUseCase, deps.UpdateMessageTemplateUseCase, deps.DeleteMessageTemplateUseCase, deps.ListMessageCampaignsUseCase, deps.CreateMessageCampaignUseCase, deps.GetMessageCampaignUseCase, deps.SendMessageCampaignUseCase, deps.ListMessageRecipientsUseCase)
	protected.GET("/message-templates", messagesHandler.ListTemplates)
	protected.POST("/message-templates", messagesHandler.CreateTemplate)
	protected.GET("/message-templates/:id", messagesHandler.GetTemplate)
	protected.PUT("/message-templates/:id", messagesHandler.UpdateTemplate)
	protected.DELETE("/message-templates/:id", messagesHandler.DeleteTemplate)
	protected.GET("/message-campaigns", messagesHandler.ListCampaigns)
	protected.POST("/message-campaigns", messagesHandler.CreateCampaign)
	protected.GET("/message-campaigns/:id", messagesHandler.GetCampaign)
	protected.GET("/message-campaigns/:id/preview", messagesHandler.PreviewCampaign)
	protected.POST("/message-campaigns/:id/send", messagesHandler.SendCampaign)
	protected.GET("/message-campaigns/:id/recipients", messagesHandler.ListRecipients)

	reportsHandler := handlers.NewReportsHandler(deps.EligibleStudentsReportUseCase, deps.PendingRewardsReportUseCase, deps.MonthlyFrequencyReportUseCase, deps.ReportExporter)
	protected.GET("/reports/eligible-students", reportsHandler.EligibleStudents)
	protected.GET("/reports/pending-rewards", reportsHandler.PendingRewards)
	protected.GET("/reports/monthly-frequency", reportsHandler.MonthlyFrequency)

	return router
}
