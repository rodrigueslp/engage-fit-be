package repositories

import "context"

// TransactionManager runs all repository calls made with the provided context
// in the same database transaction.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, operation func(context.Context) error) error
}
