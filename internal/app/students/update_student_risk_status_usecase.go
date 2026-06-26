package students

import (
	"context"
	"fmt"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type UpdateStudentRiskStatusUseCase struct {
	students repositories.StudentRepository
}

func NewUpdateStudentRiskStatusUseCase(students repositories.StudentRepository) UpdateStudentRiskStatusUseCase {
	return UpdateStudentRiskStatusUseCase{students: students}
}

func (uc UpdateStudentRiskStatusUseCase) Execute(ctx context.Context, boxID, studentID domain.ID, status domain.StudentRiskStatus) error {
	if !validRiskStatus(status) {
		return fmt.Errorf("invalid risk status")
	}
	return uc.students.UpdateRiskStatus(ctx, boxID, studentID, status)
}

func validRiskStatus(status domain.StudentRiskStatus) bool {
	switch status {
	case domain.StudentRiskStatusActive,
		domain.StudentRiskStatusObserving,
		domain.StudentRiskStatusPaused,
		domain.StudentRiskStatusNotInterested:
		return true
	default:
		return false
	}
}
