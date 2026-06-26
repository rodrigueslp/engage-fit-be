package services

import (
	"context"
	"io"
	"time"

	"boxengage/backend/internal/domain"
)

type ParsedCheckin struct {
	StudentName       string
	StudentEmail      string
	StudentPhone      string
	StudentExternalID string
	CheckinDate       time.Time
	CheckinTime       *time.Time
	Source            domain.Source
}

type CheckinFileParser interface {
	Parse(ctx context.Context, reader io.Reader, source domain.Source, filename string) ([]ParsedCheckin, error)
}
