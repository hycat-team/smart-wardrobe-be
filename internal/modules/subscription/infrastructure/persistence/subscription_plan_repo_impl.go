package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionPlanRepository struct {
	*shared_repos.GenericRepository[entities.SubscriptionPlan, uuid.UUID]
}

func NewSubscriptionPlanRepository(db *gorm.DB) repositories.ISubscriptionPlanRepository {
	relations := []string{}
	return &SubscriptionPlanRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.SubscriptionPlan, uuid.UUID](db, relations),
	}
}

func (r *SubscriptionPlanRepository) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
	err := r.GetDB(ctx).
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

func (r *SubscriptionPlanRepository) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
	err := r.GetDB(ctx).
		Where("slug = ? AND is_active = ?", slug, true).
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
