package dto

type RewardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
}

type RewardResponse struct {
	ID                  string `json:"id"`
	CampaignID          string `json:"campaign_id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Quantity            int    `json:"quantity"`
	PendingDeliveries   int    `json:"pending_deliveries"`
	DeliveredDeliveries int    `json:"delivered_deliveries"`
	AvailableQuantity   int    `json:"available_quantity"`
}

type RewardDeliveryResponse struct {
	ID           string `json:"id"`
	CampaignID   string `json:"campaign_id,omitempty"`
	CampaignName string `json:"campaign_name,omitempty"`
	RewardID     string `json:"reward_id"`
	RewardName   string `json:"reward_name,omitempty"`
	StudentID    string `json:"student_id"`
	StudentName  string `json:"student_name,omitempty"`
	StudentPhone string `json:"student_phone,omitempty"`
	Delivered    bool   `json:"delivered"`
	DeliveredAt  string `json:"delivered_at,omitempty"`
}
