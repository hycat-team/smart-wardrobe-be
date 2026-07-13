package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PricingConfigRepository struct {
	shared_persist.GenericRepository[entities.AIPricingConfig, uuid.UUID]
}

func NewPricingConfigRepository(db *gorm.DB) repositories.IPricingConfigRepository {
	return &PricingConfigRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.AIPricingConfig, uuid.UUID](db, nil),
	}
}

func (r *PricingConfigRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.AIPricingConfig, error) {
	var items []*entities.AIPricingConfig
	err := r.GetDB(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

func (r *PricingConfigRepository) GetActiveByProviderAndModel(ctx context.Context, provider aiprovider.Provider, model aimodel.Model) (*entities.AIPricingConfig, error) {
	var item entities.AIPricingConfig
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
