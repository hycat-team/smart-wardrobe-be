package contract

import (
	"context"

	"github.com/google/uuid"
)

type ChannelEnum string

const (
	ChannelEmail ChannelEnum = "email"
)

type TemplateEnum string

const (
	TemplateOtp                   TemplateEnum = "otp"
	TemplatePaymentSuccess        TemplateEnum = "payment_success"
	TemplateTopupSuccess          TemplateEnum = "topup_success"
	TemplateRenewalSuccess        TemplateEnum = "renewal_success"
	TemplateRenewalFailed         TemplateEnum = "renewal_failed"
	TemplateWalletWarning         TemplateEnum = "wallet_warning"
	TemplateWalletPurchaseSuccess TemplateEnum = "wallet_purchase_success"
)

type INotificationContract interface {
	Send(ctx context.Context, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error
	SendByUserID(ctx context.Context, channel ChannelEnum, template TemplateEnum, params map[string]any, userID uuid.UUID) error
}
