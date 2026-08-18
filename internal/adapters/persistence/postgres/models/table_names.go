package models

func (OrganizationModel) TableName() string {
	return "organizations"
}

func (BoxModel) TableName() string {
	return "boxes"
}

func (ModalityModel) TableName() string            { return "modalities" }
func (UnitModalityModel) TableName() string        { return "unit_modalities" }
func (ServiceOfferingModel) TableName() string     { return "service_offerings" }
func (UnitServiceOfferingModel) TableName() string { return "unit_service_offerings" }
func (SpaceModel) TableName() string               { return "spaces" }
func (CatalogAuditEventModel) TableName() string   { return "catalog_audit_events" }

func (MembershipPlanModel) TableName() string            { return "membership_plans" }
func (MembershipPlanVersionModel) TableName() string     { return "membership_plan_versions" }
func (MembershipPlanEntitlementModel) TableName() string { return "membership_plan_entitlements" }
func (MembershipPlanCommandRequestModel) TableName() string {
	return "membership_plan_command_requests"
}
func (MembershipPlanAuditEventModel) TableName() string { return "membership_plan_audit_events" }

func (BillingPlanModel) TableName() string         { return "billing_plans" }
func (BillingCustomerModel) TableName() string     { return "billing_customers" }
func (BillingSubscriptionModel) TableName() string { return "billing_subscriptions" }
func (BillingInvoiceModel) TableName() string      { return "billing_invoices" }
func (BillingWebhookEventModel) TableName() string { return "billing_webhook_events" }

func (UserModel) TableName() string {
	return "users"
}

func (StudentModel) TableName() string {
	return "students"
}

func (PersonModel) TableName() string                 { return "persons" }
func (PersonAddressModel) TableName() string          { return "person_addresses" }
func (PersonUnitRelationshipModel) TableName() string { return "person_unit_relationships" }
func (PersonCreationRequestModel) TableName() string  { return "person_creation_requests" }
func (PeopleAuditEventModel) TableName() string       { return "people_audit_events" }

func (ImportHistoryModel) TableName() string {
	return "import_histories"
}

func (CheckinModel) TableName() string {
	return "checkins"
}

func (SelfCheckinSessionModel) TableName() string {
	return "self_checkin_sessions"
}

func (RetentionInterventionModel) TableName() string {
	return "retention_interventions"
}

func (CheckinIngestionSourceModel) TableName() string {
	return "checkin_ingestion_sources"
}

func (CheckinIngestionBatchModel) TableName() string {
	return "checkin_ingestion_batches"
}

func (CampaignModel) TableName() string {
	return "campaigns"
}

func (CampaignGoalModel) TableName() string {
	return "campaign_goals"
}

func (CampaignProgressModel) TableName() string {
	return "campaign_progresses"
}

func (RewardModel) TableName() string {
	return "rewards"
}

func (RewardDeliveryModel) TableName() string {
	return "reward_deliveries"
}

func (WhatsappSettingsModel) TableName() string {
	return "whatsapp_settings"
}

func (MessageTemplateModel) TableName() string {
	return "message_templates"
}

func (MessageCampaignModel) TableName() string {
	return "message_campaigns"
}

func (MessageRecipientModel) TableName() string {
	return "message_recipients"
}

func (EmailSettingsModel) TableName() string {
	return "email_settings"
}

func (EmailTemplateModel) TableName() string {
	return "email_templates"
}

func (EmailCampaignModel) TableName() string {
	return "email_campaigns"
}

func (EmailRecipientModel) TableName() string {
	return "email_recipients"
}

func (AutomationRunModel) TableName() string {
	return "automation_runs"
}

func (AutomationScheduleModel) TableName() string {
	return "automation_schedules"
}

func (WorkoutModel) TableName() string {
	return "workouts"
}

func (WorkoutMessageDraftModel) TableName() string {
	return "workout_message_drafts"
}

func (WorkoutMessageRecipientModel) TableName() string {
	return "workout_message_recipients"
}

func (LLMGenerationLogModel) TableName() string {
	return "llm_generation_logs"
}
