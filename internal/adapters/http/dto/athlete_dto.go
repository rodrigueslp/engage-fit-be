package dto

import "boxengage/backend/internal/domain"

type AthleteInvitationResponse struct {
	BoxName     string `json:"box_name"`
	StudentName string `json:"student_name"`
	ExpiresAt   string `json:"expires_at"`
}

type CreateAthleteInvitationResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type ClaimAthleteInvitationRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AthleteLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AthletePasswordResetRequest struct {
	Email string `json:"email"`
}
type AthletePasswordResetConfirmRequest struct {
	Password string `json:"password"`
}

type AthleteResultEntryRequest struct {
	SectionIndex   int     `json:"section_index"`
	SectionType    string  `json:"section_type"`
	Movement       string  `json:"movement"`
	ScoreType      string  `json:"score_type"`
	TimeSeconds    int     `json:"time_seconds"`
	Rounds         int     `json:"rounds"`
	Repetitions    int     `json:"repetitions"`
	LoadKG         float64 `json:"load_kg"`
	DistanceMeters int     `json:"distance_meters"`
	Calories       int     `json:"calories"`
	Completed      bool    `json:"completed"`
}
type SaveAthleteWorkoutResultRequest struct {
	Scale       string                      `json:"scale"`
	Entries     []AthleteResultEntryRequest `json:"entries"`
	RPE         int                         `json:"rpe"`
	Notes       string                      `json:"notes"`
	PerformedAt string                      `json:"performed_at"`
}

type AthleteMembershipResponse struct {
	ID       string `json:"id"`
	BoxID    string `json:"box_id"`
	BoxName  string `json:"box_name"`
	JoinedAt string `json:"joined_at"`
}

type AthleteMeResponse struct {
	ID            string                      `json:"id"`
	Name          string                      `json:"name"`
	Email         string                      `json:"email"`
	Memberships   []AthleteMembershipResponse `json:"memberships"`
	EmailVerified bool                        `json:"email_verified"`
}

type AthleteWorkoutResponse struct {
	WorkoutResponse
	BoxName         string                        `json:"box_name"`
	MembershipID    string                        `json:"membership_id"`
	Result          *domain.AthleteWorkoutResult  `json:"result,omitempty"`
	Personalization domain.AthletePersonalization `json:"personalization"`
}
