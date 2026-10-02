package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminGrantBatchReqShape is the part of the request body the limiter needs. It
// is declared here rather than importing the subscription DTO: `shared` must not
// depend on a service's dto package, and the two fields below are the entire
// contract the limiter relies on.
type AdminGrantBatchReqShape struct {
	Items []struct {
		UserIdentifier string `json:"userIdentifier"`
		PlanSlug       string `json:"planSlug"`
	} `json:"items"`
}

// AdminGrantRateLimitMiddleware caps the administrator bulk-grant endpoint with
// two independent tiers (024 T099, FR-078/FR-079).
//
// Tiers rather than one number, because a single number cannot express both
// "60 single grants" and "6 bulk operations": the stricter of the two alone would
// make the bulk feature unusable, and the looser alone would let one request cost
// 500 writes.
//
// Counted per ADMINISTRATOR rather than per IP, unlike RateLimitMiddleware in
// this same file. On a shared office connection a per-IP limit lets one
// administrator exhaust everyone else's budget; and it can be side-stepped by
// changing address, which a per-administrator limit cannot.
//
// ⚠️ This limiter FAILS CLOSED, unlike LoginRateLimitMiddleware above, which
// fails open. The divergence is deliberate. A failed login costs one attempt; a
// grant writes up to 500 customer entitlements, so proceeding without a limit is
// not a degraded feature but an unbounded bulk write. It matches the brand-claim
// limiter, which is the closest precedent for money and entitlements.
type AdminGrantRateLimitMiddleware struct {
	limiter       shared_caching.IRateLimiter
	allLimit      int
	multiLimit    int
	windowSeconds int
}

// NewAdminGrantRateLimitMiddleware builds the two-tier administrator rule.
func NewAdminGrantRateLimitMiddleware(cfg *config.Config, limiter shared_caching.IRateLimiter) *AdminGrantRateLimitMiddleware {
	limits := cfg.AdminGrantRateLimit
	return &AdminGrantRateLimitMiddleware{
		limiter:       limiter,
		allLimit:      limits.AllPerMinute,
		multiLimit:    limits.MultiPerMinute,
		windowSeconds: limits.WindowSeconds,
	}
}

// Handle must run AFTER authentication, because the key is the administrator's
// identity.
func (m *AdminGrantRateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, _ := contextutils.GetUserId(c)
		if adminID == uuid.Nil || m.limiter == nil {
			// Unauthenticated, or no limiter wired: the auth middleware and the
			// DI graph are responsible for this, and guessing an identity here
			// would put every unauthenticated caller on one shared bucket.
			c.Error(apperror.NewUnauthorized("Vui lòng đăng nhập để thực hiện thao tác cấp gói."))
			c.Abort()
			return
		}
		if m.allLimit <= 0 || m.multiLimit <= 0 || m.windowSeconds <= 0 {
			c.Error(apperror.NewServiceUnavailable(adminGrantLimitUnavailableMsg))
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		admin := adminID.String()

		// ⚠️ THE `all` TIER IS CONSUMED BEFORE THE BODY IS READ, AND THAT ORDER IS
		// LOAD-BEARING. It used to run after the body was parsed, with an early
		// `c.Next()` for anything that was not a well-formed batch body — and that
		// skipped the limit entirely for every request without that shape: every
		// GET, and every malformed POST.
		//
		// On the grant route the hole was harmless, because the handler refuses a
		// body it cannot bind. On a READ route it is the whole problem: 025 FR-071
		// attaches this same limiter to the operator campaign look-up, and with the
		// old ordering a look-up consumed no budget at all — so the endpoint would
		// have been "rate limited" in name only.
		//
		// The fix is the ordering, not a new limit. `all` answers "how many requests
		// may this administrator make", and that question does not depend on what is
		// inside the request — which is exactly why it belongs first. Only the
		// `multi` tier needs the body, because "is this a bulk operation" is a
		// question about the body, and a request with no batch shape simply is not
		// one.
		//
		// The effect on the grant route is that a malformed body now costs one `all`
		// token before the handler refuses it with its own field-level message. That
		// is the intended fail-closed direction for an endpoint that writes customer
		// entitlements, and it is why the malformed-body `c.Next()` below stays.
		if !m.allow(ctx, "rl:admin-grant:"+admin+":all", m.allLimit) {
			c.Error(adminGrantRateLimited(m.windowSeconds))
			c.Abort()
			return
		}

		// The body is read only to decide which tier applies, then handed back to
		// the handler unchanged.
		//
		// ⚠️ The body is read with io.ReadAll and explicitly RESTORED. Gin's
		// `ShouldBindBodyWith` is not enough here: it caches the bytes for its own
		// purposes, but `ShouldBindJSON` in the handler binds straight from
		// `c.Request.Body`, which this middleware would already have drained. The
		// handler would then see an empty body and reject a perfectly valid
		// request. Restoring the reader is what makes the two cooperate.
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		var shape AdminGrantBatchReqShape
		if err := json.Unmarshal(bodyBytes, &shape); err != nil {
			// A malformed body is the handler's problem, not the limiter's: it must
			// be reported with the field-level detail the handler produces. The
			// `all` tier has already been charged above, so letting the request
			// through here is bounded.
			c.Next()
			return
		}

		// The bulk tier counts only requests of more than one item, so an
		// administrator doing single grants is not charged for the bulk budget.
		if len(shape.Items) > 1 && !m.allow(ctx, "rl:admin-grant:"+admin+":multi", m.multiLimit) {
			c.Error(adminGrantRateLimited(m.windowSeconds))
			c.Abort()
			return
		}
		c.Next()
	}
}

// allow consumes one unit and reports whether the request may proceed.
//
// A limiter error is treated as DENIED, which is the fail-closed choice
// documented on the type.
func (m *AdminGrantRateLimitMiddleware) allow(ctx context.Context, key string, limit int) bool {
	allowed, err := m.limiter.Allow(ctx, key, limit, m.windowSeconds)
	if err != nil {
		return false
	}
	return allowed
}

const adminGrantLimitUnavailableMsg = "Chưa thể kiểm tra giới hạn tần suất cấp gói. Vui lòng thử lại sau."

// adminGrantRateLimited states how long to wait, rather than the login
// middleware's "ít phút": an administrator mid-batch needs to know whether to
// wait or to come back after doing something else.
func adminGrantRateLimited(retryAfterSeconds int) *apperror.Error {
	return apperror.NewTooManyRequest(fmt.Sprintf("Bạn đang thao tác quá nhanh. Vui lòng thử lại sau %d giây.", retryAfterSeconds))
}
