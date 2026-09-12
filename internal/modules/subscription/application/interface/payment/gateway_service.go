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

// ErrCodeDuplicateOrder is the PayOS business error returned when orderCode was
// already registered on the merchant. It is retryable with a freshly generated
// order code (see application/ordercode).
const ErrCodeDuplicateOrder = "231"

// IsDuplicateOrderCode reports whether the checkout result failed only because
// the order code already exists on the payment provider.
func IsDuplicateOrderCode(result *CheckoutSessionResult) bool {
	return result != nil && result.ErrorCode == ErrCodeDuplicateOrder
}

type ProviderPaymentStatus string

const (
	ProviderPending   ProviderPaymentStatus = "pending"
	ProviderPaid      ProviderPaymentStatus = "PAID"
	ProviderCancelled ProviderPaymentStatus = "cancelled"
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
	VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) error
	GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*PaymentLinkInfo, error)
	CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error
}
