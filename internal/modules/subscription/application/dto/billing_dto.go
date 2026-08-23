package dto

import (
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/walletstatementtype"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	"time"

	"github.com/google/uuid"
)

type GetWalletStatementsQueryReq struct {
	shared_dto.PaginationQuery
}

type WalletDTO struct {
	UserID    uuid.UUID         `json:"userID"`
	Balance   float64           `json:"balance"`
	Currency  currency.Currency `json:"currency"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type WalletStatementDTO struct {
	ID              uuid.UUID                               `json:"id"`
	UserID          uuid.UUID                               `json:"userID"`
	Amount          float64                                 `json:"amount"`
	TransactionType walletstatementtype.WalletStatementType `json:"transactionType"`
	PreviousBalance float64                                 `json:"previousBalance"`
	NewBalance      float64                                 `json:"newBalance"`
	Description     string                                  `json:"description"`
	CreatedAt       time.Time                               `json:"createdAt"`
}

type WalletTopUpReq struct {
	Amount    float64 `json:"amount" binding:"required,gt=0" label:"số tiền nạp"`
	ReturnUrl string  `json:"returnUrl"`
	CancelUrl string  `json:"cancelUrl"`
}

type DirectPurchaseReq struct {
	PlanSlug  string `json:"planSlug" binding:"required" label:"gói hội viên"`
	ReturnUrl string `json:"returnUrl"`
	CancelUrl string `json:"cancelUrl"`
}

type PaymentLinkDTO struct {
	PaymentUrl           string                      `json:"paymentUrl"`
	OrderCode            int64                       `json:"orderCode"`
	PaymentStatus        depositstatus.DepositStatus `json:"paymentStatus"`
	ExpiresAt            *time.Time                  `json:"expiresAt,omitempty"`
	NextReconciliationAt *time.Time                  `json:"nextReconciliationAt,omitempty"`
}

type PayOSWebhookData struct {
	OrderCode           int64  `json:"orderCode"`
	Amount              int    `json:"amount"`
	Description         string `json:"description"`
	AccountNumber       string `json:"accountNumber"`
	Reference           string `json:"reference"`
	TransactionDateTime string `json:"transactionDateTime"`
	PaymentLinkId       string `json:"paymentLinkId"`
	Code                string `json:"code"`
	Desc                string `json:"desc"`
}

type PayOSWebhookReq struct {
	Code      string           `json:"code"`
	Desc      string           `json:"desc"`
	Data      PayOSWebhookData `json:"data"`
	Signature string           `json:"signature"`
}

type SubscriptionPlanDTO struct {
	ID                 uuid.UUID         `json:"id"`
	Slug               string            `json:"slug"`
	Name               string            `json:"name"`
	Price              float64           `json:"price"`
	MaxWardrobeItems   int               `json:"maxWardrobeItems"`
	MaxOutfits         int               `json:"maxOutfits"`
	AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int               `json:"aiChatDailyQuota"`
	DurationDays       *int              `json:"durationDays,omitempty"`
	PlanKind           plankind.PlanKind `json:"planKind"`
	TierRank           int               `json:"tierRank"`
	IsActive           bool              `json:"isActive"`
}

type SubscriptionPlanBriefRes struct {
	ID                 uuid.UUID         `json:"id"`
	Slug               string            `json:"slug"`
	Name               string            `json:"name"`
	Price              float64           `json:"price"`
	MaxWardrobeItems   int               `json:"maxWardrobeItems"`
	MaxOutfits         int               `json:"maxOutfits"`
	AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int               `json:"aiChatDailyQuota"`
	DurationDays       *int              `json:"durationDays,omitempty"`
	PlanKind           plankind.PlanKind `json:"planKind"`
	TierRank           int               `json:"tierRank"`
}

type CreateSubscriptionPlanReq struct {
	Slug               string            `json:"slug" binding:"required" label:"mã định danh gói"`
	Name               string            `json:"name" binding:"required" label:"tên gói"`
	Price              float64           `json:"price" binding:"gte=0" label:"giá gói"`
	MaxWardrobeItems   int               `json:"maxWardrobeItems" binding:"gte=0" label:"số đồ tối đa"`
	MaxOutfits         int               `json:"maxOutfits" binding:"gte=0" label:"số trang phục tối đa"`
	AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota" binding:"gte=0" label:"hạn mức AI outfit mỗi ngày"`
	AiChatDailyQuota   int               `json:"aiChatDailyQuota" binding:"gte=0" label:"hạn mức AI chat mỗi ngày"`
	DurationDays       *int              `json:"durationDays" label:"thời hạn gói (ngày)"`
	PlanKind           plankind.PlanKind `json:"planKind" label:"loại gói"`
	TierRank           int               `json:"tierRank" label:"cấp bậc gói"`
	IsActive           *bool             `json:"isActive" label:"trạng thái hoạt động"`
}

type UpdateSubscriptionPlanReq struct {
	Name               *string  `json:"name" label:"tên gói"`
	Price              *float64 `json:"price" label:"giá gói"`
	MaxWardrobeItems   *int     `json:"maxWardrobeItems" label:"số đồ tối đa"`
	MaxOutfits         *int     `json:"maxOutfits" label:"số trang phục tối đa"`
	AiOutfitDailyQuota *int     `json:"aiOutfitDailyQuota" label:"hạn mức AI outfit mỗi ngày"`
	AiChatDailyQuota   *int     `json:"aiChatDailyQuota" label:"hạn mức AI chat mỗi ngày"`
	DurationDays       *int     `json:"durationDays" label:"thời hạn gói (ngày)"`
	TierRank           *int     `json:"tierRank" label:"cấp bậc gói"`
	IsActive           *bool    `json:"isActive" label:"trạng thái hoạt động"`
}

type SetPlanActiveReq struct {
	IsActive bool `json:"isActive" label:"trạng thái hoạt động"`
}

type AdminDepositTransactionListRes = shared_dto.PaginationResult[*DepositTransactionRes]

type DepositTransactionRes struct {
	ID                   uuid.UUID                   `json:"id"`
	UserID               uuid.UUID                   `json:"userId"`
	OrderCode            int64                       `json:"orderCode"`
	Amount               float64                     `json:"amount"`
	Currency             string                      `json:"currency"`
	Status               depositstatus.DepositStatus `json:"status"`
	TransactionType      string                      `json:"transactionType"`
	TargetPlanID         *uuid.UUID                  `json:"targetPlanId,omitempty"`
	PaymentUrl           *string                     `json:"paymentUrl,omitempty"`
	Reference            *string                     `json:"reference,omitempty"`
	PaymentLinkId        *string                     `json:"paymentLinkId,omitempty"`
	FailureReason        *string                     `json:"failureReason,omitempty"`
	CancelledAt          *time.Time                  `json:"cancelledAt,omitempty"`
	NextReconciliationAt *time.Time                  `json:"nextReconciliationAt,omitempty"`
	CreatedAt            time.Time                   `json:"createdAt"`
	UpdatedAt            time.Time                   `json:"updatedAt"`
}

// PurchaseWithWalletReq dùng cho endpoint mua gói bằng ví nội bộ
type PurchaseWithWalletReq struct {
	PlanSlug string `json:"planSlug" binding:"required" label:"gói hội viên"`
}
