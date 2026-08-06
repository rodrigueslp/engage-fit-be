package services

import (
	"context"

	"boxengage/backend/internal/domain"
)

type AthleteAccountMailer interface {
	SendAthleteAccountLink(ctx context.Context, athleteID domain.ID, email, name, purpose, token string) error
}

type AthleteExplanationInput struct{ AthleteName, BoxName, WorkoutText, DeterministicContext string }
type AthleteExplanationOutput struct{ Provider, Model, Body string }
type AthleteExplanationGenerator interface {
	GenerateAthleteExplanation(ctx context.Context, input AthleteExplanationInput) (*AthleteExplanationOutput, error)
}
