package payment

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type CheckoutSessionReq struct {
	OrderCode   int64
	Amount      decimal.Decimal
	Description string
	ReturnUrl   string
	CancelUrl   string
	ExpiresAt   time.Time
}

type GatewayOutcome string

const (
	OutcomeSucceeded    GatewayOutcome = "SUCCEEDED"
	OutcomeKnownFailure GatewayOutcome = "KNOWN_FAILURE"
	OutcomeUnknown      GatewayOutcome = "UNKNOWN"
)

type CheckoutSessionResult struct {
	CheckoutURL   string
	PaymentLinkID string
	Outcome       GatewayOutcome
	Retryable     bool
	ErrorCode     string
}

type ProviderPaymentStatus string

const (
	ProviderPending   ProviderPaymentStatus = "PENDING"
	ProviderPaid      ProviderPaymentStatus = "PAID"
	ProviderCancelled ProviderPaymentStatus = "CANCELLED"
	ProviderUnknown   ProviderPaymentStatus = "UNKNOWN"
)

type PaymentLinkInfo struct {
	OrderCode     int64
	PaymentLinkID string
	Reference     string
	Amount        decimal.Decimal
	AmountPaid    decimal.Decimal
	Currency      string
	Status        ProviderPaymentStatus
	CheckoutURL   string
}

type IPaymentGatewayService interface {
	CreateCheckoutSession(ctx context.Context, req *CheckoutSessionReq) (*CheckoutSessionResult, error)
	VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) (map[string]any, error)
	GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*PaymentLinkInfo, error)
	CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error
}
