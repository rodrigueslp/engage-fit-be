package dto

type TeamMemberResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

type CreateCoachRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateCoachRequest struct {
	Name   string `json:"name" binding:"required"`
	Active bool   `json:"active"`
}

type ResetCoachPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}
