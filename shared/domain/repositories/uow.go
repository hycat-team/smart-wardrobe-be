package repositories

import "context"

// IUnitOfWork defines the contract for managing database transactions
type IUnitOfWork interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}
