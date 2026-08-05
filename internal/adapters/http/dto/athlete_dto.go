package dto

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

type AthleteMembershipResponse struct {
	ID       string `json:"id"`
	BoxID    string `json:"box_id"`
	BoxName  string `json:"box_name"`
	JoinedAt string `json:"joined_at"`
}

type AthleteMeResponse struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Email       string                      `json:"email"`
	Memberships []AthleteMembershipResponse `json:"memberships"`
}

type AthleteWorkoutResponse struct {
	WorkoutResponse
	BoxName string `json:"box_name"`
}
