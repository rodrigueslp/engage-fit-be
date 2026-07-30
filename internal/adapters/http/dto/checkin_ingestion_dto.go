package dto

type CreateCheckinIngestionSourceRequest struct {
	Name   string `json:"name" binding:"required"`
	Source string `json:"source" binding:"required"`
}

type UpdateCheckinIngestionSourceRequest struct {
	Name    string `json:"name" binding:"required"`
	Enabled bool   `json:"enabled"`
}

type CheckinIngestionSourceResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Source         string  `json:"source"`
	Enabled        bool    `json:"enabled"`
	LastIngestedAt *string `json:"last_ingested_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	Token          string  `json:"token,omitempty"`
}

type CheckinIngestionBatchResponse struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	IdempotentReplay bool    `json:"idempotent_replay"`
	ImportHistoryID  string  `json:"import_history_id,omitempty"`
	TotalRecords     int     `json:"total_records"`
	StudentsCreated  int     `json:"students_created"`
	CheckinsCreated  int     `json:"checkins_created"`
	CreatedAt        string  `json:"created_at"`
	CompletedAt      *string `json:"completed_at"`
}
