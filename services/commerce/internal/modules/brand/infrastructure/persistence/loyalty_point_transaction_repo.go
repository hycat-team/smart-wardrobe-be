package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoyaltyPointTransactionRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyPointTransaction, uuid.UUID]
}

func NewLoyaltyPointTransactionRepository(db *gorm.DB) repositories.ILoyaltyPointTransactionRepository {
	relations := []string{"LoyaltyAccount", "Brand", "BrandCustomer"}
	return &LoyaltyPointTransactionRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyPointTransaction, uuid.UUID](db, relations),
	}
}

func (r *LoyaltyPointTransactionRepository) GetByBrandAndIdempotencyKey(ctx context.Context, brandID uuid.UUID, idempotencyKey string) (*entities.LoyaltyPointTransaction, error) {
	var tx entities.LoyaltyPointTransaction
	err := r.GetDB(ctx).Where("brand_id = ? AND idempotency_key = ?", brandID, idempotencyKey).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *LoyaltyPointTransactionRepository) GetByLoyaltyAccountID(ctx context.Context, loyaltyAccountID uuid.UUID) ([]*entities.LoyaltyPointTransaction, error) {
	var transactions []*entities.LoyaltyPointTransaction
	err := r.GetDB(ctx).
		Where("loyalty_account_id = ?", loyaltyAccountID).
		Order("created_at DESC").
		Find(&transactions).Error
	return transactions, err
}

func (r *LoyaltyPointTransactionRepository) GetByLoyaltyAccountIDPaginated(ctx context.Context, filter repositories.LoyaltyTransactionFilter) (*repositories.LoyaltyTransactionListResult, error) {
	db := r.GetDB(ctx).Model(&entities.LoyaltyPointTransaction{}).Where("loyalty_account_id = ?", filter.LoyaltyAccountID)

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  filter.Page,
		Limit: filter.Limit,
	}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	var transactions []*entities.LoyaltyPointTransaction
	if err := db.Order("created_at DESC").Find(&transactions).Error; err != nil {
		return nil, err
	}

	return &repositories.LoyaltyTransactionListResult{
		Transactions: transactions,
		TotalCount:   totalCount,
	}, nil
}
