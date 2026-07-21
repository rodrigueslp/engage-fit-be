package repositories

import (
	"context"

	"boxengage/backend/internal/domain"
)

type PrivacyRepository interface {
	ExportStudent(ctx context.Context, boxID, studentID domain.ID) (*domain.StudentPrivacyExport, error)
	AnonymizeStudent(ctx context.Context, boxID, studentID, actorUserID domain.ID, reason string) error
	IsIdentitySuppressed(ctx context.Context, boxID domain.ID, source domain.Source, externalID string) (bool, error)
	RecordAudit(ctx context.Context, boxID, studentID, actorUserID domain.ID, action, reason string) error
}
