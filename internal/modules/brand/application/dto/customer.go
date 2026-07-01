package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

type CreateOfflineBrandCustomerReq struct {
	CustomerName         *string `json:"customerName" binding:"omitempty,max=255" label:"tên khách hàng"`
	PhoneE164            string  `json:"phoneE164" binding:"required,max=50" label:"số điện thoại"`
	ExternalCustomerCode *string `json:"externalCustomerCode" binding:"omitempty,max=100" label:"mã khách hàng liên kết"`
}

type BrandCustomerRes struct {
	ID                   uuid.UUID                                           `json:"id"`
	BrandID              uuid.UUID                                           `json:"brandId"`
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
	User                 *BrandCustomerUserRes                               `json:"user,omitempty"`
	LoyaltyAccount       *CustomerLoyaltyAccountRes                          `json:"loyaltyAccount,omitempty"`
}

type BrandCustomerUserRes struct {
	ID        uuid.UUID     `json:"id"`
	Username  string        `json:"username"`
	FirstName string        `json:"firstName"`
	LastName  string        `json:"lastName,omitempty"`
	Gender    gender.Gender `json:"gender"`
	AvatarUrl *string       `json:"avatarUrl,omitempty"`
}

type CustomerLoyaltyAccountRes struct {
	ID             uuid.UUID            `json:"id"`
	CurrentPoints  int                  `json:"currentPoints"`
	LifetimePoints int                  `json:"lifetimePoints"`
	TotalSpend     float64              `json:"totalSpend"`
	CurrentTier    *LoyaltyTierBriefRes `json:"currentTier,omitempty"`
}

type LoyaltyTierBriefRes struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
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
