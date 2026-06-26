package students

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type GetStudentUseCase struct {
	students repositories.StudentRepository
}

func NewGetStudentUseCase(students repositories.StudentRepository) GetStudentUseCase {
	return GetStudentUseCase{students: students}
}

func (uc GetStudentUseCase) Execute(ctx context.Context, boxID, studentID domain.ID) (*domain.Student, error) {
	return uc.students.FindByID(ctx, boxID, studentID)
}
