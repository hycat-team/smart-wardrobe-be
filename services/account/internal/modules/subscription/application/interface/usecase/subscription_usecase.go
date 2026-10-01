package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/shared/observability/workerlog"

	"github.com/google/uuid"
)

type ISubscriptionUseCase interface {
	GetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewRes, error)
	GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionOverviewRes, error)
	// EnsureDefaultSubscription provisions the default free plan for a
	// brand-new account, together with its history row and its outbox record.
	//
	// 024 T196 / FR-013: SC-006 requires the entitlement to exist within 60
	// seconds of account creation WITHOUT any subscription read, which the lazy
	// path cannot deliver. The caller has just inserted the user row and MUST
	// still be inside its unit of work, because
	// `user_subscriptions.user_id` is a foreign key onto `identity.users(id)`
	// and a second transaction would run on a connection that cannot see the
	// uncommitted user. Idempotent: an account that already has a subscription
	// is left untouched.
	EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error
	ProcessScheduledRenewals(ctx context.Context, run *workerlog.Run) error
	ProcessUpcomingExpiryNotifications(ctx context.Context) error
	SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error)
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanBriefRes, error)
	AdminList(ctx context.Context, planCode *string, status *usersubscriptionstatus.UserSubscriptionStatus, page, limit int) ([]*dto.AdminUserSubscriptionItemDTO, int64, error)
	AdminCancel(ctx context.Context, userID uuid.UUID) error
	// 024 T083/T086 (FR-060): one grant method, and it takes a LIST. A single
	// account is a list of one.
	//
	// The previous AdminGrant(ctx, userID, planID, actorAdminID) is gone: two
	// methods meant two places to keep the rules in step, and the single-account
	// one is the one that drifted - it wrote no audit record, so there was no way
	// to answer "who gave this customer a free Premium".
	//
	// identifierKey is the administrator's Idempotency-Key, recorded with the
	// audit entry so a replayed request is visible as a replay.
	AdminGrantBatch(ctx context.Context, actorAdminID uuid.UUID, items []dto.AdminGrantBatchItem, identifierKey string, requestIP *string) (*dto.AdminGrantBatchRes, error)
	// RestorePolicyGrants re-publishes plan-change events so the AI service can
	// rebuild entitlements that went missing (024 FR-088).
	//
	// It is a REPAIR, not a grant: no subscription is changed. dryRun defaults to
	// true at the transport layer because a bulk write to customer entitlements
	// has to be seen before it is applied.
	//
	// requestIP is the administrator's address, written to the audit record
	// (SC-021). It travels as an argument rather than being read from the
	// context because the application layer must not depend on a transport
	// concept.
	RestorePolicyGrants(ctx context.Context, actorAdminID uuid.UUID, dryRun bool, requestIP *string) (*dto.RestorePolicyGrantsRes, error)
}
