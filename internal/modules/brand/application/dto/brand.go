package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"

	"github.com/google/uuid"
)

type UploadSignatureResult = shared_dto.UploadSignatureResult

type CreateBrandReq struct {
	Slug         string  `json:"slug" binding:"required,max=100" label:"slug brand"`
	Name         string  `json:"name" binding:"required,max=255" label:"ten brand"`
	Description  *string `json:"description" binding:"omitempty" label:"mo ta"`
	LogoURL      *string `json:"logoUrl" binding:"omitempty,url" label:"logo brand"`
	LogoPublicID *string `json:"logoPublicId" binding:"omitempty,max=255" label:"cloudinary public id logo brand"`
}

type UpdateBrandLogoReq struct {
	LogoURL      string `json:"logoUrl" binding:"required,url" label:"logo brand"`
	LogoPublicID string `json:"logoPublicId" binding:"required,max=255" label:"cloudinary public id logo brand"`
}

type UpdateBrandStatusReq struct {
	Status brandstatus.BrandStatus `json:"status" binding:"required" label:"trang thai brand"`
}

type AddBrandMemberReq struct {
	UserID uuid.UUID                       `json:"userId" binding:"required" label:"ma user"`
	Role   brandmemberrole.BrandMemberRole `json:"role" binding:"required" label:"vai tro brand"`
}

type CreateOfflineBrandCustomerReq struct {
	CustomerName         *string `json:"customerName" binding:"omitempty,max=255"`
	PhoneE164            string  `json:"phoneE164" binding:"required,max=50"`
	ExternalCustomerCode *string `json:"externalCustomerCode" binding:"omitempty,max=100"`
}

type GrantLoyaltyPointsReq struct {
	UserID               *uuid.UUID                                    `json:"userId" binding:"omitempty" label:"ma user"`
	Phone                *string                                       `json:"phone" binding:"omitempty,max=50" label:"so dien thoai"`
	CustomerName         *string                                       `json:"customerName" binding:"omitempty,max=255" label:"ten khach hang"`
	ExternalCustomerCode *string                                       `json:"externalCustomerCode" binding:"omitempty,max=100" label:"ma khach hang ngoai"`
	PurchaseAmount       *float64                                      `json:"purchaseAmount" binding:"omitempty,min=0" label:"gia tri mua hang"`
	PointsDelta          *int                                          `json:"pointsDelta" binding:"omitempty" label:"diem thay doi"`
	TransactionType      loyaltytransactiontype.LoyaltyTransactionType `json:"transactionType" binding:"required" label:"loai giao dich"`
	Reason               *string                                       `json:"reason" binding:"omitempty,max=255" label:"ly do"`
	ReferenceType        *string                                       `json:"referenceType" binding:"omitempty,max=100" label:"loai tham chieu"`
	ReferenceID          *uuid.UUID                                    `json:"referenceId" binding:"omitempty" label:"ma tham chieu"`
	IdempotencyKey       *string                                       `json:"idempotencyKey" binding:"omitempty,max=100" label:"khoa idempotency"`
}

type BrandRes struct {
	ID               uuid.UUID               `json:"id"`
	Slug             string                  `json:"slug"`
	Name             string                  `json:"name"`
	Description      *string                 `json:"description"`
	LogoURL          *string                 `json:"logoUrl"`
	LogoPublicID     *string                 `json:"logoPublicId"`
	Status           brandstatus.BrandStatus `json:"status"`
	CreatedByUserID  uuid.UUID               `json:"createdByUserId"`
	ApprovedByUserID *uuid.UUID              `json:"approvedByUserId"`
	ApprovedAt       *time.Time              `json:"approvedAt"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
}

type BrandMemberRes struct {
	ID        uuid.UUID                           `json:"id"`
	BrandID   uuid.UUID                           `json:"brandId"`
	UserID    uuid.UUID                           `json:"userId"`
	Role      brandmemberrole.BrandMemberRole     `json:"role"`
	Status    brandmemberstatus.BrandMemberStatus `json:"status"`
	CreatedAt time.Time                           `json:"createdAt"`
	UpdatedAt time.Time                           `json:"updatedAt"`
}

type BrandCustomerRes struct {
	ID                   uuid.UUID                                           `json:"id"`
	BrandID              uuid.UUID                                           `json:"brandId"`
	UserID               *uuid.UUID                                          `json:"userId"`
	CustomerName         *string                                             `json:"customerName"`
	PhoneE164            *string                                             `json:"phoneE164"`
	ExternalCustomerCode *string                                             `json:"externalCustomerCode"`
	JoinedSource         brandcustomerjoinedsource.BrandCustomerJoinedSource `json:"joinedSource"`
	Status               brandcustomerstatus.BrandCustomerStatus             `json:"status"`
	JoinedAt             time.Time                                           `json:"joinedAt"`
	ClaimedAt            *time.Time                                          `json:"claimedAt"`
	CreatedByMemberID    *uuid.UUID                                          `json:"createdByMemberId"`
	CreatedAt            time.Time                                           `json:"createdAt"`
	UpdatedAt            time.Time                                           `json:"updatedAt"`
}

type LoyaltyTierBriefRes struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type LoyaltyPointsTransactionRes struct {
	TransactionID   uuid.UUID                               `json:"transactionId"`
	BrandID         uuid.UUID                               `json:"brandId"`
	BrandCustomerID uuid.UUID                               `json:"brandCustomerId"`
	UserID          *uuid.UUID                              `json:"userId"`
	CustomerStatus  brandcustomerstatus.BrandCustomerStatus `json:"customerStatus"`
	PointsDelta     int                                     `json:"pointsDelta"`
	BalanceAfter    int                                     `json:"balanceAfter"`
	TotalSpend      float64                                 `json:"totalSpend"`
	CurrentTier     *LoyaltyTierBriefRes                    `json:"currentTier"`
}

type LoyaltyProgramRes struct {
	ID              uuid.UUID `json:"id"`
	BrandID         uuid.UUID `json:"brandId"`
	Name            string    `json:"name"`
	AmountPerPoint  float64   `json:"amountPerPoint"`
	PointExpiryDays *int      `json:"pointExpiryDays"`
	RoundingMode    string    `json:"roundingMode"`
	IsActive        bool      `json:"isActive"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type LoyaltyTierRes struct {
	ID            uuid.UUID `json:"id"`
	BrandID       uuid.UUID `json:"brandId"`
	Name          string    `json:"name"`
	Rank          int       `json:"rank"`
	MinTotalSpend float64   `json:"minTotalSpend"`
	Description   *string   `json:"description"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type LoyaltyPointLotRes struct {
	ID              uuid.UUID  `json:"id"`
	EarnedPoints    int        `json:"earnedPoints"`
	RemainingPoints int        `json:"remainingPoints"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	Status          string     `json:"status"`
}

type BrandLoyaltyRes struct {
	BrandID                 uuid.UUID            `json:"brandId"`
	Brand                   *BrandRes            `json:"brand,omitempty"`
	BrandCustomerID         uuid.UUID            `json:"brandCustomerId"`
	LoyaltyAccountID        uuid.UUID            `json:"loyaltyAccountId"`
	CurrentPoints           int                  `json:"currentPoints"`
	LifetimePoints          int                  `json:"lifetimePoints"`
	TotalSpend              float64              `json:"totalSpend"`
	CurrentTier             *LoyaltyTierBriefRes `json:"currentTier"`
	NearestExpiringPointLot *LoyaltyPointLotRes  `json:"nearestExpiringPointLot"`
}

type LoyaltyPointTransactionDetailRes struct {
	ID               uuid.UUID  `json:"id"`
	LoyaltyAccountID uuid.UUID  `json:"loyaltyAccountId"`
	BrandID          uuid.UUID  `json:"brandId"`
	BrandCustomerID  uuid.UUID  `json:"brandCustomerId"`
	UserID           *uuid.UUID `json:"userId"`
	PointsDelta      int        `json:"pointsDelta"`
	BalanceAfter     int        `json:"balanceAfter"`
	TransactionType  string     `json:"transactionType"`
	Reason           *string    `json:"reason"`
	SpendAmount      *float64   `json:"spendAmount"`
	ReferenceType    *string    `json:"referenceType"`
	ReferenceID      *uuid.UUID `json:"referenceId"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	IdempotencyKey   *string    `json:"idempotencyKey"`
	CreatedByUserID  *uuid.UUID `json:"createdByUserId"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type CreateBrandBenefitReq struct {
	Name           string      `json:"name" binding:"required,max=255" label:"ten quyen loi"`
	Description    *string     `json:"description" binding:"omitempty" label:"mo ta"`
	BenefitType    string      `json:"benefitType" binding:"required" label:"loai quyen loi"`
	UnlockType     string      `json:"unlockType" binding:"required" label:"loai mo khoa"`
	RequiredPoints *int        `json:"requiredPoints" binding:"omitempty,min=0" label:"diem yeu cau"`
	RequiredTierID *uuid.UUID  `json:"requiredTierId" binding:"omitempty" label:"ma hang yeu cau"`
	FeatureCode    *string     `json:"featureCode" binding:"omitempty,max=100" label:"ma tinh nang"`
	FeatureConfig  interface{} `json:"featureConfig" binding:"omitempty" label:"cau hinh tinh nang"`
}

type UpdateBenefitStatusReq struct {
	Status string `json:"status" binding:"required" label:"trang thai"`
}

type BrandBenefitRes struct {
	ID             uuid.UUID   `json:"id"`
	BrandID        uuid.UUID   `json:"brandId"`
	Name           string      `json:"name"`
	Description    *string     `json:"description"`
	BenefitType    string      `json:"benefitType"`
	UnlockType     string      `json:"unlockType"`
	RequiredPoints *int        `json:"requiredPoints"`
	RequiredTierID *uuid.UUID  `json:"requiredTierId"`
	FeatureCode    *string     `json:"featureCode"`
	FeatureConfig  interface{} `json:"featureConfig"`
	Status         string      `json:"status"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

type BenefitRedemptionRes struct {
	ID              uuid.UUID  `json:"id"`
	BenefitID       uuid.UUID  `json:"benefitId"`
	BrandID         uuid.UUID  `json:"brandId"`
	BrandCustomerID uuid.UUID  `json:"brandCustomerId"`
	UserID          *uuid.UUID `json:"userId"`
	PointsSpent     int        `json:"pointsSpent"`
	Status          string     `json:"status"`
	RedeemedAt      time.Time  `json:"redeemedAt"`
	UsedAt          *time.Time `json:"usedAt"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type SendBrandChatMessageReq struct {
	Message string `json:"message" binding:"required,min=1" label:"noi dung tin nhan"`
}

type BrandConversationRes struct {
	ID              uuid.UUID  `json:"id"`
	BrandID         uuid.UUID  `json:"brandId"`
	UserID          uuid.UUID  `json:"userId"`
	CustomerName    *string    `json:"customerName"`
	UserDisplayName *string    `json:"userDisplayName"`
	Status          string     `json:"status"`
	LastMessageAt   *time.Time `json:"lastMessageAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type BrandConversationMessageRes struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversationId"`
	SenderRole     string     `json:"senderRole"`
	SenderUserID   *uuid.UUID `json:"senderUserId"`
	Message        string     `json:"message"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type CreateBrandItemReq struct {
	CategoryID    *uuid.UUID `json:"categoryId" binding:"omitempty"`
	ImageUrl      string     `json:"imageUrl" binding:"required,url"`
	ImagePublicID string     `json:"imagePublicId" binding:"required"`
	ProductCode   *string    `json:"productCode" binding:"omitempty,max=100"`
	Name          string     `json:"name" binding:"required,max=255"`
	Description   *string    `json:"description" binding:"omitempty"`
	Price         *float64   `json:"price" binding:"omitempty,gt=0"`
	ItemType      string     `json:"itemType" binding:"required"` // E.g. "BRAND_RETAIL" or "DIGITAL_SAMPLE"
	Status        string     `json:"status" binding:"omitempty"`  // DRAFT, ACTIVE, ARCHIVED
}

type UpdateBrandItemReq struct {
	Name        string   `json:"name" binding:"required,max=255"`
	Description *string  `json:"description" binding:"omitempty"`
	Price       *float64 `json:"price" binding:"omitempty,gt=0"`
	Status      string   `json:"status" binding:"required"` // DRAFT, ACTIVE, ARCHIVED
}

type UpdateBrandItemStatusReq struct {
	Status string `json:"status" binding:"required" label:"trang thai item"`
}

type BrandItemRes struct {
	ID            uuid.UUID `json:"id"`
	BrandID       uuid.UUID `json:"brandId"`
	FashionItemID uuid.UUID `json:"fashionItemId"`
	ProductCode   *string   `json:"productCode"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	Price         *float64  `json:"price"`
	ItemType      string    `json:"itemType"`
	Status        string    `json:"status"`
	FashionItem   any       `json:"fashionItem,omitempty"` // Detailed fashion metadata
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SubmitSampleFeedbackReq struct {
	OutfitID     *uuid.UUID `json:"outfitId" binding:"omitempty"`
	VoteType     *string    `json:"voteType" binding:"omitempty"` // UP, DOWN
	Rating       *int       `json:"rating" binding:"omitempty,min=1,max=5"`
	FeedbackText *string    `json:"feedbackText" binding:"omitempty"`
}

type DigitalSampleResponseRes struct {
	ID           uuid.UUID  `json:"id"`
	BrandItemID  uuid.UUID  `json:"brandItemId"`
	UserID       uuid.UUID  `json:"userId"`
	OutfitID     *uuid.UUID `json:"outfitId"`
	VoteType     *string    `json:"voteType"`
	Rating       *int       `json:"rating"`
	FeedbackText *string    `json:"feedbackText"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type CreateClaimTokenRes struct {
	ClaimToken string    `json:"claimToken"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type ClaimOfflineAccountReq struct {
	ClaimToken string `json:"claimToken" binding:"required"`
}
