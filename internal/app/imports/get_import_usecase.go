package imports

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type GetImportUseCase struct {
	imports repositories.ImportHistoryRepository
}

func NewGetImportUseCase(imports repositories.ImportHistoryRepository) GetImportUseCase {
	return GetImportUseCase{imports: imports}
}

func (uc GetImportUseCase) Execute(ctx context.Context, boxID, importID domain.ID) (*domain.ImportHistory, error) {
	return uc.imports.FindByID(ctx, boxID, importID)
}
