package main

import (
	"log"

	"boxengage/backend/internal/adapters/http"
	"boxengage/backend/internal/adapters/parsers"
	"boxengage/backend/internal/adapters/persistence/postgres"
	pgrepo "boxengage/backend/internal/adapters/persistence/postgres/repositories"
	reportadapters "boxengage/backend/internal/adapters/reports"
	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/adapters/whatsapp"
	"boxengage/backend/internal/app/auth"
	"boxengage/backend/internal/app/boxes"
	"boxengage/backend/internal/app/campaigns"
	"boxengage/backend/internal/app/dashboard"
	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/app/messages"
	reportsapp "boxengage/backend/internal/app/reports"
	"boxengage/backend/internal/app/rewards"
	"boxengage/backend/internal/app/students"
	whatsappapp "boxengage/backend/internal/app/whatsapp"
	"boxengage/backend/internal/config"
)

func main() {
	cfg := config.Load()

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	boxRepository := pgrepo.NewBoxGormRepository(db)
	userRepository := pgrepo.NewUserGormRepository(db)
	studentRepository := pgrepo.NewStudentGormRepository(db)
	checkinRepository := pgrepo.NewCheckinGormRepository(db)
	importRepository := pgrepo.NewImportHistoryGormRepository(db)
	campaignRepository := pgrepo.NewCampaignGormRepository(db)
	rewardRepository := pgrepo.NewRewardGormRepository(db)
	whatsappSettingsRepository := pgrepo.NewWhatsappSettingsGormRepository(db)
	messageRepository := pgrepo.NewMessageGormRepository(db)

	passwordService := security.NewPasswordService()
	tokenService := security.NewJWTService(cfg.JWTSecret)
	metaCloudClient := whatsapp.NewMetaCloudClient()
	twilioClient := whatsapp.NewTwilioClient()
	providerGateway := whatsapp.NewProviderGateway(metaCloudClient, twilioClient)
	whatsappGateway := whatsapp.NewSafeGateway(providerGateway, cfg.AppEnv, cfg.WhatsappAllowRealSend, cfg.WhatsappDevRecipientPhone, cfg.WhatsappDevAllowedRecipientPhones)
	checkinParser := parsers.NewCheckinParser()

	loginUseCase := auth.NewLoginUseCase(userRepository, passwordService, tokenService)
	currentUserUseCase := auth.NewGetCurrentUserUseCase(userRepository)
	createBoxUseCase := boxes.NewCreateBoxUseCase(boxRepository, userRepository, passwordService)
	getBoxUseCase := boxes.NewGetBoxUseCase(boxRepository)
	updateBoxUseCase := boxes.NewUpdateBoxUseCase(boxRepository)

	listStudentsUseCase := students.NewListStudentsUseCase(studentRepository)
	getStudentUseCase := students.NewGetStudentUseCase(studentRepository)
	listStudentCheckinsUseCase := students.NewListStudentCheckinsUseCase(checkinRepository)
	updateStudentRiskStatusUseCase := students.NewUpdateStudentRiskStatusUseCase(studentRepository)

	listImportsUseCase := imports.NewListImportsUseCase(importRepository)
	getImportUseCase := imports.NewGetImportUseCase(importRepository)
	importCheckinsUseCase := imports.NewImportCheckinsUseCase(checkinParser, importRepository, studentRepository, checkinRepository, campaignRepository, rewardRepository)

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

	listRewardsUseCase := rewards.NewListRewardsUseCase(rewardRepository)
	createRewardUseCase := rewards.NewCreateRewardUseCase(rewardRepository)
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

	getWhatsappSettingsUseCase := whatsappapp.NewGetSettingsUseCase(whatsappSettingsRepository)
	updateWhatsappSettingsUseCase := whatsappapp.NewUpdateSettingsUseCase(whatsappSettingsRepository)
	testWhatsappSettingsUseCase := whatsappapp.NewTestSettingsUseCase(whatsappGateway)

	listMessageTemplatesUseCase := messages.NewListMessageTemplatesUseCase(messageRepository)
	createMessageTemplateUseCase := messages.NewCreateMessageTemplateUseCase(messageRepository)
	getMessageTemplateUseCase := messages.NewGetMessageTemplateUseCase(messageRepository)
	updateMessageTemplateUseCase := messages.NewUpdateMessageTemplateUseCase(messageRepository)
	deleteMessageTemplateUseCase := messages.NewDeleteMessageTemplateUseCase(messageRepository)
	listMessageCampaignsUseCase := messages.NewListMessageCampaignsUseCase(messageRepository)
	createMessageCampaignUseCase := messages.NewCreateMessageCampaignUseCase(messageRepository)
	getMessageCampaignUseCase := messages.NewGetMessageCampaignUseCase(messageRepository)
	sendMessageCampaignUseCase := messages.NewSendMessageCampaignUseCase(messageRepository, boxRepository, studentRepository, checkinRepository, campaignRepository, rewardRepository, whatsappSettingsRepository, whatsappGateway)
	listMessageRecipientsUseCase := messages.NewListMessageRecipientsUseCase(messageRepository)

	reportExporter := reportadapters.NewCSVExporter()
	eligibleStudentsReportUseCase := reportsapp.NewEligibleStudentsReportUseCase(campaignRepository)
	pendingRewardsReportUseCase := reportsapp.NewPendingRewardsReportUseCase(rewardRepository)
	monthlyFrequencyReportUseCase := reportsapp.NewMonthlyFrequencyReportUseCase(checkinRepository)

	router := http.NewRouter(http.RouterDependencies{
		TokenService:       tokenService,
		LoginUseCase:       loginUseCase,
		CurrentUserUseCase: currentUserUseCase,
		CreateBoxUseCase:   createBoxUseCase,
		GetBoxUseCase:      getBoxUseCase,
		UpdateBoxUseCase:   updateBoxUseCase,

		ListStudentsUseCase:            listStudentsUseCase,
		GetStudentUseCase:              getStudentUseCase,
		ListStudentCheckinsUseCase:     listStudentCheckinsUseCase,
		UpdateStudentRiskStatusUseCase: updateStudentRiskStatusUseCase,

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

		EligibleStudentsReportUseCase: eligibleStudentsReportUseCase,
		PendingRewardsReportUseCase:   pendingRewardsReportUseCase,
		MonthlyFrequencyReportUseCase: monthlyFrequencyReportUseCase,
		ReportExporter:                reportExporter,
	})

	if err := router.Run(cfg.HTTPAddress()); err != nil {
		log.Fatalf("failed to start api: %v", err)
	}
}
