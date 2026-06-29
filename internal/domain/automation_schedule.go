package domain

import "time"

type AutomationSchedule struct {
	ID          ID
	BoxID       ID
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
