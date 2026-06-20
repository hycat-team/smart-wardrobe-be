package benefitresolution

type BenefitResolution string

const (
	SubscriptionActivated               BenefitResolution = "SUBSCRIPTION_ACTIVATED"
	SubscriptionExtended                BenefitResolution = "SUBSCRIPTION_EXTENDED"
	SubscriptionUpgraded                BenefitResolution = "SUBSCRIPTION_UPGRADED"
	LifetimeOverlaidByFinite            BenefitResolution = "LIFETIME_OVERLAID_BY_FINITE"
	LifetimeReplaced                    BenefitResolution = "LIFETIME_REPLACED"
	SameLifetimePaymentCreditedToWallet BenefitResolution = "SAME_LIFETIME_PAYMENT_CREDITED_TO_WALLET"
	LowerTierPaymentCreditedToWallet    BenefitResolution = "LOWER_TIER_PAYMENT_CREDITED_TO_WALLET"
	WalletTopupCredited                 BenefitResolution = "WALLET_TOPUP_CREDITED"
)
