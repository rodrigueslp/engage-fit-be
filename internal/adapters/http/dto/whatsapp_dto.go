package dto

type WhatsappSettingsRequest struct {
	Provider     string `json:"provider"`
	BaseURL      string `json:"base_url"`
	InstanceName string `json:"instance_name"`
	APIKey       string `json:"api_key"`
	Enabled      bool   `json:"enabled"`
}

type WhatsappSettingsResponse struct {
	ID           string `json:"id"`
	BoxID        string `json:"box_id"`
	Provider     string `json:"provider"`
	BaseURL      string `json:"base_url"`
	InstanceName string `json:"instance_name"`
	HasAPIKey    bool   `json:"has_api_key"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	Enabled      bool   `json:"enabled"`
}
