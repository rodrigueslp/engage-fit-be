package domain

import "time"

type RewardDelivery struct {
	ID           ID
	CampaignID   ID
	CampaignName string
	RewardID     ID
	RewardName   string
	StudentID    ID
	StudentName  string
	StudentPhone string
	Delivered    bool
	DeliveredAt  *time.Time
}
