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
	AICostPolicy       AuditTargetType = "ai_cost_policy"
)

// 024 T178: `AIPolicyGrant` ("ai_policy_grant") is GONE, and this is the note
// explaining why.
//
// Nothing writes it. Grants stopped being created by the Account service when FR-035
// removed that direct-write route; the AI service now owns the table and no audit row
// on the Account side names it as a target.
//
// It is safe to remove even though rows carrying the string exist in
// `shared.admin_audit_logs`: the value is stored as data, not as a reference to this
// constant, and no code reads it back. The historical rows keep their value — do NOT
// go and "clean up" rows to match.
