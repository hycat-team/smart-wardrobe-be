package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"time"

	"github.com/google/uuid"
)

type ClaimedDepositTransaction struct {
	Transaction       *entities.DepositTransaction
	ClaimedFromStatus depositstatus.DepositStatus
	ProcessingToken   uuid.UUID
}

type IDepositTransactionRepository interface {
	shared_repos.IGenericRepository[entities.DepositTransaction, uuid.UUID]
	GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error)
	GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
	GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
	HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error)
	GetActiveDirectPurchase(ctx context.Context, userID uuid.UUID) (*entities.DepositTransaction, error)
	ClaimReconciliationCandidates(ctx context.Context, limit int, lease time.Duration) ([]*ClaimedDepositTransaction, error)
	UpdateWithToken(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error)
}
