package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ModelConfigRepository struct {
	shared_persist.GenericRepository[entities.AIModelConfig, uuid.UUID]
}

type PricingConfigRepository = ModelConfigRepository

func NewPricingConfigRepository(db *gorm.DB) repositories.IPricingConfigRepository {
	return &ModelConfigRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.AIModelConfig, uuid.UUID](db, nil),
	}
}

func (r *ModelConfigRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.AIModelConfig, error) {
	var items []*entities.AIModelConfig
	err := r.GetDB(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

func (r *ModelConfigRepository) GetActiveByProviderAndModel(ctx context.Context, provider string, model string) (*entities.AIModelConfig, error) {
	var item entities.AIModelConfig
	now := time.Now()
	err := r.GetDB(ctx).
		Where("provider = ? AND model = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", provider, model, now, now).
		Order("effective_from DESC").
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *ModelConfigRepository) ListAdmin(ctx context.Context) ([]*entities.AIModelConfig, error) {
	var items []*entities.AIModelConfig
	err := r.GetDB(ctx).
		Order("provider ASC, model ASC, effective_from DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
