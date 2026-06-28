package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IBrandRepository interface {
	shared_repos.IGenericRepository[entities.Brand, uuid.UUID]
	GetBySlug(ctx context.Context, slug string) (*entities.Brand, error)
	GetActive(ctx context.Context) ([]*entities.Brand, error)
}

type IBrandMemberRepository interface {
	shared_repos.IGenericRepository[entities.BrandMember, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error)
}

type IBrandCustomerRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomer, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error)
	GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error)
	GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error)
}

type ILoyaltyProgramRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyProgram, uuid.UUID]
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
	GetByBrandCustomerIDForUpdate(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.LoyaltyAccount, error)
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.LoyaltyAccount, error)
}

type ILoyaltyPointTransactionRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyPointTransaction, uuid.UUID]
	GetByBrandAndIdempotencyKey(ctx context.Context, brandID uuid.UUID, idempotencyKey string) (*entities.LoyaltyPointTransaction, error)
	GetByLoyaltyAccountID(ctx context.Context, loyaltyAccountID uuid.UUID) ([]*entities.LoyaltyPointTransaction, error)
}

type ILoyaltyPointLotRepository interface {
	shared_repos.IGenericRepository[entities.LoyaltyPointLot, uuid.UUID]
	ListRedeemableLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error)
	ListExpiredLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error)
	UpdateLotRemainingAndStatus(ctx context.Context, lotID uuid.UUID, remainingPoints int, status loyaltypointlotstatus.LoyaltyPointLotStatus) error
	ListAccountsWithExpiredLots(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error)
}

type IBrandCustomerClaimRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomerClaim, uuid.UUID]
	GetByTokenHash(ctx context.Context, tokenHash string) (*entities.BrandCustomerClaim, error)
}
