package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/notification/application/dto"

	"github.com/google/uuid"
)

// Channel/Template enums are owned by application/dto (feature 017); aliased
// here to keep the contract API unchanged.
type ChannelEnum = dto.ChannelEnum

type TemplateEnum = dto.TemplateEnum

const (
	ChannelEmail = dto.ChannelEmail
)

const (
	TemplateOtp                   = dto.TemplateOtp
	TemplatePaymentSuccess        = dto.TemplatePaymentSuccess
	TemplateTopupSuccess          = dto.TemplateTopupSuccess
	TemplateRenewalSuccess        = dto.TemplateRenewalSuccess
	TemplateRenewalFailed         = dto.TemplateRenewalFailed
	TemplateWalletWarning         = dto.TemplateWalletWarning
	TemplateWalletPurchaseSuccess = dto.TemplateWalletPurchaseSuccess
)

type INotificationContract interface {
	Send(ctx context.Context, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error
	SendByUserID(ctx context.Context, channel ChannelEnum, template TemplateEnum, params map[string]any, userID uuid.UUID) error
}
