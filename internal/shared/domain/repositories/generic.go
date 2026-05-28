package repositories

import "context"

type IPreloadableRepository interface {
	GetPreloadRelations() []string
}

type IGenericRepository[T any, ID any] interface {
	FindByID(ctx context.Context, id ID) (*T, error)
	FindAll(ctx context.Context) ([]*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id ID) error
}
