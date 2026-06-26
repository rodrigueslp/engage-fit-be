package students

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type ListStudentCheckinsUseCase struct {
	checkins repositories.CheckinRepository
}

func NewListStudentCheckinsUseCase(checkins repositories.CheckinRepository) ListStudentCheckinsUseCase {
	return ListStudentCheckinsUseCase{checkins: checkins}
}

func (uc ListStudentCheckinsUseCase) Execute(ctx context.Context, boxID, studentID domain.ID) ([]domain.Checkin, error) {
	return uc.checkins.ListByStudent(ctx, boxID, studentID)
}
