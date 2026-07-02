package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandMemberRepository struct {
	shared_persist.GenericRepository[entities.BrandMember, uuid.UUID]
}

func NewBrandMemberRepository(db *gorm.DB) repositories.IBrandMemberRepository {
	return &BrandMemberRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandMember, uuid.UUID](db, []string{"User", "Brand"}),
	}
}

func (r *BrandMemberRepository) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error) {
	var member entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND user_id = ?", brandID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

func (r *BrandMemberRepository) GetByBrandAndUserIDs(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) ([]*entities.BrandMember, error) {
	if len(userIDs) == 0 {
		return []*entities.BrandMember{}, nil
	}
	var members []*entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND user_id IN ?", brandID, userIDs).
		Find(&members).Error
	return members, err
}

func (r *BrandMemberRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ?", brandID).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *BrandMemberRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Preload("Brand").
		Where("user_id = ? AND status = ?", userID, brandmemberstatus.Active).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}
