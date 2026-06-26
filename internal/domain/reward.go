package domain

type Reward struct {
	ID                  ID
	CampaignID          ID
	Name                string
	Description         string
	Quantity            int
	PendingDeliveries   int
	DeliveredDeliveries int
}
