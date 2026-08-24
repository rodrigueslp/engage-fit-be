package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type ContractGormRepository struct {
	db *gorm.DB
}

func NewContractGormRepository(db *gorm.DB) *ContractGormRepository {
	return &ContractGormRepository{db: db}
}

// Templates

func (r *ContractGormRepository) FindTemplateByID(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) (*domain.ContractTemplate, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractTemplateModel
	query := db.Where("id = ? AND organization_id = ?", stringID(templateID), stringID(organizationID))
	if homeUnitID != "" {
		query = query.Where("home_unit_id = ?", stringID(homeUnitID))
	}
	if err := query.Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrContractTemplateNotFound
		}
		return nil, err
	}
	return r.loadTemplate(db, row)
}

func (r *ContractGormRepository) FindTemplateByNameKey(ctx context.Context, organizationID domain.ID, nameKey string) (*domain.ContractTemplate, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractTemplateModel
	if err := db.Where("organization_id = ? AND name_key = ?", stringID(organizationID), nameKey).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.loadTemplate(db, row)
}

func (r *ContractGormRepository) ListTemplates(ctx context.Context, organizationID, homeUnitID domain.ID, includeArchived bool) ([]domain.ContractTemplate, error) {
	db := databaseForContext(r.db, ctx)
	var rows []models.ContractTemplateModel
	query := db.Where("organization_id = ? AND home_unit_id = ?", stringID(organizationID), stringID(homeUnitID))
	if !includeArchived {
		query = query.Where("status = ?", "active")
	}
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]domain.ContractTemplate, 0, len(rows))
	for _, row := range rows {
		tmpl, err := r.loadTemplate(db, row)
		if err != nil {
			return nil, err
		}
		result = append(result, *tmpl)
	}
	return result, nil
}

func (r *ContractGormRepository) SaveTemplateDraft(ctx context.Context, template *domain.ContractTemplate, draft *domain.ContractTemplateVersion) error {
	if err := ensureID(&template.ID); err != nil {
		return err
	}
	if err := ensureID(&draft.ID); err != nil {
		return err
	}
	draft.TemplateID = template.ID
	draft.OrganizationID = template.OrganizationID

	tmplRow := models.ContractTemplateModel{
		ID:             stringID(template.ID),
		OrganizationID: stringID(template.OrganizationID),
		HomeUnitID:     stringID(template.HomeUnitID),
		Code:           template.Code,
		Name:           template.Name,
		NameKey:        template.NameKey,
		ContractKind:   string(template.ContractKind),
		Status:         string(template.Status),
		CreatedAt:      template.CreatedAt,
		UpdatedAt:      template.UpdatedAt,
	}

	draftRow := models.ContractTemplateVersionModel{
		ID:                       stringID(draft.ID),
		OrganizationID:           stringID(draft.OrganizationID),
		TemplateID:               stringID(draft.TemplateID),
		VersionNumber:            draft.VersionNumber,
		Revision:                 draft.Revision,
		Status:                   string(draft.Status),
		Title:                    draft.Title,
		BodyMarkdown:             draft.BodyMarkdown,
		AcceptanceStatement:      draft.AcceptanceStatement,
		PlaceholderSchemaVersion: draft.PlaceholderSchemaVersion,
		RendererVersion:          draft.RendererVersion,
		CreatedAt:                draft.CreatedAt,
		UpdatedAt:                draft.UpdatedAt,
	}

	db := databaseForContext(r.db, ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tmplRow).Error; err != nil {
			return err
		}
		return tx.Create(&draftRow).Error
	})
	return contractWriteError(err)
}

func (r *ContractGormRepository) UpdateTemplateDraft(ctx context.Context, organizationID, homeUnitID, templateID domain.ID, draft *domain.ContractTemplateVersion, expectedRevision int) error {
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		var tmplRow models.ContractTemplateModel
		if err := tx.Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(templateID), stringID(organizationID), stringID(homeUnitID)).Take(&tmplRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return portrepo.ErrContractTemplateNotFound
			}
			return err
		}

		var draftRow models.ContractTemplateVersionModel
		if err := tx.Raw("SELECT * FROM contract_template_versions WHERE template_id = ? AND status = 'draft' FOR UPDATE", stringID(templateID)).Scan(&draftRow).Error; err != nil {
			return err
		}
		if draftRow.ID == "" {
			return portrepo.ErrContractTemplateNotFound
		}
		if draftRow.Revision != expectedRevision {
			return portrepo.ErrContractTemplateRevisionConflict
		}

		res := tx.Model(&models.ContractTemplateVersionModel{}).
			Where("id = ? AND revision = ?", draftRow.ID, expectedRevision).
			Updates(map[string]any{
				"title":                draft.Title,
				"body_markdown":        draft.BodyMarkdown,
				"acceptance_statement": draft.AcceptanceStatement,
				"revision":             gorm.Expr("revision + 1"),
				"updated_at":           time.Now().UTC(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return portrepo.ErrContractTemplateRevisionConflict
		}
		return nil
	})
}

func (r *ContractGormRepository) PublishTemplateVersion(ctx context.Context, organizationID, homeUnitID, templateID, versionID domain.ID, draftRevision int, publishedAt time.Time, contentSHA256 string) error {
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		var tmpl models.ContractTemplateModel
		if err := tx.Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(templateID), stringID(organizationID), stringID(homeUnitID)).Take(&tmpl).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return portrepo.ErrContractTemplateNotFound
			}
			return err
		}

		var draft models.ContractTemplateVersionModel
		if err := tx.Raw("SELECT * FROM contract_template_versions WHERE id = ? AND template_id = ? AND status = 'draft' FOR UPDATE", stringID(versionID), stringID(templateID)).Scan(&draft).Error; err != nil {
			return err
		}
		if draft.ID == "" {
			return portrepo.ErrContractTemplateNotFound
		}
		if draft.Revision != draftRevision {
			return portrepo.ErrContractTemplateRevisionConflict
		}

		// Retire any previously published version
		if err := tx.Model(&models.ContractTemplateVersionModel{}).
			Where("template_id = ? AND status = 'published'", stringID(templateID)).
			Updates(map[string]any{
				"status":     "retired",
				"updated_at": publishedAt,
			}).Error; err != nil {
			return err
		}

		// Publish current draft
		res := tx.Model(&models.ContractTemplateVersionModel{}).
			Where("id = ? AND status = 'draft' AND revision = ?", stringID(versionID), draftRevision).
			Updates(map[string]any{
				"status":         "published",
				"published_at":   publishedAt,
				"content_sha256": contentSHA256,
				"revision":       gorm.Expr("revision + 1"),
				"updated_at":     publishedAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return portrepo.ErrContractTemplateRevisionConflict
		}
		return nil
	})
}

func (r *ContractGormRepository) CreateNewTemplateVersion(ctx context.Context, organizationID, homeUnitID, templateID domain.ID, newDraft *domain.ContractTemplateVersion) error {
	if err := ensureID(&newDraft.ID); err != nil {
		return err
	}
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		var tmpl models.ContractTemplateModel
		if err := tx.Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(templateID), stringID(organizationID), stringID(homeUnitID)).Take(&tmpl).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return portrepo.ErrContractTemplateNotFound
			}
			return err
		}

		var existingDraft models.ContractTemplateVersionModel
		if err := tx.Where("template_id = ? AND status = 'draft'", stringID(templateID)).Take(&existingDraft).Error; err == nil {
			return portrepo.ErrContractTemplateDraftExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var maxVer int
		if err := tx.Model(&models.ContractTemplateVersionModel{}).
			Where("template_id = ?", stringID(templateID)).
			Select("COALESCE(MAX(version_number), 0)").
			Scan(&maxVer).Error; err != nil {
			return err
		}

		newDraft.VersionNumber = maxVer + 1
		newDraft.TemplateID = templateID
		newDraft.OrganizationID = organizationID
		newDraft.Status = domain.ContractTemplateVersionStatusDraft
		newDraft.Revision = 1

		row := models.ContractTemplateVersionModel{
			ID:                       stringID(newDraft.ID),
			OrganizationID:           stringID(newDraft.OrganizationID),
			TemplateID:               stringID(newDraft.TemplateID),
			VersionNumber:            newDraft.VersionNumber,
			Revision:                 newDraft.Revision,
			Status:                   string(newDraft.Status),
			Title:                    newDraft.Title,
			BodyMarkdown:             newDraft.BodyMarkdown,
			AcceptanceStatement:      newDraft.AcceptanceStatement,
			PlaceholderSchemaVersion: newDraft.PlaceholderSchemaVersion,
			RendererVersion:          newDraft.RendererVersion,
			CreatedAt:                newDraft.CreatedAt,
			UpdatedAt:                newDraft.UpdatedAt,
		}
		return contractWriteError(tx.Create(&row).Error)
	})
}

func (r *ContractGormRepository) ArchiveTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) error {
	db := databaseForContext(r.db, ctx)
	res := db.Model(&models.ContractTemplateModel{}).
		Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(templateID), stringID(organizationID), stringID(homeUnitID)).
		Updates(map[string]any{
			"status":     "archived",
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return portrepo.ErrContractTemplateNotFound
	}
	return nil
}

func (r *ContractGormRepository) ReactivateTemplate(ctx context.Context, organizationID, homeUnitID, templateID domain.ID) error {
	db := databaseForContext(r.db, ctx)
	res := db.Model(&models.ContractTemplateModel{}).
		Where("id = ? AND organization_id = ? AND home_unit_id = ?", stringID(templateID), stringID(organizationID), stringID(homeUnitID)).
		Updates(map[string]any{
			"status":     "active",
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return portrepo.ErrContractTemplateNotFound
	}
	return nil
}

func (r *ContractGormRepository) FindTemplateVersionByID(ctx context.Context, organizationID, versionID domain.ID) (*domain.ContractTemplateVersion, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractTemplateVersionModel
	if err := db.Where("id = ? AND organization_id = ?", stringID(versionID), stringID(organizationID)).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrContractTemplateNotFound
		}
		return nil, err
	}
	v := templateVersionToDomain(row)
	return &v, nil
}

// Instances

func (r *ContractGormRepository) FindInstanceByID(ctx context.Context, organizationID, unitID, instanceID domain.ID) (*domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractInstanceModel
	if err := db.Where("id = ? AND organization_id = ? AND unit_id = ?", stringID(instanceID), stringID(organizationID), stringID(unitID)).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, portrepo.ErrContractInstanceNotFound
		}
		return nil, err
	}
	return r.loadInstance(db, row)
}

func (r *ContractGormRepository) FindActiveInstanceByEnrollmentID(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) (*domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractInstanceModel
	if err := db.Where("organization_id = ? AND unit_id = ? AND enrollment_id = ? AND status <> 'discarded'", stringID(organizationID), stringID(unitID), stringID(enrollmentID)).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.loadInstance(db, row)
}

func (r *ContractGormRepository) ListInstancesForEnrollment(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var rows []models.ContractInstanceModel
	if err := db.Where("organization_id = ? AND unit_id = ? AND enrollment_id = ?", stringID(organizationID), stringID(unitID), stringID(enrollmentID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ContractInstance, 0, len(rows))
	for _, row := range rows {
		inst, err := r.loadInstance(db, row)
		if err != nil {
			return nil, err
		}
		result = append(result, *inst)
	}
	return result, nil
}

func (r *ContractGormRepository) ListInstancesForPerson(ctx context.Context, organizationID, unitID, personID domain.ID) ([]domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var rows []models.ContractInstanceModel
	if err := db.Where("organization_id = ? AND unit_id = ? AND person_id = ?", stringID(organizationID), stringID(unitID), stringID(personID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ContractInstance, 0, len(rows))
	for _, row := range rows {
		inst, err := r.loadInstance(db, row)
		if err != nil {
			return nil, err
		}
		result = append(result, *inst)
	}
	return result, nil
}

func (r *ContractGormRepository) SaveInstanceDraft(ctx context.Context, instance *domain.ContractInstance) error {
	if err := ensureID(&instance.ID); err != nil {
		return err
	}
	commSnapJSON, err := json.Marshal(instance.CommercialSnapshot)
	if err != nil {
		return err
	}
	partySnapJSON, err := json.Marshal(instance.PartySnapshot)
	if err != nil {
		return err
	}

	row := models.ContractInstanceModel{
		ID:                         stringID(instance.ID),
		OrganizationID:             stringID(instance.OrganizationID),
		UnitID:                     stringID(instance.UnitID),
		PersonID:                   stringID(instance.PersonID),
		EnrollmentID:               stringID(instance.EnrollmentID),
		TemplateVersionID:          stringID(instance.TemplateVersionID),
		ContractKind:               string(instance.ContractKind),
		Status:                     string(instance.Status),
		Revision:                   instance.Revision,
		SnapshotSchemaVersion:      instance.SnapshotSchemaVersion,
		CommercialSnapshot:         string(commSnapJSON),
		CommercialSnapshotSHA256:   instance.CommercialSnapshotSHA256,
		PartySnapshot:              string(partySnapJSON),
		DocumentTitle:              instance.DocumentTitle,
		DocumentBodyMarkdown:       instance.DocumentBodyMarkdown,
		DocumentSHA256:             instance.DocumentSHA256,
		AcceptanceStatement:        instance.AcceptanceStatement,
		AcceptanceStatementVersion: instance.AcceptanceStatementVersion,
		CreatedByUserID:            stringID(instance.CreatedByUserID),
		RetentionClass:             string(instance.RetentionClass),
		RetentionDueAt:             instance.RetentionDueAt,
		CreatedAt:                  instance.CreatedAt,
		UpdatedAt:                  instance.UpdatedAt,
	}

	db := databaseForContext(r.db, ctx)
	return contractWriteError(db.Create(&row).Error)
}

func (r *ContractGormRepository) IssueInstance(ctx context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, issuedAt time.Time) (*domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var instance *domain.ContractInstance
	err := db.Transaction(func(tx *gorm.DB) error {
		var row models.ContractInstanceModel
		if err := tx.Raw("SELECT * FROM contract_instances WHERE id = ? AND organization_id = ? AND unit_id = ? FOR UPDATE", stringID(instanceID), stringID(organizationID), stringID(unitID)).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == "" {
			return portrepo.ErrContractInstanceNotFound
		}
		if row.Status != "draft" {
			return portrepo.ErrContractAlreadyIssued
		}
		if row.Revision != expectedRevision {
			return portrepo.ErrContractRevisionConflict
		}

		actorStr := stringID(actorUserID)
		res := tx.Model(&models.ContractInstanceModel{}).
			Where("id = ? AND revision = ? AND status = 'draft'", stringID(instanceID), expectedRevision).
			Updates(map[string]any{
				"status":            "pending_acceptance",
				"issued_at":         issuedAt,
				"issued_by_user_id": &actorStr,
				"revision":          gorm.Expr("revision + 1"),
				"updated_at":        issuedAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return portrepo.ErrContractRevisionConflict
		}

		var updated models.ContractInstanceModel
		if err := tx.Where("id = ?", stringID(instanceID)).Take(&updated).Error; err != nil {
			return err
		}
		var loadErr error
		instance, loadErr = r.loadInstance(tx, updated)
		return loadErr
	})
	if err != nil {
		return nil, contractWriteError(err)
	}
	return instance, nil
}

func (r *ContractGormRepository) DiscardInstance(ctx context.Context, organizationID, unitID, instanceID, actorUserID domain.ID, expectedRevision int, discardedAt time.Time) (*domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var instance *domain.ContractInstance
	err := db.Transaction(func(tx *gorm.DB) error {
		var row models.ContractInstanceModel
		if err := tx.Raw("SELECT * FROM contract_instances WHERE id = ? AND organization_id = ? AND unit_id = ? FOR UPDATE", stringID(instanceID), stringID(organizationID), stringID(unitID)).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == "" {
			return portrepo.ErrContractInstanceNotFound
		}
		if row.Status != "draft" {
			return portrepo.ErrContractAlreadyIssued
		}
		if row.Revision != expectedRevision {
			return portrepo.ErrContractRevisionConflict
		}

		actorStr := stringID(actorUserID)
		res := tx.Model(&models.ContractInstanceModel{}).
			Where("id = ? AND revision = ? AND status = 'draft'", stringID(instanceID), expectedRevision).
			Updates(map[string]any{
				"status":               "discarded",
				"discarded_at":         discardedAt,
				"discarded_by_user_id": &actorStr,
				"revision":             gorm.Expr("revision + 1"),
				"updated_at":           discardedAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return portrepo.ErrContractRevisionConflict
		}

		var updated models.ContractInstanceModel
		if err := tx.Where("id = ?", stringID(instanceID)).Take(&updated).Error; err != nil {
			return err
		}
		var loadErr error
		instance, loadErr = r.loadInstance(tx, updated)
		return loadErr
	})
	if err != nil {
		return nil, contractWriteError(err)
	}
	return instance, nil
}

func (r *ContractGormRepository) AcceptInstance(ctx context.Context, organizationID, unitID, instanceID, sessionID domain.ID, acceptedAt time.Time, evidence map[string]any, adapter, adapterRef, docSHA256 string, statementVersion int, personID domain.ID) (*domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var instance *domain.ContractInstance
	err := db.Transaction(func(tx *gorm.DB) error {
		var row models.ContractInstanceModel
		if err := tx.Raw("SELECT * FROM contract_instances WHERE id = ? AND organization_id = ? FOR UPDATE", stringID(instanceID), stringID(organizationID)).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == "" {
			return portrepo.ErrContractInstanceNotFound
		}
		if row.Status == "accepted" {
			var loadErr error
			instance, loadErr = r.loadInstance(tx, row)
			return loadErr
		}
		if row.Status != "pending_acceptance" {
			return portrepo.ErrContractRevisionConflict
		}

		// Lock and update session
		var sessRow models.ContractAcceptanceSessionModel
		if err := tx.Raw("SELECT * FROM contract_acceptance_sessions WHERE id = ? AND organization_id = ? FOR UPDATE", stringID(sessionID), stringID(organizationID)).Scan(&sessRow).Error; err != nil {
			return err
		}
		if sessRow.ID == "" || sessRow.Status != "active" {
			return portrepo.ErrContractAcceptanceSessionInvalid
		}

		if err := tx.Model(&models.ContractAcceptanceSessionModel{}).
			Where("id = ?", stringID(sessionID)).
			Updates(map[string]any{
				"status":     "used",
				"used_at":    acceptedAt,
				"updated_at": acceptedAt,
			}).Error; err != nil {
			return err
		}

		evidenceJSON, err := json.Marshal(nonNilAuditData(evidence))
		if err != nil {
			return err
		}

		acceptanceID := domain.ID(uuid.NewString())
		acceptanceRow := models.ContractAcceptanceModel{
			ID:                         stringID(acceptanceID),
			OrganizationID:             stringID(organizationID),
			ContractInstanceID:         stringID(instanceID),
			AcceptanceSessionID:        stringID(sessionID),
			Adapter:                    adapter,
			AdapterReference:           adapterRef,
			EvidenceSchemaVersion:      domain.EvidenceSchemaVersion1,
			PersonID:                   stringID(personID),
			DocumentSHA256:             docSHA256,
			AcceptanceStatementVersion: statementVersion,
			Evidence:                   string(evidenceJSON),
			AcceptedAt:                 acceptedAt,
			CreatedAt:                  acceptedAt,
		}
		if err := tx.Create(&acceptanceRow).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ContractInstanceModel{}).
			Where("id = ? AND status = 'pending_acceptance'", stringID(instanceID)).
			Updates(map[string]any{
				"status":      "accepted",
				"accepted_at": acceptedAt,
				"revision":    gorm.Expr("revision + 1"),
				"updated_at":  acceptedAt,
			}).Error; err != nil {
			return err
		}

		var updated models.ContractInstanceModel
		if err := tx.Where("id = ?", stringID(instanceID)).Take(&updated).Error; err != nil {
			return err
		}
		var loadErr error
		instance, loadErr = r.loadInstance(tx, updated)
		return loadErr
	})
	if err != nil {
		return nil, contractWriteError(err)
	}
	return instance, nil
}

// Acceptance Sessions

func (r *ContractGormRepository) FindActiveAcceptanceSessionByInstanceID(ctx context.Context, organizationID, instanceID domain.ID) (*domain.ContractAcceptanceSession, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractAcceptanceSessionModel
	if err := db.Where("organization_id = ? AND contract_instance_id = ? AND status = 'active'", stringID(organizationID), stringID(instanceID)).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	s := acceptanceSessionToDomain(row)
	return &s, nil
}

func (r *ContractGormRepository) FindAcceptanceSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.ContractAcceptanceSession, *domain.ContractInstance, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractAcceptanceSessionModel
	if err := db.Where("token_hash = ?", tokenHash).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, portrepo.ErrContractAcceptanceSessionInvalid
		}
		return nil, nil, err
	}

	session := acceptanceSessionToDomain(row)

	var instRow models.ContractInstanceModel
	if err := db.Where("id = ? AND organization_id = ?", row.ContractInstanceID, row.OrganizationID).Take(&instRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, portrepo.ErrContractInstanceNotFound
		}
		return nil, nil, err
	}

	instance, err := r.loadInstance(db, instRow)
	if err != nil {
		return nil, nil, err
	}
	return &session, instance, nil
}

func (r *ContractGormRepository) SaveAcceptanceSession(ctx context.Context, session *domain.ContractAcceptanceSession) error {
	if err := ensureID(&session.ID); err != nil {
		return err
	}
	row := models.ContractAcceptanceSessionModel{
		ID:                 stringID(session.ID),
		OrganizationID:     stringID(session.OrganizationID),
		ContractInstanceID: stringID(session.ContractInstanceID),
		Adapter:            session.Adapter,
		AdapterReference:   session.AdapterReference,
		TokenHash:          session.TokenHash,
		Status:             string(session.Status),
		ExpiresAt:          session.ExpiresAt,
		UsedAt:             session.UsedAt,
		CreatedAt:          session.CreatedAt,
		UpdatedAt:          session.UpdatedAt,
	}
	db := databaseForContext(r.db, ctx)
	return contractWriteError(db.Create(&row).Error)
}

func (r *ContractGormRepository) SupersedeActiveAcceptanceSessions(ctx context.Context, organizationID, instanceID domain.ID) error {
	db := databaseForContext(r.db, ctx)
	return db.Model(&models.ContractAcceptanceSessionModel{}).
		Where("organization_id = ? AND contract_instance_id = ? AND status = 'active'", stringID(organizationID), stringID(instanceID)).
		Updates(map[string]any{
			"status":     "superseded",
			"updated_at": time.Now().UTC(),
		}).Error
}

// Commands & Events

func (r *ContractGormRepository) FindCommandRequest(ctx context.Context, organizationID domain.ID, command, idempotencyKey string) (*domain.ContractCommandRequest, error) {
	db := databaseForContext(r.db, ctx)
	var row models.ContractCommandRequestModel
	if err := db.Where("organization_id = ? AND command = ? AND idempotency_key = ?", stringID(organizationID), command, idempotencyKey).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	response := map[string]any{}
	if err := json.Unmarshal([]byte(row.ResponseData), &response); err != nil {
		return nil, err
	}
	var resID *domain.ID
	if row.ResultID != nil {
		id := domainID(*row.ResultID)
		resID = &id
	}
	return &domain.ContractCommandRequest{
		ID:             domainID(row.ID),
		OrganizationID: domainID(row.OrganizationID),
		Command:        row.Command,
		IdempotencyKey: row.IdempotencyKey,
		Fingerprint:    row.RequestFingerprint,
		ResultID:       resID,
		ResponseData:   response,
		CreatedAt:      row.CreatedAt,
	}, nil
}

func (r *ContractGormRepository) SaveCommandRequest(ctx context.Context, req *domain.ContractCommandRequest) error {
	if err := ensureID(&req.ID); err != nil {
		return err
	}
	response, err := json.Marshal(nonNilAuditData(req.ResponseData))
	if err != nil {
		return err
	}
	var resID *string
	if req.ResultID != nil {
		v := stringID(*req.ResultID)
		resID = &v
	}
	row := models.ContractCommandRequestModel{
		ID:                 stringID(req.ID),
		OrganizationID:     stringID(req.OrganizationID),
		Command:            req.Command,
		IdempotencyKey:     req.IdempotencyKey,
		RequestFingerprint: req.Fingerprint,
		ResultID:           resID,
		ResponseData:       string(response),
		CreatedAt:          req.CreatedAt,
	}
	db := databaseForContext(r.db, ctx)
	return contractWriteError(db.Create(&row).Error)
}

func (r *ContractGormRepository) RecordEvent(ctx context.Context, event *domain.ContractEvent) error {
	if err := ensureID(&event.ID); err != nil {
		return err
	}
	before, err := json.Marshal(nonNilAuditData(event.BeforeData))
	if err != nil {
		return err
	}
	after, err := json.Marshal(nonNilAuditData(event.AfterData))
	if err != nil {
		return err
	}
	var actorID *string
	if event.ActorUserID != nil {
		v := stringID(*event.ActorUserID)
		actorID = &v
	}
	row := models.ContractEventModel{
		ID:                 stringID(event.ID),
		OrganizationID:     stringID(event.OrganizationID),
		UnitID:             stringID(event.UnitID),
		PersonID:           stringID(event.PersonID),
		EnrollmentID:       stringID(event.EnrollmentID),
		ContractInstanceID: stringID(event.ContractInstanceID),
		ActorType:          string(event.ActorType),
		ActorUserID:        actorID,
		Action:             event.Action,
		BeforeData:         string(before),
		AfterData:          string(after),
		CreatedAt:          event.CreatedAt,
	}
	db := databaseForContext(r.db, ctx)
	return contractWriteError(db.Create(&row).Error)
}

func (r *ContractGormRepository) RecordTemplateAuditEvent(ctx context.Context, event *domain.ContractTemplateAuditEvent) error {
	if err := ensureID(&event.ID); err != nil {
		return err
	}
	before, err := json.Marshal(nonNilAuditData(event.BeforeData))
	if err != nil {
		return err
	}
	after, err := json.Marshal(nonNilAuditData(event.AfterData))
	if err != nil {
		return err
	}
	var verID *string
	if event.TemplateVersionID != nil {
		v := stringID(*event.TemplateVersionID)
		verID = &v
	}
	row := models.ContractTemplateAuditEventModel{
		ID:                stringID(event.ID),
		OrganizationID:    stringID(event.OrganizationID),
		UnitID:            stringID(event.UnitID),
		TemplateID:        stringID(event.TemplateID),
		TemplateVersionID: verID,
		ActorUserID:       stringID(event.ActorUserID),
		Action:            event.Action,
		BeforeData:        string(before),
		AfterData:         string(after),
		CreatedAt:         event.CreatedAt,
	}
	db := databaseForContext(r.db, ctx)
	return contractWriteError(db.Create(&row).Error)
}

// Privacy & Retention

func (r *ContractGormRepository) ListPrivacy(ctx context.Context, organizationID, personID domain.ID) ([]domain.ContractPrivacyExport, error) {
	return r.ListPrivacyContracts(ctx, organizationID, personID)
}

func (r *ContractGormRepository) ListPrivacyContracts(ctx context.Context, organizationID, personID domain.ID) ([]domain.ContractPrivacyExport, error) {
	db := databaseForContext(r.db, ctx)
	var rows []models.ContractInstanceModel
	if err := db.Where("organization_id = ? AND person_id = ?", stringID(organizationID), stringID(personID)).
		Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]domain.ContractPrivacyExport, 0, len(rows))
	for _, row := range rows {
		instance, err := r.loadInstance(db, row)
		if err != nil {
			return nil, err
		}

		var tmplVerRow models.ContractTemplateVersionModel
		if err := db.Where("id = ? AND organization_id = ?", row.TemplateVersionID, row.OrganizationID).Take(&tmplVerRow).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		tmplVer := templateVersionToDomain(tmplVerRow)

		var sessRows []models.ContractAcceptanceSessionModel
		if err := db.Where("contract_instance_id = ?", row.ID).Order("created_at ASC").Find(&sessRows).Error; err != nil {
			return nil, err
		}
		sessions := make([]domain.ContractAcceptanceSession, 0, len(sessRows))
		for _, sRow := range sessRows {
			s := acceptanceSessionToDomain(sRow)
			s.TokenHash = "" // never expose internal token hash in privacy export
			sessions = append(sessions, s)
		}

		var eventRows []models.ContractEventModel
		if err := db.Where("contract_instance_id = ?", row.ID).Order("created_at ASC").Find(&eventRows).Error; err != nil {
			return nil, err
		}
		events := make([]domain.ContractEvent, 0, len(eventRows))
		for _, eRow := range eventRows {
			events = append(events, contractEventToDomain(eRow))
		}

		result = append(result, domain.ContractPrivacyExport{
			Instance:   *instance,
			Template:   tmplVer,
			Acceptance: instance.Acceptance,
			Sessions:   sessions,
			Events:     events,
		})
	}
	return result, nil
}

func (r *ContractGormRepository) PurgeExpiredSimulatorRecords(ctx context.Context, now time.Time, apply bool) (int64, error) {
	db := databaseForContext(r.db, ctx)
	var count int64
	err := db.Transaction(func(tx *gorm.DB) error {
		var expiredInstances []models.ContractInstanceModel
		if err := tx.Where("retention_class = 'local_simulator' AND retention_due_at <= ?", now).Find(&expiredInstances).Error; err != nil {
			return err
		}
		count = int64(len(expiredInstances))

		if apply && count > 0 {
			instanceIDs := make([]string, 0, len(expiredInstances))
			for _, inst := range expiredInstances {
				instanceIDs = append(instanceIDs, inst.ID)
			}
			if err := tx.Where("contract_instance_id IN ?", instanceIDs).Delete(&models.ContractAcceptanceModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("contract_instance_id IN ?", instanceIDs).Delete(&models.ContractAcceptanceSessionModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("contract_instance_id IN ?", instanceIDs).Delete(&models.ContractEventModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", instanceIDs).Delete(&models.ContractInstanceModel{}).Error; err != nil {
				return err
			}
		}

		// Also purge expired sessions older than 24h
		cutoff24h := now.Add(-24 * time.Hour)
		var expiredSessionsCount int64
		if err := tx.Model(&models.ContractAcceptanceSessionModel{}).
			Where("(status = 'expired' OR (status = 'active' AND expires_at < ?)) AND created_at < ?", now, cutoff24h).
			Count(&expiredSessionsCount).Error; err != nil {
			return err
		}
		count += expiredSessionsCount

		if apply && expiredSessionsCount > 0 {
			if err := tx.Where("(status = 'expired' OR (status = 'active' AND expires_at < ?)) AND created_at < ?", now, cutoff24h).
				Delete(&models.ContractAcceptanceSessionModel{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	return count, err
}

// Helpers

func (r *ContractGormRepository) loadTemplate(db *gorm.DB, row models.ContractTemplateModel) (*domain.ContractTemplate, error) {
	tmpl := templateToDomain(row)
	var verRows []models.ContractTemplateVersionModel
	if err := db.Where("template_id = ? AND organization_id = ?", row.ID, row.OrganizationID).Order("version_number DESC").Find(&verRows).Error; err != nil {
		return nil, err
	}
	tmpl.Versions = make([]domain.ContractTemplateVersion, 0, len(verRows))
	for _, vRow := range verRows {
		v := templateVersionToDomain(vRow)
		tmpl.Versions = append(tmpl.Versions, v)
		if v.Status == domain.ContractTemplateVersionStatusDraft {
			copyV := v
			tmpl.CurrentDraft = &copyV
		}
		if v.Status == domain.ContractTemplateVersionStatusPublished {
			copyV := v
			tmpl.CurrentVersion = &copyV
		}
	}
	return &tmpl, nil
}

func (r *ContractGormRepository) loadInstance(db *gorm.DB, row models.ContractInstanceModel) (*domain.ContractInstance, error) {
	inst, err := instanceToDomain(row)
	if err != nil {
		return nil, err
	}

	var tmplVerRow models.ContractTemplateVersionModel
	if err := db.Where("id = ? AND organization_id = ?", row.TemplateVersionID, row.OrganizationID).Take(&tmplVerRow).Error; err == nil {
		v := templateVersionToDomain(tmplVerRow)
		inst.TemplateVersion = &v
	}

	var sessRow models.ContractAcceptanceSessionModel
	if err := db.Where("contract_instance_id = ? AND status = 'active'", row.ID).Take(&sessRow).Error; err == nil {
		s := acceptanceSessionToDomain(sessRow)
		inst.ActiveSession = &s
	}

	var accRow models.ContractAcceptanceModel
	if err := db.Where("contract_instance_id = ?", row.ID).Take(&accRow).Error; err == nil {
		a := acceptanceToDomain(accRow)
		inst.Acceptance = &a
	}

	var eventRows []models.ContractEventModel
	if err := db.Where("contract_instance_id = ?", row.ID).Order("created_at ASC").Find(&eventRows).Error; err == nil {
		events := make([]domain.ContractEvent, 0, len(eventRows))
		for _, eRow := range eventRows {
			events = append(events, contractEventToDomain(eRow))
		}
		inst.Events = events
	}

	return &inst, nil
}

func templateToDomain(row models.ContractTemplateModel) domain.ContractTemplate {
	return domain.ContractTemplate{
		ID:             domainID(row.ID),
		OrganizationID: domainID(row.OrganizationID),
		HomeUnitID:     domainID(row.HomeUnitID),
		Code:           row.Code,
		Name:           row.Name,
		NameKey:        row.NameKey,
		ContractKind:   domain.ContractKind(row.ContractKind),
		Status:         domain.ContractTemplateStatus(row.Status),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func templateVersionToDomain(row models.ContractTemplateVersionModel) domain.ContractTemplateVersion {
	sha := ""
	if row.ContentSHA256 != nil {
		sha = *row.ContentSHA256
	}
	return domain.ContractTemplateVersion{
		ID:                       domainID(row.ID),
		OrganizationID:           domainID(row.OrganizationID),
		TemplateID:               domainID(row.TemplateID),
		VersionNumber:            row.VersionNumber,
		Revision:                 row.Revision,
		Status:                   domain.ContractTemplateVersionStatus(row.Status),
		Title:                    row.Title,
		BodyMarkdown:             row.BodyMarkdown,
		AcceptanceStatement:      row.AcceptanceStatement,
		PlaceholderSchemaVersion: row.PlaceholderSchemaVersion,
		RendererVersion:          row.RendererVersion,
		ContentSHA256:            sha,
		PublishedAt:              row.PublishedAt,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
}

func instanceToDomain(row models.ContractInstanceModel) (domain.ContractInstance, error) {
	var commSnap domain.ContractCommercialSnapshot
	if err := json.Unmarshal([]byte(row.CommercialSnapshot), &commSnap); err != nil {
		return domain.ContractInstance{}, fmt.Errorf("unmarshal commercial_snapshot: %w", err)
	}

	var partySnap domain.ContractPartySnapshot
	if err := json.Unmarshal([]byte(row.PartySnapshot), &partySnap); err != nil {
		return domain.ContractInstance{}, fmt.Errorf("unmarshal party_snapshot: %w", err)
	}

	var issuedBy, discardedBy *domain.ID
	if row.IssuedByUserID != nil {
		v := domainID(*row.IssuedByUserID)
		issuedBy = &v
	}
	if row.DiscardedByUserID != nil {
		v := domainID(*row.DiscardedByUserID)
		discardedBy = &v
	}

	return domain.ContractInstance{
		ID:                         domainID(row.ID),
		OrganizationID:             domainID(row.OrganizationID),
		UnitID:                     domainID(row.UnitID),
		PersonID:                   domainID(row.PersonID),
		EnrollmentID:               domainID(row.EnrollmentID),
		TemplateVersionID:          domainID(row.TemplateVersionID),
		ContractKind:               domain.ContractKind(row.ContractKind),
		Status:                     domain.ContractInstanceStatus(row.Status),
		Revision:                   row.Revision,
		SnapshotSchemaVersion:      row.SnapshotSchemaVersion,
		CommercialSnapshot:         commSnap,
		CommercialSnapshotSHA256:   row.CommercialSnapshotSHA256,
		PartySnapshot:              partySnap,
		DocumentTitle:              row.DocumentTitle,
		DocumentBodyMarkdown:       row.DocumentBodyMarkdown,
		DocumentSHA256:             row.DocumentSHA256,
		AcceptanceStatement:        row.AcceptanceStatement,
		AcceptanceStatementVersion: row.AcceptanceStatementVersion,
		CreatedByUserID:            domainID(row.CreatedByUserID),
		IssuedByUserID:             issuedBy,
		DiscardedByUserID:          discardedBy,
		IssuedAt:                   row.IssuedAt,
		AcceptedAt:                 row.AcceptedAt,
		DiscardedAt:                row.DiscardedAt,
		RetentionClass:             domain.ContractRetentionClass(row.RetentionClass),
		RetentionDueAt:             row.RetentionDueAt,
		CreatedAt:                  row.CreatedAt,
		UpdatedAt:                  row.UpdatedAt,
	}, nil
}

func acceptanceSessionToDomain(row models.ContractAcceptanceSessionModel) domain.ContractAcceptanceSession {
	return domain.ContractAcceptanceSession{
		ID:                 domainID(row.ID),
		OrganizationID:     domainID(row.OrganizationID),
		ContractInstanceID: domainID(row.ContractInstanceID),
		Adapter:            row.Adapter,
		AdapterReference:   row.AdapterReference,
		TokenHash:          row.TokenHash,
		Status:             domain.ContractAcceptanceSessionStatus(row.Status),
		ExpiresAt:          row.ExpiresAt,
		UsedAt:             row.UsedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func acceptanceToDomain(row models.ContractAcceptanceModel) domain.ContractAcceptance {
	evidence := map[string]any{}
	_ = json.Unmarshal([]byte(row.Evidence), &evidence)
	return domain.ContractAcceptance{
		ID:                         domainID(row.ID),
		OrganizationID:             domainID(row.OrganizationID),
		ContractInstanceID:         domainID(row.ContractInstanceID),
		AcceptanceSessionID:        domainID(row.AcceptanceSessionID),
		Adapter:                    row.Adapter,
		AdapterReference:           row.AdapterReference,
		EvidenceSchemaVersion:      row.EvidenceSchemaVersion,
		PersonID:                   domainID(row.PersonID),
		DocumentSHA256:             row.DocumentSHA256,
		AcceptanceStatementVersion: row.AcceptanceStatementVersion,
		Evidence:                   evidence,
		AcceptedAt:                 row.AcceptedAt,
		CreatedAt:                  row.CreatedAt,
	}
}

func contractEventToDomain(row models.ContractEventModel) domain.ContractEvent {
	before := map[string]any{}
	after := map[string]any{}
	_ = json.Unmarshal([]byte(row.BeforeData), &before)
	_ = json.Unmarshal([]byte(row.AfterData), &after)
	var actorID *domain.ID
	if row.ActorUserID != nil {
		v := domainID(*row.ActorUserID)
		actorID = &v
	}
	return domain.ContractEvent{
		ID:                 domainID(row.ID),
		OrganizationID:     domainID(row.OrganizationID),
		UnitID:             domainID(row.UnitID),
		PersonID:           domainID(row.PersonID),
		EnrollmentID:       domainID(row.EnrollmentID),
		ContractInstanceID: domainID(row.ContractInstanceID),
		ActorType:          domain.ContractActorType(row.ActorType),
		ActorUserID:        actorID,
		Action:             row.Action,
		BeforeData:         before,
		AfterData:          after,
		CreatedAt:          row.CreatedAt,
	}
}

func contractWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			switch pgErr.ConstraintName {
			case "contract_templates_org_name_key":
				return portrepo.ErrContractTemplateNameConflict
			case "idx_contract_template_versions_one_draft":
				return portrepo.ErrContractTemplateDraftExists
			case "idx_contract_instances_one_active_per_enrollment":
				return portrepo.ErrContractInstanceExists
			case "contract_command_requests_scope_key":
				return portrepo.ErrContractCommandIdempotencyReused
			case "contract_acceptances_instance_unique", "contract_acceptances_session_unique":
				return portrepo.ErrContractRevisionConflict
			}
		case "55000": // object_not_in_prerequisite_state (our triggers)
			return portrepo.ErrContractRevisionConflict
		}
	}
	return err
}
