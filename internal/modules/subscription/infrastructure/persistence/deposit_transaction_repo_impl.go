package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DepositTransactionRepository struct {
	*shared_repos.GenericRepository[entities.DepositTransaction, uuid.UUID]
}

func NewDepositTransactionRepository(dbConn *gorm.DB) repositories.IDepositTransactionRepository {
	relations := []string{}
	return &DepositTransactionRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.DepositTransaction, uuid.UUID](dbConn, relations),
	}
}

func (r *DepositTransactionRepository) GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Where("gateway_reference = ?", reference).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *DepositTransactionRepository) GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Where("order_code = ?", orderCode).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *DepositTransactionRepository) GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_code = ?", orderCode).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *DepositTransactionRepository) HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.DepositTransaction{}).
		Where("user_id = ? AND status = ? AND transaction_type = ?", userID, depositstatus.Pending, deposittransactiontype.DirectPurchase).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
