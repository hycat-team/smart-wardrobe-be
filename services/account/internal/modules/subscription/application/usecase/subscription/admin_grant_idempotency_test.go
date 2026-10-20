package subscription

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// FR-068: a retried request must not extend the subscription a second time.
//
// This is the failure the Idempotency-Key exists to prevent. A network retry of a
// batch that appeared to fail would otherwise hand the customer two months of
// Premium for one decision, and nothing in the system would object.
//
// The replay is reported as `skipped`, not `success`: the administrator has to be
// able to tell a repeat from a fresh grant.
func TestAdminGrantBatchRetryIsSkippedAndDoesNotExtendExpiry(t *testing.T) {
	user := grantTestUser("retry", "retry@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	items := grantTestItems([2]string{"retry@example.com", "premium"})

	first, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-A", nil)
	if err != nil {
		t.Fatalf("lần gọi đầu lỗi: %v", err)
	}
	if first.Succeeded != 1 {
		t.Fatalf("lần gọi đầu phải thành công, có = %d", first.Succeeded)
	}
	firstExpiry := h.subs.subs[user.ID].ExpiresAt

	// Same key, same list: a network retry.
	second, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-A", nil)
	if err != nil {
		t.Fatalf("lần gọi lại lỗi: %v", err)
	}

	if second.Results[0].Status != sub_dto.AdminGrantStatusSkipped {
		t.Errorf("lần gọi lại phải là skipped, có = %s", second.Results[0].Status)
	}
	if second.Results[0].Reason != sub_dto.AdminGrantReasonDuplicateRequest {
		t.Errorf("phải báo duplicate_request, có = %s", second.Results[0].Reason)
	}
	if second.Skipped != 1 || second.Succeeded != 0 {
		t.Errorf("tổng hợp phải là 1 skipped, có succeeded=%d skipped=%d", second.Succeeded, second.Skipped)
	}

	// The decisive assertion: the expiry did not move.
	if got := h.subs.subs[user.ID].ExpiresAt; got == nil || !got.Equal(*firstExpiry) {
		t.Errorf("thời hạn không được thay đổi khi gọi lại, cũ = %v mới = %v", firstExpiry, got)
	}
	if h.subs.subs[user.ID].Version != 1 {
		t.Errorf("bản ghi không được áp dụng lần thứ hai, có version = %d", h.subs.subs[user.ID].Version)
	}
	if len(h.audit.records) != 1 {
		t.Errorf("chỉ được ghi một nhận ký cho một quyết định, có = %d", len(h.audit.records))
	}
}

// FR-068: the key covers the whole request, so each LINE is deduplicated by its
// own index. A retry must return the same outcome per line, not skip the whole
// list.
func TestAdminGrantBatchRetryDeduplicatesPerItemIndex(t *testing.T) {
	first := grantTestUser("i1", "i1@example.com", roleslug.User, false)
	second := grantTestUser("i2", "i2@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)

	items := grantTestItems(
		[2]string{"i1@example.com", "premium"},
		[2]string{"i2@example.com", "premium"},
	)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-B", nil); err != nil {
		t.Fatalf("lần gọi đầu lỗi: %v", err)
	}
	replay, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-B", nil)
	if err != nil {
		t.Fatalf("lần gọi lại lỗi: %v", err)
	}

	for idx, result := range replay.Results {
		if result.Status != sub_dto.AdminGrantStatusSkipped {
			t.Errorf("phần tử %d phải skipped, có = %s", idx, result.Status)
		}
		if result.ResolvedUserID == nil {
			t.Errorf("phần tử %d vẫn phải trả về tài khoản đã phân giải", idx)
		}
	}
}

// A DIFFERENT key is a different decision, so it grants again rather than replaying
// the first result. It does NOT add days onto the previous term: 027/D-08 made an
// administrative grant restart the period at `now`, because nobody paid for the days
// that stacking would hand out. Deliberate extension therefore means granting a longer
// duration or a different plan, not calling the same grant twice.
func TestAdminGrantBatchDifferentKeyGrantsAgain(t *testing.T) {
	user := grantTestUser("twice", "twice@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	items := grantTestItems([2]string{"twice@example.com", "premium"})
	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-C", nil); err != nil {
		t.Fatalf("lần gọi đầu lỗi: %v", err)
	}
	before := *h.subs.subs[user.ID].ExpiresAt

	second, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-D", nil)
	if err != nil {
		t.Fatalf("lần gọi thứ hai lỗi: %v", err)
	}
	if second.Succeeded != 1 {
		t.Fatalf("khoá khác phải được cấp lại, có = %d", second.Succeeded)
	}
	after := h.subs.subs[user.ID].ExpiresAt
	if after == nil {
		t.Fatal("cấp lại phải để lại ngày hết hạn")
	}
	// The harness uses the real clock, so "restart at now" lands a few microseconds
	// after the previous value. What must NOT happen is a whole extra duration: that
	// would be the stacking 027/D-08 removed.
	delta := after.Sub(before)
	if delta < 0 || delta > time.Minute {
		t.Errorf("cấp hành chính phải tính lại kỳ từ lúc cấp, không cộng dồn (027/D-08): "+
			"cũ = %v mới = %v chênh = %v", before, *after, delta)
	}
}

// The event key is what the receiver and the history table deduplicate on, so its
// shape is part of the contract rather than an internal detail.
func TestAdminGrantBatchWritesIdempotencyShapedEventKey(t *testing.T) {
	user := grantTestUser("keyed", "keyed@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"keyed@example.com", "premium"}), "idem-E", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if len(h.events.keys) != 1 {
		t.Fatalf("phải ghi đúng một sự kiện, có = %d", len(h.events.keys))
	}
	if h.events.keys[0] != "ADMIN_GRANT:idem-E:0" {
		t.Errorf("khoá sự kiện phải theo mẫu ADMIN_GRANT:{key}:{index}, có = %s", h.events.keys[0])
	}
}

// FR-069: a row that changed between the batch's read and the item's transaction
// is reported, not overwritten. Overwriting would discard whatever the other
// write was — most often a real customer purchase.
func TestAdminGrantBatchReportsVersionChangedSincePlan(t *testing.T) {
	user := grantTestUser("conflict", "conflict@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	future := time.Now().AddDate(0, 0, 10)
	h.subs.subs[user.ID] = &entities.UserSubscription{
		UserID:                 user.ID,
		SubscriptionPlanID:     uuid.New(),
		CurrentPlanCode:        "free",
		CurrentBenefitSnapshot: entities.JSONDocument("{}"),
		ExpiresAt:              &future,
		Version:                4,
	}
	// The row moves on between the pre-pass read (version 4) and the transaction.
	h.subs.lockVersionOffset = 1

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"conflict@example.com", "premium"}), "idem-F", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Reason != sub_dto.AdminGrantReasonConcurrentModif {
		t.Errorf("phải báo concurrent_modification, có = %s", res.Results[0].Reason)
	}
	if h.subs.subs[user.ID].CurrentPlanCode != "free" {
		t.Errorf("không được ghi đè thay đổi của thao tác khác, có = %s", h.subs.subs[user.ID].CurrentPlanCode)
	}
	// 024 T129: a version conflict is an EXPECTED outcome the administrator can
	// act on, so it must not also be logged as an infrastructure failure. Only
	// the non-conflict branch logs, and asserting nothing was logged keeps the
	// two paths from merging again.
	if lg, ok := h.uc.log.(*grantTestLogger); ok && len(lg.entries) != 0 {
		t.Errorf("xung đột version không được ghi nhận như lỗi hệ thống, có = %v", lg.entries)
	}
}

// 024 T129: a failure that is NOT a version conflict must NOT be dressed up as
// one.
//
// Before this task every error from an item's transaction was reported as
// `concurrent_modification`, which names a cause the operator cannot act on and
// tells them to retry - so a dead dependency was retried forever and a refused
// audit write looked like a busy account.
func TestAdminGrantBatchDoesNotReportInfrastructureFailureAsConcurrency(t *testing.T) {
	user := grantTestUser("boom", "boom@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.audit.err = errors.New("audit storage down")

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"boom@example.com", "premium"}), "idem-G", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Status != sub_dto.AdminGrantStatusError {
		t.Fatalf("phần tử phải thất bại, có = %s", res.Results[0].Status)
	}
	if res.Results[0].Reason == sub_dto.AdminGrantReasonConcurrentModif {
		t.Error("lỗi ghi kiểm toán không được báo là concurrent_modification")
	}
	if res.Results[0].Reason != sub_dto.AdminGrantReasonGrantFailed {
		t.Errorf("phải báo grant_failed, có = %s", res.Results[0].Reason)
	}
	// The cause is logged for an operator, never shown to the caller.
	lg, ok := h.uc.log.(*grantTestLogger)
	if !ok {
		t.Fatal("harness phải dùng grantTestLogger")
	}
	if len(lg.entries) == 0 {
		t.Error("phải ghi log nguyên nhân để vận hành điều tra được")
	}
	if strings.Contains(res.Results[0].Message, "audit storage down") {
		t.Error("không được lộ chi tiết lỗi hệ thống cho quản trị viên")
	}
}

// An account with NO subscription yet is not a conflict. Its first grant is a
// creation, and treating the missing row as a change would refuse every grant to
// a new customer.
func TestAdminGrantBatchTreatsMissingSubscriptionAsNotAConflict(t *testing.T) {
	user := grantTestUser("fresh", "fresh@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"fresh@example.com", "premium"}), "idem-G", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Succeeded != 1 {
		t.Errorf("cấp lần đầu cho tài khoản chưa có gói phải thành công, có = %d", res.Succeeded)
	}
	if h.subs.subs[user.ID] == nil || h.subs.subs[user.ID].CurrentPlanCode != "premium" {
		t.Error("bản ghi gói phải được tạo")
	}
}

// FR-096: valid items are collected into ONE pass. The identifier lookup, the
// plan lookup and the version read each happen once for the whole list, not once
// per line.
func TestAdminGrantBatchDoesNotDeclareIdentifierTypeOrReQueryPerItem(t *testing.T) {
	good := grantTestUser("good1", "good1@example.com", roleslug.User, false)
	h := newGrantHarness(t, good)
	// A per-item pre-pass would read the version of this account once per line.
	h.subs.plannedReadCalls = 0

	items := grantTestItems(
		[2]string{"good1@example.com", "premium"},
		[2]string{"  ", "premium"},
		[2]string{"khong-co-that", "premium"},
		[2]string{"good1@example.com", "premium"},
	)
	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-H", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if h.subs.plannedReadCalls != 1 {
		t.Errorf("phiên bản gói phải được đọc một lần cho cả danh sách, có = %d", h.subs.plannedReadCalls)
	}
	if h.plans.slugCalls != 1 {
		t.Errorf("mã gói phải được tra cứu một lần cho cả danh sách, có = %d", h.plans.slugCalls)
	}
}

// FR-093: a blank identifier is that item's own error. The request still runs and
// the other items are still granted — the order of the list is preserved.
func TestAdminGrantBatchBlankIdentifierIsItsOwnErrorOnly(t *testing.T) {
	first := grantTestUser("b1", "b1@example.com", roleslug.User, false)
	second := grantTestUser("b2", "b2@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)

	items := grantTestItems(
		[2]string{"b1@example.com", "premium"},
		[2]string{"", "premium"},
		[2]string{"   ", "premium"},
		[2]string{"b2@example.com", "premium"},
	)
	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-I", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Total != 4 || res.Succeeded != 2 || res.Failed != 2 {
		t.Fatalf("phải có 2 thành công / 2 lỗi / tổng 4, có = %d/%d/%d", res.Succeeded, res.Failed, res.Total)
	}
	for _, idx := range []int{1, 2} {
		if res.Results[idx].Reason != sub_dto.AdminGrantReasonIdentifierBlank {
			t.Errorf("phần tử %d phải báo identifier_blank, có = %s", idx, res.Results[idx].Reason)
		}
		if res.Results[idx].Message == "" {
			t.Errorf("phần tử %d phải có thông báo tiếng Việt nêu rõ nguyên nhân", idx)
		}
	}
	// The echo still carries what was sent, so the console can highlight the line.
	if res.Results[1].UserIdentifier != "" || res.Results[2].UserIdentifier != "   " {
		t.Error("phải giữ nguyên chuỗi đã gọi để quản trị viên định vị dòng lỗi")
	}
	// The accounts on either side of the bad lines were still granted.
	if h.subs.subs[first.ID] == nil || h.subs.subs[second.ID] == nil {
		t.Error("hai tài khoản hợp lệ phải được cấp dù danh sách có phần tử rỗng")
	}
}

// An account that resolves but whose plan is unusable is that item's error, and
// the same rules apply in either order.
func TestAdminGrantBatchRulesApplyPerItemRegardlessOfOrder(t *testing.T) {
	user := grantTestUser("order", "order@example.com", roleslug.User, false)
	// 024 T132: the valid line needs its OWN account. It used to name the same
	// account as the failing line, which now reads as a duplicate and stops
	// testing what this test is about: that a bad line does not block the others.
	valid := grantTestUser("ok", "ok@example.com", roleslug.User, false)
	adminUser := grantTestUser("boss", "boss@example.com", roleslug.Admin, false)
	h := newGrantHarness(t, user, adminUser, valid)

	items := grantTestItems(
		[2]string{"boss@example.com", "premium"},
		[2]string{"order@example.com", "khong-co"},
		[2]string{"ok@example.com", "premium"},
	)
	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "idem-J", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	wantReasons := []string{
		sub_dto.AdminGrantReasonTargetIsAdmin,
		sub_dto.AdminGrantReasonPlanNotFound,
		"",
	}
	for idx, want := range wantReasons {
		if res.Results[idx].Reason != want {
			t.Errorf("phần tử %d: lý do mong đợi %q, có = %q", idx, want, res.Results[idx].Reason)
		}
	}
	if res.Results[2].Status != sub_dto.AdminGrantStatusSuccess {
		t.Errorf("phần tử hợp lệ phải được cấp, có = %s", res.Results[2].Status)
	}
	if res.Succeeded != 1 || res.Failed != 2 {
		t.Errorf("tổng hợp phải là 1/2, có = %d/%d", res.Succeeded, res.Failed)
	}
}

// FR-064: two identical identifiers in one list, differing only by case, resolve
// to one account and grant once.
func TestAdminGrantBatchTreatsCaseVariantIdentifiersAsDuplicates(t *testing.T) {
	user := grantTestUser("Case", "Case@Example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"case@example.com", "premium"},
		[2]string{"CASE@EXAMPLE.COM", "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Succeeded != 1 || res.Skipped != 1 || res.Failed != 0 {
		t.Errorf("phai cap dung MOT lan va BO QUA MOT lan, co succeeded=%d skipped=%d failed=%d", res.Succeeded, res.Skipped, res.Failed)
	}
	if res.Results[1].Reason != sub_dto.AdminGrantReasonDuplicateInRequest {
		t.Errorf("bản viết hoa khác phải được coi là trùng, có = %s", res.Results[1].Reason)
	}
}

// An empty list is refused at the use case too, not only by the request binding,
// so a caller that bypasses HTTP still cannot process nothing.
func TestAdminGrantBatchRefusesEmptyList(t *testing.T) {
	h := newGrantHarness(t)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, nil, "idem-L", nil); err == nil {
		t.Error("danh sách rỗng phải bị từ chối")
	}
}

// The user brief used by the plan pass is the one the resolver produced, so the
// pre-pass never re-queries identity for accounts already resolved.
func TestAdminGrantBatchReusesResolvedIdentityForVersionPass(t *testing.T) {
	first := grantTestUser("r1", "r1@example.com", roleslug.User, false)
	second := grantTestUser("r2", "r2@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)
	var seen []uuid.UUID
	h.subs.onPlannedRead = func(ids []uuid.UUID) { seen = ids }

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"r1@example.com", "premium"},
		[2]string{"r2@example.com", "premium"},
		[2]string{"r1@example.com", "premium"},
	), "idem-M", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if len(seen) != 2 {
		t.Errorf("mỗi tài khoản chỉ được đưa vào lượt đọc phiên bản một lần, có = %d id", len(seen))
	}
}

// compile-time guard: the resolver output type used by the batch path.
var _ = func() *identity_dto.UserBriefContractDTO { return nil }

// FR-062: an over-long identifier is reported as a FORMAT error, not as
// "user_not_found".
//
// The distinction is what makes the response actionable. "Không tìm thấy tài
// khoản" sends the administrator to re-check a spelling that may well be
// correct; naming the limit tells them what to do instead.
func TestAdminGrantBatchReportsOverlongIdentifierAsFormatError(t *testing.T) {
	good := grantTestUser("ok", "ok@example.com", roleslug.User, false)
	h := newGrantHarness(t, good)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{strings.Repeat("x", MaxIdentifierLength+1), "premium"},
		[2]string{"ok@example.com", "premium"},
	), "idem-LONG", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Results[0].Reason != sub_dto.AdminGrantReasonIdentifierTooLong {
		t.Errorf("mã định danh quá dài phải báo identifier_too_long, có = %s", res.Results[0].Reason)
	}
	if res.Results[0].Reason == sub_dto.AdminGrantReasonUserNotFound {
		t.Error("không được báo user_not_found cho một mã định danh quá dài")
	}
	if !strings.Contains(res.Results[0].Message, "255") {
		t.Errorf("thông báo phải nêu giới hạn cụ thể, có = %q", res.Results[0].Message)
	}
	// The valid item in the same request is unaffected (FR-064).
	if res.Results[1].Status != sub_dto.AdminGrantStatusSuccess {
		t.Errorf("phần tử hợp lệ phải vẫn được cấp, có = %s", res.Results[1].Status)
	}
}

// SC-021: the repair is traceable to the administrator AND their connection.
func TestRestorePolicyGrantsRecordsAuditWithRequestIP(t *testing.T) {
	user := grantTestUser("restore", "restore@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.uc.userSubRepo = newRestoreScanRepo(map[uuid.UUID]*entities.UserSubscription{
		user.ID: {UserID: user.ID, CurrentPlanCode: "premium", Version: 1, StartedAt: time.Now()},
	})
	h.uc.planRepo = &fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{
		"premium": renewalTestPlan(),
	}}

	ip := "203.0.113.7"
	if _, err := h.uc.RestorePolicyGrants(context.Background(), h.adminID, false, &ip); err != nil {
		t.Fatalf("RestorePolicyGrants lỗi: %v", err)
	}

	if len(h.audit.records) != 1 {
		t.Fatalf("phải ghi đúng một bản ghi kiểm toán cho cả lượt, có = %d", len(h.audit.records))
	}
	if h.audit.records[0].RequestIP == nil {
		t.Fatal("bản ghi kiểm toán phải mang địa chỉ mạng")
	}
	if *h.audit.records[0].RequestIP != ip {
		t.Errorf("địa chỉ mạng ghi sai, có = %s", *h.audit.records[0].RequestIP)
	}
}

// Constitution V: the whole repair - scan, plan lookup, re-publish, audit - runs
// on ONE transaction. Otherwise a failure between the repairs and the audit
// leaves a repaired system with no record of who ran it.
func TestRestorePolicyGrunsInsideASingleTransaction(t *testing.T) {
	user := grantTestUser("tx", "tx@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.uc.userSubRepo = newRestoreScanRepo(map[uuid.UUID]*entities.UserSubscription{
		user.ID: {UserID: user.ID, CurrentPlanCode: "premium", Version: 1, StartedAt: time.Now()},
	})
	h.uc.planRepo = &fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{
		"premium": renewalTestPlan(),
	}}
	uow := &countingUow{inner: &fakeRenewalUow{}}
	h.uc.uow = uow

	if _, err := h.uc.RestorePolicyGrants(context.Background(), h.adminID, false, nil); err != nil {
		t.Fatalf("RestorePolicyGrants lỗi: %v", err)
	}
	if uow.calls != 1 {
		t.Errorf("cả lượt khôi phục phải nằm trong MỘT giao dịch, có = %d", uow.calls)
	}
}

// countingUow counts how many transactions were opened.
type countingUow struct {
	inner *fakeRenewalUow
	calls int
}

func (c *countingUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	c.calls++
	return c.inner.Execute(ctx, fn)
}

// T125: an over-long plan slug is refused as a FORMAT error, and - the point of
// the guard - it never reaches the database.
//
// A slug far longer than the 100-character column cannot use an index, so one
// such item would turn the whole request's slug lookup into a table scan.
func TestAdminGrantBatchRejectsOverlongPlanSlugWithoutQuerying(t *testing.T) {
	good := grantTestUser("sl2", "sl2@example.com", roleslug.User, false)
	// 024 T132: the two items name DIFFERENT accounts. The same account twice
	// is a duplicate by design, which would mask what this test checks.
	h := newGrantHarness(t, good, grantTestUser("sl1", "sl1@example.com", roleslug.User, false))

	overlongSlug := strings.Repeat("s", mapper.MaxPlanSlugLength+1)
	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"sl1@example.com", overlongSlug},
		[2]string{"sl2@example.com", "premium"},
	), "idem-SLUG", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Results[0].Reason != sub_dto.AdminGrantReasonPlanSlugTooLong {
		t.Errorf("mã gói quá dài phải báo plan_slug_too_long, có = %s", res.Results[0].Reason)
	}
	if res.Results[0].Reason == sub_dto.AdminGrantReasonPlanNotFound {
		t.Error("không được báo plan_not_found cho một mã gói quá dài")
	}
	if !strings.Contains(res.Results[0].Message, "100") {
		t.Errorf("thông báo phải nêu giới hạn cụ thể, có = %q", res.Results[0].Message)
	}

	// Exactly ONE lookup for the whole request, and - the part the counter could
	// never check - the over-long slug was not part of it.
	//
	// 024 T150 turned this from a loop of per-slug queries into a single batch, so
	// "one query" alone would now pass even if the guard had stopped working and
	// the oversized slug were sent along with the rest. Asserting the CONTENT of
	// the batch is what keeps the guard honest.
	if h.plans.slugBatchCalls != 1 {
		t.Errorf("mỗi yêu cầu chỉ được tra cứu đúng một số dữ liệu, thực tế = %d lần tra cứu", h.plans.slugBatchCalls)
	}
	for _, sent := range h.plans.slugBatchSlugs {
		if sent == overlongSlug {
			t.Errorf("mã gọi quá dài đã tới cơ sở dữ liệu: %d ký tự", len(sent))
		}
	}
	if h.plans.slugCalls != 1 {
		t.Errorf("mã gọi quá dài không được gọi xuống cơ sở dữ liệu, thực tế = %d lần tra cứu", h.plans.slugCalls)
	}
	if res.Results[1].Status != sub_dto.AdminGrantStatusSuccess {
		t.Errorf("phần tử hợp lệ phải vẫn được cấp, có = %s", res.Results[1].Status)
	}
}

// The slug limit matches the column width and is inclusive, exactly like the
// identifier limit.
func TestMaxPlanSlugLengthIsInclusiveAndMatchesTheColumn(t *testing.T) {
	if !ExceedsPlanSlugLength(strings.Repeat("s", mapper.MaxPlanSlugLength+1)) {
		t.Error("mã gói dài hơn giới hạn phải bị từ chối")
	}
	if ExceedsPlanSlugLength(strings.Repeat("s", mapper.MaxPlanSlugLength)) {
		t.Error("mã gói đúng bằng giới hạn phải được chấp nhận")
	}
	if mapper.MaxPlanSlugLength != 100 {
		t.Errorf("giới hạn phải khớp bề rộng cột subscription.subscription_plans.slug (100), có = %d", mapper.MaxPlanSlugLength)
	}
}

// ---- 024 T143: FR-076's time budget, and one lookup for the whole UUID list ----

// The message quotes the same numbers the use case enforces. The mapper package
// cannot import the use case (that would be a cycle), so the two copies are
// separate literals and this test is what stops them drifting apart - a message
// promising a 30-second limit the code does not honour is worse than no message.
func TestMsgAdminGrantBatchTimeoutMatchesTheEnforcedBudget(t *testing.T) {
	for _, budget := range []struct {
		name    string
		actual  time.Duration
		seconds int
	}{
		{"mot phan tu", adminGrantSingleItemBudget, 5},
		{"danh sach", adminGrantBulkBudget, 60},
	} {
		if int(budget.actual/time.Second) != budget.seconds {
			t.Errorf("ngan sach %s phai la %d giay, co = %v", budget.name, budget.seconds, budget.actual)
		}
		if !strings.Contains(mapper.MsgAdminGrantBatchTimeout, fmt.Sprintf("%d giây", budget.seconds)) {
			t.Errorf("thong bao phai nho muc %d giay cho %s", budget.seconds, budget.name)
		}
	}
}

// FR-076 states the two bounds explicitly: a one-item list answers within 5
// seconds, a full list within 60. A single list is the common case and gets the
// tighter one.
func TestGrantBudgetFor(t *testing.T) {
	if got := grantBudgetFor(1); got != adminGrantSingleItemBudget {
		t.Errorf("danh sach 1 phan tu phai co ngan sach 5s, co = %v", got)
	}
	if got := grantBudgetFor(2); got != adminGrantBulkBudget {
		t.Errorf("tu 2 phan tu tro len phai dung ngan sach 60s, co = %v", got)
	}
	if got := grantBudgetFor(500); got != adminGrantBulkBudget {
		t.Errorf("500 phan tu phai dung ngan sach 60s, co = %v", got)
	}
}

// A batch that runs out of time must STOP, not keep granting. The items already
// granted stand; the rest are reported rather than silently dropped, because a
// short response with no explanation is indistinguishable from a completed one.
func TestAdminGrantBatchStopsGrantingWhenItsTimeBudgetRunsOut(t *testing.T) {
	first := grantTestUser("t1", "t1@example.com", roleslug.User, false)
	second := grantTestUser("t2", "t2@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)

	// A context already past its deadline: the loop must see it on its first
	// check rather than discovering it one database round trip at a time.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	res, err := h.uc.AdminGrantBatch(ctx, h.adminID, grantTestItems(
		[2]string{"t1@example.com", "premium"},
		[2]string{"t2@example.com", "premium"},
	), "idem-TIMEOUT", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}
	if res.Succeeded != 0 {
		t.Errorf("khong duoc cap goi khi het ngan sach, co succeeded=%d", res.Succeeded)
	}
	if res.Total != 2 {
		t.Fatalf("moi phan tu phai co mot dong ket qua, co total=%d", res.Total)
	}
	for i, r := range res.Results {
		if r.Reason != sub_dto.AdminGrantReasonBatchTimeout {
			t.Errorf("phan tu %d phai bao batch_timeout, co = %s", i, r.Reason)
		}
		if r.UserIdentifier == "" || r.PlanSlug == "" {
			t.Errorf("dong ket qua %d phai giu lai dinh danh de van canh, co = %+v", i, r)
		}
	}
	if !strings.Contains(res.Results[0].Message, "Idempotency-Key") {
		t.Errorf("thong bao phai chi ro cach gui lai, co = %q", res.Results[0].Message)
	}
}

// The resolver's doc comment promises the whole list resolves in one call per
// kind. For UUIDs it used to loop one id at a time, so a 500-UUID batch was 500
// sequential round trips.
func TestAdminUserResolverResolvesAllUUIDsInOneCall(t *testing.T) {
	ids := make([]string, 0, 5)
	want := make(map[string]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		id := uuid.New()
		ids = append(ids, id.String())
		want[id.String()] = id
	}

	contract := &fakeIdentityBriefContract{byID: map[uuid.UUID]*identity_dto.UserBriefContractDTO{}}
	for i, id := range ids {
		u := uuid.MustParse(id)
		contract.byID[u] = &identity_dto.UserBriefContractDTO{ID: u, Username: "uuiduser" + string(rune('A'+i))}
	}
	resolver := NewAdminUserResolver(contract)

	got, err := resolver.Resolve(context.Background(), ids)
	if err != nil {
		t.Fatalf("Resolve loi: %v", err)
	}

	if contract.byIDCalls != 1 {
		t.Errorf("phai goi GetByIDs DUNG MOT LAN cho ca danh sach, co = %d", contract.byIDCalls)
	}
	if len(contract.lastByIDs) != 5 {
		t.Errorf("mot lan goi phai mang ca 5 uuid, co = %d", len(contract.lastByIDs))
	}
	for _, id := range ids {
		entry := got[strings.ToLower(id)]
		if entry == nil || entry.UserID != want[id] {
			t.Errorf("uuid %s phai phan giai dung, co = %+v", id, entry)
		}
	}
}
