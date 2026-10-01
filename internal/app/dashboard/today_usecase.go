package dashboard

import (
	"context"
	"fmt"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type GetTodayOperationsUseCase struct {
	boxes  repositories.BoxRepository
	reader repositories.TodayOperationsRepository
	now    func() time.Time
}

func NewGetTodayOperationsUseCase(boxes repositories.BoxRepository, reader repositories.TodayOperationsRepository) GetTodayOperationsUseCase {
	return GetTodayOperationsUseCase{boxes: boxes, reader: reader, now: time.Now}
}

func (uc GetTodayOperationsUseCase) Execute(ctx context.Context, unitID domain.ID) (*domain.TodayOperations, error) {
	box, err := uc.boxes.FindByID(ctx, unitID)
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(box.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid unit timezone: %w", err)
	}
	now := uc.now()
	local := now.In(location)
	from := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	result, err := uc.reader.ReadToday(ctx, box.OrganizationID, unitID, from.Format("2006-01-02"), from, from.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	result.Timezone = box.Timezone
	result.GeneratedAt = now
	return result, nil
}
