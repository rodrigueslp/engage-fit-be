package domain

import "time"

type MessagingLimitError struct{ Reason string }

func (e MessagingLimitError) Error() string { return e.Reason }

type MessagingPolicyScope string

const (
	MessagingPolicyScopeBox      MessagingPolicyScope = "box"
	MessagingPolicyScopePlatform MessagingPolicyScope = "platform"
)

type MessagingPolicy struct {
	ID                            ID
	Scope                         MessagingPolicyScope
	BoxID                         ID
	DailyMessageLimit             int
	MonthlyMessageLimit           int
	PerDispatchLimit              int
	EstimatedCostMicrosPerMessage int64
	DailyCostLimitMicros          int64
	MonthlyCostLimitMicros        int64
	Currency                      string
	WarningPercent                int
	Timezone                      string
	Blocked                       bool
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

type MessagingUsage struct {
	DailyAccepted              int
	DailyReserved              int
	MonthlyAccepted            int
	MonthlyReserved            int
	DailyEstimatedCostMicros   int64
	DailyReservedCostMicros    int64
	MonthlyEstimatedCostMicros int64
	MonthlyReservedCostMicros  int64
}

type MessageDispatch struct {
	ID                  ID
	BoxID               ID
	RequestedByUserID   ID
	SourceType          string
	SourceID            ID
	ConnectionMode      WhatsappConnectionMode
	RecipientsTotal     int
	ReservedMessages    int
	AcceptedMessages    int
	FailedMessages      int
	EstimatedCostMicros int64
	Currency            string
	Status              string
	BlockReason         string
	CreatedAt           time.Time
	CompletedAt         *time.Time
}

type MessagingBoxOverview struct {
	Box            Box
	ConnectionMode WhatsappConnectionMode
	Policy         MessagingPolicy
	Usage          MessagingUsage
}

type AdminAuditLog struct {
	ID          ID
	AdminUserID ID
	Action      string
	TargetType  string
	TargetID    string
	BeforeData  []byte
	AfterData   []byte
	Reason      string
	IPAddress   string
	CreatedAt   time.Time
}
