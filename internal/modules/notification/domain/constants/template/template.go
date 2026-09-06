package template

// Template is the email template identifier rendered by the notification module.
type Template string

const (
	Otp                   Template = "otp"
	PaymentSuccess        Template = "payment_success"
	TopupSuccess          Template = "topup_success"
	RenewalSuccess        Template = "renewal_success"
	RenewalFailed         Template = "renewal_failed"
	WalletWarning         Template = "wallet_warning"
	WalletPurchaseSuccess Template = "wallet_purchase_success"
)
