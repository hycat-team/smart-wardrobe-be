package repositories

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"

	"smart-wardrobe-be/shared/infrastructure/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var preloadFieldCache sync.Map

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
		if relation != "" && fieldExistsOnType[T](relation) {
			query = query.Preload(relation)
		}
	}

	return query
}

func fieldExistsOnType[T any](fieldPath string) bool {
	typeKey := reflect.TypeOf((*T)(nil)).Elem().String()
	cacheKey := typeKey + "|" + fieldPath
	if val, ok := preloadFieldCache.Load(cacheKey); ok {
		return val.(bool)
	}

	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	parts := strings.Split(fieldPath, ".")
	current := t
	for i, part := range parts {
		field, ok := current.FieldByName(part)
		if !ok {
			preloadFieldCache.Store(cacheKey, false)
			return false
		}
		if i < len(parts)-1 {
			ft := field.Type
			for ft.Kind() == reflect.Ptr || ft.Kind() == reflect.Slice {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				current = ft
			} else {
				preloadFieldCache.Store(cacheKey, false)
				return false
			}
		}
	}

	preloadFieldCache.Store(cacheKey, true)
	return true
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

func (r *GenericRepository[T, ID]) GetByIDs(ctx context.Context, ids []ID) ([]*T, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var entities []*T
	query := r.GetQueryWithPreload(ctx)
	err := query.Where("id IN ?", ids).Find(&entities).Error
	return entities, err
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

func (r *GenericRepository[T, ID]) BatchUpdate(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, e := range entities {
			if err := tx.Omit(clause.Associations).Save(e).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GenericRepository[T, ID]) BatchUpdateMap(ctx context.Context, ids []ID, values map[string]any) error {
	if len(ids) == 0 {
		return nil
	}
	return r.GetDB(ctx).Model(new(T)).Where("id IN ?", ids).Updates(values).Error
}
