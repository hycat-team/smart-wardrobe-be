package repositories

import "context"

type IGenericRepository[T any, ID any] interface {
	GetByID(ctx context.Context, id ID) (*T, error)
	GetByIDs(ctx context.Context, ids []ID) ([]*T, error)
	GetAll(ctx context.Context) ([]*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id ID) error
	BatchUpdate(ctx context.Context, entities []*T) error
	BatchUpdateMap(ctx context.Context, ids []ID, values map[string]any) error
}
