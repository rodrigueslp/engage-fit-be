package dto

type BoxResponse struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	RiskInactiveDays        int    `json:"risk_inactive_days"`
	RiskMessageCooldownDays int    `json:"risk_message_cooldown_days"`
}

type UpdateBoxRequest struct {
	Name                    string `json:"name"`
	RiskInactiveDays        int    `json:"risk_inactive_days"`
	RiskMessageCooldownDays int    `json:"risk_message_cooldown_days"`
}
