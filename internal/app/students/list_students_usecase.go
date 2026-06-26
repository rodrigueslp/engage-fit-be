package students

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type ListStudentsUseCase struct {
	students repositories.StudentRepository
}

func NewListStudentsUseCase(students repositories.StudentRepository) ListStudentsUseCase {
	return ListStudentsUseCase{students: students}
}

func (uc ListStudentsUseCase) Execute(ctx context.Context, boxID domain.ID, filters repositories.StudentFilters) ([]domain.Student, error) {
	return uc.students.List(ctx, boxID, filters)
}
