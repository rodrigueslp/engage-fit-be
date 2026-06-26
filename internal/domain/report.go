package domain

import "time"

type EligibleStudentReportRow struct {
	CampaignID         ID
	CampaignName       string
	StudentID          ID
	StudentName        string
	StudentPhone       string
	Source             Source
	CurrentCheckins    int
	TargetCheckins     int
	RemainingCheckins  int
	ProgressPercentage float64
	RewardName         string
}

type MonthlyFrequencyReportRow struct {
	StudentID    ID
	StudentName  string
	StudentPhone string
	Source       Source
	Checkins     int
	FirstCheckin *time.Time
	LastCheckin  *time.Time
}
