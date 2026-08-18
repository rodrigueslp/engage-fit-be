package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type transactionStub struct{}

func (transactionStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type repositoryStub struct {
	current            domain.CatalogContext
	created            *domain.Modality
	association        *domain.UnitModality
	audits             []domain.CatalogAuditEvent
	auditError         error
	createdService     *domain.ServiceOffering
	serviceAssociation *domain.UnitServiceOffering
	services           []domain.ServiceOffering
}

func (s *repositoryStub) GetContext(context.Context, domain.ID) (*domain.CatalogContext, error) {
	value := s.current
	return &value, nil
}
func (*repositoryStub) ListModalities(context.Context, domain.ID, domain.ID, bool) ([]domain.Modality, error) {
	return nil, nil
}
func (*repositoryStub) FindModality(context.Context, domain.ID, domain.ID, domain.ID) (*domain.Modality, error) {
	return nil, repositories.ErrCatalogModalityNotFound
}
func (s *repositoryStub) CreateModality(_ context.Context, value *domain.Modality) error {
	value.ID = "modality-1"
	copy := *value
	s.created = &copy
	return nil
}
func (*repositoryStub) UpdateModality(context.Context, domain.Modality) error { return nil }
func (s *repositoryStub) CreateUnitModality(_ context.Context, value *domain.UnitModality) error {
	copy := *value
	s.association = &copy
	return nil
}
func (*repositoryStub) SetUnitModalityActive(context.Context, domain.ID, domain.ID, domain.ID, bool) error {
	return nil
}
func (s *repositoryStub) ListServiceOfferings(context.Context, domain.ID, domain.ID, bool) ([]domain.ServiceOffering, error) {
	return append([]domain.ServiceOffering(nil), s.services...), nil
}
func (s *repositoryStub) FindServiceOffering(_ context.Context, organizationID, _ domain.ID, serviceOfferingID domain.ID) (*domain.ServiceOffering, error) {
	for _, item := range s.services {
		if item.ID == serviceOfferingID && item.OrganizationID == organizationID {
			copy := item
			return &copy, nil
		}
	}
	return nil, repositories.ErrCatalogServiceOfferingNotFound
}
func (s *repositoryStub) CreateServiceOffering(_ context.Context, value *domain.ServiceOffering) error {
	value.ID = "service-1"
	copy := *value
	s.createdService = &copy
	s.services = append(s.services, copy)
	return nil
}
func (*repositoryStub) UpdateServiceOffering(context.Context, domain.ServiceOffering) error {
	return nil
}
func (s *repositoryStub) CreateUnitServiceOffering(_ context.Context, value *domain.UnitServiceOffering) error {
	copy := *value
	s.serviceAssociation = &copy
	return nil
}
func (*repositoryStub) SetUnitServiceOfferingActive(context.Context, domain.ID, domain.ID, domain.ID, bool) error {
	return nil
}
func (*repositoryStub) ListSpaces(context.Context, domain.ID, bool) ([]domain.Space, error) {
	return nil, nil
}
func (*repositoryStub) FindSpace(context.Context, domain.ID, domain.ID) (*domain.Space, error) {
	return nil, repositories.ErrCatalogSpaceNotFound
}
func (*repositoryStub) CreateSpace(context.Context, *domain.Space) error { return nil }
func (*repositoryStub) UpdateSpace(context.Context, domain.Space) error  { return nil }
func (s *repositoryStub) CreateAuditEvent(_ context.Context, value *domain.CatalogAuditEvent) error {
	if s.auditError != nil {
		return s.auditError
	}
	s.audits = append(s.audits, *value)
	return nil
}

func TestCreateModalityNormalizesAndPersistsAtomicParts(t *testing.T) {
	repository := &repositoryStub{current: domain.CatalogContext{Organization: domain.Organization{ID: "organization-1"}, Unit: domain.Box{ID: "unit-1"}}}
	service := NewService(repository, transactionStub{})
	service.now = func() time.Time { return time.Date(2026, 8, 11, 20, 0, 0, 0, time.UTC) }

	item, err := service.CreateModality(context.Background(), "unit-1", "owner-1", "  Cross\t Training ", "#a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Cross Training" || item.NameKey != "cross training" || item.Color != "#A1B2C3" || !item.ActiveInUnit {
		t.Fatalf("unexpected modality: %+v", item)
	}
	if repository.association == nil || repository.association.ModalityID != item.ID || !repository.association.Active {
		t.Fatalf("association not created: %+v", repository.association)
	}
	if len(repository.audits) != 2 || repository.audits[0].Action != "modality.created" || repository.audits[1].Action != "unit_modality.activated" {
		t.Fatalf("unexpected audits: %+v", repository.audits)
	}
}

func TestCatalogUseCasesRejectInvalidInputAndCrossUnit(t *testing.T) {
	repository := &repositoryStub{current: domain.CatalogContext{Organization: domain.Organization{ID: "organization-1"}, Unit: domain.Box{ID: "unit-1"}}}
	service := NewService(repository, transactionStub{})
	if _, err := service.CreateModality(context.Background(), "unit-1", "owner-1", " ", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("blank name = %v", err)
	}
	if _, err := service.CreateSpace(context.Background(), "unit-1", "other-unit", "owner-1", "Sala", nil); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("cross-unit create = %v", err)
	}
	if _, err := service.ListModalities(context.Background(), "unit-1", "other-unit", false); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("cross-unit list = %v", err)
	}
}

func TestCreateModalityPropagatesAuditFailure(t *testing.T) {
	expected := errors.New("audit unavailable")
	repository := &repositoryStub{current: domain.CatalogContext{Organization: domain.Organization{ID: "organization-1"}, Unit: domain.Box{ID: "unit-1"}}, auditError: expected}
	service := NewService(repository, transactionStub{})
	if _, err := service.CreateModality(context.Background(), "unit-1", "owner-1", "Pilates", ""); !errors.Is(err, expected) {
		t.Fatalf("audit failure = %v", err)
	}
}

func TestCreateServiceOfferingNormalizesOptionalFieldsAndAuditsAssociation(t *testing.T) {
	repository := &repositoryStub{current: domain.CatalogContext{Organization: domain.Organization{ID: "organization-1"}, Unit: domain.Box{ID: "unit-1"}}}
	service := NewService(repository, transactionStub{})
	service.now = func() time.Time { return time.Date(2026, 8, 18, 20, 0, 0, 0, time.UTC) }

	item, err := service.CreateServiceOffering(context.Background(), "unit-1", "owner-1", "  Avaliação\t física ", "  Sessão inicial  ", "#a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Avaliação física" || item.NameKey != "avaliação física" || item.Description != "Sessão inicial" || item.Color != "#A1B2C3" || !item.ActiveInUnit {
		t.Fatalf("unexpected service offering: %+v", item)
	}
	if repository.serviceAssociation == nil || repository.serviceAssociation.ServiceOfferingID != item.ID || !repository.serviceAssociation.Active {
		t.Fatalf("association not created: %+v", repository.serviceAssociation)
	}
	if len(repository.audits) != 2 || repository.audits[0].Action != "service_offering.created" || repository.audits[1].Action != "unit_service_offering.activated" {
		t.Fatalf("unexpected audits: %+v", repository.audits)
	}
}

func TestServiceOfferingValidationAndArchivedActivation(t *testing.T) {
	repository := &repositoryStub{current: domain.CatalogContext{Organization: domain.Organization{ID: "organization-1"}, Unit: domain.Box{ID: "unit-1"}}, services: []domain.ServiceOffering{{ID: "service-1", OrganizationID: "organization-1", Status: domain.CatalogStatusArchived}}}
	service := NewService(repository, transactionStub{})
	if _, err := service.CreateServiceOffering(context.Background(), "unit-1", "owner-1", "Recovery", string(make([]rune, 281)), ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("long description = %v", err)
	}
	if _, err := service.ListServiceOfferings(context.Background(), "unit-1", "other-unit", false); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("cross-unit list = %v", err)
	}
	if _, err := service.SetServiceOfferingActive(context.Background(), "unit-1", "unit-1", "owner-1", "service-1", true); !errors.Is(err, ErrServiceOfferingArchived) {
		t.Fatalf("activate archived = %v", err)
	}
}
