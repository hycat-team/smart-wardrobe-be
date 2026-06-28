package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type IBrandCoreUseCase interface {
	CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error)
	GetActiveBrands(ctx context.Context) ([]*dto.BrandRes, error)
	GetActiveBrand(ctx context.Context, brandID uuid.UUID) (*dto.BrandRes, error)
	GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandRes, error)
	GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.BrandRes, error)
	GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	GetBrandItemUploadSignature(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	UpdateBrandLogo(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandLogoReq) (*dto.BrandRes, error)
	AddBrandMember(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMemberReq) (*dto.BrandMemberRes, error)
	GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error)
	GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandCustomerRes, error)
	GetBrandCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.BrandCustomerRes, error)
	JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerRes, error)
	CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error)
	GrantLoyaltyPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*dto.LoyaltyPointsTransactionRes, error)
	ListUserBrandLoyalties(ctx context.Context, userID uuid.UUID) ([]*dto.BrandLoyaltyRes, error)
	GetUserBrandLoyalty(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandLoyaltyRes, error)
	GetUserBrandLoyaltyTransactions(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyPointTransactionDetailRes, error)
	GetLoyaltyAccountTransactionsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID) ([]*dto.LoyaltyPointTransactionDetailRes, error)
	GetLoyaltyProgramForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) (*dto.LoyaltyProgramRes, error)
	GetLoyaltyTiersForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyTierRes, error)
	ProcessExpiredLoyaltyPointLots(ctx context.Context, now time.Time, batchSize int) (int, error)
	RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error
	CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error)
	ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error)
	ListActiveBenefitsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error)
	GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error)
	ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID) ([]*dto.BenefitRedemptionRes, error)
	RedeemBenefit(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error)
	CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error)
	UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error)
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, filter interface{}) (interface{}, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *entities.BrandItem, error)
	GetUserConversation(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandConversationRes, error)
	SendUserMessage(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error)
	ListBrandConversations(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandConversationRes, error)
	ListConversationMessages(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) ([]*dto.BrandConversationMessageRes, error)
	SendStaffMessage(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error)

	CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error)
	GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error)
	GetBrandItemForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error)
	GetBrandItemForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error)
	UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error)
	UpdateBrandItemStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, status string) (*dto.BrandItemRes, error)
	GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error)
	ListBrandItemsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error)
	SubmitSampleFeedback(ctx context.Context, userID uuid.UUID, brandItemID uuid.UUID, input dto.SubmitSampleFeedbackReq) (*dto.DigitalSampleResponseRes, error)
	CreateBrandCustomerClaim(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.CreateClaimTokenRes, error)
	ClaimBrandCustomer(ctx context.Context, userID uuid.UUID, claimToken string) (*dto.BrandCustomerRes, error)
}
