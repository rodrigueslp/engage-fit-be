package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"boxengage/backend/internal/adapters/http/handlers"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/announcements"
	athleteapp "boxengage/backend/internal/app/athlete"
	"boxengage/backend/internal/app/attendance"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/automation"
	billingapp "boxengage/backend/internal/app/billing"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/catalog"
	"boxengage/backend/internal/app/checkiningestion"
	commerceapp "boxengage/backend/internal/app/commerce"
	"boxengage/backend/internal/app/contactactivation"
	"boxengage/backend/internal/app/contractlifecycle"
	"boxengage/backend/internal/app/contracts"
	"boxengage/backend/internal/app/dashboard"
	"boxengage/backend/internal/app/email"
	"boxengage/backend/internal/app/healthforms"
	"boxengage/backend/internal/app/imports"
	memberbillingapp "boxengage/backend/internal/app/memberbilling"
	"boxengage/backend/internal/app/memberships"
	"boxengage/backend/internal/app/messages"
	"boxengage/backend/internal/app/organizations"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/app/platformadmin"
	"boxengage/backend/internal/app/reports"
	reservationsapp "boxengage/backend/internal/app/reservations"
	"boxengage/backend/internal/app/retention"
	"boxengage/backend/internal/app/rewards"
	schedulingapp "boxengage/backend/internal/app/scheduling"
	"boxengage/backend/internal/app/students"
	"boxengage/backend/internal/app/team"
	"boxengage/backend/internal/app/whatsapp"
	"boxengage/backend/internal/app/workouts"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type RouterDependencies struct {
	AppEnv                        string
	TokenService                  services.TokenService
	UserRepository                repositories.UserRepository
	BoxRepository                 repositories.BoxRepository
	LoginUseCase                  auth.LoginUseCase
	CurrentUserUseCase            auth.GetCurrentUserUseCase
	ChangePasswordUseCase         auth.ChangePasswordUseCase
	LogoutUseCase                 auth.LogoutUseCase
	ResetOwnerPasswordUseCase     platformadmin.ResetOwnerPasswordUseCase
	BoxAdminUseCases              platformadmin.BoxAdminUseCases
	CreateBoxUseCase              boxes.CreateBoxUseCase
	GetBoxUseCase                 boxes.GetBoxUseCase
	UpdateBoxUseCase              boxes.UpdateBoxUseCase
	CatalogService                *catalog.Service
	PeopleService                 *people.Service
	MembershipService             *memberships.Service
	EnrollmentService             *memberships.EnrollmentService
	ContractService               *contracts.Service
	ContractLifecycleService      *contractlifecycle.Service
	HealthFormService             *healthforms.Service
	ContractSignatureWebhookToken string
	SchedulingService             *schedulingapp.Service
	SchedulingGridService         *schedulingapp.GridService
	ReservationService            *reservationsapp.Service
	OrganizationService           *organizations.Service
	UnitAccessAuthorizer          repositories.UnitAccessAuthorizer
	SwitchUnitContextUseCase      auth.SwitchUnitContextUseCase

	ListStudentsUseCase            students.ListStudentsUseCase
	GetStudentUseCase              students.GetStudentUseCase
	ListStudentCheckinsUseCase     students.ListStudentCheckinsUseCase
	UpdateStudentRiskStatusUseCase students.UpdateStudentRiskStatusUseCase
	ExportStudentDataUseCase       students.ExportStudentDataUseCase
	UpdateContactPreferenceUseCase students.UpdateContactPreferenceUseCase
	AnonymizeStudentUseCase        students.AnonymizeStudentUseCase
	ContactActivationService       *contactactivation.Service
	AttendanceService              *attendance.Service
	AttendanceAccessService        *attendance.AccessService

	ListImportsUseCase      imports.ListImportsUseCase
	GetImportUseCase        imports.GetImportUseCase
	ImportCheckinsUseCase   imports.ImportCheckinsUseCase
	CheckinIngestionService *checkiningestion.Service
	TeamService             *team.Service

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
	DashboardTodayUseCase           dashboard.GetTodayOperationsUseCase
	RetentionService                retention.Service

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
	BillingService                *billingapp.Service
	MemberBillingService          *memberbillingapp.Service
	AthleteService                *athleteapp.Service
	AnnouncementService           *announcements.Service
	CommerceService               *commerceapp.Service
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
	if deps.OrganizationService != nil {
		authHandler = authHandler.WithUnitContext(deps.SwitchUnitContextUseCase)
	}
	loginLimiter := middleware.NewWindowRateLimiter(deps.LoginRateLimitRequests, time.Duration(deps.LoginRateLimitWindowSeconds)*time.Second)
	api.POST("/auth/login", middleware.JSONRateLimit(loginLimiter, "email"), authHandler.Login)
	if deps.AthleteService != nil {
		athleteSession := athleteSessionConfig(deps.SessionConfig)
		athleteHandler := handlers.NewAthleteHandler(deps.AthleteService, athleteSession)
		athleteLimiter := middleware.NewWindowRateLimiter(10, time.Minute)
		api.GET("/athlete/invitations/:token", athleteHandler.PreviewInvitation)
		api.POST("/athlete/invitations/:token/claim", middleware.JSONRateLimit(athleteLimiter, "email"), athleteHandler.ClaimInvitation)
		api.POST("/athlete/auth/login", middleware.JSONRateLimit(athleteLimiter, "email"), athleteHandler.Login)
		api.POST("/athlete/auth/password-reset", middleware.JSONRateLimit(athleteLimiter, "email"), athleteHandler.RequestPasswordReset)
		api.POST("/athlete/auth/password-reset/:token", middleware.JSONRateLimit(athleteLimiter, "token"), athleteHandler.ResetPassword)
		api.POST("/athlete/auth/verify-email/:token", middleware.JSONRateLimit(athleteLimiter, "token"), athleteHandler.VerifyEmail)
		athleteAuthenticated := api.Group("/athlete")
		athleteAuthenticated.Use(middleware.AthleteAuth(deps.AthleteService, athleteSession), middleware.CSRF(athleteSession))
		athleteAuthenticated.GET("/me", athleteHandler.Me)
		athleteAuthenticated.GET("/portal", athleteHandler.Portal)
		athleteAuthenticated.POST("/memberships/:membership_id/sessions/:session_id/book", athleteHandler.BookClass)
		athleteAuthenticated.POST("/memberships/:membership_id/sessions/:session_id/reservations/:reservation_id/cancel", athleteHandler.CancelClassReservation)
		athleteAuthenticated.GET("/workouts", athleteHandler.Workouts)
		athleteAuthenticated.PUT("/workouts/:id/result", athleteHandler.SaveResult)
		athleteAuthenticated.POST("/workouts/:id/explanation", athleteHandler.ExplainWorkout)
		athleteAuthenticated.GET("/results", athleteHandler.Results)
		athleteAuthenticated.GET("/personal-records", athleteHandler.PersonalRecords)
		athleteAuthenticated.POST("/personal-records", athleteHandler.RecordPersonalBest)
		athleteAuthenticated.POST("/personal-records/:id/confirm", athleteHandler.ConfirmPersonalRecord)
		athleteAuthenticated.POST("/auth/verify-email", athleteHandler.RequestEmailVerification)
		athleteAuthenticated.POST("/auth/logout", athleteHandler.Logout)
		if deps.AnnouncementService != nil {
			announcementHandler := handlers.NewAnnouncementHandler(deps.AnnouncementService)
			athleteAuthenticated.GET("/announcements", announcementHandler.AthleteList)
			athleteAuthenticated.POST("/announcements/:id/view", announcementHandler.MarkViewed)
		}
	}

	setupHandler := handlers.NewSetupHandler(deps.CreateBoxUseCase)
	setupLimiter := middleware.NewWindowRateLimiter(deps.SetupRateLimitRequests, time.Duration(deps.SetupRateLimitWindowSeconds)*time.Second)
	api.POST("/setup/owner", middleware.JSONRateLimit(setupLimiter, "owner_email"), middleware.OwnerSetupAccess(deps.OwnerSetupEnabled, deps.OwnerSetupToken), setupHandler.CreateOwner)

	if deps.BillingService != nil {
		billingHandler := handlers.NewBillingHandler(deps.BillingService)
		api.POST("/webhooks/asaas", billingHandler.Webhook)
	}
	if deps.CheckinIngestionService != nil {
		ingestionHandler := handlers.NewCheckinIngestionHandler(deps.CheckinIngestionService)
		api.POST("/checkin-ingestion/:sourceId/imports", ingestionHandler.Ingest)
	}
	if deps.ContactActivationService != nil {
		activationHandler := handlers.NewContactActivationHandler(deps.ContactActivationService)
		activationLimiter := middleware.NewWindowRateLimiter(10, time.Minute)
		api.GET("/public/contact-activation/:code", activationHandler.PublicConfig)
		api.POST("/public/contact-activation/:code", middleware.JSONRateLimit(activationLimiter, "name"), activationHandler.Start)
		api.POST("/webhooks/twilio/whatsapp", activationHandler.Inbound)
	}
	if deps.AttendanceService != nil {
		attendanceHandler := handlers.NewAttendanceHandler(deps.AttendanceService)
		selfCheckinLimiter := middleware.NewWindowRateLimiter(60, time.Minute)
		api.GET("/public/self-checkin/:token", attendanceHandler.PublicSession)
		api.POST("/public/self-checkin/:token", middleware.JSONRateLimit(selfCheckinLimiter, "phone"), attendanceHandler.SelfCheckin)
	}
	authenticated := api.Group("")
	if deps.UnitAccessAuthorizer != nil {
		authenticated.Use(middleware.AuthWithUnitAccess(deps.TokenService, deps.UserRepository, deps.BoxRepository, deps.UnitAccessAuthorizer, deps.SessionConfig), middleware.CSRF(deps.SessionConfig))
	} else {
		authenticated.Use(middleware.Auth(deps.TokenService, deps.UserRepository, deps.BoxRepository, deps.SessionConfig), middleware.CSRF(deps.SessionConfig))
	}
	authenticated.POST("/auth/logout", authHandler.Logout)
	authenticated.GET("/auth/me", authHandler.Me)
	authenticated.PUT("/auth/password", authHandler.ChangePassword)
	if deps.OrganizationService != nil {
		authenticated.POST("/auth/unit-context", authHandler.SwitchUnitContext)
	}

	protected := authenticated.Group("")
	if deps.AttendanceAccessService != nil {
		attendanceAccessHandler := handlers.NewAttendanceAccessHandler(deps.AttendanceAccessService)
		attendanceAccess := protected.Group("/units/:unit_id/attendance")
		attendanceAccess.GET("/today", attendanceAccessHandler.Today)
		attendanceAccess.POST("/admissions", attendanceAccessHandler.Admit)
	}
	protected.Use(middleware.Tenant(), middleware.TenantRoleAccess())
	if deps.OrganizationService != nil {
		organizationHandler := handlers.NewOrganizationHandler(deps.OrganizationService)
		protected.GET("/units", organizationHandler.ListUnits)
		ownerUnits := protected.Group("")
		ownerUnits.Use(middleware.Owner())
		ownerUnits.POST("/units", organizationHandler.CreateUnit)
		ownerUnits.PATCH("/units/:unit_id", organizationHandler.UpdateUnit)
		ownerUnits.GET("/team/coaches/:id/units", organizationHandler.MemberUnitAccess)
		ownerUnits.PUT("/team/coaches/:id/units", organizationHandler.ReplaceMemberUnitAccess)
	}

	governanceHandler := handlers.NewMessagingGovernanceHandler(deps.MessagingAdminUseCases, deps.TenantMessagingUsageUseCase)
	protected.GET("/messaging/usage", governanceHandler.TenantUsage)
	if deps.TeamService != nil {
		teamHandler := handlers.NewTeamHandler(deps.TeamService)
		protected.GET("/team/members", teamHandler.List)
		ownerTeam := protected.Group("/team")
		ownerTeam.Use(middleware.Owner())
		ownerTeam.POST("/coaches", teamHandler.CreateCoach)
		ownerTeam.PATCH("/coaches/:id", teamHandler.UpdateCoach)
		ownerTeam.PUT("/coaches/:id/password", teamHandler.ResetPassword)
	}

	admin := authenticated.Group("/admin")
	admin.Use(middleware.PlatformAdmin())
	admin.GET("/messaging/boxes", governanceHandler.ListBoxes)
	admin.GET("/messaging/boxes/:id/policy", governanceHandler.GetBoxPolicy)
	admin.PUT("/messaging/boxes/:id/policy", governanceHandler.UpdateBoxPolicy)
	admin.GET("/messaging/platform/policy", governanceHandler.GetPlatformPolicy)
	admin.PUT("/messaging/platform/policy", governanceHandler.UpdatePlatformPolicy)
	adminAccessHandler := handlers.NewPlatformAdminAccessHandler(deps.ResetOwnerPasswordUseCase)
	admin.PUT("/boxes/:id/owner-password", adminAccessHandler.ResetOwnerPassword)
	adminBoxesHandler := handlers.NewAdminBoxesHandler(deps.BoxAdminUseCases)
	admin.GET("/boxes", adminBoxesHandler.List)
	admin.POST("/boxes", adminBoxesHandler.Create)
	admin.PATCH("/boxes/:id", adminBoxesHandler.Update)
	admin.POST("/boxes/:id/suspend", adminBoxesHandler.Suspend)
	admin.POST("/boxes/:id/reactivate", adminBoxesHandler.Reactivate)
	admin.POST("/boxes/:id/archive", adminBoxesHandler.Archive)
	if deps.OrganizationService != nil {
		admin.POST("/organizations/:organization_id/units", handlers.NewOrganizationHandler(deps.OrganizationService).CreateUnitAsPlatformAdmin)
	}

	if deps.BillingService != nil {
		billingHandler := handlers.NewBillingHandler(deps.BillingService)
		admin.GET("/billing/summary", billingHandler.Summary)
		admin.GET("/billing/plans", billingHandler.ListPlans)
		admin.POST("/billing/plans", billingHandler.CreatePlan)
		admin.PUT("/billing/plans/:id", billingHandler.UpdatePlan)
		admin.GET("/billing/boxes", billingHandler.ListBoxes)
		admin.GET("/billing/boxes/:id", billingHandler.GetAdminBox)
		admin.PUT("/billing/boxes/:id/customer", billingHandler.SaveCustomer)
		admin.POST("/billing/boxes/:id/subscription", billingHandler.CreateSubscription)
		admin.POST("/billing/boxes/:id/subscription/cancel", billingHandler.CancelSubscription)
		admin.POST("/billing/boxes/:id/subscription/grace", billingHandler.GrantGrace)
		admin.POST("/billing/reconcile", billingHandler.Reconcile)
		protected.GET("/billing", billingHandler.GetOwnerBilling)
	}

	boxesHandler := handlers.NewBoxesHandler(deps.GetBoxUseCase, deps.UpdateBoxUseCase)
	protected.GET("/box", boxesHandler.Get)
	protected.PUT("/box", boxesHandler.Update)
	if deps.AnnouncementService != nil {
		announcementHandler := handlers.NewAnnouncementHandler(deps.AnnouncementService)
		ownerAnnouncements := protected.Group("/units/:unit_id/announcements")
		ownerAnnouncements.Use(middleware.Owner())
		ownerAnnouncements.GET("", announcementHandler.List)
		ownerAnnouncements.POST("", announcementHandler.Create)
		ownerAnnouncements.PATCH("/:id", announcementHandler.Update)
		ownerAnnouncements.POST("/:id/publish", announcementHandler.Publish)
		ownerAnnouncements.POST("/:id/archive", announcementHandler.Archive)
	}
	if deps.CommerceService != nil {
		commerceHandler := handlers.NewCommerceHandler(deps.CommerceService)
		ownerCommerce := protected.Group("/units/:unit_id/commerce")
		ownerCommerce.Use(middleware.Owner())
		ownerCommerce.GET("/products", commerceHandler.ListProducts)
		ownerCommerce.POST("/products", commerceHandler.CreateProduct)
		ownerCommerce.PATCH("/products/:id", commerceHandler.UpdateProduct)
		ownerCommerce.GET("/sales", commerceHandler.ListSales)
		ownerCommerce.POST("/sales", commerceHandler.CreateSale)
	}
	if deps.CatalogService != nil {
		catalogHandler := handlers.NewCatalogHandler(deps.CatalogService)
		protected.GET("/organization", catalogHandler.Organization)
		protected.GET("/modalities", catalogHandler.ListModalities)
		protected.GET("/service-offerings", catalogHandler.ListServiceOfferings)
		protected.GET("/units/:unit_id/spaces", catalogHandler.ListSpaces)
		ownerCatalog := protected.Group("")
		ownerCatalog.Use(middleware.Owner())
		ownerCatalog.POST("/modalities", catalogHandler.CreateModality)
		ownerCatalog.PATCH("/modalities/:id", catalogHandler.UpdateModality)
		ownerCatalog.PUT("/units/:unit_id/modalities/:modality_id", catalogHandler.SetModalityActive)
		ownerCatalog.POST("/service-offerings", catalogHandler.CreateServiceOffering)
		ownerCatalog.PATCH("/service-offerings/:id", catalogHandler.UpdateServiceOffering)
		ownerCatalog.PUT("/units/:unit_id/service-offerings/:service_offering_id", catalogHandler.SetServiceOfferingActive)
		ownerCatalog.POST("/units/:unit_id/spaces", catalogHandler.CreateSpace)
		ownerCatalog.PATCH("/units/:unit_id/spaces/:id", catalogHandler.UpdateSpace)
	}
	if deps.PeopleService != nil {
		peopleHandler := handlers.NewPeopleHandler(deps.PeopleService)
		protected.GET("/units/:unit_id/people", peopleHandler.List)
		protected.POST("/units/:unit_id/people/search", peopleHandler.Search)
		protected.GET("/units/:unit_id/people/:id", peopleHandler.Get)
		ownerPeople := protected.Group("")
		ownerPeople.Use(middleware.Owner())
		ownerPeople.POST("/units/:unit_id/people/duplicate-check", peopleHandler.DuplicateCheck)
		ownerPeople.POST("/units/:unit_id/people", peopleHandler.Create)
		ownerPeople.PATCH("/units/:unit_id/people/:id", peopleHandler.Update)
		ownerPeople.POST("/units/:unit_id/people/:id/link", peopleHandler.LinkUnit)
		ownerPeople.POST("/units/:unit_id/people/:id/external-identities", peopleHandler.LinkExternalIdentity)
		ownerPeople.POST("/units/:unit_id/people/:id/merge", peopleHandler.Merge)
		ownerPeople.GET("/units/:unit_id/people/identity-reviews", peopleHandler.ListIdentityReviews)
		ownerPeople.POST("/units/:unit_id/people/identity-reviews/:review_id/resolve", peopleHandler.ResolveIdentityReview)
	}
	if deps.MembershipService != nil {
		membershipHandler := handlers.NewMembershipHandler(deps.MembershipService)
		ownerMemberships := protected.Group("")
		ownerMemberships.Use(middleware.Owner())
		ownerMemberships.GET("/membership-plans", membershipHandler.List)
		ownerMemberships.POST("/membership-plans", membershipHandler.Create)
		ownerMemberships.GET("/membership-plans/:id", membershipHandler.Get)
		ownerMemberships.PATCH("/membership-plans/:id/draft", membershipHandler.UpdateDraft)
		ownerMemberships.POST("/membership-plans/:id/publish", membershipHandler.Publish)
		ownerMemberships.POST("/membership-plans/:id/versions", membershipHandler.NewVersion)
		ownerMemberships.POST("/membership-plans/:id/archive", membershipHandler.Archive)
		ownerMemberships.POST("/membership-plans/:id/reactivate", membershipHandler.Reactivate)
	}
	if deps.EnrollmentService != nil {
		enrollmentHandler := handlers.NewEnrollmentHandler(deps.EnrollmentService)
		ownerEnrollments := protected.Group("")
		ownerEnrollments.Use(middleware.Owner())
		ownerEnrollments.GET("/units/:unit_id/students/:student_id/enrollments", enrollmentHandler.ListForStudent)
		ownerEnrollments.POST("/units/:unit_id/students/:student_id/enrollments", enrollmentHandler.Create)
		ownerEnrollments.GET("/units/:unit_id/enrollments/roster", enrollmentHandler.Roster)
		ownerEnrollments.GET("/units/:unit_id/enrollments/:enrollment_id/access-status", enrollmentHandler.AccessStatus)
		ownerEnrollments.GET("/units/:unit_id/enrollments/:enrollment_id", enrollmentHandler.Get)
		ownerEnrollments.POST("/units/:unit_id/enrollments/:enrollment_id/end", enrollmentHandler.End)
	}
	if deps.MemberBillingService != nil {
		memberBillingHandler := handlers.NewMemberBillingHandler(deps.MemberBillingService)
		ownerMemberBilling := protected.Group("")
		ownerMemberBilling.Use(middleware.Owner())
		ownerMemberBilling.GET("/units/:unit_id/enrollments/:enrollment_id/member-billing", memberBillingHandler.Snapshot)
		ownerMemberBilling.POST("/units/:unit_id/enrollments/:enrollment_id/member-billing", memberBillingHandler.Start)
		ownerMemberBilling.POST("/units/:unit_id/member-billing/contracts/:contract_id/materialize", memberBillingHandler.Materialize)
		ownerMemberBilling.POST("/units/:unit_id/member-billing/charges/:charge_id/manual-payments", memberBillingHandler.Manual)
		ownerMemberBilling.POST("/units/:unit_id/member-billing/charges/:charge_id/reconcile", memberBillingHandler.Reconcile)
		ownerMemberBilling.POST("/units/:unit_id/member-billing/charges/:charge_id/refunds", memberBillingHandler.Refund)
	}
	if deps.ContractService != nil {
		contractHandler := handlers.NewContractHandler(deps.ContractService)
		api.GET("/public/contract-acceptance/:token", contractHandler.GetPublicAcceptanceSession)
		api.POST("/public/contract-acceptance/:token/accept", contractHandler.AcceptContract)

		ownerContracts := protected.Group("")
		ownerContracts.Use(middleware.Owner())
		ownerContracts.GET("/contract-templates", contractHandler.ListTemplates)
		ownerContracts.POST("/contract-templates", contractHandler.CreateTemplate)
		ownerContracts.GET("/contract-templates/:template_id", contractHandler.GetTemplate)
		ownerContracts.PATCH("/contract-templates/:template_id/draft", contractHandler.UpdateTemplateDraft)
		ownerContracts.PUT("/contract-templates/:template_id/draft", contractHandler.UpdateTemplateDraft)
		ownerContracts.POST("/contract-templates/:template_id/publish", contractHandler.PublishTemplate)
		ownerContracts.POST("/contract-templates/:template_id/versions", contractHandler.CreateNewVersion)
		ownerContracts.POST("/contract-templates/:template_id/archive", contractHandler.ArchiveTemplate)
		ownerContracts.POST("/contract-templates/:template_id/reactivate", contractHandler.ReactivateTemplate)

		ownerContracts.GET("/units/:unit_id/enrollments/:enrollment_id/contracts", contractHandler.ListForEnrollment)
		ownerContracts.POST("/units/:unit_id/enrollments/:enrollment_id/contracts/preview", contractHandler.PreviewContract)
		ownerContracts.POST("/units/:unit_id/enrollments/:enrollment_id/contracts", contractHandler.CreateInstanceDraft)
		ownerContracts.GET("/units/:unit_id/contracts/:contract_id", contractHandler.GetInstance)
		ownerContracts.POST("/units/:unit_id/contracts/:contract_id/issue", contractHandler.IssueInstance)
		ownerContracts.POST("/units/:unit_id/contracts/:contract_id/discard", contractHandler.DiscardInstance)
		ownerContracts.POST("/units/:unit_id/contracts/:contract_id/acceptance-sessions", contractHandler.CreateAcceptanceSession)
	}
	if deps.ContractLifecycleService != nil {
		handler := handlers.NewContractLifecycleHandler(deps.ContractLifecycleService, deps.ContractSignatureWebhookToken)
		api.POST("/webhooks/contracts/:provider", handler.ReconcileSignature)
		owner := protected.Group("")
		owner.Use(middleware.Owner())
		owner.GET("/units/:unit_id/contracts/:contract_id/signature-envelopes", handler.ListSignatureEnvelopes)
		owner.POST("/units/:unit_id/contracts/:contract_id/signature-envelopes", handler.CreateSignatureEnvelope)
		owner.GET("/units/:unit_id/enrollments/:enrollment_id/legal-actions", handler.ListLegalActions)
		owner.POST("/units/:unit_id/enrollments/:enrollment_id/legal-actions", handler.CreateLegalAction)
	}
	if deps.HealthFormService != nil {
		handler := handlers.NewHealthFormHandler(deps.HealthFormService)
		healthFormLimiter := middleware.NewWindowRateLimiter(20, time.Minute)
		api.GET("/public/health-forms/:token", handler.GetPublicSession)
		api.POST("/public/health-forms/:token", middleware.JSONRateLimit(healthFormLimiter, "token"), handler.SubmitPublic)
		owner := protected.Group("")
		owner.Use(middleware.Owner())
		owner.GET("/units/:unit_id/enrollments/:enrollment_id/health-forms", handler.List)
		owner.POST("/units/:unit_id/enrollments/:enrollment_id/health-forms", handler.Submit)
		owner.POST("/units/:unit_id/enrollments/:enrollment_id/health-form-sessions", handler.CreateSession)
		owner.GET("/units/:unit_id/health-forms/:form_id", handler.Get)
		owner.POST("/units/:unit_id/health-forms/:form_id/revoke-consent", handler.Revoke)
	}

	if deps.SchedulingService != nil {
		schedulingHandler := handlers.NewSchedulingHandler(deps.SchedulingService)
		protected.GET("/units/:unit_id/scheduling/professionals", schedulingHandler.List)
		ownerScheduling := protected.Group("")
		ownerScheduling.Use(middleware.Owner())
		ownerScheduling.GET("/units/:unit_id/scheduling/professional-account-candidates", schedulingHandler.ListCandidates)
		ownerScheduling.POST("/units/:unit_id/scheduling/professionals", schedulingHandler.Create)
		ownerScheduling.PATCH("/units/:unit_id/scheduling/professionals/:professional_id", schedulingHandler.Update)
		ownerScheduling.POST("/units/:unit_id/scheduling/professionals/:professional_id/archive", schedulingHandler.Archive)
		ownerScheduling.POST("/units/:unit_id/scheduling/professionals/:professional_id/reactivate", schedulingHandler.Reactivate)
		ownerScheduling.GET("/units/:unit_id/scheduling/professionals/:professional_id/privacy-export", schedulingHandler.PrivacyExport)
		ownerScheduling.POST("/units/:unit_id/scheduling/professionals/:professional_id/anonymize", schedulingHandler.Anonymize)
	}
	if deps.SchedulingGridService != nil {
		gridHandler := handlers.NewSchedulingGridHandler(deps.SchedulingGridService)
		protected.GET("/units/:unit_id/scheduling/sessions", gridHandler.ListSessions)
		protected.GET("/units/:unit_id/scheduling/sessions/:session_id", gridHandler.GetSession)
		ownerGrid := protected.Group("")
		ownerGrid.Use(middleware.Owner())
		ownerGrid.GET("/units/:unit_id/scheduling/series", gridHandler.ListSeries)
		ownerGrid.GET("/units/:unit_id/scheduling/series/:series_id", gridHandler.GetSeries)
		ownerGrid.GET("/units/:unit_id/scheduling/preview", gridHandler.PreviewGrid)
		ownerGrid.POST("/units/:unit_id/scheduling/series/preview", gridHandler.PreviewCandidate)
		ownerGrid.POST("/units/:unit_id/scheduling/series", gridHandler.CreateSeries)
		ownerGrid.PATCH("/units/:unit_id/scheduling/series/:series_id", gridHandler.UpdateSeries)
		ownerGrid.PATCH("/units/:unit_id/scheduling/series/:series_id/draft", gridHandler.UpdateSeries)
		ownerGrid.POST("/units/:unit_id/scheduling/series/:series_id/archive", gridHandler.ArchiveSeries)
		ownerGrid.POST("/units/:unit_id/scheduling/series/:series_id/reactivate", gridHandler.ReactivateSeries)
		ownerGrid.POST("/units/:unit_id/scheduling/series/:series_id/publish", gridHandler.PublishSeries)
		ownerGrid.POST("/units/:unit_id/scheduling/series/:series_id/changes/preview", gridHandler.PreviewSeriesChange)
		ownerGrid.POST("/units/:unit_id/scheduling/series/:series_id/changes", gridHandler.ChangeSeries)
		ownerGrid.POST("/units/:unit_id/scheduling/series/:series_id/cancel", gridHandler.CancelSeries)
		ownerGrid.POST("/units/:unit_id/scheduling/sessions", gridHandler.CreateSession)
		ownerGrid.POST("/units/:unit_id/scheduling/sessions/:session_id/change", gridHandler.ChangeSession)
		ownerGrid.POST("/units/:unit_id/scheduling/sessions/:session_id/cancel", gridHandler.CancelSession)
	}
	if deps.ReservationService != nil {
		reservationHandler := handlers.NewReservationHandler(deps.ReservationService)
		protected.GET("/units/:unit_id/reservations/summaries", reservationHandler.SessionSummaries)
		protected.POST("/units/:unit_id/reservations", reservationHandler.Book)
		protected.POST("/units/:unit_id/reservations/:reservation_id/cancel", reservationHandler.Cancel)
		protected.POST("/units/:unit_id/reservations/:reservation_id/no-show", reservationHandler.NoShow)
		protected.GET("/units/:unit_id/scheduling/sessions/:session_id/reservations", reservationHandler.ListSession)
		protected.GET("/units/:unit_id/people/:id/reservations", reservationHandler.ListPerson)
	}

	dashboardHandler := handlers.NewDashboardHandler(deps.DashboardSummaryUseCase, deps.DashboardActiveCampaignsUseCase, deps.DashboardNearGoalUseCase, deps.DashboardAtRiskUseCase, deps.ListPendingRewardDeliveriesUseCase, deps.DashboardTodayUseCase)
	protected.GET("/dashboard/today", dashboardHandler.Today)
	protected.GET("/dashboard/summary", dashboardHandler.Summary)
	protected.GET("/dashboard/active-campaigns", dashboardHandler.ActiveCampaigns)
	protected.GET("/dashboard/near-goal-students", dashboardHandler.NearGoalStudents)
	protected.GET("/dashboard/at-risk-students", dashboardHandler.AtRiskStudents)
	protected.GET("/dashboard/pending-rewards", dashboardHandler.PendingRewards)

	retentionHandler := handlers.NewRetentionHandler(deps.RetentionService)
	protected.GET("/retention/radar", retentionHandler.Radar)
	protected.GET("/retention/rules", retentionHandler.Rules)
	protected.GET("/retention/summary", retentionHandler.Summary)
	protected.GET("/retention/onboarding", retentionHandler.OnboardingJourney)
	protected.GET("/students/:id/retention-interventions", retentionHandler.ListInterventions)
	protected.POST("/students/:id/retention-interventions", retentionHandler.CreateIntervention)
	protected.PATCH("/retention-interventions/:id", retentionHandler.UpdateIntervention)
	protected.PATCH("/students/:id/membership-start", retentionHandler.UpdateMembershipStart)
	protected.PATCH("/students/:id/retention-monitoring", retentionHandler.UpdateMonitoring)

	studentsHandler := handlers.NewStudentsHandler(deps.ListStudentsUseCase, deps.GetStudentUseCase, deps.ListStudentCheckinsUseCase, deps.UpdateStudentRiskStatusUseCase, deps.ExportStudentDataUseCase, deps.UpdateContactPreferenceUseCase, deps.AnonymizeStudentUseCase)
	protected.GET("/students", studentsHandler.List)
	protected.GET("/students/:id", studentsHandler.Get)
	protected.PATCH("/students/:id/risk-status", studentsHandler.UpdateRiskStatus)
	protected.GET("/students/:id/checkins", studentsHandler.Checkins)
	if deps.AthleteService != nil {
		athleteHandler := handlers.NewAthleteHandler(deps.AthleteService, athleteSessionConfig(deps.SessionConfig))
		ownerAthletes := protected.Group("")
		ownerAthletes.Use(middleware.Owner())
		ownerAthletes.POST("/students/:id/athlete-invitations", athleteHandler.CreateInvitation)
	}
	if deps.AttendanceService != nil {
		attendanceHandler := handlers.NewAttendanceHandler(deps.AttendanceService)
		ownerAttendance := protected.Group("")
		ownerAttendance.Use(middleware.Owner())
		ownerAttendance.POST("/self-checkin-sessions", attendanceHandler.CreateSession)
		ownerAttendance.POST("/students/:id/checkins/manual", attendanceHandler.ManualCheckin)
	}
	protected.GET("/students/:id/campaign-progress", studentsHandler.CampaignProgress)
	protected.GET("/students/:id/privacy-export", studentsHandler.ExportData)
	protected.PATCH("/students/:id/contact-preference", studentsHandler.UpdateContactPreference)
	protected.POST("/students/:id/anonymize", studentsHandler.Anonymize)
	if deps.ContactActivationService != nil {
		activationHandler := handlers.NewContactActivationHandler(deps.ContactActivationService)
		protected.GET("/contact-activations/summary", activationHandler.Summary)
		protected.GET("/contact-activations", activationHandler.List)
		protected.POST("/students/:id/contact-activation", activationHandler.StartForStudent)
		protected.POST("/contact-activations/:id/resolve", activationHandler.Resolve)
		ownerActivation := protected.Group("")
		ownerActivation.Use(middleware.Owner())
		ownerActivation.POST("/contact-activations/:id/create-student", activationHandler.CreateStudentFromReview)
		ownerActivation.POST("/contact-activations/:id/cancel", activationHandler.CancelReview)
	}

	importsHandler := handlers.NewImportsHandler(deps.ImportCheckinsUseCase, deps.ListImportsUseCase, deps.GetImportUseCase)
	protected.POST("/imports", importsHandler.Create)
	protected.GET("/imports", importsHandler.List)
	protected.GET("/imports/:id", importsHandler.Get)
	if deps.CheckinIngestionService != nil {
		ingestionHandler := handlers.NewCheckinIngestionHandler(deps.CheckinIngestionService)
		protected.GET("/checkin-ingestion/sources", ingestionHandler.ListSources)
		protected.POST("/checkin-ingestion/sources", ingestionHandler.CreateSource)
		protected.PATCH("/checkin-ingestion/sources/:id", ingestionHandler.UpdateSource)
		protected.POST("/checkin-ingestion/sources/:id/rotate-token", ingestionHandler.RotateToken)
	}

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

func athleteSessionConfig(base middleware.SessionConfig) middleware.SessionConfig {
	base.CookieName = "engagefit_athlete_session"
	base.CSRFCookieName = "engagefit_athlete_csrf"
	base.MaxAgeSeconds = 30 * 24 * 60 * 60
	return base
}
