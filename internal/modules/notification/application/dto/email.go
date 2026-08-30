package dto

// EmailPayload là payload JSON của sự kiện notification.email.* gửi qua queue.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Channel string `json:"channel"`
}

// ChannelEnum và TemplateEnum là các loại kênh/template email mà contract
// public của module hỗ trợ; contract package alias lại để giữ API (feature 017).
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
