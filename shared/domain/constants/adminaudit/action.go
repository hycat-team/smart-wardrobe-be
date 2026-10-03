package adminaudit

type AuditAction string

const (
	UserCreate                 AuditAction = "user.create"
	UserRoleChange             AuditAction = "user.role_change"
	UserDeactivate             AuditAction = "user.deactivate"
	UserActivate               AuditAction = "user.activate"
	UserSessionsRevoke         AuditAction = "user.sessions_revoke"
	BrandMemberAdd             AuditAction = "brand.member_add"
	BrandMemberRoleChange      AuditAction = "brand.member_role_change"
	BrandMemberRemove          AuditAction = "brand.member_remove"
	BrandOwnershipTransfer     AuditAction = "brand.ownership_transfer"
	SubscriptionPlanCreate     AuditAction = "subscription.plan_create"
	SubscriptionPlanUpdate     AuditAction = "subscription.plan_update"
	SubscriptionPlanDeactivate AuditAction = "subscription.plan_deactivate"
	SubscriptionCancel         AuditAction = "subscription.cancel"
	SubscriptionGrant          AuditAction = "subscription.grant"
	// 024 T145: `subscription.bulk_grant` is GONE. It was declared to record a
	// batch request as one extra row, and never written - so the verification SQL in
	// quickstart.md, which filtered on it, returned nothing at all.
	//
	// Removing it is better than wiring it up. FR-081/FR-082 ask for one audit row
	// PER GRANTED ACCOUNT, and every one of those rows already carries
	// `identifierKey` and `itemIndex`, which is what identifies the request. A
	// separate request-level action would mean a second outbox row per call
	// (FR-015) to answer a question the payload fields already answer, and would put
	// a non-grant row in the same table an operator uses to answer "who gave away
	// Premium".
	// quickstart.md now groups by `payload->>'identifierKey'`.
	//
	// 024 T110 SubscriptionGrantRestore records a repair that re-published plan
	// events for accounts missing an entitlement. It is a separate action
	// because it writes entitlements without granting anything: an operator
	// reviewing who gave away Premium must not see restores mixed in with grants.
	SubscriptionGrantRestore     AuditAction = "subscription.grant_restore"
	PaymentRetryInvestigation    AuditAction = "payment.retry_investigation"
	PaymentResolvePaid           AuditAction = "payment.resolve_paid"
	PaymentResolveCancelled      AuditAction = "payment.resolve_cancelled"
	PaymentResolveCreationFailed AuditAction = "payment.resolve_creation_failed"
	AIPricingUpdate              AuditAction = "ai.pricing_update"
	AICostPolicyUpdate           AuditAction = "ai.cost_policy_update"

	// 026 (FR-060): the campaign lifecycle, written by the admin registry use case
	// into the SAME transaction as the campaign row it describes.
	//
	// Dotted names, because they are actions ON the `campaign` target type rather
	// than on the subscription: an operator reading "who granted away Premium"
	// filters on `subscription.grant`, and a campaign open/close must not show up
	// in that answer — it moved no entitlement (FR-033).
	CampaignCreate AuditAction = "campaign.create"
	CampaignUpdate AuditAction = "campaign.update"
	CampaignClose  AuditAction = "campaign.close"
)

// 024 T178: `AIGrantCreate` ("ai.grant_create") is GONE, and this is the note
// explaining why.
//
// Nothing wrote it. The route it belonged to was removed in FR-035, which deleted the
// direct-write path from Account into the AI service's tables; every AI grant is now
// created by the event receiver, and the audit action for a cost-policy edit is
// `AICostPolicyUpdate`. A constant nobody writes is a name a reader assumes is in use.
//
// It is safe to remove even though rows carrying the string exist: the value lives in
// `shared.admin_audit_logs` as data, not as a reference to this constant, and no code
// reads it back. So the historical rows keep their value and nothing needs to be
// rewritten. The same reasoning applies to the `AIPolicyGrant` target type in
// targettype.go — do NOT go and "clean up" rows to match.
