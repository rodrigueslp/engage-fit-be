package domain

import "time"

type User struct {
	ID           ID
	BoxID        ID
	Name         string
	Email        string
	PasswordHash string
	AuthVersion  int
	Role         UserRole
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
