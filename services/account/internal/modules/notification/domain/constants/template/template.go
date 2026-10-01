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
	// AdminGrant is sent when an administrator grants a plan (024 FR-083). ONE
	// template serves both a single-account grant and a batch of 500, because
	// FR-083 requires identical content: an administrator should not be able to
	// infer how many other accounts were touched from the wording a recipient
	// receives.
	AdminGrant Template = "admin_grant"
)
