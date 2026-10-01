package subscription

import (
	"context"
	"errors"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// 024 T151: FR-073 says a committed grant must not leave a stale cached overview
// behind.
//
// The delete used to run on the caller's context, after the commit. That is the one
// moment the context is most likely to be dead: the batch's own FR-076 budget can
// expire between the last `ctx.Err()` check and the delete, because the grant in
// between is an unbounded database transaction, and a client disconnect cancels the
// request context outright.
//
// `Del` on a cancelled context fails immediately, the error was discarded with `_ =`,
// and the cache kept serving the OLD plan until its TTL expired. The grant had
// committed, the administrator saw `success`, and the customer saw the previous
// plan. Nothing anywhere recorded that it had happened.
//
// These tests drive `invalidateOverviewCache` directly with an already-cancelled
// context, which is the exact state the old code mishandled and cannot be produced
// reliably through the whole batch path.
func TestInvalidateOverviewCacheSurvivesCancelledCallerContext(t *testing.T) {
	user := grantTestUser("t151", "t151@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	cache := &recordingOverviewCache{}
	h.uc.overviewCache = cache

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the caller is gone before the delete runs

	h.uc.invalidateOverviewCache(ctx, user.ID)

	if len(cache.invalidated) != 1 {
		t.Fatalf("bộ nhớ đệm phải được xoá dù context của caller đã hủy, có = %d lần xoá", len(cache.invalidated))
	}
	if !cache.sawLiveContext {
		t.Error("context truyền cho Invalidate vẫn còn hiệu lực - lỗi xoá cache sẽ bị nuốt im lặng")
	}
}

// A cache failure must be reported, not swallowed. It still must NOT become a
// failed grant - the plan really was applied, and an operator re-granting it would
// be worse than a stale cache - but it has to leave a trace.
func TestInvalidateOverviewCacheFailureIsLoggedNotSwallowed(t *testing.T) {
	user := grantTestUser("t151b", "t151b@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	logger := &grantTestLogger{}
	h.uc.log = logger
	cache := &recordingOverviewCache{invalidateErr: errors.New("redis unavailable")}
	h.uc.overviewCache = cache

	h.uc.invalidateOverviewCache(context.Background(), user.ID)

	if len(logger.entries) == 0 {
		t.Fatal("lỗi xoá bộ nhớ đệm phải được ghi log, không được bỏ qua")
	}
	// grantTestLogger records "level: message".
	found := false
	for _, entry := range logger.entries {
		if strings.HasPrefix(entry, "error:") && strings.Contains(entry, "cache") {
			found = true
		}
	}
	if !found {
		t.Errorf("phải ghi log ở mức error có nhắc tới cache, đã ghi: %v", logger.entries)
	}
	if len(cache.invalidated) != 0 {
		t.Error("cache từ chối thì không được tính là đã xoá")
	}
}

// The grant itself must still report success when the cache is down. Turning a
// cache outage into a failed grant would send the administrator off to re-grant a
// plan the customer already has - the failure mode this task must not introduce.
func TestAdminGrantBatchStillSucceedsWhenCacheIsDown(t *testing.T) {
	user := grantTestUser("t151c", "t151c@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.uc.overviewCache = &recordingOverviewCache{invalidateErr: errors.New("redis unavailable")}

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"t151c@example.com", "premium"},
	), "idem-CACHE-DOWN", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Status != "success" {
		t.Errorf("gói đã cấp thành công thì cache hỏng không được biến thành thất bại, thực tế = %s", res.Results[0].Status)
	}
}

// A nil cache must stay a no-op rather than a panic: the use case is constructed
// with and without it depending on deployment wiring.
func TestInvalidateOverviewCacheToleratesNilCache(t *testing.T) {
	h := newGrantHarness(t, grantTestUser("t151d", "t151d@example.com", roleslug.User, false))
	h.uc.overviewCache = nil

	h.uc.invalidateOverviewCache(context.Background(), uuid.New())
}
