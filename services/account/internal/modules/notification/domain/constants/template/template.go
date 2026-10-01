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

	// CampaignGrant is sent when a campaign grants a promotional plan (025 T070,
	// FR-044).
	//
	// A SEPARATE template from AdminGrant, on purpose, and the reason is that the
	// two audiences must never be able to read each other's promise. `admin_grant`
	// is written for the manual grant path and its wording is deliberately vague
	// about the source; reusing it would make a campaign recipient read text
	// authored for an administrator, and would make any later edit to the manual
	// wording silently change what a campaign customer is told (research.md R12).
	//
	// It is also named for what it DOES, not for the launch: the first campaign is
	// not the last, and a template named `launch_promo` would have to be reused —
	// and read as a lie — for every campaign after it.
	CampaignGrant Template = "campaign_grant"
)
