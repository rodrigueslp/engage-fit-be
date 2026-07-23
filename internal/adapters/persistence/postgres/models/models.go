package models

import "time"

type BoxModel struct {
	ID                      string `gorm:"primaryKey"`
	Name                    string
	Status                  string `gorm:"default:active"`
	StatusReason            string
	StatusChangedAt         *time.Time
	StatusChangedBy         *string
	BillingAccessBlocked    bool
	BillingAccessReason     string
	BillingAccessChangedAt  *time.Time
	RiskInactiveDays        int
	RiskMessageCooldownDays int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type BillingPlanModel struct {
	ID                  string `gorm:"primaryKey"`
	Code                string
	Version             int
	Name                string
	Description         string
	MonthlyPriceCents   int64
	Currency            string
	MonthlyMessageLimit int
	DailyMessageLimit   int
	PerDispatchLimit    int
	WarningPercent      int
	GracePeriodDays     int
	Active              bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type BillingCustomerModel struct {
	ID                   string `gorm:"primaryKey"`
	BoxID                string `gorm:"uniqueIndex"`
	Provider             string
	ProviderCustomerID   string
	LegalName            string
	CPFCNPJ              string `gorm:"column:cpf_cnpj"`
	Email                string
	Phone                string
	PostalCode           string
	Address              string
	AddressNumber        string
	Complement           string
	Province             string
	City                 string
	State                string
	NotificationDisabled bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type BillingSubscriptionModel struct {
	ID                     string `gorm:"primaryKey"`
	BoxID                  string `gorm:"index"`
	BillingCustomerID      string
	PlanID                 string
	Provider               string
	ProviderSubscriptionID string
	Status                 string `gorm:"index"`
	BillingType            string
	NextDueDate            time.Time
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	GraceUntil             *time.Time
	StartedAt              time.Time
	CanceledAt             *time.Time
	CancelAtPeriodEnd      bool
	ExternalReference      string
	LastReconciledAt       *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type BillingInvoiceModel struct {
	ID                string `gorm:"primaryKey"`
	BoxID             string `gorm:"index"`
	SubscriptionID    string `gorm:"index"`
	Provider          string
	ProviderPaymentID string `gorm:"uniqueIndex"`
	Status            string `gorm:"index"`
	BillingType       string
	ValueCents        int64
	NetValueCents     *int64
	DueDate           time.Time
	OriginalDueDate   *time.Time
	ConfirmedAt       *time.Time
	ReceivedAt        *time.Time
	InvoiceURL        string
	BankSlipURL       string
	ExternalReference string
	Description       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type BillingWebhookEventModel struct {
	ID                string `gorm:"primaryKey"`
	Provider          string
	ProviderEventID   string
	EventType         string
	ProviderPaymentID string
	Payload           []byte `gorm:"type:jsonb"`
	Status            string `gorm:"index"`
	Attempts          int
	ErrorMessage      string
	ReceivedAt        time.Time
	ProcessedAt       *time.Time
}

type UserModel struct {
	ID           string  `gorm:"primaryKey"`
	BoxID        *string `gorm:"index"`
	Name         string
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	AuthVersion  int
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type StudentModel struct {
	ID                     string `gorm:"primaryKey"`
	BoxID                  string `gorm:"index"`
	Name                   string
	Email                  string
	Phone                  string
	Source                 string `gorm:"index"`
	ExternalID             string `gorm:"index"`
	RiskStatus             string
	RiskLastMessageAt      *time.Time
	ContactStatus          string
	ContactStatusUpdatedAt *time.Time
	ContactStatusSource    string
	AnonymizedAt           *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ImportHistoryModel struct {
	ID           string `gorm:"primaryKey"`
	BoxID        string `gorm:"index"`
	Filename     string
	Source       string
	TotalRecords int
	ImportedAt   time.Time
}

type CheckinModel struct {
	ID              string `gorm:"primaryKey"`
	BoxID           string `gorm:"index"`
	StudentID       string `gorm:"index"`
	CheckinDate     time.Time
	CheckinTime     *string
	Source          string `gorm:"index"`
	ImportHistoryID string
	CreatedAt       time.Time
}

type CampaignModel struct {
	ID          string `gorm:"primaryKey"`
	BoxID       string `gorm:"index"`
	Name        string
	Description string
	StartDate   time.Time
	EndDate     time.Time
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CampaignGoalModel struct {
	ID             string `gorm:"primaryKey"`
	CampaignID     string `gorm:"index:idx_campaign_goal_source,unique"`
	Source         string `gorm:"index:idx_campaign_goal_source,unique"`
	TargetCheckins int
}

type CampaignProgressModel struct {
	ID                 string `gorm:"primaryKey"`
	CampaignID         string `gorm:"index:idx_campaign_student,unique"`
	StudentID          string `gorm:"index:idx_campaign_student,unique"`
	CurrentCheckins    int
	TargetCheckins     int
	ProgressPercentage float64
	Achieved           bool
	NearGoal           bool
	UpdatedAt          time.Time
}

type RewardModel struct {
	ID                  string `gorm:"primaryKey"`
	CampaignID          string `gorm:"index"`
	Name                string
	Description         string
	Quantity            int
	PendingDeliveries   int `gorm:"column:pending_deliveries;->"`
	DeliveredDeliveries int `gorm:"column:delivered_deliveries;->"`
}

type RewardDeliveryModel struct {
	ID           string `gorm:"primaryKey"`
	CampaignID   string `gorm:"column:campaign_id;->"`
	CampaignName string `gorm:"column:campaign_name;->"`
	RewardID     string `gorm:"index:idx_reward_student,unique"`
	RewardName   string `gorm:"column:reward_name;->"`
	StudentID    string `gorm:"index:idx_reward_student,unique"`
	StudentName  string `gorm:"column:student_name;->"`
	StudentPhone string `gorm:"column:student_phone;->"`
	Delivered    bool
	DeliveredAt  *time.Time
}

type WhatsappSettingsModel struct {
	ID              string `gorm:"primaryKey"`
	BoxID           string `gorm:"uniqueIndex"`
	ConnectionMode  string
	Provider        string
	BaseURL         string
	InstanceName    string
	APIKeyEncrypted string
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MessageTemplateModel struct {
	ID             string `gorm:"primaryKey"`
	BoxID          string `gorm:"index"`
	Name           string
	Content        string
	ContentSID     string `gorm:"column:content_sid"`
	TemplateType   string `gorm:"column:template_type"`
	Provider       string
	ApprovalStatus string
	Language       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MessageCampaignModel struct {
	ID           string `gorm:"primaryKey"`
	BoxID        string `gorm:"index"`
	CampaignID   string `gorm:"index"`
	Name         string
	Audience     string
	TemplateID   string
	TemplateType string `gorm:"column:template_type"`
	SentAt       *time.Time
	CreatedAt    time.Time
}

type MessageRecipientModel struct {
	ID                 string `gorm:"primaryKey"`
	MessageCampaignID  string `gorm:"index"`
	StudentID          string
	Phone              string
	Status             string
	ErrorMessage       string
	ProviderMessageSID *string `gorm:"column:provider_message_sid"`
	ProviderStatus     string
	DispatchID         *string
	SentAt             *time.Time
	CreatedAt          time.Time
}

type EmailSettingsModel struct {
	ID                string `gorm:"primaryKey"`
	BoxID             string `gorm:"uniqueIndex"`
	Provider          string
	SMTPHost          string `gorm:"column:smtp_host"`
	SMTPPort          int    `gorm:"column:smtp_port"`
	Username          string
	PasswordEncrypted string
	FromEmail         string
	FromName          string
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type EmailTemplateModel struct {
	ID        string `gorm:"primaryKey"`
	BoxID     string `gorm:"index"`
	Name      string
	Subject   string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EmailCampaignModel struct {
	ID         string `gorm:"primaryKey"`
	BoxID      string `gorm:"index"`
	CampaignID string `gorm:"index"`
	Name       string
	Audience   string
	TemplateID string
	SentAt     *time.Time
	CreatedAt  time.Time
}

type EmailRecipientModel struct {
	ID              string `gorm:"primaryKey"`
	EmailCampaignID string `gorm:"index"`
	StudentID       string
	Email           string
	Status          string
	ErrorMessage    string
	SentAt          *time.Time
	CreatedAt       time.Time
}

type AutomationRunModel struct {
	ID                      string `gorm:"primaryKey"`
	BoxID                   string `gorm:"index"`
	ScheduleID              *string
	ExecutionKey            string
	ScheduledFor            *time.Time
	Status                  string `gorm:"index"`
	Source                  string
	Filename                string
	Imported                bool
	RecalculatedCampaigns   int
	SkippedMessageCampaigns int
	SentMessages            int
	FailedMessages          int
	ErrorMessage            string
	StartedAt               time.Time
	FinishedAt              *time.Time
}

type AutomationScheduleModel struct {
	ID          string `gorm:"primaryKey"`
	BoxID       string `gorm:"index"`
	Name        string
	Mode        string
	RunTime     string
	Timezone    string
	DaysOfWeek  string
	AllowResend bool
	Enabled     bool
	LastRunAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkoutModel struct {
	ID          string `gorm:"primaryKey"`
	BoxID       string `gorm:"index"`
	WorkoutDate time.Time
	Title       string
	Goal        string
	Movements   string
	CoachNotes  string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkoutMessageDraftModel struct {
	ID               string `gorm:"primaryKey"`
	BoxID            string `gorm:"index"`
	WorkoutID        string `gorm:"index"`
	CampaignID       *string
	Audience         string `gorm:"index"`
	GeneratedBody    string
	ApprovedBody     string
	Status           string `gorm:"index"`
	TotalRecipients  int
	SentRecipients   int
	FailedRecipients int
	GeneratedAt      time.Time
	ApprovedAt       *time.Time
	SentAt           *time.Time
}

type WorkoutMessageRecipientModel struct {
	ID                    string `gorm:"primaryKey"`
	WorkoutMessageDraftID string `gorm:"index"`
	StudentID             string
	Phone                 string
	Status                string `gorm:"index"`
	ErrorMessage          string
	ProviderMessageSID    *string `gorm:"column:provider_message_sid"`
	ProviderStatus        string
	DispatchID            *string
	SentAt                *time.Time
	CreatedAt             time.Time
}

type LLMGenerationLogModel struct {
	ID            string `gorm:"primaryKey"`
	BoxID         string `gorm:"index"`
	WorkoutID     string `gorm:"index"`
	DraftID       string `gorm:"index"`
	Provider      string
	Model         string
	PromptSummary string
	Status        string
	ErrorMessage  string
	CreatedAt     time.Time
}
