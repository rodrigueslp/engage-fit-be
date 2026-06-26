package dto

type ImportResponse struct {
	ID           string `json:"id,omitempty"`
	Filename     string `json:"filename,omitempty"`
	Source       string `json:"source,omitempty"`
	TotalRecords int    `json:"total_records"`
	Students     int    `json:"students,omitempty"`
	Checkins     int    `json:"checkins,omitempty"`
	ImportedAt   string `json:"imported_at,omitempty"`
}
