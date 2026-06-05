package payment

import (
	"context"

	"github.com/shopspring/decimal"
)

type CheckoutSessionReq struct {
	OrderCode   int64
	Amount      decimal.Decimal
	Description string
	ReturnUrl   string
	CancelUrl   string
}

type IPaymentGatewayService interface {
	CreateCheckoutSession(ctx context.Context, req *CheckoutSessionReq) (string, error)
	VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) (map[string]any, error)
}
