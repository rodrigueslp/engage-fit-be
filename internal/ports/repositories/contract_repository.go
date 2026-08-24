package repositories

import (
	"context"
	"errors"
	"time"

	"boxengage/backend/internal/domain"
)

var (
	ErrContractTemplateNotFound          = errors.New("contract template not found")
	ErrContractTemplateNameConflict      = errors.New("contract template name conflict")
	ErrContractTemplateDraftExists       = errors.New("contract template draft exists")
	ErrContractTemplateRevisionConflict  = errors.New("contract template revision conflict")
	ErrContractTemplatePublishedNotFound = errors.New("contract template published version not found")
	ErrContractInstanceNotFound          = errors.New("contract instance not found")
	ErrContractInstanceExists            = errors.New("contract instance already exists for enrollment")
	ErrContractRevisionConflict          = errors.New("contract revision conflict")
	ErrContractAlreadyIssued             = errors.New("contract already issued or accepted")
	ErrContractAcceptanceSessionInvalid  = errors.New("contract acceptance session invalid")
	ErrContractCommandIdempotencyReused  = errors.New("contract command idempotency key reused")
)

type ContractRepository interface {
	FindTemplateByID(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) (*domain.ContractTemplate, error)
	FindTemplateByNameKey(ctx context.Context, organizationID domain.ID, nameKey string) (*domain.ContractTemplate, error)
	ListTemplates(ctx context.Context, organizationID, homeUnitID domain.ID, includeArchived bool) ([]domain.ContractTemplate, error)
	SaveTemplateDraft(ctx context.Context, template *domain.ContractTemplate, draft *domain.ContractTemplateVersion) error
	UpdateTemplateDraft(ctx context.Context, organizationID, homeUnitID, templateID domain.ID, draft *domain.ContractTemplateVersion, expectedRevision int) error
	PublishTemplateVersion(ctx context.Context, organizationID, homeUnitID, templateID, versionID domain.ID, draftRevision int, publishedAt time.Time, contentSHA256 string) error
	CreateNewTemplateVersion(ctx context.Context, organizationID, homeUnitID, templateID domain.ID, newDraft *domain.ContractTemplateVersion) error
	ArchiveTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) error
	ReactivateTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) error
	FindTemplateVersionByID(ctx context.Context, organizationID, versionID domain.ID) (*domain.ContractTemplateVersion, error)

	FindInstanceByID(ctx context.Context, organizationID, unitID, instanceID domain.ID) (*domain.ContractInstance, error)
	FindActiveInstanceByEnrollmentID(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.ContractInstance, error)
	ListInstancesForEnrollment(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractInstance, error)
	ListInstancesForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) ([]domain.ContractInstance, error)
	SaveInstanceDraft(ctx context.Context, instance *domain.ContractInstance) error
	IssueInstance(ctx context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, issuedAt time.Time) (*domain.ContractInstance, error)
	DiscardInstance(ctx context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, discardedAt time.Time) (*domain.ContractInstance, error)
	AcceptInstance(ctx context.Context, organizationID, unitID, instanceID, sessionID domain.ID, acceptedAt time.Time, evidence map[string]any, adapter, adapterRef, docSHA256 string, statementVersion int, personID domain.ID) (*domain.ContractInstance, error)

	FindActiveAcceptanceSessionByInstanceID(ctx context.Context, organizationID, instanceID domain.ID) (*domain.ContractAcceptanceSession, error)
	FindAcceptanceSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.ContractAcceptanceSession, *domain.ContractInstance, error)
	SaveAcceptanceSession(ctx context.Context, session *domain.ContractAcceptanceSession) error
	SupersedeActiveAcceptanceSessions(ctx context.Context, organizationID, instanceID domain.ID) error

	FindCommandRequest(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.ContractCommandRequest, error)
	SaveCommandRequest(ctx context.Context, req *domain.ContractCommandRequest) error

	RecordEvent(ctx context.Context, event *domain.ContractEvent) error
	RecordTemplateAuditEvent(ctx context.Context, event *domain.ContractTemplateAuditEvent) error

	ListPrivacyContracts(ctx context.Context, organizationID, personID domain.ID) ([]domain.ContractPrivacyExport, error)
	PurgeExpiredSimulatorRecords(ctx context.Context, now time.Time, apply bool) (int64, error)
}
