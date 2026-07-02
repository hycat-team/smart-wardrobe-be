package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"

	"github.com/google/uuid"
)

type IBrandLoyaltyUseCase interface {
	GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandCustomersQueryReq) (*dto.BrandCustomerListRes, error)
	GetBrandCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.BrandCustomerRes, error)
	JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerRes, error)
	CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error)
	GrantLoyaltyPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*dto.LoyaltyPointsTransactionRes, error)
	ListUserBrandLoyalties(ctx context.Context, userID uuid.UUID) ([]*dto.BrandLoyaltyRes, error)
	GetUserBrandLoyalty(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandLoyaltyRes, error)
	GetUserBrandLoyaltyTransactions(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyPointTransactionDetailRes, error)
	GetUserBrandLoyaltyLots(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.ListLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error)
	GetLoyaltyAccountTransactionsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID, query dto.GetLoyaltyTransactionsQueryReq) (*dto.LoyaltyTransactionListRes, error)
	GetLoyaltyAccountLotsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID, query dto.ListLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error)
	GetLoyaltyProgramForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) (*dto.LoyaltyProgramRes, error)
	UpsertLoyaltyProgram(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.UpsertLoyaltyProgramReq) (*dto.LoyaltyProgramRes, error)
	GetLoyaltyTiersForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyTierRes, error)
	CreateLoyaltyTier(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateLoyaltyTierReq) (*dto.LoyaltyTierRes, error)
	UpdateLoyaltyTier(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, tierID uuid.UUID, input dto.UpdateLoyaltyTierReq) (*dto.LoyaltyTierRes, error)
	ProcessExpiredLoyaltyPointLots(ctx context.Context, now time.Time, batchSize int) (int, error)
}
