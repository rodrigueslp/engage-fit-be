package repositories

import (
	"context"

	portrepo "boxengage/backend/internal/ports/repositories"
	"gorm.io/gorm"
)

type transactionContextKey struct{}

type GormTransactionManager struct {
	db *gorm.DB
}

func NewGormTransactionManager(db *gorm.DB) GormTransactionManager {
	return GormTransactionManager{db: db}
}

func (m GormTransactionManager) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return databaseForContext(m.db, ctx).Transaction(func(tx *gorm.DB) error {
		return operation(context.WithValue(ctx, transactionContextKey{}, tx))
	})
}

func databaseForContext(db *gorm.DB, ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

var _ portrepo.TransactionManager = GormTransactionManager{}
