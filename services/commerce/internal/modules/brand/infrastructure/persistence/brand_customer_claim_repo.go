package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandCustomerClaimRepository struct {
	shared_persist.GenericRepository[entities.BrandCustomerClaim, uuid.UUID]
}

func NewBrandCustomerClaimRepository(db *gorm.DB) repositories.IBrandCustomerClaimRepository {
	relations := []string{"BrandCustomer"}
	return &BrandCustomerClaimRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandCustomerClaim, uuid.UUID](db, relations),
	}
}

func (r *BrandCustomerClaimRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*entities.BrandCustomerClaim, error) {
	var claim entities.BrandCustomerClaim
	err := r.GetDB(ctx).Where("claim_token_hash = ?", tokenHash).First(&claim).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &claim, nil
}

func (r *BrandCustomerClaimRepository) GetActiveByCustomerID(ctx context.Context, brandCustomerID uuid.UUID, now time.Time) ([]*entities.BrandCustomerClaim, error) {
	var claims []*entities.BrandCustomerClaim
	err := r.GetDB(ctx).
		Where("brand_customer_id = ? AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > ?", brandCustomerID, now).
		Order("created_at ASC").
		Find(&claims).Error
	return claims, err
}

func (r *BrandCustomerClaimRepository) GetByCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BrandCustomerClaim, error) {
	var claims []*entities.BrandCustomerClaim
	err := r.GetDB(ctx).
		Where("brand_customer_id = ?", brandCustomerID).
		Order("created_at DESC").
		Find(&claims).Error
	return claims, err
}
