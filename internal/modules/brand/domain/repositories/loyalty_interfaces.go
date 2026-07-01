package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type ILoyaltyProgramRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyProgram, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) (*entities.LoyaltyProgram, error)
	GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) (*entities.LoyaltyProgram, error)
}

type ILoyaltyTierRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyTier, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.LoyaltyTier, error)
	GetHighestEligibleBySpend(ctx context.Context, brandID uuid.UUID, totalSpend float64) (*entities.LoyaltyTier, error)
}

type ILoyaltyAccountRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyAccount, uuid.UUID]
	GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error)
	GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.LoyaltyAccount, error)
	GetByBrandCustomerIDForUpdate(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.LoyaltyAccount, error)
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.LoyaltyAccount, error)
}

type LoyaltyTransactionFilter struct {
	LoyaltyAccountID uuid.UUID
	Page             int
	Limit            int
}

type LoyaltyTransactionListResult struct {
	Transactions []*entities.LoyaltyPointTransaction
	TotalCount   int64
}

type ILoyaltyPointTransactionRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyPointTransaction, uuid.UUID]
	GetByBrandAndIdempotencyKey(ctx context.Context, brandID uuid.UUID, idempotencyKey string) (*entities.LoyaltyPointTransaction, error)
	GetByLoyaltyAccountID(ctx context.Context, loyaltyAccountID uuid.UUID) ([]*entities.LoyaltyPointTransaction, error)
	GetByLoyaltyAccountIDPaginated(ctx context.Context, filter LoyaltyTransactionFilter) (*LoyaltyTransactionListResult, error)
}

type ILoyaltyPointLotRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyPointLot, uuid.UUID]
	ListRedeemableLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error)
	ListExpiredLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error)
	UpdateLotRemainingAndStatus(ctx context.Context, lotID uuid.UUID, remainingPoints int, status loyaltypointlotstatus.LoyaltyPointLotStatus) error
	ListAccountsWithExpiredLots(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error)
	GetNearestExpiringActiveLot(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) (*entities.LoyaltyPointLot, error)
	ListByAccountID(ctx context.Context, loyaltyAccountID uuid.UUID, status *loyaltypointlotstatus.LoyaltyPointLotStatus, expiresAt *time.Time, page int, limit int) ([]*entities.LoyaltyPointLot, error)
}
