package contract

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/notification/domain/constants/channel"
	"smart-wardrobe-be/services/account/internal/modules/notification/domain/constants/template"

	"github.com/google/uuid"
)

// Channel/Template enums are owned by domain/constants (DTO Convention);
// aliased here to keep the contract API unchanged.
type ChannelEnum = channel.Channel

type TemplateEnum = template.Template

const (
	ChannelEmail = channel.Email
)

const (
	TemplateOtp                   = template.Otp
	TemplatePaymentSuccess        = template.PaymentSuccess
	TemplateTopupSuccess          = template.TopupSuccess
	TemplateRenewalSuccess        = template.RenewalSuccess
	TemplateRenewalFailed         = template.RenewalFailed
	TemplateWalletWarning         = template.WalletWarning
	TemplateWalletPurchaseSuccess = template.WalletPurchaseSuccess
	// TemplateAdminGrant is the administrator-grant notification (024 FR-083).
	TemplateAdminGrant = template.AdminGrant
	// TemplateCampaignGrant is the campaign-grant notification (025 T070). Its own
	// template because the promise it makes is not the administrator's (R12).
	TemplateCampaignGrant = template.CampaignGrant
)

type INotificationContract interface {
	Send(ctx context.Context, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error
	SendByUserID(ctx context.Context, channel ChannelEnum, template TemplateEnum, params map[string]any, userID uuid.UUID) error
}
