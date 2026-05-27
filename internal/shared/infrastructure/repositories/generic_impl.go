package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type GenericRepository[T any, ID any] struct {
	DB *gorm.DB
}

func NewGenericRepository[T any, ID any](db *gorm.DB) *GenericRepository[T, ID] {
	return &GenericRepository[T, ID]{
		DB: db,
	}
}

func (r *GenericRepository[T, ID]) FindByID(ctx context.Context, id ID) (*T, error) {
	var entity T
	err := r.DB.WithContext(ctx).First(&entity, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *GenericRepository[T, ID]) FindAll(ctx context.Context) ([]*T, error) {
	var entities []*T
	err := r.DB.WithContext(ctx).Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *GenericRepository[T, ID]) Create(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Create(entity).Error
}

func (r *GenericRepository[T, ID]) Update(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Save(entity).Error
}

func (r *GenericRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	var entity T
	return r.DB.WithContext(ctx).Delete(&entity, "id = ?", id).Error
}
