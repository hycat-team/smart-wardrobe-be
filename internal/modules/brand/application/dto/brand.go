package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltyroundingmode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"

	"github.com/google/uuid"
)

type UploadSignatureResult = shared_dto.UploadSignatureResult

type CreateBrandReq struct {
	Slug         string  `json:"slug" binding:"required,max=100" label:"slug thương hiệu"`
	Name         string  `json:"name" binding:"required,max=255" label:"tên thương hiệu"`
	Description  *string `json:"description" binding:"omitempty" label:"mô tả"`
	LogoURL      *string `json:"logoUrl" binding:"omitempty,url" label:"logo thương hiệu"`
	LogoPublicID *string `json:"logoPublicId" binding:"omitempty,max=255" label:"mã ảnh logo thương hiệu"`
}

type UpdateBrandLogoReq struct {
	LogoURL      string `json:"logoUrl" binding:"required,url" label:"logo thương hiệu"`
	LogoPublicID string `json:"logoPublicId" binding:"required,max=255" label:"mã ảnh logo thương hiệu"`
}

type UpdateBrandStatusReq struct {
	Status brandstatus.BrandStatus `json:"status" binding:"required" label:"trạng thái thương hiệu"`
}

type AddBrandMembersReq struct {
	Members []AddBrandMemberItemReq `json:"members" binding:"required,min=1,max=50,dive" label:"danh sách thành viên"`
}

type AddBrandMemberItemReq struct {
	EmailOrUsername string                          `json:"emailOrUsername" binding:"required,max=255" label:"email hoặc tên đăng nhập"`
	Role            brandmemberrole.BrandMemberRole `json:"role" binding:"required" label:"vai trò thành viên"`
}

type AddBrandMemberItemResult struct {
	EmailOrUsername string          `json:"emailOrUsername"`
	Member          *BrandMemberRes `json:"member,omitempty"`
	ReasonCode      string          `json:"reasonCode,omitempty"`
	Message         string          `json:"message,omitempty"`
}

type AddBrandMembersRes struct {
	Created []AddBrandMemberItemResult `json:"created"`
	Updated []AddBrandMemberItemResult `json:"updated"`
	Failed  []AddBrandMemberItemResult `json:"failed"`
}

type CreateOfflineBrandCustomerReq struct {
	CustomerName         *string `json:"customerName" binding:"omitempty,max=255" label:"tên khách hàng"`
	PhoneE164            string  `json:"phoneE164" binding:"required,max=50" label:"số điện thoại"`
	ExternalCustomerCode *string `json:"externalCustomerCode" binding:"omitempty,max=100" label:"mã khách hàng liên kết"`
}

type GrantLoyaltyPointsReq struct {
	UserID               *uuid.UUID                                    `json:"userId" binding:"omitempty" label:"mã người dùng"`
	Phone                *string                                       `json:"phone" binding:"omitempty,max=50" label:"số điện thoại"`
	CustomerName         *string                                       `json:"customerName" binding:"omitempty,max=255" label:"tên khách hàng"`
	ExternalCustomerCode *string                                       `json:"externalCustomerCode" binding:"omitempty,max=100" label:"mã khách hàng liên kết"`
	PurchaseAmount       *float64                                      `json:"purchaseAmount" binding:"omitempty,min=0" label:"giá trị mua hàng"`
	PointsDelta          *int                                          `json:"pointsDelta" binding:"omitempty" label:"điểm thay đổi"`
	TransactionType      loyaltytransactiontype.LoyaltyTransactionType `json:"transactionType" binding:"required" label:"loại giao dịch"`
	Reason               *string                                       `json:"reason" binding:"omitempty,max=255" label:"lý do"`
	ReferenceType        *string                                       `json:"referenceType" binding:"omitempty,max=100" label:"loại tham chiếu"`
	ReferenceID          *uuid.UUID                                    `json:"referenceId" binding:"omitempty" label:"mã tham chiếu"`
	IdempotencyKey       *string                                       `json:"idempotencyKey" binding:"omitempty,max=100" label:"khóa idempotency"`
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

type PortalBrandRes struct {
	BrandRes
	MemberID     uuid.UUID                           `json:"memberId"`
	MemberRole   brandmemberrole.BrandMemberRole     `json:"memberRole"`
	MemberStatus brandmemberstatus.BrandMemberStatus `json:"memberStatus"`
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

type UpsertLoyaltyProgramReq struct {
	Name            string                                  `json:"name" binding:"required,max=255" label:"tên chương trình"`
	AmountPerPoint  float64                                 `json:"amountPerPoint" binding:"required,gt=0" label:"số tiền trên mỗi điểm"`
	PointExpiryDays *int                                    `json:"pointExpiryDays" binding:"omitempty,min=0" label:"số ngày hết hạn điểm"`
	RoundingMode    loyaltyroundingmode.LoyaltyRoundingMode `json:"roundingMode" binding:"required,oneof=floor round ceil" label:"chế độ làm tròn"`
	IsActive        *bool                                   `json:"isActive" binding:"omitempty" label:"trạng thái hoạt động"`
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
	ID                uuid.UUID  `json:"id"`
	EarnedPoints      int        `json:"earnedPoints"`
	RemainingPoints   int        `json:"remainingPoints"`
	ExpiresAt         *time.Time `json:"expiresAt"`
	Status            string     `json:"status"`
	EarnTransactionID uuid.UUID  `json:"earnTransactionId,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
}

type ListLoyaltyPointLotsQueryReq struct {
	Status    *string    `form:"status" binding:"omitempty" label:"trạng thái lô điểm"`
	ExpiresAt *time.Time `form:"expiresAt" binding:"omitempty" label:"ngày hết hạn"`
	Page      int        `form:"page" binding:"omitempty,min=1" label:"trang"`
	Limit     int        `form:"limit" binding:"omitempty,min=1,max=100" label:"số lượng"`
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
	Name           string      `json:"name" binding:"required,max=255" label:"tên quyền lợi"`
	Description    *string     `json:"description" binding:"omitempty" label:"mô tả"`
	BenefitType    string      `json:"benefitType" binding:"required" label:"loại quyền lợi"`
	UnlockType     string      `json:"unlockType" binding:"required" label:"loại mở khóa"`
	RequiredPoints *int        `json:"requiredPoints" binding:"omitempty,min=0" label:"điểm yêu cầu"`
	RequiredTierID *uuid.UUID  `json:"requiredTierId" binding:"omitempty" label:"mã hạng yêu cầu"`
	FeatureCode    *string     `json:"featureCode" binding:"omitempty,max=100" label:"mã tính năng"`
	FeatureConfig  interface{} `json:"featureConfig" binding:"omitempty" label:"cấu hình tính năng"`
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
	Message string `json:"message" binding:"required,min=1" label:"nội dung tin nhắn"`
}

type BrandConversationRes struct {
	ID               uuid.UUID  `json:"id"`
	BrandID          uuid.UUID  `json:"brandId"`
	UserID           uuid.UUID  `json:"userId"`
	CustomerName     *string    `json:"customerName"`
	UserDisplayName  *string    `json:"userDisplayName"`
	Status           string     `json:"status"`
	LastMessageAt    *time.Time `json:"lastMessageAt"`
	UserLastReadAt   *time.Time `json:"userLastReadAt"`
	StaffLastReadAt  *time.Time `json:"staffLastReadAt"`
	UserUnreadCount  int        `json:"userUnreadCount"`
	StaffUnreadCount int        `json:"staffUnreadCount"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
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
	CategoryID    *uuid.UUID `json:"categoryId" binding:"omitempty" label:"mã danh mục"`
	ImageUrl      string     `json:"imageUrl" binding:"required,url" label:"đường dẫn hình ảnh"`
	ImagePublicID string     `json:"imagePublicId" binding:"required" label:"mã hình ảnh"`
	ProductCode   *string    `json:"productCode" binding:"omitempty,max=100" label:"mã sản phẩm"`
	Name          string     `json:"name" binding:"required,max=255" label:"tên sản phẩm"`
	Description   *string    `json:"description" binding:"omitempty" label:"mô tả"`
	Price         *float64   `json:"price" binding:"omitempty,gt=0" label:"giá sản phẩm"`
	ItemType      string     `json:"itemType" binding:"required" label:"loại sản phẩm"` // E.g. "BRAND_RETAIL" or "DIGITAL_SAMPLE"
	Status        string     `json:"status" binding:"omitempty" label:"trạng thái"`     // DRAFT, ACTIVE, ARCHIVED
}

type UpdateBrandItemReq struct {
	Name        string   `json:"name" binding:"required,max=255" label:"tên sản phẩm"`
	Description *string  `json:"description" binding:"omitempty" label:"mô tả"`
	Price       *float64 `json:"price" binding:"omitempty,gt=0" label:"giá sản phẩm"`
	Status      string   `json:"status" binding:"required" label:"trạng thái"` // DRAFT, ACTIVE, ARCHIVED
}

type UpdateBrandItemStatusReq struct {
	Status string `json:"status" binding:"required" label:"trạng thái sản phẩm"`
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
	OutfitID     *uuid.UUID `json:"outfitId" binding:"omitempty" label:"mã trang phục"`
	VoteType     *string    `json:"voteType" binding:"omitempty" label:"loại bình chọn"` // like, dislike, would_buy, not_interested
	Rating       *int       `json:"rating" binding:"omitempty,min=1,max=5" label:"đánh giá"`
	FeedbackText *string    `json:"feedbackText" binding:"omitempty" label:"nội dung phản hồi"`
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

type ClaimTokenRes struct {
	ID              uuid.UUID  `json:"id"`
	BrandCustomerID uuid.UUID  `json:"brandCustomerId"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	ConsumedAt      *time.Time `json:"consumedAt"`
	RevokedAt       *time.Time `json:"revokedAt"`
	RevokedByUserID *uuid.UUID `json:"revokedByUserId"`
	RevokedReason   *string    `json:"revokedReason"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type RevokeClaimTokenReq struct {
	Reason *string `json:"reason" binding:"omitempty,max=255" label:"lý do thu hồi"`
}

type ClaimOfflineAccountReq struct {
	ClaimToken string `json:"claimToken" binding:"required" label:"mã nhận tài khoản"`
}

type GetBrandsAdminQueryReq struct {
	shared_dto.PaginationQuery
	Status *brandstatus.BrandStatus `form:"status" binding:"omitempty" label:"trạng thái"`
	Query  *string                  `form:"q" binding:"omitempty" label:"từ khóa tìm kiếm"`
}

type AdminBrandListRes = shared_dto.PaginationResult[*BrandRes]
