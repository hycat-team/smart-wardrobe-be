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

// ProviderPaymentStatus mirrors the status values payOS returns from
// GET /v2/payment-requests/{id} (see the official SDK PaymentLinkStatus enum).
// They are compared as-is, so the constants must stay uppercase.
type ProviderPaymentStatus string

const (
	ProviderPending    ProviderPaymentStatus = "PENDING"
	ProviderProcessing ProviderPaymentStatus = "PROCESSING"
	ProviderPaid       ProviderPaymentStatus = "PAID"
	ProviderCancelled  ProviderPaymentStatus = "CANCELLED"
	ProviderExpired    ProviderPaymentStatus = "EXPIRED"
	ProviderUnderpaid  ProviderPaymentStatus = "UNDERPAID"
	ProviderFailed     ProviderPaymentStatus = "FAILED"
	ProviderUnknown    ProviderPaymentStatus = "UNKNOWN"
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
	// CancelledAt and CancellationReason come from payOS when Status is
	// ProviderCancelled. Both are optional on the provider's response.
	CancelledAt        *time.Time
	CancellationReason string
}

type IPaymentGatewayService interface {
	CreateCheckoutSession(ctx context.Context, req *CheckoutSessionReq) (*CheckoutSessionResult, error)
	VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) error
	GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*PaymentLinkInfo, error)
	CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error
}
