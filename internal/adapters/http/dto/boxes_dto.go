package dto

type BoxResponse struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Status                  string `json:"status"`
	RiskInactiveDays        int    `json:"risk_inactive_days"`
	RiskMessageCooldownDays int    `json:"risk_message_cooldown_days"`
}

type AdminBoxResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	StatusReason    string `json:"status_reason"`
	StatusChangedAt string `json:"status_changed_at,omitempty"`
	OwnerID         string `json:"owner_id"`
	OwnerName       string `json:"owner_name"`
	OwnerEmail      string `json:"owner_email"`
	CreatedAt       string `json:"created_at"`
}

type CreateAdminBoxRequest struct {
	BoxName    string `json:"box_name"`
	OwnerName  string `json:"owner_name"`
	OwnerEmail string `json:"owner_email"`
	Password   string `json:"password"`
	Reason     string `json:"reason"`
}

type UpdateAdminBoxRequest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type ChangeAdminBoxStatusRequest struct {
	Reason string `json:"reason"`
}

type UpdateBoxRequest struct {
	Name                    string `json:"name"`
	RiskInactiveDays        int    `json:"risk_inactive_days"`
	RiskMessageCooldownDays int    `json:"risk_message_cooldown_days"`
}
