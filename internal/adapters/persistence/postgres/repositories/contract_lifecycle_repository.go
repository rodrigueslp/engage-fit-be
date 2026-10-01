package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r ContractLifecycleGormRepository) CreateSignatureEnvelope(ctx context.Context, envelope *domain.ContractSignatureEnvelope, signer domain.ContractSignerQualification) error {
	plain, err := json.Marshal(signer)
	if err != nil {
		return err
	}
	encrypted, err := r.cipher.Encrypt(string(plain), domain.ContractSignerAAD(envelope.OrganizationID, envelope.ID))
	if err != nil {
		return err
	}
	ref := envelope.ProviderReference
	row := models.ContractSignatureEnvelopeModel{ID: stringID(envelope.ID), OrganizationID: stringID(envelope.OrganizationID), UnitID: stringID(envelope.UnitID), ContractInstanceID: stringID(envelope.ContractInstanceID), Provider: envelope.Provider, ProviderReference: &ref, Status: envelope.Status, SignerQualificationEncrypted: encrypted, ExpiresAt: envelope.ExpiresAt, CompletedAt: envelope.CompletedAt, CreatedByUserID: stringID(envelope.CreatedByUserID), CreatedAt: envelope.CreatedAt, UpdatedAt: envelope.UpdatedAt}
	if err := databaseForContext(r.db, ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return portrepo.ErrSignatureEnvelopeExists
		}
		return err
	}
	return nil
}

func (r ContractLifecycleGormRepository) ListSignatureEnvelopes(ctx context.Context, organizationID, unitID, contractID domain.ID) ([]domain.ContractSignatureEnvelope, error) {
	var rows []models.ContractSignatureEnvelopeModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND contract_instance_id = ?", stringID(organizationID), stringID(unitID), stringID(contractID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ContractSignatureEnvelope, 0, len(rows))
	for _, row := range rows {
		result = append(result, signatureEnvelopeFromModel(row))
	}
	return result, nil
}

func (r ContractLifecycleGormRepository) FindSignatureEnvelopeByReference(ctx context.Context, organizationID, unitID domain.ID, provider, providerReference string) (*domain.ContractSignatureEnvelope, error) {
	var row models.ContractSignatureEnvelopeModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND provider = ? AND provider_reference = ?", stringID(organizationID), stringID(unitID), provider, providerReference).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, portrepo.ErrSignatureEnvelopeNotFound
		}
		return nil, err
	}
	value := signatureEnvelopeFromModel(row)
	return &value, nil
}

func (r ContractLifecycleGormRepository) ReconcileSignatureEnvelope(ctx context.Context, organizationID, unitID domain.ID, provider, providerReference, eventID, status string, evidence map[string]any, at time.Time) (*domain.ContractSignatureEnvelope, bool, error) {
	var output *domain.ContractSignatureEnvelope
	replayed := false
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var envelope models.ContractSignatureEnvelopeModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND unit_id = ? AND provider = ? AND provider_reference = ?", stringID(organizationID), stringID(unitID), provider, providerReference).Take(&envelope).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return portrepo.ErrSignatureEnvelopeNotFound
			}
			return err
		}
		plain, _ := json.Marshal(evidence)
		encrypted, err := r.cipher.Encrypt(string(plain), "contracts:"+string(organizationID)+":"+eventID+":webhook:v1")
		if err != nil {
			return err
		}
		webhook := models.ContractSignatureWebhookEventModel{ID: uuid.NewString(), OrganizationID: envelope.OrganizationID, UnitID: envelope.UnitID, EnvelopeID: envelope.ID, Provider: provider, ProviderEventID: eventID, NormalizedStatus: status, EvidenceEncrypted: encrypted, CreatedAt: at}
		created := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}, {Name: "provider"}, {Name: "provider_event_id"}}, DoNothing: true}).Create(&webhook)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			replayed = true
			value := signatureEnvelopeFromModel(envelope)
			output = &value
			return nil
		}
		if envelope.Status == "accepted" {
			if status != "accepted" {
				return portrepo.ErrSignatureEnvelopeExists
			}
			value := signatureEnvelopeFromModel(envelope)
			output = &value
			return nil
		}
		if envelope.Status == "declined" || envelope.Status == "expired" || envelope.Status == "failed" {
			return portrepo.ErrSignatureEnvelopeExists
		}
		updates := map[string]any{"status": status, "webhook_event_id": eventID, "evidence_encrypted": encrypted, "updated_at": at}
		if status == "accepted" {
			updates["completed_at"] = at
		}
		if err := tx.Model(&models.ContractSignatureEnvelopeModel{}).Where("id = ? AND organization_id = ?", envelope.ID, envelope.OrganizationID).Updates(updates).Error; err != nil {
			return err
		}
		if status == "accepted" {
			var instance models.ContractInstanceModel
			if err := tx.Where("id = ? AND organization_id = ? AND unit_id = ? AND status = 'pending_acceptance'", envelope.ContractInstanceID, envelope.OrganizationID, envelope.UnitID).Take(&instance).Error; err != nil {
				return err
			}
			sessionID := uuid.NewString()
			sum := sha256.Sum256([]byte("external:" + provider + ":" + eventID))
			tokenHash := fmt.Sprintf("%x", sum[:])
			session := models.ContractAcceptanceSessionModel{ID: sessionID, OrganizationID: envelope.OrganizationID, ContractInstanceID: envelope.ContractInstanceID, Adapter: "external_provider", AdapterReference: providerReference, TokenHash: tokenHash, Status: "used", ExpiresAt: at, UsedAt: &at, CreatedAt: at, UpdatedAt: at}
			if err := tx.Create(&session).Error; err != nil {
				return err
			}
			minimal, _ := json.Marshal(map[string]any{"provider": provider, "provider_event_id": eventID, "envelope_id": envelope.ID, "verified": true})
			acceptance := models.ContractAcceptanceModel{ID: uuid.NewString(), OrganizationID: envelope.OrganizationID, ContractInstanceID: envelope.ContractInstanceID, AcceptanceSessionID: sessionID, Adapter: "external_provider", AdapterReference: providerReference, EvidenceSchemaVersion: 1, PersonID: instance.PersonID, DocumentSHA256: instance.DocumentSHA256, AcceptanceStatementVersion: instance.AcceptanceStatementVersion, Evidence: string(minimal), AcceptedAt: at, CreatedAt: at}
			if err := tx.Create(&acceptance).Error; err != nil {
				return err
			}
			result := tx.Model(&models.ContractInstanceModel{}).Where("id = ? AND organization_id = ? AND status = 'pending_acceptance'", instance.ID, instance.OrganizationID).Updates(map[string]any{"status": "accepted", "revision": gorm.Expr("revision + 1"), "accepted_at": at, "updated_at": at})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return portrepo.ErrContractRevisionConflict
			}
			before, _ := json.Marshal(map[string]any{"status": "pending_acceptance"})
			after, _ := json.Marshal(map[string]any{"status": "accepted", "adapter": "external_provider", "document_sha256": instance.DocumentSHA256})
			event := models.ContractEventModel{ID: uuid.NewString(), OrganizationID: instance.OrganizationID, UnitID: instance.UnitID, PersonID: instance.PersonID, EnrollmentID: instance.EnrollmentID, ContractInstanceID: instance.ID, ActorType: "system", Action: "contract.accepted", BeforeData: string(before), AfterData: string(after), CreatedAt: at}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}
		}
		envelope.Status = status
		envelope.WebhookEventID = &eventID
		envelope.EvidenceEncrypted = &encrypted
		envelope.UpdatedAt = at
		if status == "accepted" {
			envelope.CompletedAt = &at
		}
		value := signatureEnvelopeFromModel(envelope)
		output = &value
		return nil
	})
	return output, replayed, err
}

func (r ContractLifecycleGormRepository) CreateLegalAction(ctx context.Context, action *domain.ContractLegalAction) error {
	terms, _ := json.Marshal(action.TermsSnapshot)
	financial, _ := json.Marshal(action.FinancialInstruction)
	effective, _ := time.Parse("2006-01-02", action.EffectiveOn)
	var ends *time.Time
	if action.EndsOn != nil {
		parsed, _ := time.Parse("2006-01-02", *action.EndsOn)
		ends = &parsed
	}
	row := models.ContractLegalActionModel{ID: stringID(action.ID), OrganizationID: stringID(action.OrganizationID), UnitID: stringID(action.UnitID), PersonID: stringID(action.PersonID), EnrollmentID: stringID(action.EnrollmentID), SourceContractInstanceID: stringID(action.SourceContractInstanceID), ActionType: action.ActionType, Status: action.Status, EffectiveOn: effective, EndsOn: ends, Reason: action.Reason, TermsSnapshot: string(terms), FinancialInstruction: string(financial), Revision: action.Revision, CreatedByUserID: stringID(action.CreatedByUserID), CreatedAt: action.CreatedAt, UpdatedAt: action.UpdatedAt}
	db := databaseForContext(r.db, ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueViolation(err) {
				return portrepo.ErrContractLegalConflict
			}
			return err
		}
		data, _ := json.Marshal(map[string]any{"action_type": action.ActionType, "effective_on": action.EffectiveOn, "revision": 1})
		return tx.Create(&models.ContractLegalEventModel{ID: uuid.NewString(), OrganizationID: row.OrganizationID, UnitID: row.UnitID, LegalActionID: row.ID, ActorUserID: row.CreatedByUserID, Action: "legal_action.recorded", Data: string(data), CreatedAt: row.CreatedAt}).Error
	})
}

func (r ContractLifecycleGormRepository) ListLegalActions(ctx context.Context, organizationID, unitID, enrollmentID domain.ID) ([]domain.ContractLegalAction, error) {
	var rows []models.ContractLegalActionModel
	if err := databaseForContext(r.db, ctx).Where("organization_id = ? AND unit_id = ? AND enrollment_id = ?", stringID(organizationID), stringID(unitID), stringID(enrollmentID)).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ContractLegalAction, 0, len(rows))
	for _, row := range rows {
		result = append(result, legalActionFromModel(row))
	}
	return result, nil
}

func signatureEnvelopeFromModel(row models.ContractSignatureEnvelopeModel) domain.ContractSignatureEnvelope {
	ref := ""
	if row.ProviderReference != nil {
		ref = *row.ProviderReference
	}
	return domain.ContractSignatureEnvelope{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), ContractInstanceID: domainID(row.ContractInstanceID), Provider: row.Provider, ProviderReference: ref, Status: row.Status, ExpiresAt: row.ExpiresAt, CompletedAt: row.CompletedAt, CreatedByUserID: domainID(row.CreatedByUserID), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func legalActionFromModel(row models.ContractLegalActionModel) domain.ContractLegalAction {
	terms := map[string]any{}
	financial := map[string]any{}
	_ = json.Unmarshal([]byte(row.TermsSnapshot), &terms)
	_ = json.Unmarshal([]byte(row.FinancialInstruction), &financial)
	effective := row.EffectiveOn.Format("2006-01-02")
	var ends *string
	if row.EndsOn != nil {
		value := row.EndsOn.Format("2006-01-02")
		ends = &value
	}
	var supersedes *domain.ID
	if row.SupersedesActionID != nil {
		value := domainID(*row.SupersedesActionID)
		supersedes = &value
	}
	return domain.ContractLegalAction{ID: domainID(row.ID), OrganizationID: domainID(row.OrganizationID), UnitID: domainID(row.UnitID), PersonID: domainID(row.PersonID), EnrollmentID: domainID(row.EnrollmentID), SourceContractInstanceID: domainID(row.SourceContractInstanceID), ActionType: row.ActionType, Status: row.Status, EffectiveOn: effective, EndsOn: ends, Reason: row.Reason, TermsSnapshot: terms, FinancialInstruction: financial, Revision: row.Revision, SupersedesActionID: supersedes, CreatedByUserID: domainID(row.CreatedByUserID), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
