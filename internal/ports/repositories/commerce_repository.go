package repositories

import (
	"context"
	"errors"

	"boxengage/backend/internal/domain"
)

var (
	ErrCommerceNotFound          = errors.New("commerce not found")
	ErrCommerceConflict          = errors.New("commerce conflict")
	ErrCommerceIdempotencyReused = errors.New("commerce idempotency reused")
)

type CommerceRepository interface {
	UnitOrganization(context.Context, domain.ID) (domain.ID, error)
	ListProducts(context.Context, domain.ID, domain.ID, bool) ([]domain.CommerceProduct, error)
	GetProduct(context.Context, domain.ID, domain.ID, domain.ID) (*domain.CommerceProduct, error)
	GetProductsForSale(context.Context, domain.ID, domain.ID, []domain.ID) ([]domain.CommerceProduct, error)
	CreateProduct(context.Context, *domain.CommerceProduct) error
	UpdateProduct(context.Context, *domain.CommerceProduct) error
	ListSales(context.Context, domain.ID, domain.ID, int) ([]domain.CommerceSale, error)
	GetSale(context.Context, domain.ID, domain.ID, domain.ID) (*domain.CommerceSale, error)
	CreateSale(context.Context, *domain.CommerceSale) error
	LockCommand(context.Context, domain.ID, domain.ID, string, string) error
	FindCommand(context.Context, domain.ID, domain.ID, string, string) (*domain.CommerceCommand, error)
	CreateCommand(context.Context, *domain.CommerceCommand) error
	ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.CommerceSale, error)
}

type CommercePrivacyReader interface {
	ListPrivacy(context.Context, domain.ID, domain.ID) ([]domain.CommerceSale, error)
}
