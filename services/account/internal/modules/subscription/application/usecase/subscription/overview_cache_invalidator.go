package subscription

import (
	"context"
	"time"

	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// overviewCacheInvalidateTimeout bounds the post-commit cache delete.
//
// The timeout is not optional. Detaching the context from cancellation is what makes
// the delete reliable; the timeout is what stops a hung Redis from holding the
// response open. Handing `WithoutCancel` a context with no deadline would convert a
// cache problem into a request that never returns.
const overviewCacheInvalidateTimeout = 2 * time.Second

// InvalidateOverviewCache drops one account's cached overview, after a commit.
//
// 024 T151 introduced this for the admin grant path. 024 T154 found the same defect
// copied into three more places, each with its own
// `_ = uc.overviewCache.Invalidate(ctx, ...)`:
//
//	admin grant      admin_grant_batch_uc.go
//	auto-renew toggle subscription_uc.go
//	scheduled renewal  subscription_renewal_workflow.go
//	wallet top-up      purchase/wallet.go
//	payment webhook    webhook/webhook.go
//
// Four copies of one bug is not four coincidences: duplication is the mechanism, so
// the fix is one function plus an assertion that keeps it that way. Without the
// assertion the fifth copy is a matter of time.
//
// It is a package-level function rather than a method because `PurchaseUseCase` and
// `PaymentWebhookUseCase` live in sibling packages and already import this one, so
// they can share it without introducing a new dependency. The cache and logger are
// passed in because each of those use cases holds its own.
//
// The failure mode this exists to prevent: the caller's context is the wrong thing to
// pass, for two reasons.
//
//   - The FR-076 time budget can expire between the last `ctx.Err()` check and this
//     call, because the work in between is a database transaction of unbounded length.
//   - The client can disconnect at any moment, cancelling the request context.
//
// In both cases `Del(ctx)` fails immediately, the cached overview keeps describing the
// OLD plan until its TTL expires, and FR-073 is violated: the change committed, the
// caller saw success, and the customer saw the previous plan for the rest of the cache
// window. Under `_ =` nothing anywhere reported it - a scheduled renewal whose context
// had already lapsed was the quietest version of the bug, because no client was even
// waiting to notice.
func InvalidateOverviewCache(
	ctx context.Context,
	log logger.Interface,
	cache subscriptioncaching.ISubscriptionOverviewCache,
	userID uuid.UUID,
) {
	if cache == nil {
		return
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), overviewCacheInvalidateTimeout)
	defer cancel()

	if err := cache.Invalidate(cacheCtx, userID); err != nil {
		// Logged rather than discarded, but deliberately NOT turned into a failed
		// operation: the plan really was applied, and reporting an error would invite
		// the administrator to re-grant what the customer already has.
		//
		// The nil-logger guard matters: a panic in post-commit cleanup would turn a
		// committed operation into a 500, which is strictly worse than the stale cache
		// we were trying to avoid.
		if log != nil {
			log.Error("failed to invalidate subscription overview cache after commit",
				zap.String("user_id", userID.String()),
				zap.Duration("timeout", overviewCacheInvalidateTimeout),
				zap.Error(err),
			)
		}
	}
}
