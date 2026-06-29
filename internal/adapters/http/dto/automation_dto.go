package dto

type AutomationRunRequest struct {
	Source                  string `json:"source"`
	Filename                string `json:"filename"`
	Imported                bool   `json:"imported"`
	RecalculatedCampaigns   int    `json:"recalculated_campaigns"`
	SkippedMessageCampaigns int    `json:"skipped_message_campaigns"`
	SentMessages            int    `json:"sent_messages"`
	FailedMessages          int    `json:"failed_messages"`
	ErrorMessage            string `json:"error_message"`
}

type AutomationRunUpdateRequest struct {
	Status                  string `json:"status"`
	Imported                bool   `json:"imported"`
	RecalculatedCampaigns   int    `json:"recalculated_campaigns"`
	SkippedMessageCampaigns int    `json:"skipped_message_campaigns"`
	SentMessages            int    `json:"sent_messages"`
	FailedMessages          int    `json:"failed_messages"`
	ErrorMessage            string `json:"error_message"`
}

type AutomationRunResponse struct {
	ID                      string `json:"id"`
	Status                  string `json:"status"`
	Source                  string `json:"source"`
	Filename                string `json:"filename"`
	Imported                bool   `json:"imported"`
	RecalculatedCampaigns   int    `json:"recalculated_campaigns"`
	SkippedMessageCampaigns int    `json:"skipped_message_campaigns"`
	SentMessages            int    `json:"sent_messages"`
	FailedMessages          int    `json:"failed_messages"`
	ErrorMessage            string `json:"error_message,omitempty"`
	StartedAt               string `json:"started_at"`
	FinishedAt              string `json:"finished_at,omitempty"`
}

type AutomationScheduleRequest struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	RunTime     string `json:"run_time"`
	Timezone    string `json:"timezone"`
	DaysOfWeek  string `json:"days_of_week"`
	AllowResend bool   `json:"allow_resend"`
	Enabled     bool   `json:"enabled"`
}

type AutomationScheduleResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	RunTime     string `json:"run_time"`
	Timezone    string `json:"timezone"`
	DaysOfWeek  string `json:"days_of_week"`
	AllowResend bool   `json:"allow_resend"`
	Enabled     bool   `json:"enabled"`
	LastRunAt   string `json:"last_run_at,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
