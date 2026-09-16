package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// AdminBillingFilter filters deposit transactions for the admin list endpoint.
type AdminBillingFilter struct {
	Status    *depositstatus.DepositStatus
	FromDate  *time.Time
	ToDate    *time.Time
	OrderCode *int64
	UserID    *uuid.UUID
}

type IAdminBillingUseCase interface {
	ListTransactions(ctx context.Context, filter AdminBillingFilter, page, limit int) ([]*entities.DepositTransaction, int64, error)
	GetTransaction(ctx context.Context, id uuid.UUID) (*entities.DepositTransaction, error)
	GetTransactionByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
}
