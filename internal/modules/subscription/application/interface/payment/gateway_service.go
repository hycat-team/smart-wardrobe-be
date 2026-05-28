package payment

import (
	"context"
)

type CheckoutSessionReq struct {
	OrderCode   int64
	Amount      float64
	Description string
	ReturnUrl   string
	CancelUrl   string
}

type IPaymentGatewayService interface {
	CreateCheckoutSession(ctx context.Context, req *CheckoutSessionReq) (string, error)
	VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) (map[string]any, error)
}
