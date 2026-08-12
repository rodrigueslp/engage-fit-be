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
	current     domain.CatalogContext
	created     *domain.Modality
	association *domain.UnitModality
	audits      []domain.CatalogAuditEvent
	auditError  error
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
