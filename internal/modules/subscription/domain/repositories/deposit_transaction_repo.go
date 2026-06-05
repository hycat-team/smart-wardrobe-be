package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IDepositTransactionRepository interface {
	shared_repos.IGenericRepository[entities.DepositTransaction, uuid.UUID]
	GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error)
	GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
	GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
	HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error)
}
