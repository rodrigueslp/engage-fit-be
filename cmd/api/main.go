package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boxengage/backend/internal/adapters/email"
	apphttp "boxengage/backend/internal/adapters/http"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/adapters/llm"
	"boxengage/backend/internal/adapters/parsers"
	"boxengage/backend/internal/adapters/persistence/postgres"
	pgrepo "boxengage/backend/internal/adapters/persistence/postgres/repositories"
	reportadapters "boxengage/backend/internal/adapters/reports"
	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/adapters/whatsapp"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/automation"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/dashboard"
	emailapp "boxengage/backend/internal/app/email"
	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/app/messages"
	messagingapp "boxengage/backend/internal/app/messaging"
	"boxengage/backend/internal/app/platformadmin"
	reportsapp "boxengage/backend/internal/app/reports"
	"boxengage/backend/internal/app/rewards"
	"boxengage/backend/internal/app/students"
	whatsappapp "boxengage/backend/internal/app/whatsapp"
	"boxengage/backend/internal/app/workouts"
	"boxengage/backend/internal/config"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/observability"
	portservices "boxengage/backend/internal/ports/services"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	if cfg.BuildVersion != "" {
		version = cfg.BuildVersion
	}
	if cfg.BuildCommit != "" {
		commit = cfg.BuildCommit
	}
	if cfg.BuildTime != "" {
		buildTime = cfg.BuildTime
	}
	if cfg.OTelServiceVersion == "dev" && version != "dev" {
		cfg.OTelServiceVersion = version
	}
	var secretCipher portservices.SecretCipher
	encryptionDisabled := cfg.DataEncryptionActiveKeyID == "" || cfg.DataEncryptionKeys == ""
	if encryptionDisabled {
		secretCipher = security.NewPlaintextSecretCipher()
	} else {
		configuredCipher, cipherErr := security.NewSecretCipher(cfg.DataEncryptionActiveKeyID, cfg.DataEncryptionKeys)
		if cipherErr != nil {
			log.Fatalf("invalid data encryption configuration: %v", cipherErr)
		}
		secretCipher = configuredCipher
	}
	telemetry, err := observability.Setup(context.Background(), observability.Config{
		Enabled:          cfg.OTelEnabled,
		ServiceName:      cfg.OTelServiceName,
		ServiceVersion:   cfg.OTelServiceVersion,
		Environment:      cfg.AppEnv,
		TraceSampleRatio: cfg.OTelTraceSampleRatio,
		Prometheus:       cfg.PrometheusEnabled,
	})
	if err != nil {
		log.Fatalf("failed to configure observability: %v", err)
	}
	if encryptionDisabled {
		slog.Warn("data_encryption_disabled", "reason", "DATA_ENCRYPTION_ACTIVE_KEY_ID or DATA_ENCRYPTION_KEYS is empty")
	}

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	sqlDB, err := postgres.ConfigurePool(db, postgres.PoolConfig{
		MaxOpenConnections: cfg.DBMaxOpenConnections,
		MaxIdleConnections: cfg.DBMaxIdleConnections,
		MaxLifetime:        time.Duration(cfg.DBConnectionMaxLifetimeSeconds) * time.Second,
		MaxIdleTime:        time.Duration(cfg.DBConnectionMaxIdleTimeSeconds) * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to configure database pool: %v", err)
	}
	defer sqlDB.Close()
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
	if err := sqlDB.PingContext(startupContext); err != nil {
		cancelStartup()
		log.Fatalf("failed to ping database: %v", err)
	}
	cancelStartup()

	boxRepository := pgrepo.NewBoxGormRepository(db)
	userRepository := pgrepo.NewUserGormRepository(db)
	studentRepository := pgrepo.NewStudentGormRepository(db)
	privacyRepository := pgrepo.NewPrivacyGormRepository(db)
	checkinRepository := pgrepo.NewCheckinGormRepository(db)
	importRepository := pgrepo.NewImportHistoryGormRepository(db)
	campaignRepository := pgrepo.NewCampaignGormRepository(db)
	rewardRepository := pgrepo.NewRewardGormRepository(db)
	whatsappSettingsRepository := pgrepo.NewWhatsappSettingsGormRepository(db, secretCipher)
	messageRepository := pgrepo.NewMessageGormRepository(db)
	emailSettingsRepository := pgrepo.NewEmailSettingsGormRepository(db, secretCipher)
	emailRepository := pgrepo.NewEmailGormRepository(db)
	automationRepository := pgrepo.NewAutomationGormRepository(db)
	workoutRepository := pgrepo.NewWorkoutGormRepository(db)
	messagingGovernanceRepository := pgrepo.NewMessagingGovernanceGormRepository(db)

	passwordService := security.NewPasswordService()
	ensureAdminUseCase := platformadmin.NewEnsureAdminUseCase(userRepository, passwordService)
	if err := ensureAdminUseCase.Execute(context.Background(), cfg.PlatformAdminName, cfg.PlatformAdminEmail, cfg.PlatformAdminPassword); err != nil {
		log.Fatalf("failed to ensure platform admin: %v", err)
	}
	tokenService := security.NewJWTService(cfg.JWTSecret)
	metaCloudClient := whatsapp.NewMetaCloudClient()
	twilioClient := whatsapp.NewTwilioClient()
	providerGateway := whatsapp.NewProviderGateway(metaCloudClient, twilioClient)
	whatsappGateway := whatsapp.NewSafeGateway(providerGateway, cfg.AppEnv, cfg.WhatsappAllowRealSend, cfg.WhatsappDevRecipientPhone, cfg.WhatsappDevAllowedRecipientPhones)
	platformWhatsappSettings := domain.WhatsappSettings{
		ConnectionMode:  domain.WhatsappConnectionPlatform,
		Provider:        "twilio",
		BaseURL:         cfg.WhatsappPlatformBaseURL,
		InstanceName:    cfg.WhatsappPlatformSender,
		APIKeyEncrypted: cfg.WhatsappPlatformAccountSID + ":" + cfg.WhatsappPlatformAuthToken,
		Enabled:         cfg.WhatsappPlatformEnabled,
		ContentSIDs: map[domain.MessageTemplateType]string{
			domain.MessageTemplateAlmostThere: cfg.WhatsappPlatformAlmostThereSID,
			domain.MessageTemplateGoalReached: cfg.WhatsappPlatformGoalReachedSID,
			domain.MessageTemplateWeMissYou:   cfg.WhatsappPlatformWeMissYouSID,
		},
	}
	whatsappSettingsResolver := whatsappapp.NewSettingsResolver(whatsappSettingsRepository, platformWhatsappSettings)
	messagingGovernanceService := messagingapp.NewGovernanceService(messagingGovernanceRepository)
	messagingAdminUseCases := platformadmin.NewMessagingAdminUseCases(boxRepository, whatsappSettingsResolver, messagingGovernanceRepository)
	tenantMessagingUsageUseCase := platformadmin.NewGetTenantMessagingUsageUseCase(messagingGovernanceRepository)
	emailGateway := email.NewSMTPGateway(cfg.AppEnv, cfg.EmailAllowRealSend, cfg.EmailDevRecipientEmail)
	llmGenerator := llm.NewOpenAIGenerator(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAITimeoutSeconds)
	checkinParser := parsers.NewCheckinParser()

	loginUseCase := auth.NewLoginUseCase(userRepository, boxRepository, passwordService, tokenService)
	currentUserUseCase := auth.NewGetCurrentUserUseCase(userRepository)
	changePasswordUseCase := auth.NewChangePasswordUseCase(userRepository, passwordService)
	logoutUseCase := auth.NewLogoutUseCase(userRepository)
	resetOwnerPasswordUseCase := platformadmin.NewResetOwnerPasswordUseCase(userRepository, passwordService, messagingGovernanceRepository)
	createBoxUseCase := boxes.NewCreateBoxUseCase(boxRepository, userRepository, passwordService)
	boxAdminUseCases := platformadmin.NewBoxAdminUseCases(boxRepository, userRepository, createBoxUseCase, messagingGovernanceRepository)
	getBoxUseCase := boxes.NewGetBoxUseCase(boxRepository)
	updateBoxUseCase := boxes.NewUpdateBoxUseCase(boxRepository)

	listStudentsUseCase := students.NewListStudentsUseCase(studentRepository)
	getStudentUseCase := students.NewGetStudentUseCase(studentRepository)
	listStudentCheckinsUseCase := students.NewListStudentCheckinsUseCase(checkinRepository)
	updateStudentRiskStatusUseCase := students.NewUpdateStudentRiskStatusUseCase(studentRepository)
	exportStudentDataUseCase := students.NewExportStudentDataUseCase(privacyRepository)
	updateContactPreferenceUseCase := students.NewUpdateContactPreferenceUseCase(studentRepository, privacyRepository)
	anonymizeStudentUseCase := students.NewAnonymizeStudentUseCase(privacyRepository)

	listImportsUseCase := imports.NewListImportsUseCase(importRepository)
	getImportUseCase := imports.NewGetImportUseCase(importRepository)
	importCheckinsUseCase := imports.NewImportCheckinsUseCase(checkinParser, importRepository, studentRepository, checkinRepository, campaignRepository, rewardRepository, privacyRepository)

	listCampaignsUseCase := campaigns.NewListCampaignsUseCase(campaignRepository)
	createCampaignUseCase := campaigns.NewCreateCampaignUseCase(campaignRepository)
	getCampaignUseCase := campaigns.NewGetCampaignUseCase(campaignRepository)
	updateCampaignUseCase := campaigns.NewUpdateCampaignUseCase(campaignRepository)
	closeCampaignUseCase := campaigns.NewCloseCampaignUseCase(campaignRepository)
	deleteCampaignUseCase := campaigns.NewDeleteCampaignUseCase(campaignRepository)
	listCampaignGoalsUseCase := campaigns.NewListCampaignGoalsUseCase(campaignRepository)
	upsertCampaignGoalUseCase := campaigns.NewUpsertCampaignGoalUseCase(campaignRepository)
	deleteCampaignGoalUseCase := campaigns.NewDeleteCampaignGoalUseCase(campaignRepository)
	listCampaignProgressUseCase := campaigns.NewListCampaignProgressUseCase(campaignRepository)
	recalculateCampaignProgressUseCase := campaigns.NewRecalculateCampaignProgressUseCase(campaignRepository, studentRepository, checkinRepository, rewardRepository)

	listRewardsUseCase := rewards.NewListRewardsUseCase(rewardRepository, campaignRepository)
	createRewardUseCase := rewards.NewCreateRewardUseCase(rewardRepository, campaignRepository)
	getRewardUseCase := rewards.NewGetRewardUseCase(rewardRepository)
	updateRewardUseCase := rewards.NewUpdateRewardUseCase(rewardRepository)
	deleteRewardUseCase := rewards.NewDeleteRewardUseCase(rewardRepository)
	listRewardDeliveriesUseCase := rewards.NewListRewardDeliveriesUseCase(rewardRepository)
	listPendingRewardDeliveriesUseCase := rewards.NewListPendingRewardDeliveriesUseCase(rewardRepository)
	markRewardDeliveredUseCase := rewards.NewMarkRewardDeliveredUseCase(rewardRepository)

	dashboardSummaryUseCase := dashboard.NewGetDashboardSummaryUseCase(boxRepository, studentRepository, checkinRepository, campaignRepository, rewardRepository)
	dashboardActiveCampaignsUseCase := dashboard.NewListActiveCampaignsUseCase(campaignRepository)
	dashboardNearGoalUseCase := dashboard.NewListNearGoalStudentsUseCase(studentRepository, campaignRepository)
	dashboardAtRiskUseCase := dashboard.NewListAtRiskStudentsUseCase(boxRepository, studentRepository, checkinRepository)

	getWhatsappSettingsUseCase := whatsappapp.NewGetSettingsUseCase(whatsappSettingsRepository, whatsappSettingsResolver)
	updateWhatsappSettingsUseCase := whatsappapp.NewUpdateSettingsUseCase(whatsappSettingsRepository)
	testWhatsappSettingsUseCase := whatsappapp.NewTestSettingsUseCase(whatsappGateway, whatsappSettingsResolver)

	listMessageTemplatesUseCase := messages.NewListMessageTemplatesUseCase(messageRepository)
	createMessageTemplateUseCase := messages.NewCreateMessageTemplateUseCase(messageRepository)
	getMessageTemplateUseCase := messages.NewGetMessageTemplateUseCase(messageRepository)
	updateMessageTemplateUseCase := messages.NewUpdateMessageTemplateUseCase(messageRepository)
	deleteMessageTemplateUseCase := messages.NewDeleteMessageTemplateUseCase(messageRepository)
	listMessageCampaignsUseCase := messages.NewListMessageCampaignsUseCase(messageRepository)
	createMessageCampaignUseCase := messages.NewCreateMessageCampaignUseCase(messageRepository)
	getMessageCampaignUseCase := messages.NewGetMessageCampaignUseCase(messageRepository)
	sendMessageCampaignUseCase := messages.NewSendMessageCampaignUseCase(messageRepository, boxRepository, studentRepository, checkinRepository, campaignRepository, rewardRepository, whatsappSettingsResolver, whatsappGateway, messagingGovernanceService)
	listMessageRecipientsUseCase := messages.NewListMessageRecipientsUseCase(messageRepository)

	getEmailSettingsUseCase := emailapp.NewGetSettingsUseCase(emailSettingsRepository)
	updateEmailSettingsUseCase := emailapp.NewUpdateSettingsUseCase(emailSettingsRepository)
	testEmailSettingsUseCase := emailapp.NewTestSettingsUseCase(emailGateway)
	listEmailTemplatesUseCase := emailapp.NewListEmailTemplatesUseCase(emailRepository)
	createEmailTemplateUseCase := emailapp.NewCreateEmailTemplateUseCase(emailRepository)
	getEmailTemplateUseCase := emailapp.NewGetEmailTemplateUseCase(emailRepository)
	updateEmailTemplateUseCase := emailapp.NewUpdateEmailTemplateUseCase(emailRepository)
	deleteEmailTemplateUseCase := emailapp.NewDeleteEmailTemplateUseCase(emailRepository)
	listEmailCampaignsUseCase := emailapp.NewListEmailCampaignsUseCase(emailRepository)
	createEmailCampaignUseCase := emailapp.NewCreateEmailCampaignUseCase(emailRepository)
	getEmailCampaignUseCase := emailapp.NewGetEmailCampaignUseCase(emailRepository)
	sendEmailCampaignUseCase := emailapp.NewSendEmailCampaignUseCase(emailRepository, boxRepository, studentRepository, checkinRepository, campaignRepository, rewardRepository, emailSettingsRepository, emailGateway)
	listEmailRecipientsUseCase := emailapp.NewListEmailRecipientsUseCase(emailRepository)

	listAutomationRunsUseCase := automation.NewListRunsUseCase(automationRepository)
	getAutomationRunUseCase := automation.NewGetRunUseCase(automationRepository)
	createAutomationRunUseCase := automation.NewCreateRunUseCase(automationRepository)
	updateAutomationRunUseCase := automation.NewUpdateRunUseCase(automationRepository)
	listAutomationSchedulesUseCase := automation.NewListSchedulesUseCase(automationRepository)
	getAutomationScheduleUseCase := automation.NewGetScheduleUseCase(automationRepository)
	createAutomationScheduleUseCase := automation.NewCreateScheduleUseCase(automationRepository)
	updateAutomationScheduleUseCase := automation.NewUpdateScheduleUseCase(automationRepository)
	deleteAutomationScheduleUseCase := automation.NewDeleteScheduleUseCase(automationRepository)
	executeAutomationScheduleUseCase := automation.NewExecuteScheduleUseCase(automationRepository, campaignRepository, messageRepository, recalculateCampaignProgressUseCase, sendMessageCampaignUseCase, time.Duration(cfg.AutomationStaleRunMinutes)*time.Minute, time.Duration(cfg.AutomationCatchupWindowMinutes)*time.Minute)

	listWorkoutsUseCase := workouts.NewListWorkoutsUseCase(workoutRepository)
	createWorkoutUseCase := workouts.NewCreateWorkoutUseCase(workoutRepository)
	getWorkoutUseCase := workouts.NewGetWorkoutUseCase(workoutRepository)
	updateWorkoutUseCase := workouts.NewUpdateWorkoutUseCase(workoutRepository)
	deleteWorkoutUseCase := workouts.NewDeleteWorkoutUseCase(workoutRepository)
	listWorkoutDraftsUseCase := workouts.NewListWorkoutDraftsUseCase(workoutRepository)
	generateWorkoutDraftUseCase := workouts.NewGenerateWorkoutDraftUseCase(workoutRepository, boxRepository, studentRepository, checkinRepository, campaignRepository, llmGenerator)
	getWorkoutDraftUseCase := workouts.NewGetWorkoutDraftUseCase(workoutRepository)
	updateWorkoutDraftUseCase := workouts.NewUpdateWorkoutDraftUseCase(workoutRepository)
	approveWorkoutDraftUseCase := workouts.NewApproveWorkoutDraftUseCase(workoutRepository)
	sendWorkoutDraftUseCase := workouts.NewSendWorkoutDraftUseCase(workoutRepository, boxRepository, studentRepository, checkinRepository, campaignRepository, whatsappSettingsResolver, whatsappGateway, messagingGovernanceService)
	listWorkoutRecipientsUseCase := workouts.NewListWorkoutRecipientsUseCase(workoutRepository)

	reportExporter := reportadapters.NewCSVExporter()
	eligibleStudentsReportUseCase := reportsapp.NewEligibleStudentsReportUseCase(campaignRepository)
	pendingRewardsReportUseCase := reportsapp.NewPendingRewardsReportUseCase(rewardRepository)
	monthlyFrequencyReportUseCase := reportsapp.NewMonthlyFrequencyReportUseCase(checkinRepository)

	router := apphttp.NewRouter(apphttp.RouterDependencies{
		AppEnv:                    cfg.AppEnv,
		TokenService:              tokenService,
		UserRepository:            userRepository,
		BoxRepository:             boxRepository,
		LoginUseCase:              loginUseCase,
		CurrentUserUseCase:        currentUserUseCase,
		ChangePasswordUseCase:     changePasswordUseCase,
		LogoutUseCase:             logoutUseCase,
		ResetOwnerPasswordUseCase: resetOwnerPasswordUseCase,
		BoxAdminUseCases:          boxAdminUseCases,
		CreateBoxUseCase:          createBoxUseCase,
		GetBoxUseCase:             getBoxUseCase,
		UpdateBoxUseCase:          updateBoxUseCase,

		ListStudentsUseCase:            listStudentsUseCase,
		GetStudentUseCase:              getStudentUseCase,
		ListStudentCheckinsUseCase:     listStudentCheckinsUseCase,
		UpdateStudentRiskStatusUseCase: updateStudentRiskStatusUseCase,
		ExportStudentDataUseCase:       exportStudentDataUseCase,
		UpdateContactPreferenceUseCase: updateContactPreferenceUseCase,
		AnonymizeStudentUseCase:        anonymizeStudentUseCase,

		ListImportsUseCase:    listImportsUseCase,
		GetImportUseCase:      getImportUseCase,
		ImportCheckinsUseCase: importCheckinsUseCase,

		ListCampaignsUseCase:               listCampaignsUseCase,
		CreateCampaignUseCase:              createCampaignUseCase,
		GetCampaignUseCase:                 getCampaignUseCase,
		UpdateCampaignUseCase:              updateCampaignUseCase,
		CloseCampaignUseCase:               closeCampaignUseCase,
		DeleteCampaignUseCase:              deleteCampaignUseCase,
		ListCampaignGoalsUseCase:           listCampaignGoalsUseCase,
		UpsertCampaignGoalUseCase:          upsertCampaignGoalUseCase,
		DeleteCampaignGoalUseCase:          deleteCampaignGoalUseCase,
		ListCampaignProgressUseCase:        listCampaignProgressUseCase,
		RecalculateCampaignProgressUseCase: recalculateCampaignProgressUseCase,

		ListRewardsUseCase:                 listRewardsUseCase,
		CreateRewardUseCase:                createRewardUseCase,
		GetRewardUseCase:                   getRewardUseCase,
		UpdateRewardUseCase:                updateRewardUseCase,
		DeleteRewardUseCase:                deleteRewardUseCase,
		ListRewardDeliveriesUseCase:        listRewardDeliveriesUseCase,
		ListPendingRewardDeliveriesUseCase: listPendingRewardDeliveriesUseCase,
		MarkRewardDeliveredUseCase:         markRewardDeliveredUseCase,

		DashboardSummaryUseCase:         dashboardSummaryUseCase,
		DashboardActiveCampaignsUseCase: dashboardActiveCampaignsUseCase,
		DashboardNearGoalUseCase:        dashboardNearGoalUseCase,
		DashboardAtRiskUseCase:          dashboardAtRiskUseCase,

		GetWhatsappSettingsUseCase:    getWhatsappSettingsUseCase,
		UpdateWhatsappSettingsUseCase: updateWhatsappSettingsUseCase,
		TestWhatsappSettingsUseCase:   testWhatsappSettingsUseCase,

		ListMessageTemplatesUseCase:  listMessageTemplatesUseCase,
		CreateMessageTemplateUseCase: createMessageTemplateUseCase,
		GetMessageTemplateUseCase:    getMessageTemplateUseCase,
		UpdateMessageTemplateUseCase: updateMessageTemplateUseCase,
		DeleteMessageTemplateUseCase: deleteMessageTemplateUseCase,
		ListMessageCampaignsUseCase:  listMessageCampaignsUseCase,
		CreateMessageCampaignUseCase: createMessageCampaignUseCase,
		GetMessageCampaignUseCase:    getMessageCampaignUseCase,
		SendMessageCampaignUseCase:   sendMessageCampaignUseCase,
		ListMessageRecipientsUseCase: listMessageRecipientsUseCase,

		GetEmailSettingsUseCase:    getEmailSettingsUseCase,
		UpdateEmailSettingsUseCase: updateEmailSettingsUseCase,
		TestEmailSettingsUseCase:   testEmailSettingsUseCase,
		ListEmailTemplatesUseCase:  listEmailTemplatesUseCase,
		CreateEmailTemplateUseCase: createEmailTemplateUseCase,
		GetEmailTemplateUseCase:    getEmailTemplateUseCase,
		UpdateEmailTemplateUseCase: updateEmailTemplateUseCase,
		DeleteEmailTemplateUseCase: deleteEmailTemplateUseCase,
		ListEmailCampaignsUseCase:  listEmailCampaignsUseCase,
		CreateEmailCampaignUseCase: createEmailCampaignUseCase,
		GetEmailCampaignUseCase:    getEmailCampaignUseCase,
		SendEmailCampaignUseCase:   sendEmailCampaignUseCase,
		ListEmailRecipientsUseCase: listEmailRecipientsUseCase,

		ListAutomationRunsUseCase:        listAutomationRunsUseCase,
		GetAutomationRunUseCase:          getAutomationRunUseCase,
		CreateAutomationRunUseCase:       createAutomationRunUseCase,
		UpdateAutomationRunUseCase:       updateAutomationRunUseCase,
		ListAutomationSchedulesUseCase:   listAutomationSchedulesUseCase,
		GetAutomationScheduleUseCase:     getAutomationScheduleUseCase,
		CreateAutomationScheduleUseCase:  createAutomationScheduleUseCase,
		UpdateAutomationScheduleUseCase:  updateAutomationScheduleUseCase,
		DeleteAutomationScheduleUseCase:  deleteAutomationScheduleUseCase,
		ExecuteAutomationScheduleUseCase: executeAutomationScheduleUseCase,

		ListWorkoutsUseCase:          listWorkoutsUseCase,
		CreateWorkoutUseCase:         createWorkoutUseCase,
		GetWorkoutUseCase:            getWorkoutUseCase,
		UpdateWorkoutUseCase:         updateWorkoutUseCase,
		DeleteWorkoutUseCase:         deleteWorkoutUseCase,
		ListWorkoutDraftsUseCase:     listWorkoutDraftsUseCase,
		GenerateWorkoutDraftUseCase:  generateWorkoutDraftUseCase,
		GetWorkoutDraftUseCase:       getWorkoutDraftUseCase,
		UpdateWorkoutDraftUseCase:    updateWorkoutDraftUseCase,
		ApproveWorkoutDraftUseCase:   approveWorkoutDraftUseCase,
		SendWorkoutDraftUseCase:      sendWorkoutDraftUseCase,
		ListWorkoutRecipientsUseCase: listWorkoutRecipientsUseCase,

		EligibleStudentsReportUseCase: eligibleStudentsReportUseCase,
		PendingRewardsReportUseCase:   pendingRewardsReportUseCase,
		MonthlyFrequencyReportUseCase: monthlyFrequencyReportUseCase,
		ReportExporter:                reportExporter,
		MessagingAdminUseCases:        messagingAdminUseCases,
		TenantMessagingUsageUseCase:   tenantMessagingUsageUseCase,
		OwnerSetupEnabled:             cfg.OwnerSetupEnabled,
		OwnerSetupToken:               cfg.OwnerSetupToken,
		HTTPMaxBodyBytes:              int64(cfg.HTTPMaxBodyBytes),
		ImportMaxUploadBytes:          int64(cfg.ImportMaxUploadBytes),
		LoginRateLimitRequests:        cfg.LoginRateLimitRequests,
		LoginRateLimitWindowSeconds:   cfg.LoginRateLimitWindowSeconds,
		SetupRateLimitRequests:        cfg.SetupRateLimitRequests,
		SetupRateLimitWindowSeconds:   cfg.SetupRateLimitWindowSeconds,
		TrustedProxies:                cfg.TrustedProxyList(),
		ReadinessCheck: func(ctx context.Context) error {
			checkContext, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			return sqlDB.PingContext(checkContext)
		},
		MetricsHandler:       telemetry.MetricsHandler,
		MetricsBearerToken:   cfg.PrometheusBearerToken,
		ObservabilityService: cfg.OTelServiceName,
		SessionConfig: middleware.SessionConfig{
			CookieName: cfg.AuthCookieName, CSRFCookieName: cfg.AuthCookieName + "_csrf",
			Secure: cfg.AuthCookieSecure, SameSite: middleware.ParseSameSite(cfg.AuthCookieSameSite),
			MaxAgeSeconds: cfg.AuthSessionMaxAgeSeconds,
		},
		CORSAllowedOrigins: cfg.CORSAllowedOriginList(),
		Capabilities: middleware.Capabilities{
			Whatsapp: cfg.FeatureWhatsappEnabled, Email: cfg.FeatureEmailEnabled,
			Automation: cfg.FeatureAutomationEnabled, Workouts: cfg.FeatureWorkoutsEnabled, LLM: cfg.FeatureLLMEnabled,
		},
		BuildVersion: version,
		BuildCommit:  commit,
		BuildTime:    buildTime,
	})

	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var workerDone <-chan struct{}
	if cfg.AutomationWorkerEnabled {
		interval := time.Duration(cfg.AutomationWorkerIntervalSeconds) * time.Second
		workerDone = automation.NewWorker(executeAutomationScheduleUseCase, interval).Start(runContext)
		log.Printf("automation worker enabled with interval %s", interval)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           router,
		ReadHeaderTimeout: time.Duration(cfg.HTTPReadHeaderTimeoutSeconds) * time.Second,
		ReadTimeout:       time.Duration(cfg.HTTPReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.HTTPWriteTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.HTTPIdleTimeoutSeconds) * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http_server_failed", "error", err)
		}
		stop()
	case <-runContext.Done():
		slog.Info("shutdown_started", "reason", runContext.Err())
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), time.Duration(cfg.HTTPShutdownTimeoutSeconds)*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		slog.Error("http_server_shutdown_failed", "error", err)
		_ = server.Close()
	}
	if workerDone != nil {
		select {
		case <-workerDone:
		case <-shutdownContext.Done():
			slog.Warn("automation_worker_shutdown_timed_out")
		}
	}
	slog.Info("shutdown_completed")
	if err := telemetry.Shutdown(shutdownContext); err != nil {
		slog.Error("observability_shutdown_failed", "error", err)
	}
}
