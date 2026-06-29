package models

import "time"

type BoxModel struct {
	ID                      string `gorm:"primaryKey"`
	Name                    string
	RiskInactiveDays        int
	RiskMessageCooldownDays int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type UserModel struct {
	ID           string `gorm:"primaryKey"`
	BoxID        string `gorm:"index"`
	Name         string
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type StudentModel struct {
	ID                string `gorm:"primaryKey"`
	BoxID             string `gorm:"index"`
	Name              string
	Email             string
	Phone             string
	Source            string `gorm:"index"`
	ExternalID        string `gorm:"index"`
	RiskStatus        string
	RiskLastMessageAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
	Provider        string
	BaseURL         string
	InstanceName    string
	APIKeyEncrypted string
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MessageTemplateModel struct {
	ID         string `gorm:"primaryKey"`
	BoxID      string `gorm:"index"`
	Name       string
	Content    string
	ContentSID string `gorm:"column:content_sid"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type MessageCampaignModel struct {
	ID         string `gorm:"primaryKey"`
	BoxID      string `gorm:"index"`
	CampaignID string `gorm:"index"`
	Name       string
	Audience   string
	TemplateID string
	SentAt     *time.Time
	CreatedAt  time.Time
}

type MessageRecipientModel struct {
	ID                string `gorm:"primaryKey"`
	MessageCampaignID string `gorm:"index"`
	StudentID         string
	Phone             string
	Status            string
	ErrorMessage      string
	SentAt            *time.Time
	CreatedAt         time.Time
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
