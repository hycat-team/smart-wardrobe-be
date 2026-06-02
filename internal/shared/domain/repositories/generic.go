package repositories

import "context"

type IGenericRepository[T any, ID any] interface {
	GetByID(ctx context.Context, id ID) (*T, error)
	GetAll(ctx context.Context) ([]*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id ID) error
}
