package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"boxengage/backend/internal/adapters/http/handlers"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/automation"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/dashboard"
	"boxengage/backend/internal/app/email"
	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/app/messages"
	"boxengage/backend/internal/app/platformadmin"
	"boxengage/backend/internal/app/reports"
	"boxengage/backend/internal/app/rewards"
	"boxengage/backend/internal/app/students"
	"boxengage/backend/internal/app/whatsapp"
	"boxengage/backend/internal/app/workouts"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type RouterDependencies struct {
	AppEnv                    string
	TokenService              services.TokenService
	UserRepository            repositories.UserRepository
	LoginUseCase              auth.LoginUseCase
	CurrentUserUseCase        auth.GetCurrentUserUseCase
	ChangePasswordUseCase     auth.ChangePasswordUseCase
	LogoutUseCase             auth.LogoutUseCase
	ResetOwnerPasswordUseCase platformadmin.ResetOwnerPasswordUseCase
	CreateBoxUseCase          boxes.CreateBoxUseCase
	GetBoxUseCase             boxes.GetBoxUseCase
	UpdateBoxUseCase          boxes.UpdateBoxUseCase

	ListStudentsUseCase            students.ListStudentsUseCase
	GetStudentUseCase              students.GetStudentUseCase
	ListStudentCheckinsUseCase     students.ListStudentCheckinsUseCase
	UpdateStudentRiskStatusUseCase students.UpdateStudentRiskStatusUseCase
	ExportStudentDataUseCase       students.ExportStudentDataUseCase
	UpdateContactPreferenceUseCase students.UpdateContactPreferenceUseCase
	AnonymizeStudentUseCase        students.AnonymizeStudentUseCase

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

	GetEmailSettingsUseCase    email.GetSettingsUseCase
	UpdateEmailSettingsUseCase email.UpdateSettingsUseCase
	TestEmailSettingsUseCase   email.TestSettingsUseCase
	ListEmailTemplatesUseCase  email.ListEmailTemplatesUseCase
	CreateEmailTemplateUseCase email.CreateEmailTemplateUseCase
	GetEmailTemplateUseCase    email.GetEmailTemplateUseCase
	UpdateEmailTemplateUseCase email.UpdateEmailTemplateUseCase
	DeleteEmailTemplateUseCase email.DeleteEmailTemplateUseCase
	ListEmailCampaignsUseCase  email.ListEmailCampaignsUseCase
	CreateEmailCampaignUseCase email.CreateEmailCampaignUseCase
	GetEmailCampaignUseCase    email.GetEmailCampaignUseCase
	SendEmailCampaignUseCase   email.SendEmailCampaignUseCase
	ListEmailRecipientsUseCase email.ListEmailRecipientsUseCase

	ListAutomationRunsUseCase        automation.ListRunsUseCase
	GetAutomationRunUseCase          automation.GetRunUseCase
	CreateAutomationRunUseCase       automation.CreateRunUseCase
	UpdateAutomationRunUseCase       automation.UpdateRunUseCase
	ListAutomationSchedulesUseCase   automation.ListSchedulesUseCase
	GetAutomationScheduleUseCase     automation.GetScheduleUseCase
	CreateAutomationScheduleUseCase  automation.CreateScheduleUseCase
	UpdateAutomationScheduleUseCase  automation.UpdateScheduleUseCase
	DeleteAutomationScheduleUseCase  automation.DeleteScheduleUseCase
	ExecuteAutomationScheduleUseCase automation.ExecuteScheduleUseCase

	ListWorkoutsUseCase          workouts.ListWorkoutsUseCase
	CreateWorkoutUseCase         workouts.CreateWorkoutUseCase
	GetWorkoutUseCase            workouts.GetWorkoutUseCase
	UpdateWorkoutUseCase         workouts.UpdateWorkoutUseCase
	DeleteWorkoutUseCase         workouts.DeleteWorkoutUseCase
	ListWorkoutDraftsUseCase     workouts.ListWorkoutDraftsUseCase
	GenerateWorkoutDraftUseCase  workouts.GenerateWorkoutDraftUseCase
	GetWorkoutDraftUseCase       workouts.GetWorkoutDraftUseCase
	UpdateWorkoutDraftUseCase    workouts.UpdateWorkoutDraftUseCase
	ApproveWorkoutDraftUseCase   workouts.ApproveWorkoutDraftUseCase
	SendWorkoutDraftUseCase      workouts.SendWorkoutDraftUseCase
	ListWorkoutRecipientsUseCase workouts.ListWorkoutRecipientsUseCase

	EligibleStudentsReportUseCase reports.EligibleStudentsReportUseCase
	PendingRewardsReportUseCase   reports.PendingRewardsReportUseCase
	MonthlyFrequencyReportUseCase reports.MonthlyFrequencyReportUseCase
	ReportExporter                services.ReportExporter
	MessagingAdminUseCases        platformadmin.MessagingAdminUseCases
	TenantMessagingUsageUseCase   platformadmin.GetTenantMessagingUsageUseCase
	OwnerSetupEnabled             bool
	OwnerSetupToken               string
	HTTPMaxBodyBytes              int64
	ImportMaxUploadBytes          int64
	LoginRateLimitRequests        int
	LoginRateLimitWindowSeconds   int
	SetupRateLimitRequests        int
	SetupRateLimitWindowSeconds   int
	TrustedProxies                []string
	ReadinessCheck                func(context.Context) error
	MetricsHandler                http.Handler
	MetricsBearerToken            string
	ObservabilityService          string
	SessionConfig                 middleware.SessionConfig
	CORSAllowedOrigins            []string
	Capabilities                  middleware.Capabilities
	BuildVersion                  string
	BuildCommit                   string
	BuildTime                     string
}

func NewRouter(deps RouterDependencies) *gin.Engine {
	if deps.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	if err := router.SetTrustedProxies(deps.TrustedProxies); err != nil {
		panic("invalid TRUSTED_PROXIES: " + err.Error())
	}
	router.Use(middleware.BodySizeLimit(deps.HTTPMaxBodyBytes, deps.ImportMaxUploadBytes))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(deps.CORSAllowedOrigins))
	router.Use(middleware.RequestID())
	if deps.ObservabilityService != "" {
		router.Use(otelgin.Middleware(deps.ObservabilityService))
	}
	router.Use(middleware.HTTPMetrics())
	router.Use(middleware.Logger())
	router.Use(gin.Recovery())

	liveness := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	router.GET("/health", liveness)
	router.GET("/health/live", liveness)
	router.GET("/health/ready", func(c *gin.Context) {
		if deps.ReadinessCheck != nil {
			if err := deps.ReadinessCheck(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.GET("/health/build", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": deps.BuildVersion, "commit": deps.BuildCommit, "build_time": deps.BuildTime})
	})
	if deps.MetricsHandler != nil {
		router.GET("/metrics", middleware.MetricsAccess(deps.MetricsBearerToken), gin.WrapH(deps.MetricsHandler))
	}

	api := router.Group("/api/v1")
	api.GET("/capabilities", func(c *gin.Context) { c.JSON(http.StatusOK, deps.Capabilities) })
	api.Use(middleware.FeatureGates(deps.Capabilities))

	authHandler := handlers.NewAuthHandler(deps.LoginUseCase, deps.CurrentUserUseCase, deps.ChangePasswordUseCase, deps.LogoutUseCase, deps.SessionConfig)
	loginLimiter := middleware.NewWindowRateLimiter(deps.LoginRateLimitRequests, time.Duration(deps.LoginRateLimitWindowSeconds)*time.Second)
	api.POST("/auth/login", middleware.JSONRateLimit(loginLimiter, "email"), authHandler.Login)

	setupHandler := handlers.NewSetupHandler(deps.CreateBoxUseCase)
	setupLimiter := middleware.NewWindowRateLimiter(deps.SetupRateLimitRequests, time.Duration(deps.SetupRateLimitWindowSeconds)*time.Second)
	api.POST("/setup/owner", middleware.JSONRateLimit(setupLimiter, "owner_email"), middleware.OwnerSetupAccess(deps.OwnerSetupEnabled, deps.OwnerSetupToken), setupHandler.CreateOwner)

	authenticated := api.Group("")
	authenticated.Use(middleware.Auth(deps.TokenService, deps.UserRepository, deps.SessionConfig), middleware.CSRF(deps.SessionConfig))
	authenticated.POST("/auth/logout", authHandler.Logout)
	authenticated.GET("/auth/me", authHandler.Me)
	authenticated.PUT("/auth/password", authHandler.ChangePassword)

	protected := authenticated.Group("")
	protected.Use(middleware.Tenant())

	governanceHandler := handlers.NewMessagingGovernanceHandler(deps.MessagingAdminUseCases, deps.TenantMessagingUsageUseCase)
	protected.GET("/messaging/usage", governanceHandler.TenantUsage)

	admin := authenticated.Group("/admin")
	admin.Use(middleware.PlatformAdmin())
	admin.GET("/messaging/boxes", governanceHandler.ListBoxes)
	admin.GET("/messaging/boxes/:id/policy", governanceHandler.GetBoxPolicy)
	admin.PUT("/messaging/boxes/:id/policy", governanceHandler.UpdateBoxPolicy)
	admin.GET("/messaging/platform/policy", governanceHandler.GetPlatformPolicy)
	admin.PUT("/messaging/platform/policy", governanceHandler.UpdatePlatformPolicy)
	adminAccessHandler := handlers.NewPlatformAdminAccessHandler(deps.ResetOwnerPasswordUseCase)
	admin.PUT("/boxes/:id/owner-password", adminAccessHandler.ResetOwnerPassword)

	boxesHandler := handlers.NewBoxesHandler(deps.GetBoxUseCase, deps.UpdateBoxUseCase)
	protected.GET("/box", boxesHandler.Get)
	protected.PUT("/box", boxesHandler.Update)

	dashboardHandler := handlers.NewDashboardHandler(deps.DashboardSummaryUseCase, deps.DashboardActiveCampaignsUseCase, deps.DashboardNearGoalUseCase, deps.DashboardAtRiskUseCase, deps.ListPendingRewardDeliveriesUseCase)
	protected.GET("/dashboard/summary", dashboardHandler.Summary)
	protected.GET("/dashboard/active-campaigns", dashboardHandler.ActiveCampaigns)
	protected.GET("/dashboard/near-goal-students", dashboardHandler.NearGoalStudents)
	protected.GET("/dashboard/at-risk-students", dashboardHandler.AtRiskStudents)
	protected.GET("/dashboard/pending-rewards", dashboardHandler.PendingRewards)

	studentsHandler := handlers.NewStudentsHandler(deps.ListStudentsUseCase, deps.GetStudentUseCase, deps.ListStudentCheckinsUseCase, deps.UpdateStudentRiskStatusUseCase, deps.ExportStudentDataUseCase, deps.UpdateContactPreferenceUseCase, deps.AnonymizeStudentUseCase)
	protected.GET("/students", studentsHandler.List)
	protected.GET("/students/:id", studentsHandler.Get)
	protected.PATCH("/students/:id/risk-status", studentsHandler.UpdateRiskStatus)
	protected.GET("/students/:id/checkins", studentsHandler.Checkins)
	protected.GET("/students/:id/campaign-progress", studentsHandler.CampaignProgress)
	protected.GET("/students/:id/privacy-export", studentsHandler.ExportData)
	protected.PATCH("/students/:id/contact-preference", studentsHandler.UpdateContactPreference)
	protected.POST("/students/:id/anonymize", studentsHandler.Anonymize)

	importsHandler := handlers.NewImportsHandler(deps.ImportCheckinsUseCase, deps.ListImportsUseCase, deps.GetImportUseCase)
	protected.POST("/imports", importsHandler.Create)
	protected.GET("/imports", importsHandler.List)
	protected.GET("/imports/:id", importsHandler.Get)

	messagesHandler := handlers.NewMessagesHandler(deps.ListMessageTemplatesUseCase, deps.CreateMessageTemplateUseCase, deps.GetMessageTemplateUseCase, deps.UpdateMessageTemplateUseCase, deps.DeleteMessageTemplateUseCase, deps.ListMessageCampaignsUseCase, deps.CreateMessageCampaignUseCase, deps.GetMessageCampaignUseCase, deps.SendMessageCampaignUseCase, deps.ListMessageRecipientsUseCase)

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
	protected.GET("/campaigns/:id/whatsapp-templates/preview", messagesHandler.PreviewOfficialTemplates)

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
	protected.POST("/whatsapp/settings/test", whatsappHandler.TestSettings)
	admin.GET("/messaging/boxes/:id/whatsapp-settings", whatsappHandler.AdminGetSettings)
	admin.PUT("/messaging/boxes/:id/whatsapp-settings", whatsappHandler.AdminUpdateSettings)
	admin.POST("/messaging/boxes/:id/whatsapp-settings/test", whatsappHandler.AdminTestSettings)

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

	workoutsHandler := handlers.NewWorkoutsHandler(deps.ListWorkoutsUseCase, deps.CreateWorkoutUseCase, deps.GetWorkoutUseCase, deps.UpdateWorkoutUseCase, deps.DeleteWorkoutUseCase, deps.ListWorkoutDraftsUseCase, deps.GenerateWorkoutDraftUseCase, deps.GetWorkoutDraftUseCase, deps.UpdateWorkoutDraftUseCase, deps.ApproveWorkoutDraftUseCase, deps.SendWorkoutDraftUseCase, deps.ListWorkoutRecipientsUseCase)
	protected.GET("/workouts", workoutsHandler.List)
	protected.POST("/workouts", workoutsHandler.Create)
	protected.GET("/workouts/:id", workoutsHandler.Get)
	protected.PUT("/workouts/:id", workoutsHandler.Update)
	protected.DELETE("/workouts/:id", workoutsHandler.Delete)
	protected.GET("/workouts/:id/message-drafts", workoutsHandler.ListDrafts)
	protected.POST("/workouts/:id/message-drafts", workoutsHandler.GenerateDraft)
	protected.PUT("/workout-message-drafts/:id", workoutsHandler.UpdateDraft)
	protected.POST("/workout-message-drafts/:id/approve", workoutsHandler.ApproveDraft)
	protected.POST("/workout-message-drafts/:id/send", workoutsHandler.SendDraft)
	protected.GET("/workout-message-drafts/:id/recipients", workoutsHandler.ListRecipients)

	emailHandler := handlers.NewEmailHandler(deps.GetEmailSettingsUseCase, deps.UpdateEmailSettingsUseCase, deps.TestEmailSettingsUseCase, deps.ListEmailTemplatesUseCase, deps.CreateEmailTemplateUseCase, deps.GetEmailTemplateUseCase, deps.UpdateEmailTemplateUseCase, deps.DeleteEmailTemplateUseCase, deps.ListEmailCampaignsUseCase, deps.CreateEmailCampaignUseCase, deps.GetEmailCampaignUseCase, deps.SendEmailCampaignUseCase, deps.ListEmailRecipientsUseCase)
	protected.GET("/email/settings", emailHandler.GetSettings)
	protected.PUT("/email/settings", emailHandler.UpdateSettings)
	protected.POST("/email/settings/test", emailHandler.TestSettings)
	protected.GET("/email-templates", emailHandler.ListTemplates)
	protected.POST("/email-templates", emailHandler.CreateTemplate)
	protected.GET("/email-templates/:id", emailHandler.GetTemplate)
	protected.PUT("/email-templates/:id", emailHandler.UpdateTemplate)
	protected.DELETE("/email-templates/:id", emailHandler.DeleteTemplate)
	protected.GET("/email-campaigns", emailHandler.ListCampaigns)
	protected.POST("/email-campaigns", emailHandler.CreateCampaign)
	protected.GET("/email-campaigns/:id", emailHandler.GetCampaign)
	protected.GET("/email-campaigns/:id/preview", emailHandler.PreviewCampaign)
	protected.POST("/email-campaigns/:id/send", emailHandler.SendCampaign)
	protected.GET("/email-campaigns/:id/recipients", emailHandler.ListRecipients)

	automationHandler := handlers.NewAutomationHandler(deps.ListAutomationRunsUseCase, deps.GetAutomationRunUseCase, deps.CreateAutomationRunUseCase, deps.UpdateAutomationRunUseCase, deps.ListAutomationSchedulesUseCase, deps.GetAutomationScheduleUseCase, deps.CreateAutomationScheduleUseCase, deps.UpdateAutomationScheduleUseCase, deps.DeleteAutomationScheduleUseCase, deps.ExecuteAutomationScheduleUseCase)
	protected.GET("/automation/runs", automationHandler.ListRuns)
	protected.POST("/automation/runs", automationHandler.CreateRun)
	protected.GET("/automation/runs/:id", automationHandler.GetRun)
	protected.PATCH("/automation/runs/:id", automationHandler.UpdateRun)
	protected.GET("/automation/schedules", automationHandler.ListSchedules)
	protected.POST("/automation/schedules", automationHandler.CreateSchedule)
	protected.PUT("/automation/schedules/:id", automationHandler.UpdateSchedule)
	protected.DELETE("/automation/schedules/:id", automationHandler.DeleteSchedule)
	protected.POST("/automation/schedules/:id/run", automationHandler.RunScheduleNow)

	reportsHandler := handlers.NewReportsHandler(deps.EligibleStudentsReportUseCase, deps.PendingRewardsReportUseCase, deps.MonthlyFrequencyReportUseCase, deps.ReportExporter)
	protected.GET("/reports/eligible-students", reportsHandler.EligibleStudents)
	protected.GET("/reports/pending-rewards", reportsHandler.PendingRewards)
	protected.GET("/reports/monthly-frequency", reportsHandler.MonthlyFrequency)
	protected.GET("/checkins/summary", reportsHandler.CheckinSummary)

	return router
}
