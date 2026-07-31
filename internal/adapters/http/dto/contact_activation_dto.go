package dto

import "time"

type StartContactActivationRequest struct {
	Name              string `json:"name"`
	Source            string `json:"source"`
	RecentCheckinDate string `json:"recent_checkin_date"`
	IsNewStudent      bool   `json:"is_new_student"`
	ConsentAccepted   bool   `json:"consent_accepted"`
}

type ResolveContactActivationRequest struct {
	StudentID string `json:"student_id"`
}

type ContactActivationConfigResponse struct {
	BoxName        string `json:"box_name"`
	ActivationCode string `json:"activation_code"`
	SenderPhone    string `json:"sender_phone"`
	ConsentVersion string `json:"consent_version"`
	ConsentText    string `json:"consent_text"`
}

type StartContactActivationResponse struct {
	WhatsappURL string    `json:"whatsapp_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type ContactActivationSummaryResponse struct {
	TotalStudents   int64  `json:"total_students"`
	WithPhone       int64  `json:"with_phone"`
	OptedIn         int64  `json:"opted_in"`
	OptedOut        int64  `json:"opted_out"`
	PendingReview   int64  `json:"pending_review"`
	AwaitingMessage int64  `json:"awaiting_message"`
	ActivationCode  string `json:"activation_code"`
	SenderPhone     string `json:"sender_phone"`
	WhatsappReady   bool   `json:"whatsapp_ready"`
}

type ContactActivationResponse struct {
	ID                string     `json:"id"`
	StudentID         string     `json:"student_id,omitempty"`
	StudentName       string     `json:"student_name,omitempty"`
	ClaimedName       string     `json:"claimed_name"`
	Source            string     `json:"source"`
	RecentCheckinDate *time.Time `json:"recent_checkin_date,omitempty"`
	IsNewStudent      bool       `json:"is_new_student"`
	Phone             string     `json:"phone,omitempty"`
	Status            string     `json:"status"`
	ConsentedAt       *time.Time `json:"consented_at,omitempty"`
	ExpiresAt         time.Time  `json:"expires_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
