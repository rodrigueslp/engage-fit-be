package dto

import "time"

type ManualCheckinRequest struct {
	Date string `json:"date"`
}

type SelfCheckinRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type SelfCheckinSessionResponse struct {
	Token     string    `json:"token,omitempty"`
	BoxName   string    `json:"box_name,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AttendanceCheckinResponse struct {
	StudentID       string `json:"student_id"`
	StudentName     string `json:"student_name,omitempty"`
	CheckinDate     string `json:"checkin_date"`
	AlreadyRecorded bool   `json:"already_recorded"`
}
