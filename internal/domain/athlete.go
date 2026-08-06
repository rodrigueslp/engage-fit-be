package domain

import "time"

type AthleteAccount struct {
	ID              ID
	Name            string
	Email           string
	PasswordHash    string
	Status          string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
	BoxName         string
	MembershipID    ID
	Result          *AthleteWorkoutResult
	Personalization AthletePersonalization
}

type AthleteResultEntry struct {
	SectionIndex int     `json:"section_index"`
	SectionType  string  `json:"section_type"`
	Movement     string  `json:"movement"`
	ScoreType    string  `json:"score_type"`
	TimeSeconds  int     `json:"time_seconds,omitempty"`
	Rounds       int     `json:"rounds,omitempty"`
	Repetitions  int     `json:"repetitions,omitempty"`
	LoadKG       float64 `json:"load_kg,omitempty"`
	DistanceM    int     `json:"distance_meters,omitempty"`
	Calories     int     `json:"calories,omitempty"`
	Completed    bool    `json:"completed,omitempty"`
}

type AthleteWorkoutResult struct {
	ID           ID                   `json:"id"`
	AthleteID    ID                   `json:"athlete_id"`
	WorkoutID    ID                   `json:"workout_id"`
	MembershipID ID                   `json:"membership_id"`
	Scale        string               `json:"scale"`
	Entries      []AthleteResultEntry `json:"entries"`
	RPE          int                  `json:"rpe,omitempty"`
	Notes        string               `json:"notes"`
	PerformedAt  time.Time            `json:"performed_at"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

type AthletePersonalRecord struct {
	ID             ID         `json:"id"`
	AthleteID      ID         `json:"athlete_id"`
	MovementKey    string     `json:"movement_key"`
	MovementName   string     `json:"movement_name"`
	Metric         string     `json:"metric"`
	BestValue      float64    `json:"best_value"`
	Unit           string     `json:"unit"`
	Status         string     `json:"status"`
	SourceResultID ID         `json:"source_result_id"`
	AchievedAt     time.Time  `json:"achieved_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
}

type AthleteGuidance struct {
	Movement       string  `json:"movement"`
	Message        string  `json:"message"`
	ReferenceValue float64 `json:"reference_value,omitempty"`
	ReferenceUnit  string  `json:"reference_unit,omitempty"`
	SuggestedMin   float64 `json:"suggested_min,omitempty"`
	SuggestedMax   float64 `json:"suggested_max,omitempty"`
}

type AthletePersonalization struct {
	Summary     string            `json:"summary"`
	Pacing      string            `json:"pacing"`
	Guidance    []AthleteGuidance `json:"guidance"`
	GeneratedBy string            `json:"generated_by"`
}

type AthleteAccountToken struct {
	ID        ID
	AthleteID ID
	Purpose   string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type AthleteWorkoutInsight struct {
	ID        ID        `json:"id"`
	AthleteID ID        `json:"athlete_id"`
	WorkoutID ID        `json:"workout_id"`
	InputHash string    `json:"-"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
