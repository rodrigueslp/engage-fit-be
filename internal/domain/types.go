package domain

import "time"

type ID string

type Source string

const (
	SourceWellhub   Source = "wellhub"
	SourceTotalPass Source = "totalpass"
)

type UserRole string

const (
	UserRoleOwner         UserRole = "OWNER"
	UserRolePlatformAdmin UserRole = "PLATFORM_ADMIN"
)

type StudentRiskStatus string

const (
	StudentRiskStatusActive        StudentRiskStatus = "active"
	StudentRiskStatusObserving     StudentRiskStatus = "observing"
	StudentRiskStatusPaused        StudentRiskStatus = "paused"
	StudentRiskStatusNotInterested StudentRiskStatus = "not_interested"
)

type MessageAudience string

const (
	MessageAudienceNearGoal    MessageAudience = "near_goal"
	MessageAudienceAlmostThere MessageAudience = "almost_there"
	MessageAudienceAchieved    MessageAudience = "achieved"
	MessageAudienceInactive    MessageAudience = "inactive"
	MessageAudienceAll         MessageAudience = "all"
)

type MessageTemplateType string

const (
	MessageTemplateAlmostThere MessageTemplateType = "ALMOST_THERE"
	MessageTemplateGoalReached MessageTemplateType = "GOAL_REACHED"
	MessageTemplateWeMissYou   MessageTemplateType = "WE_MISS_YOU"
)

type MessageTemplateApprovalStatus string

const (
	MessageTemplateNotConfigured MessageTemplateApprovalStatus = "NOT_CONFIGURED"
	MessageTemplatePending       MessageTemplateApprovalStatus = "PENDING"
	MessageTemplateApproved      MessageTemplateApprovalStatus = "APPROVED"
	MessageTemplateRejected      MessageTemplateApprovalStatus = "REJECTED"
)

type MessageRecipientStatus string

const (
	MessageRecipientPending MessageRecipientStatus = "pending"
	MessageRecipientSent    MessageRecipientStatus = "sent"
	MessageRecipientFailed  MessageRecipientStatus = "failed"
)

type TimeRange struct {
	Start time.Time
	End   time.Time
}
