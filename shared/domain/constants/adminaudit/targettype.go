package adminaudit

type AuditTargetType string

const (
	User               AuditTargetType = "user"
	Brand              AuditTargetType = "brand"
	BrandMember        AuditTargetType = "brand_member"
	SubscriptionPlan   AuditTargetType = "subscription_plan"
	UserSubscription   AuditTargetType = "user_subscription"
	DepositTransaction AuditTargetType = "deposit_transaction"
	AIModelConfig      AuditTargetType = "ai_model_config"
	AIPolicyGrant      AuditTargetType = "ai_policy_grant"
)
