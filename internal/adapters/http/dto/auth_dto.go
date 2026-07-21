package dto

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ResetOwnerPasswordRequest struct {
	NewPassword string `json:"new_password"`
	Reason      string `json:"reason"`
}

type CurrentUserResponse struct {
	ID    string `json:"id"`
	BoxID string `json:"box_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type CreateOwnerRequest struct {
	BoxName    string `json:"box_name"`
	OwnerName  string `json:"owner_name"`
	OwnerEmail string `json:"owner_email"`
	Password   string `json:"password"`
}

type CreateOwnerResponse struct {
	BoxID  string `json:"box_id"`
	UserID string `json:"user_id"`
}
