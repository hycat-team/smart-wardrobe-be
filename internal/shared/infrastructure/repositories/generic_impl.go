package repositories

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/shared/infrastructure/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GenericRepository[T any, ID any] struct {
	db               *gorm.DB
	preloadRelations []string
}

func NewGenericRepository[T any, ID any](dbConn *gorm.DB, preloadRelations []string) *GenericRepository[T, ID] {
	return &GenericRepository[T, ID]{
		db:               dbConn,
		preloadRelations: preloadRelations,
	}
}

func (r *GenericRepository[T, ID]) GetDB(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GenericRepository[T, ID]) GetQueryWithPreload(ctx context.Context) *gorm.DB {
	query := r.GetDB(ctx)

	for _, relation := range r.preloadRelations {
		query = query.Preload(relation)
	}

	return query
}

func (r *GenericRepository[T, ID]) GetByID(ctx context.Context, id ID) (*T, error) {
	var entity T
	query := r.GetQueryWithPreload(ctx)

	err := query.First(&entity, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *GenericRepository[T, ID]) GetAll(ctx context.Context) ([]*T, error) {
	var entities []*T
	query := r.GetQueryWithPreload(ctx)

	err := query.Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *GenericRepository[T, ID]) Create(ctx context.Context, entity *T) error {
	return r.GetDB(ctx).Omit(clause.Associations).Create(entity).Error
}

func (r *GenericRepository[T, ID]) Update(ctx context.Context, entity *T) error {
	return r.GetDB(ctx).Omit(clause.Associations).Save(entity).Error
}

func (r *GenericRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	var entity T
	return r.GetDB(ctx).Delete(&entity, "id = ?", id).Error
}
