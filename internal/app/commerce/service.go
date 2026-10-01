package commerce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	portservices "boxengage/backend/internal/ports/services"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput      = errors.New("invalid commerce input")
	ErrNotFound          = repositories.ErrCommerceNotFound
	ErrConflict          = repositories.ErrCommerceConflict
	ErrIdempotencyReused = repositories.ErrCommerceIdempotencyReused
	keyPattern           = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
)

type ProductInput struct {
	Name       string `json:"name"`
	SKU        string `json:"sku"`
	PriceCents int    `json:"price_cents"`
	Status     string `json:"status"`
}

type SaleLineInput struct {
	ProductID domain.ID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type CreateSaleInput struct {
	PersonID      *domain.ID      `json:"person_id"`
	Items         []SaleLineInput `json:"items"`
	PaymentMethod string          `json:"payment_method"`
	SoldAt        string          `json:"sold_at"`
	Note          string          `json:"note"`
}

type personResolver interface {
	Get(context.Context, domain.ID, domain.ID, domain.ID) (*domain.PersonProfile, error)
}

type Service struct {
	repository   repositories.CommerceRepository
	people       personResolver
	payments     portservices.CommercePaymentRecorder
	transactions repositories.TransactionManager
	now          func() time.Time
}

func NewService(repository repositories.CommerceRepository, people personResolver, payments portservices.CommercePaymentRecorder, transactions repositories.TransactionManager) *Service {
	return &Service{repository: repository, people: people, payments: payments, transactions: transactions, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) ListProducts(ctx context.Context, unitID domain.ID, includeInactive bool) ([]domain.CommerceProduct, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.repository.ListProducts(ctx, organizationID, unitID, includeInactive)
}

func (s *Service) CreateProduct(ctx context.Context, unitID, actorID domain.ID, input ProductInput) (*domain.CommerceProduct, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, ErrNotFound
	}
	value := &domain.CommerceProduct{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, CreatedByUserID: actorID, CreatedAt: s.now(), UpdatedAt: s.now()}
	if err := applyProduct(value, input); err != nil {
		return nil, err
	}
	if err := s.repository.CreateProduct(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) UpdateProduct(ctx context.Context, unitID, productID domain.ID, input ProductInput) (*domain.CommerceProduct, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, ErrNotFound
	}
	value, err := s.repository.GetProduct(ctx, organizationID, unitID, productID)
	if err != nil {
		return nil, err
	}
	if err := applyProduct(value, input); err != nil {
		return nil, err
	}
	value.UpdatedAt = s.now()
	if err := s.repository.UpdateProduct(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func applyProduct(value *domain.CommerceProduct, input ProductInput) error {
	value.Name, value.SKU, value.Status = strings.TrimSpace(input.Name), strings.TrimSpace(input.SKU), input.Status
	value.PriceCents = input.PriceCents
	if value.Status == "" {
		value.Status = "active"
	}
	if len(value.Name) < 2 || len(value.Name) > 120 || len(value.SKU) > 64 || value.PriceCents <= 0 || (value.Status != "active" && value.Status != "inactive") {
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) ListSales(ctx context.Context, unitID domain.ID, limit int) ([]domain.CommerceSale, error) {
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, ErrNotFound
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repository.ListSales(ctx, organizationID, unitID, limit)
}

func (s *Service) CreateSale(ctx context.Context, unitID, actorID domain.ID, key string, input CreateSaleInput) (*domain.CommerceSale, bool, error) {
	if !keyPattern.MatchString(key) || len(input.Items) == 0 || len(input.Items) > 50 || !validPaymentMethod(input.PaymentMethod) || len(strings.TrimSpace(input.Note)) > 500 {
		return nil, false, ErrInvalidInput
	}
	soldAt, err := time.Parse(time.RFC3339, input.SoldAt)
	if err != nil {
		return nil, false, ErrInvalidInput
	}
	organizationID, err := s.repository.UnitOrganization(ctx, unitID)
	if err != nil {
		return nil, false, ErrNotFound
	}
	if input.PersonID != nil {
		profile, err := s.people.Get(ctx, organizationID, unitID, *input.PersonID)
		if err != nil || profile.Person.ProfileStatus != domain.PersonProfileActive {
			return nil, false, ErrNotFound
		}
	}
	quantities := map[domain.ID]int{}
	for _, item := range input.Items {
		if item.Quantity < 1 || item.Quantity > 999 {
			return nil, false, ErrInvalidInput
		}
		quantities[item.ProductID] += item.Quantity
		if quantities[item.ProductID] > 999 {
			return nil, false, ErrInvalidInput
		}
	}
	productIDs := make([]domain.ID, 0, len(quantities))
	for id := range quantities {
		productIDs = append(productIDs, id)
	}
	sort.Slice(productIDs, func(i, j int) bool { return productIDs[i] < productIDs[j] })
	canonical := make([]SaleLineInput, 0, len(productIDs))
	for _, id := range productIDs {
		canonical = append(canonical, SaleLineInput{ProductID: id, Quantity: quantities[id]})
	}
	fingerprint := hash(struct {
		PersonID             *domain.ID
		Items                []SaleLineInput
		Method, SoldAt, Note string
	}{input.PersonID, canonical, input.PaymentMethod, input.SoldAt, strings.TrimSpace(input.Note)})
	var result *domain.CommerceSale
	replayed := false
	err = s.transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.repository.LockCommand(tx, organizationID, unitID, "create_sale", key); err != nil {
			return err
		}
		previous, err := s.repository.FindCommand(tx, organizationID, unitID, "create_sale", key)
		if err != nil {
			return err
		}
		if previous != nil {
			if previous.Fingerprint != fingerprint {
				return ErrIdempotencyReused
			}
			replayed = true
			result, err = s.repository.GetSale(tx, organizationID, unitID, previous.ResultID)
			return err
		}
		products, err := s.repository.GetProductsForSale(tx, organizationID, unitID, productIDs)
		if err != nil {
			return err
		}
		if len(products) != len(productIDs) {
			return ErrNotFound
		}
		now, total := s.now(), 0
		sale := &domain.CommerceSale{ID: domain.ID(uuid.NewString()), OrganizationID: organizationID, UnitID: unitID, PersonID: input.PersonID, Status: "completed", Currency: "BRL", Note: strings.TrimSpace(input.Note), CreatedByUserID: actorID, SoldAt: soldAt.UTC(), CreatedAt: now, Items: make([]domain.CommerceSaleItem, 0, len(products))}
		for _, product := range products {
			quantity := quantities[product.ID]
			lineTotal := product.PriceCents * quantity
			if lineTotal <= 0 || total > 100000000-lineTotal {
				return ErrInvalidInput
			}
			total += lineTotal
			sale.Items = append(sale.Items, domain.CommerceSaleItem{ID: domain.ID(uuid.NewString()), ProductID: product.ID, ProductName: product.Name, SKU: product.SKU, Quantity: quantity, UnitPriceCents: product.PriceCents, TotalCents: lineTotal})
		}
		sale.TotalCents = total
		receipt, err := s.payments.Record(tx, portservices.CommercePaymentRequest{SaleID: sale.ID, ActorUserID: actorID, Method: input.PaymentMethod, AmountCents: total, OccurredAt: soldAt.UTC()})
		if err != nil {
			return ErrInvalidInput
		}
		sale.Payment = domain.CommerceSalePayment{ID: domain.ID(uuid.NewString()), Method: input.PaymentMethod, AmountCents: total, OccurredAt: soldAt.UTC(), ActorUserID: actorID, Reference: receipt.Reference, CreatedAt: now}
		if err := s.repository.CreateSale(tx, sale); err != nil {
			return err
		}
		if err := s.repository.CreateCommand(tx, &domain.CommerceCommand{OrganizationID: organizationID, UnitID: unitID, Command: "create_sale", IdempotencyKey: key, Fingerprint: fingerprint, ResultID: sale.ID, CreatedAt: now}); err != nil {
			return err
		}
		result = sale
		return nil
	})
	return result, replayed, err
}

func validPaymentMethod(value string) bool {
	return value == "pix" || value == "cash" || value == "card_machine"
}
func hash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
