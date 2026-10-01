package commerce

import (
	"context"
	"testing"
	"time"

	commerceadapter "boxengage/backend/internal/adapters/commerce"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type commerceRepoFake struct {
	products []domain.CommerceProduct
	sales    map[domain.ID]*domain.CommerceSale
	commands map[string]*domain.CommerceCommand
}

func (f *commerceRepoFake) UnitOrganization(context.Context, domain.ID) (domain.ID, error) {
	return "org-1", nil
}
func (f *commerceRepoFake) ListProducts(context.Context, domain.ID, domain.ID, bool) ([]domain.CommerceProduct, error) {
	return f.products, nil
}
func (f *commerceRepoFake) GetProduct(context.Context, domain.ID, domain.ID, domain.ID) (*domain.CommerceProduct, error) {
	return nil, repositories.ErrCommerceNotFound
}
func (f *commerceRepoFake) GetProductsForSale(_ context.Context, _, _ domain.ID, ids []domain.ID) ([]domain.CommerceProduct, error) {
	result := []domain.CommerceProduct{}
	for _, product := range f.products {
		for _, id := range ids {
			if product.ID == id && product.Status == "active" {
				result = append(result, product)
			}
		}
	}
	return result, nil
}
func (f *commerceRepoFake) CreateProduct(context.Context, *domain.CommerceProduct) error { return nil }
func (f *commerceRepoFake) UpdateProduct(context.Context, *domain.CommerceProduct) error { return nil }
func (f *commerceRepoFake) ListSales(context.Context, domain.ID, domain.ID, int) ([]domain.CommerceSale, error) {
	return nil, nil
}
func (f *commerceRepoFake) GetSale(_ context.Context, _, _, id domain.ID) (*domain.CommerceSale, error) {
	value := *f.sales[id]
	return &value, nil
}
func (f *commerceRepoFake) CreateSale(_ context.Context, value *domain.CommerceSale) error {
	copy := *value
	f.sales[value.ID] = &copy
	return nil
}
func (f *commerceRepoFake) LockCommand(context.Context, domain.ID, domain.ID, string, string) error {
	return nil
}
func (f *commerceRepoFake) FindCommand(_ context.Context, _, _ domain.ID, command, key string) (*domain.CommerceCommand, error) {
	return f.commands[command+":"+key], nil
}
func (f *commerceRepoFake) CreateCommand(_ context.Context, value *domain.CommerceCommand) error {
	copy := *value
	f.commands[value.Command+":"+value.IdempotencyKey] = &copy
	return nil
}
func (f *commerceRepoFake) ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.CommerceSale, error) {
	return nil, nil
}

type transactionFake struct{}

func (transactionFake) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type peopleFake struct{}

func (peopleFake) Get(context.Context, domain.ID, domain.ID, domain.ID) (*domain.PersonProfile, error) {
	return &domain.PersonProfile{Person: domain.Person{ProfileStatus: domain.PersonProfileActive}}, nil
}

func TestCreateSaleFreezesPriceTotalsAndReplaysIdempotently(t *testing.T) {
	repo := &commerceRepoFake{products: []domain.CommerceProduct{{ID: "p-1", Name: "Água", PriceCents: 500, Status: "active"}, {ID: "p-2", Name: "Barra", PriceCents: 750, Status: "active"}}, sales: map[domain.ID]*domain.CommerceSale{}, commands: map[string]*domain.CommerceCommand{}}
	service := NewService(repo, peopleFake{}, commerceadapter.NewManualPaymentRecorder(), transactionFake{})
	service.now = func() time.Time { return time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC) }
	input := CreateSaleInput{Items: []SaleLineInput{{ProductID: "p-1", Quantity: 2}, {ProductID: "p-2", Quantity: 1}}, PaymentMethod: "pix", SoldAt: "2026-09-17T11:00:00Z"}
	created, replayed, err := service.CreateSale(context.Background(), "unit-1", "user-1", "sale-key-123", input)
	if err != nil || replayed {
		t.Fatalf("CreateSale() err=%v replayed=%v", err, replayed)
	}
	if created.TotalCents != 1750 || created.Payment.AmountCents != 1750 || len(created.Items) != 2 {
		t.Fatalf("unexpected sale: %#v", created)
	}
	repo.products[0].PriceCents = 900
	again, replayed, err := service.CreateSale(context.Background(), "unit-1", "user-1", "sale-key-123", input)
	if err != nil || !replayed || again.TotalCents != 1750 {
		t.Fatalf("replay err=%v replayed=%v sale=%#v", err, replayed, again)
	}
}

func TestCreateSaleRejectsInactiveAndReusedKey(t *testing.T) {
	repo := &commerceRepoFake{products: []domain.CommerceProduct{{ID: "p-1", Name: "Água", PriceCents: 500, Status: "inactive"}}, sales: map[domain.ID]*domain.CommerceSale{}, commands: map[string]*domain.CommerceCommand{}}
	service := NewService(repo, peopleFake{}, commerceadapter.NewManualPaymentRecorder(), transactionFake{})
	input := CreateSaleInput{Items: []SaleLineInput{{ProductID: "p-1", Quantity: 1}}, PaymentMethod: "cash", SoldAt: "2026-09-17T11:00:00Z"}
	if _, _, err := service.CreateSale(context.Background(), "unit-1", "user-1", "sale-key-123", input); err != ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
	repo.products[0].Status = "active"
	if _, _, err := service.CreateSale(context.Background(), "unit-1", "user-1", "sale-key-123", input); err != nil {
		t.Fatal(err)
	}
	input.Items[0].Quantity = 2
	if _, _, err := service.CreateSale(context.Background(), "unit-1", "user-1", "sale-key-123", input); err != ErrIdempotencyReused {
		t.Fatalf("want reused key, got %v", err)
	}
}
