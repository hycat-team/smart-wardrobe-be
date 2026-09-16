package admin

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

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
	return uc.depositRepo.GetByID(ctx, id)
}

func (uc *AdminBillingUseCase) GetTransactionByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	return uc.depositRepo.GetByOrderCode(ctx, orderCode)
}
