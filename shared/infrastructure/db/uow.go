package db

import (
	"context"
	"smart-wardrobe-be/shared/domain/repositories"

	"gorm.io/gorm"
)

type contextKey struct{}

var txKey = contextKey{}

// InjectTx binds a GORM transaction to ctx and returns the context callers should
// use inside the unit of work.
//
// The binding is the load-bearing part, and it is easy to miss. GORM opens a
// transaction by cloning the connection handle with
// `Session{Context: db.Statement.Context}`, and on a handle built by `gorm.Open`
// that context is `context.Background()`. Injecting that handle unchanged means the
// transaction is live but detached: `txCtx.Err()` reports the caller's deadline,
// while every query issued straight off `GetTx(txCtx)` runs with no deadline and no
// cancellation at all.
//
// That split is what FR-076 forbids. It produces the worst kind of failure - a
// request that has already given up at its 60 second deadline while its database
// work keeps going, so the caller times out on a unit of work that then commits
// anyway.
//
// Repositories fall into two groups, and before this fix only one of them was safe.
// `GenericRepository.GetDB` already re-attached the caller with `tx.WithContext(ctx)`.
// The ones that take the transaction from the context and use it directly - the
// outbox, the subscription audit, renewal attempt, provider payment event and
// webhook inbox repositories - did not, and silently ran unbounded. Binding here
// rather than in each of them fixes all of those at once and keeps a sixth caller
// from reintroducing the bug.
//
// A nil transaction is passed through unchanged so `GetTx` keeps its existing
// "no transaction in this context" answer.
func InjectTx(ctx context.Context, tx *gorm.DB) context.Context {
	if tx != nil {
		tx = tx.WithContext(ctx)
	}
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
