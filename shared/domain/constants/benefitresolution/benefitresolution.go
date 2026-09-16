package benefitresolution

type BenefitResolution string

const (
	SubscriptionActivated               BenefitResolution = "subscription_activated"
	SubscriptionExtended                BenefitResolution = "subscription_extended"
	SubscriptionUpgraded                BenefitResolution = "subscription_upgraded"
	LifetimeOverlaidByFinite            BenefitResolution = "lifetime_overlaid_by_finite"
	LifetimeReplaced                    BenefitResolution = "lifetime_replaced"
	SameLifetimePaymentCreditedToWallet BenefitResolution = "same_lifetime_payment_credited_to_wallet"
	LowerTierPaymentCreditedToWallet    BenefitResolution = "lower_tier_payment_credited_to_wallet"
	WalletTopupCredited                 BenefitResolution = "wallet_topup_credited"
)
