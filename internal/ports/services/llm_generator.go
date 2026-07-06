package services

import "context"

type WorkoutMessageGenerationInput struct {
	BoxName    string
	Date       string
	Title      string
	Goal       string
	Movements  string
	CoachNotes string
	RawText    string
	Audience   string
}

type WorkoutMessageGenerationOutput struct {
	Provider string
	Model    string
	Body     string
}

type LLMGenerator interface {
	GenerateWorkoutMessage(ctx context.Context, input WorkoutMessageGenerationInput) (*WorkoutMessageGenerationOutput, error)
}
