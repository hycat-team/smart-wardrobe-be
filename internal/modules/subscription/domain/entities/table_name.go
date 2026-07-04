package entities

func (SubscriptionPlan) TableName() string        { return "subscription_plans" }
func (UserSubscription) TableName() string        { return "user_subscriptions" }
func (UserDailyQuota) TableName() string           { return "user_daily_quotas" }
func (UserWallet) TableName() string               { return "user_wallets" }
func (DepositTransaction) TableName() string       { return "deposit_transactions" }
func (WalletStatement) TableName() string          { return "wallet_statements" }
func (ProviderPaymentEvent) TableName() string     { return "provider_payment_events" }
func (ProviderWebhookInbox) TableName() string     { return "provider_webhook_inbox" }
func (UserSubscriptionEvent) TableName() string    { return "user_subscription_events" }
func (SubscriptionRenewalAttempt) TableName() string { return "subscription_renewal_attempts" }
