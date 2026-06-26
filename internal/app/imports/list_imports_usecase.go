package imports

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type ListImportsUseCase struct {
	imports repositories.ImportHistoryRepository
}

func NewListImportsUseCase(imports repositories.ImportHistoryRepository) ListImportsUseCase {
	return ListImportsUseCase{imports: imports}
}

func (uc ListImportsUseCase) Execute(ctx context.Context, boxID domain.ID) ([]domain.ImportHistory, error) {
	return uc.imports.List(ctx, boxID)
}
