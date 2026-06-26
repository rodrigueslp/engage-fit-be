package boxes

import (
	"context"
	"fmt"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type GetBoxUseCase struct {
	boxes repositories.BoxRepository
}

func NewGetBoxUseCase(boxes repositories.BoxRepository) GetBoxUseCase {
	return GetBoxUseCase{boxes: boxes}
}

func (uc GetBoxUseCase) Execute(ctx context.Context, boxID domain.ID) (*domain.Box, error) {
	return uc.boxes.FindByID(ctx, boxID)
}

type UpdateBoxUseCase struct {
	boxes repositories.BoxRepository
}

func NewUpdateBoxUseCase(boxes repositories.BoxRepository) UpdateBoxUseCase {
	return UpdateBoxUseCase{boxes: boxes}
}

func (uc UpdateBoxUseCase) Execute(ctx context.Context, box domain.Box) error {
	if box.RiskInactiveDays < 1 || box.RiskInactiveDays > 365 {
		return fmt.Errorf("risk inactive days must be between 1 and 365")
	}
	if box.RiskMessageCooldownDays < 1 || box.RiskMessageCooldownDays > 365 {
		return fmt.Errorf("risk message cooldown days must be between 1 and 365")
	}
	return uc.boxes.Update(ctx, box)
}
