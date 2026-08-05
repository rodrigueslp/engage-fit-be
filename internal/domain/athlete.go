package domain

import "time"

type AthleteAccount struct {
	ID           ID
	Name         string
	Email        string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AthleteMembership struct {
	ID        ID
	AthleteID ID
	BoxID     ID
	BoxName   string
	Status    string
	JoinedAt  time.Time
}

type AthleteInvitation struct {
	ID              ID
	BoxID           ID
	BoxName         string
	StudentID       ID
	StudentName     string
	TokenHash       string
	CreatedByUserID ID
	ExpiresAt       time.Time
	ClaimedAt       *time.Time
	CreatedAt       time.Time
}

type AthleteSession struct {
	ID        ID
	AthleteID ID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type AthleteContext struct {
	Account     AthleteAccount
	Memberships []AthleteMembership
}

type AthleteWorkout struct {
	Workout
	BoxName string
}
