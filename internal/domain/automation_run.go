package domain

import "time"

type AutomationRun struct {
	ID                      ID
	BoxID                   ID
	Status                  string
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
