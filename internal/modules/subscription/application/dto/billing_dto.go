package dto

import (
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
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
	PaymentUrl string `json:"paymentUrl"`
	OrderCode  int64  `json:"orderCode"`
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
	ID                 uuid.UUID `json:"id"`
	Slug               string    `json:"slug"`
	Name               string    `json:"name"`
	Price              float64   `json:"price"`
	MaxWardrobeItems   int       `json:"maxWardrobeItems"`
	MaxOutfits         int       `json:"maxOutfits"`
	AiOutfitDailyQuota int       `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int       `json:"aiChatDailyQuota"`
	DurationDays       *int      `json:"durationDays,omitempty"`
}
