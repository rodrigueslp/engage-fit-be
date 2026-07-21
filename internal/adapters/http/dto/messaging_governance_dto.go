package dto

type MessagingPolicyResponse struct {
	ID                            string `json:"id"`
	Scope                         string `json:"scope"`
	BoxID                         string `json:"box_id,omitempty"`
	DailyMessageLimit             int    `json:"daily_message_limit"`
	MonthlyMessageLimit           int    `json:"monthly_message_limit"`
	PerDispatchLimit              int    `json:"per_dispatch_limit"`
	EstimatedCostMicrosPerMessage int64  `json:"estimated_cost_micros_per_message"`
	DailyCostLimitMicros          int64  `json:"daily_cost_limit_micros"`
	MonthlyCostLimitMicros        int64  `json:"monthly_cost_limit_micros"`
	Currency                      string `json:"currency"`
	WarningPercent                int    `json:"warning_percent"`
	Timezone                      string `json:"timezone"`
	Blocked                       bool   `json:"blocked"`
	UpdatedAt                     string `json:"updated_at,omitempty"`
}

type MessagingUsageResponse struct {
	DailyAccepted              int   `json:"daily_accepted"`
	DailyReserved              int   `json:"daily_reserved"`
	MonthlyAccepted            int   `json:"monthly_accepted"`
	MonthlyReserved            int   `json:"monthly_reserved"`
	DailyEstimatedCostMicros   int64 `json:"daily_estimated_cost_micros"`
	DailyReservedCostMicros    int64 `json:"daily_reserved_cost_micros"`
	MonthlyEstimatedCostMicros int64 `json:"monthly_estimated_cost_micros"`
	MonthlyReservedCostMicros  int64 `json:"monthly_reserved_cost_micros"`
}

type MessagingPolicyWithUsageResponse struct {
	Policy MessagingPolicyResponse `json:"policy"`
	Usage  MessagingUsageResponse  `json:"usage"`
}

type MessagingBoxOverviewResponse struct {
	BoxID          string                  `json:"box_id"`
	BoxName        string                  `json:"box_name"`
	ConnectionMode string                  `json:"connection_mode"`
	Policy         MessagingPolicyResponse `json:"policy"`
	Usage          MessagingUsageResponse  `json:"usage"`
}

type UpdateMessagingPolicyRequest struct {
	DailyMessageLimit             int    `json:"daily_message_limit"`
	MonthlyMessageLimit           int    `json:"monthly_message_limit"`
	PerDispatchLimit              int    `json:"per_dispatch_limit"`
	EstimatedCostMicrosPerMessage int64  `json:"estimated_cost_micros_per_message"`
	DailyCostLimitMicros          int64  `json:"daily_cost_limit_micros"`
	MonthlyCostLimitMicros        int64  `json:"monthly_cost_limit_micros"`
	Currency                      string `json:"currency"`
	WarningPercent                int    `json:"warning_percent"`
	Timezone                      string `json:"timezone"`
	Blocked                       bool   `json:"blocked"`
	Reason                        string `json:"reason"`
}
