package dto

type WorkoutRequest struct {
	WorkoutDate string `json:"workout_date"`
	Title       string `json:"title"`
	Goal        string `json:"goal"`
	Movements   string `json:"movements"`
	CoachNotes  string `json:"coach_notes"`
	Status      string `json:"status"`
}

type WorkoutResponse struct {
	ID          string `json:"id"`
	WorkoutDate string `json:"workout_date"`
	Title       string `json:"title"`
	Goal        string `json:"goal"`
	Movements   string `json:"movements"`
	CoachNotes  string `json:"coach_notes"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type GenerateWorkoutDraftRequest struct {
	Audience   string   `json:"audience"`
	CampaignID string   `json:"campaign_id"`
	StudentIDs []string `json:"student_ids"`
}

type UpdateWorkoutDraftRequest struct {
	ApprovedBody string `json:"approved_body"`
}

type WorkoutDraftResponse struct {
	ID               string `json:"id"`
	WorkoutID        string `json:"workout_id"`
	CampaignID       string `json:"campaign_id,omitempty"`
	Audience         string `json:"audience"`
	GeneratedBody    string `json:"generated_body"`
	ApprovedBody     string `json:"approved_body"`
	Status           string `json:"status"`
	TotalRecipients  int    `json:"total_recipients"`
	SentRecipients   int    `json:"sent_recipients"`
	FailedRecipients int    `json:"failed_recipients"`
	GeneratedAt      string `json:"generated_at"`
	ApprovedAt       string `json:"approved_at,omitempty"`
	SentAt           string `json:"sent_at,omitempty"`
}

type SendWorkoutDraftResponse struct {
	Total  int `json:"total"`
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
}

type WorkoutRecipientResponse struct {
	ID                    string `json:"id"`
	WorkoutMessageDraftID string `json:"workout_message_draft_id"`
	StudentID             string `json:"student_id"`
	Phone                 string `json:"phone"`
	Status                string `json:"status"`
	ErrorMessage          string `json:"error_message,omitempty"`
	ProviderMessageSID    string `json:"provider_message_sid,omitempty"`
	ProviderStatus        string `json:"provider_status,omitempty"`
	DispatchID            string `json:"dispatch_id,omitempty"`
	SentAt                string `json:"sent_at,omitempty"`
	CreatedAt             string `json:"created_at"`
}
