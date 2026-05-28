package db

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"gorm.io/gorm"
)

type contextKey struct{}

var txKey = contextKey{}

// InjectTx injects GORM database transaction into context
func InjectTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetTx retrieves GORM database transaction from context
func GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// GormUnitOfWork implements IUnitOfWork interface using GORM
type GormUnitOfWork struct {
	db *gorm.DB
}

// NewGormUnitOfWork creates a new instance of GORM-based Unit of Work
func NewGormUnitOfWork(db *gorm.DB) repositories.IUnitOfWork {
	return &GormUnitOfWork{
		db: db,
	}
}

// Execute performs database actions inside an atomic transaction
func (u *GormUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		txCtx := InjectTx(ctx, tx)
		return fn(txCtx)
	})
}
