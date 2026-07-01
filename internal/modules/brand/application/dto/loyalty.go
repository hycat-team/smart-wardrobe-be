package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltyroundingmode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"

	"github.com/google/uuid"
)

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
