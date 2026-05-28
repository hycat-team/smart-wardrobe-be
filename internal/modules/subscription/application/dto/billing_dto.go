package dto

import (
	"time"

	"github.com/google/uuid"
)

type WalletDTO struct {
	UserID    uuid.UUID `json:"userID"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type WalletStatementDTO struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userID"`
	Amount          float64   `json:"amount"`
	TransactionType string    `json:"transactionType"`
	PreviousBalance float64   `json:"previousBalance"`
	NewBalance      float64   `json:"newBalance"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"createdAt"`
}

type WalletTopUpReq struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	ReturnUrl string  `json:"returnUrl"`
	CancelUrl string  `json:"cancelUrl"`
}

type DirectPurchaseReq struct {
	SubscriptionPlanID uuid.UUID `json:"subscriptionPlanID" binding:"required"`
	ReturnUrl          string    `json:"returnUrl"`
	CancelUrl          string    `json:"cancelUrl"`
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
