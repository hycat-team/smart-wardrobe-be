package adminaudit

type AuditAction string

const (
	UserCreate                   AuditAction = "user.create"
	UserRoleChange               AuditAction = "user.role_change"
	UserDeactivate               AuditAction = "user.deactivate"
	UserActivate                 AuditAction = "user.activate"
	UserSessionsRevoke           AuditAction = "user.sessions_revoke"
	BrandMemberAdd               AuditAction = "brand.member_add"
	BrandMemberRoleChange        AuditAction = "brand.member_role_change"
	BrandMemberRemove            AuditAction = "brand.member_remove"
	BrandOwnershipTransfer       AuditAction = "brand.ownership_transfer"
	SubscriptionPlanCreate       AuditAction = "subscription.plan_create"
	SubscriptionPlanUpdate       AuditAction = "subscription.plan_update"
	SubscriptionPlanDeactivate   AuditAction = "subscription.plan_deactivate"
	SubscriptionCancel           AuditAction = "subscription.cancel"
	SubscriptionGrant            AuditAction = "subscription.grant"
	PaymentRetryInvestigation    AuditAction = "payment.retry_investigation"
	PaymentResolvePaid           AuditAction = "payment.resolve_paid"
	PaymentResolveCancelled      AuditAction = "payment.resolve_cancelled"
	PaymentResolveCreationFailed AuditAction = "payment.resolve_creation_failed"
	AIPricingUpdate              AuditAction = "ai.pricing_update"
	AIGrantCreate                AuditAction = "ai.grant_create"
)
