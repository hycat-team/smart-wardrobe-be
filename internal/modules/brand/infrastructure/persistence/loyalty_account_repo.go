package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoyaltyAccountRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyAccount, uuid.UUID]
}

func NewLoyaltyAccountRepository(db *gorm.DB) repositories.ILoyaltyAccountRepository {
	relations := []string{"Brand", "BrandCustomer", "CurrentTier"}
	return &LoyaltyAccountRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyAccount, uuid.UUID](db, relations),
	}
}

func (r *LoyaltyAccountRepository) GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).Where("brand_customer_id = ?", brandCustomerID).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *LoyaltyAccountRepository) GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.LoyaltyAccount, error) {
	if len(brandCustomerIDs) == 0 {
		return []*entities.LoyaltyAccount{}, nil
	}
	var accounts []*entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).Where("brand_customer_id IN ?", brandCustomerIDs).Find(&accounts).Error
	return accounts, err
}

func (r *LoyaltyAccountRepository) GetByBrandCustomerIDForUpdate(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("brand_customer_id = ?", brandCustomerID).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *LoyaltyAccountRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *LoyaltyAccountRepository) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).Where("brand_id = ? AND user_id = ?", brandID, userID).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}
