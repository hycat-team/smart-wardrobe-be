package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoyaltyTierRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyTier, uuid.UUID]
}

func NewLoyaltyTierRepository(db *gorm.DB) repositories.ILoyaltyTierRepository {
	relations := []string{"Brand"}
	return &LoyaltyTierRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyTier, uuid.UUID](db, relations),
	}
}

func (r *LoyaltyTierRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.LoyaltyTier, error) {
	var tiers []*entities.LoyaltyTier
	err := r.GetDB(ctx).
		Where("brand_id = ?", brandID).
		Order("rank ASC").
		Find(&tiers).Error
	return tiers, err
}

func (r *LoyaltyTierRepository) GetByBrandAndName(ctx context.Context, brandID uuid.UUID, name string) (*entities.LoyaltyTier, error) {
	var tier entities.LoyaltyTier
	err := r.GetDB(ctx).Where("brand_id = ? AND name = ?", brandID, name).First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tier, nil
}

func (r *LoyaltyTierRepository) GetByBrandAndRank(ctx context.Context, brandID uuid.UUID, rank int) (*entities.LoyaltyTier, error) {
	var tier entities.LoyaltyTier
	err := r.GetDB(ctx).Where("brand_id = ? AND rank = ?", brandID, rank).First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tier, nil
}

func (r *LoyaltyTierRepository) GetHighestEligibleBySpend(ctx context.Context, brandID uuid.UUID, totalSpend float64) (*entities.LoyaltyTier, error) {
	var tier entities.LoyaltyTier
	err := r.GetDB(ctx).
		Where("brand_id = ? AND min_total_spend <= ?", brandID, totalSpend).
		Order("min_total_spend DESC, rank DESC").
		First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tier, nil
}
