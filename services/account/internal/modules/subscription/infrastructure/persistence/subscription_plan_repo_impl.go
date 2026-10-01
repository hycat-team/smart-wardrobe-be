package persistence

import (
	"context"
	"errors"
	"strings"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionPlanRepository struct {
	*shared_repos.GenericRepository[entities.SubscriptionPlan, uuid.UUID]
}

func NewSubscriptionPlanRepository(db *gorm.DB) repositories.ISubscriptionPlanRepository {
	// relations := []string{}
	return &SubscriptionPlanRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.SubscriptionPlan, uuid.UUID](db, nil),
	}
}

func (r *SubscriptionPlanRepository) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
	err := r.GetDB(ctx).
		Where("plan_kind = ? AND is_active = ?", plankind.DefaultFree, true).
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

func (r *SubscriptionPlanRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var plans []*entities.SubscriptionPlan
	err := r.GetDB(ctx).
		Where("id IN ?", ids).
		Find(&plans).
		Error
	if err != nil {
		return nil, err
	}

	return plans, nil
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

func (r *SubscriptionPlanRepository) GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	var plan entities.SubscriptionPlan
	err := r.GetDB(ctx).
		Where("slug = ?", slug).
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

func (r *SubscriptionPlanRepository) GetBySlugsAnyStatus(ctx context.Context, slugs []string) (map[string]*entities.SubscriptionPlan, error) {
	found := make(map[string]*entities.SubscriptionPlan, len(slugs))
	if len(slugs) == 0 {
		return found, nil
	}

	var plans []entities.SubscriptionPlan
	// `slug` carries a unique index, so `IN (...)` stays index-backed however many
	// slugs arrive - which is the whole point of doing this in one round trip.
	if err := r.GetDB(ctx).Where("slug IN ?", slugs).Find(&plans).Error; err != nil {
		return nil, err
	}
	for idx := range plans {
		found[strings.ToLower(plans[idx].Slug)] = &plans[idx]
	}
	return found, nil
}

func (r *SubscriptionPlanRepository) ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	var plans []*entities.SubscriptionPlan
	err := r.GetDB(ctx).
		Order("tier_rank ASC, price ASC").
		Find(&plans).
		Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}
