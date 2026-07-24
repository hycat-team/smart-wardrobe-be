package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoyaltyProgramRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyProgram, uuid.UUID]
}

func NewLoyaltyProgramRepository(db *gorm.DB) repositories.ILoyaltyProgramRepository {
	relations := []string{"Brand"}
	return &LoyaltyProgramRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyProgram, uuid.UUID](db, relations),
	}
}

func (r *LoyaltyProgramRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) (*entities.LoyaltyProgram, error) {
	var program entities.LoyaltyProgram
	err := r.GetDB(ctx).Where("brand_id = ?", brandID).First(&program).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &program, nil
}

func (r *LoyaltyProgramRepository) GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) (*entities.LoyaltyProgram, error) {
	var program entities.LoyaltyProgram
	err := r.GetDB(ctx).Where("brand_id = ? AND is_active = ?", brandID, true).First(&program).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &program, nil
}


