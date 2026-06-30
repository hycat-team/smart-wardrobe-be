package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandFilter struct {
	Status *brandstatus.BrandStatus
	Query  *string
	Page   int
	Limit  int
}

type BrandListResult struct {
	Brands     []*entities.Brand
	TotalCount int64
}

type IBrandRepository interface {
	shared_repos.IGenericRepository[entities.Brand, uuid.UUID]
	GetBySlug(ctx context.Context, slug string) (*entities.Brand, error)
	GetActive(ctx context.Context) ([]*entities.Brand, error)
	GetBrandsForAdmin(ctx context.Context, filter BrandFilter) (*BrandListResult, error)
}

type IBrandMemberRepository interface {
	shared_repos.IGenericRepository[entities.BrandMember, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error)
	GetByBrandAndUserIDs(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) ([]*entities.BrandMember, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandMember, error)
}

type IBrandCustomerRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomer, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error)
	GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error)
	GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandCustomer, error)
}

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
	GetNearestExpiringActiveLot(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) (*entities.LoyaltyPointLot, error)
	ListByAccountID(ctx context.Context, loyaltyAccountID uuid.UUID, status *loyaltypointlotstatus.LoyaltyPointLotStatus, expiresAt *time.Time, page int, limit int) ([]*entities.LoyaltyPointLot, error)
}

type IBrandCustomerClaimRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomerClaim, uuid.UUID]
	GetByTokenHash(ctx context.Context, tokenHash string) (*entities.BrandCustomerClaim, error)
	GetActiveByCustomerID(ctx context.Context, brandCustomerID uuid.UUID, now time.Time) ([]*entities.BrandCustomerClaim, error)
	GetByCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BrandCustomerClaim, error)
}

type IBrandBenefitRepository interface {
	shared_repos.IGenericRepository[entities.BrandBenefit, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error)
	GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error)
}

type IBenefitRedemptionRepository interface {
	shared_repos.IGenericRepository[entities.BenefitRedemption, uuid.UUID]
	GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BenefitRedemption, error)
	GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.BenefitRedemption, error)
	GetActiveRedemptionByFeature(ctx context.Context, brandCustomerID uuid.UUID, featureCode string, now time.Time) (*entities.BenefitRedemption, error)
}

type IBrandConversationRepository interface {
	shared_repos.IGenericRepository[entities.BrandConversation, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandConversation, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandConversation, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.BrandConversation, error)
}

type IBrandConversationMessageRepository interface {
	shared_repos.IGenericRepository[entities.BrandConversationMessage, uuid.UUID]
	GetByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entities.BrandConversationMessage, error)
	CountUnread(ctx context.Context, conversationID uuid.UUID, senderRole string, since *time.Time) (int, error)
}

type IBrandItemRepository interface {
	shared_repos.IGenericRepository[entities.BrandItem, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error)
	GetByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) ([]*entities.BrandItem, error)
	GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error)
	GetByFashionItemID(ctx context.Context, fashionItemID uuid.UUID) (*entities.BrandItem, error)
}

type IDigitalSampleResponseRepository interface {
	shared_repos.IGenericRepository[entities.DigitalSampleResponse, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.DigitalSampleResponse, error)
	GetByBrandItemID(ctx context.Context, brandItemID uuid.UUID) ([]*entities.DigitalSampleResponse, error)
}
