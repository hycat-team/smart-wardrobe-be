package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/billing/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionPlanRepository struct {
	*shared_repos.GenericRepository[entities.SubscriptionPlan, uuid.UUID]
}

func NewSubscriptionPlanRepository(db *gorm.DB) repositories.ISubscriptionPlanRepository {
	return &SubscriptionPlanRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.SubscriptionPlan, uuid.UUID](db),
	}
}

func (r *SubscriptionPlanRepository) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
	err := r.DB.WithContext(ctx).
		Where("price = ? AND is_active = ?", 0, true).
		First(&plan).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}
