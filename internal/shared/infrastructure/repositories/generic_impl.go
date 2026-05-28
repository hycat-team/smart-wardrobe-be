package repositories

import (
	"context"
	"errors"
	domain_repo "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/shared/infrastructure/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GenericRepository[T any, ID any] struct {
	db *gorm.DB
}

func NewGenericRepository[T any, ID any](dbConn *gorm.DB) *GenericRepository[T, ID] {
	return &GenericRepository[T, ID]{
		db: dbConn,
	}
}

func (r *GenericRepository[T, ID]) GetDB(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GenericRepository[T, ID]) GetByID(ctx context.Context, id ID) (*T, error) {
	var entity T
	query := r.GetDB(ctx)

	if preloadableRepo, ok := any(r).(domain_repo.IPreloadableRepository); ok {
		for _, relation := range preloadableRepo.GetPreloadRelations() {
			query = query.Preload(relation)
		}
	}

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
	query := r.GetDB(ctx)

	if preloadableRepo, ok := any(r).(domain_repo.IPreloadableRepository); ok {
		for _, relation := range preloadableRepo.GetPreloadRelations() {
			query = query.Preload(relation)
		}
	}

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
