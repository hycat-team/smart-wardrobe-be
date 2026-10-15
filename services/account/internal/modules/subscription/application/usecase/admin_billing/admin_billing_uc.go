package admin

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

// AdminBillingUseCase provides read access to deposit transactions for the
// admin billing endpoints. It wraps the repository so handlers stay thin and
// respect the layer boundaries.
type AdminBillingUseCase struct {
	depositRepo repositories.IDepositTransactionRepository
}

func NewAdminBillingUseCase(depositRepo repositories.IDepositTransactionRepository) usecase.IAdminBillingUseCase {
	return &AdminBillingUseCase{depositRepo: depositRepo}
}

func (uc *AdminBillingUseCase) ListTransactions(ctx context.Context, filter usecase.AdminBillingFilter, page, limit int) ([]*entities.DepositTransaction, int64, error) {
	return uc.depositRepo.ListAdmin(ctx, repositories.AdminDepositTransactionFilter{
		Status:    filter.Status,
		FromDate:  filter.FromDate,
		ToDate:    filter.ToDate,
		OrderCode: filter.OrderCode,
		UserID:    filter.UserID,
	}, page, limit)
}

func (uc *AdminBillingUseCase) GetTransaction(ctx context.Context, id uuid.UUID) (*entities.DepositTransaction, error) {
	tx, err := uc.depositRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// The generic repository reports a miss as (nil, nil); the admin contract
	// promises 404 for an unknown transaction (029 data-model §8), and a 200
	// with an empty body would read as "found, no data".
	if tx == nil {
		return nil, apperror.NewNotFound("Không tìm thấy giao dịch.")
	}
	return tx, nil
}

func (uc *AdminBillingUseCase) GetTransactionByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	return uc.depositRepo.GetByOrderCode(ctx, orderCode)
}
