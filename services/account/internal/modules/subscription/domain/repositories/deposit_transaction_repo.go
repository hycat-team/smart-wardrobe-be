package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type ClaimedDepositTransaction struct {
	Transaction       *entities.DepositTransaction
	ClaimedFromStatus depositstatus.DepositStatus
	ProcessingToken   uuid.UUID
}

// AdminDepositTransactionFilter filters deposit transactions for the admin list.
//
// FromDate/ToDate are the half-open period [FromDate, ToDate) over VN calendar
// days (data-model §3, R1): callers pass StartOfDayInVN(fromDate) and
// EndOfDayInVN(toDate), and the query compares `created_at >= FromDate AND
// created_at < ToDate`. A nil side leaves that side of the range open.
type AdminDepositTransactionFilter struct {
	Status    *depositstatus.DepositStatus
	FromDate  *time.Time
	ToDate    *time.Time
	OrderCode *int64
	UserID    *uuid.UUID
}

type IDepositTransactionRepository interface {
	shared_repos.IGenericRepository[entities.DepositTransaction, uuid.UUID]
	GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error)
	GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
	GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
	ListAdmin(ctx context.Context, filter AdminDepositTransactionFilter, page, limit int) ([]*entities.DepositTransaction, int64, error)
	HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error)
	GetActiveDirectPurchase(ctx context.Context, userID uuid.UUID) (*entities.DepositTransaction, error)
	ClaimReconciliationCandidates(ctx context.Context, limit int, lease time.Duration) ([]*ClaimedDepositTransaction, error)
	UpdateWithToken(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error)
	GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*AdminSubscriptionStatsDTO, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]RevenueBreakdownDTO, error)
}
