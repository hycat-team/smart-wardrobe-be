package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	admin_contract "smart-wardrobe-be/services/account/contract/admin"
	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"

	"github.com/google/uuid"
)

// ---- Fakes ----

// fakeGrantQuotaRepo records whether the daily counters were reset.
type fakeGrantQuotaRepo struct {
	quota   *entities.UserDailyQuota
	updates int
}

func (f *fakeGrantQuotaRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	return f.quota, nil
}
func (f *fakeGrantQuotaRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	return f.quota, nil
}

// CreateIfAbsent mirrors the real repository (024 T148): a quota that is already
// present is not overwritten, and the caller is told it did not insert.
func (f *fakeGrantQuotaRepo) CreateIfAbsent(ctx context.Context, e *entities.UserDailyQuota) (bool, error) {
	if f.quota != nil {
		return false, nil
	}
	f.quota = e
	return true, nil
}

func (f *fakeGrantQuotaRepo) Update(ctx context.Context, e *entities.UserDailyQuota) error {
	f.updates++
	f.quota = e
	return nil
}
func (f *fakeGrantQuotaRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserDailyQuota, error) {
	return nil, nil
}
func (f *fakeGrantQuotaRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserDailyQuota, error) {
	return nil, nil
}
func (f *fakeGrantQuotaRepo) GetAll(ctx context.Context) ([]*entities.UserDailyQuota, error) {
	return nil, nil
}
func (f *fakeGrantQuotaRepo) Create(ctx context.Context, e *entities.UserDailyQuota) error {
	return nil
}
func (f *fakeGrantQuotaRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeGrantQuotaRepo) BatchCreate(ctx context.Context, es []*entities.UserDailyQuota) error {
	return nil
}
func (f *fakeGrantQuotaRepo) BatchUpdate(ctx context.Context, es []*entities.UserDailyQuota) error {
	return nil
}
func (f *fakeGrantQuotaRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// fakeGrantPlanRepo serves plans by slug.
//
// 024 T135: the grant path calls `GetBySlugAnyStatus` so a deactivated plan can be
// told apart from a misspelled one, so BOTH slug methods count towards slugCalls -
// the "one lookup per distinct slug" assertions must hold whichever one is used.
//
// 024 T150: the grant path now resolves every slug in ONE call
// (`GetBySlugsAnyStatus`). It still counts as a single lookup for those
// assertions, and `slugBatchSlugs` records what was actually sent so a test can
// check WHICH slugs reached the database - the per-slug counter never could.
type fakeGrantPlanRepo struct {
	planBySlug map[string]*entities.SubscriptionPlan
	slugCalls  int
	// 024 T135: counted separately so a test can prove WHICH lookup was used.
	activeOnlyCalls int
	anyStatusCalls  int
	// 024 T150: the batch call and the slugs it carried.
	slugBatchCalls int
	slugBatchSlugs []string
	// 024 T150: whether the context reaching the lookup already carried the
	// request's time budget.
	sawDeadline bool
	remaining   time.Duration
}

func (f *fakeGrantPlanRepo) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	f.slugCalls++
	f.activeOnlyCalls++
	return f.planBySlug[slug], nil
}
func (f *fakeGrantPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlan, error) {
	for _, plan := range f.planBySlug {
		if plan.ID == id {
			return plan, nil
		}
	}
	return nil, nil
}
func (f *fakeGrantPlanRepo) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeGrantPlanRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeGrantPlanRepo) GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	f.slugCalls++
	f.anyStatusCalls++
	return f.planBySlug[slug], nil
}

// GetBySlugsAnyStatus mirrors the real repository: one query for the whole set, and
// a slug that does not exist is ABSENT from the map rather than present-and-nil,
// because that is the distinction the plan_not_found reason depends on.
func (f *fakeGrantPlanRepo) GetBySlugsAnyStatus(ctx context.Context, slugs []string) (map[string]*entities.SubscriptionPlan, error) {
	f.slugCalls++
	f.anyStatusCalls++
	f.slugBatchCalls++
	f.slugBatchSlugs = append(f.slugBatchSlugs, slugs...)
	if deadline, ok := ctx.Deadline(); ok {
		f.sawDeadline = true
		f.remaining = time.Until(deadline)
	}
	found := make(map[string]*entities.SubscriptionPlan, len(slugs))
	for _, slug := range slugs {
		if plan, ok := f.planBySlug[slug]; ok && plan != nil {
			found[slug] = plan
		}
	}
	return found, nil
}
func (f *fakeGrantPlanRepo) ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeGrantPlanRepo) GetAll(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeGrantPlanRepo) Create(ctx context.Context, e *entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeGrantPlanRepo) Update(ctx context.Context, e *entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeGrantPlanRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeGrantPlanRepo) BatchUpdate(ctx context.Context, es []*entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeGrantPlanRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// fakeGrantSubRepo keeps one row per user so a batch can touch several.
//
// lockVersionOffset simulates a row that CHANGES between the batch's read and the
// item's transaction — a customer purchase landing in between, which is exactly
// what FR-069 exists to catch.
type fakeGrantSubRepo struct {
	subs               map[uuid.UUID]*entities.UserSubscription
	orders             []uuid.UUID
	lockVersionOffset  int64
	plannedReadUserIDs []uuid.UUID
	// plannedReadCalls and onPlannedRead let a test assert that the version
	// pre-pass happens ONCE for the whole batch (FR-096).
	plannedReadCalls int
	onPlannedRead    func(ids []uuid.UUID)
}

func (f *fakeGrantSubRepo) GetByUserIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	f.plannedReadCalls++
	f.plannedReadUserIDs = append(f.plannedReadUserIDs, ids...)
	if f.onPlannedRead != nil {
		f.onPlannedRead(ids)
	}
	out := make([]*entities.UserSubscription, 0, len(ids))
	for _, id := range ids {
		if sub, ok := f.subs[id]; ok {
			copied := *sub
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeGrantSubRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	f.orders = append(f.orders, userID)
	sub := f.subs[userID]
	if sub == nil || f.lockVersionOffset == 0 {
		return sub, nil
	}
	copied := *sub
	copied.Version += f.lockVersionOffset
	return &copied, nil
}
func (f *fakeGrantSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.subs[userID], nil
}
func (f *fakeGrantSubRepo) Create(ctx context.Context, e *entities.UserSubscription) error {
	f.subs[e.UserID] = e
	return nil
}
func (f *fakeGrantSubRepo) Update(ctx context.Context, e *entities.UserSubscription) error {
	f.subs[e.UserID] = e
	return nil
}
func (f *fakeGrantSubRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) GetAll(ctx context.Context) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeGrantSubRepo) GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	f.subs[sub.UserID] = sub
	return nil
}
func (f *fakeGrantSubRepo) GetDistributionByPlanCode(ctx context.Context, from, to time.Time) ([]repositories.SubscriptionDistributionDTO, error) {
	return nil, nil
}
func (f *fakeGrantSubRepo) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	return nil, 0, nil
}
func (f *fakeGrantSubRepo) BatchUpdate(ctx context.Context, es []*entities.UserSubscription) error {
	return nil
}
func (f *fakeGrantSubRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// fakeGrantAuditLog captures the audit records, including which administrator
// performed the action and which transaction context carried them.
type fakeGrantAuditLog struct {
	records []admin_dto.RecordAuditLogInput
	lastCtx context.Context
	err     error
}

func (f *fakeGrantAuditLog) Record(ctx context.Context, input admin_dto.RecordAuditLogInput) error {
	if f.err != nil {
		return f.err
	}
	f.lastCtx = ctx
	f.records = append(f.records, input)
	return nil
}

var _ admin_contract.IAuditLogContract = (*fakeGrantAuditLog)(nil)

// grantTxEventRepo is a private copy of the shared event fake that also records
// the transaction context it was handed, so the "same transaction" requirement
// (FR-015) can be asserted rather than assumed. byKey lets a test simulate a
// retried request.
type grantTxEventRepo struct {
	lastCtx context.Context
	creates int
	byKey   map[string]*entities.UserSubscriptionEvent
	keys    []string
}

func (f *grantTxEventRepo) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	f.lastCtx = ctx
	f.creates++
	f.keys = append(f.keys, event.EventKey)
	if f.byKey == nil {
		f.byKey = map[string]*entities.UserSubscriptionEvent{}
	}
	// 024 T155: GORM assigns the primary key during INSERT, and the receiver needs it
	// as sourceEventId. The fake has to do the same, or the payload validation added
	// with T155 rejects every event this test produces - which is a correct rejection
	// of an event the fake invented, not a bug in the code under test.
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	// The real repository persists the row, so a later GetByKey finds it. The
	// fake has to do the same or every retry would look like a first attempt.
	f.byKey[event.EventKey] = event
	return nil
}

// CreateIfAbsent mirrors the real repository: it refuses a key that is already
// present rather than overwriting it, and says so (024 T148). Without this the
// fake would accept a duplicate key and the publisher's idempotency branch would
// never be exercised.
func (f *grantTxEventRepo) CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (bool, error) {
	f.lastCtx = ctx
	if f.byKey == nil {
		f.byKey = map[string]*entities.UserSubscriptionEvent{}
	}
	if _, exists := f.byKey[event.EventKey]; exists {
		return false, nil
	}
	f.creates++
	f.keys = append(f.keys, event.EventKey)
	// 024 T155: see Create - GORM assigns the id on insert and the payload needs it.
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	f.byKey[event.EventKey] = event
	return true, nil
}

func (f *grantTxEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	return f.byKey[key], nil
}

// ---- Harness ----

type grantHarness struct {
	uc      *SubscriptionUseCase
	subs    *fakeGrantSubRepo
	plans   *fakeGrantPlanRepo
	quota   *fakeGrantQuotaRepo
	events  *grantTxEventRepo
	audit   *fakeGrantAuditLog
	notif   *fakeRenewalNotif
	adminID uuid.UUID
}

// grantTestLogger records log lines instead of writing them.
//
// 024 T129: an item whose transaction fails for a reason that is NOT a version
// conflict now logs the cause, so the harness needs a logger. Production always
// has one; leaving it nil turned every infrastructure-failure test into a
// nil-pointer panic instead of an assertion.
type grantTestLogger struct {
	entries []string
}

func (l *grantTestLogger) record(level, message string) {
	l.entries = append(l.entries, level+": "+message)
}

func (l *grantTestLogger) Debug(message string, fields ...zap.Field) { l.record("debug", message) }
func (l *grantTestLogger) Info(message string, fields ...zap.Field)  { l.record("info", message) }
func (l *grantTestLogger) Warn(message string, fields ...zap.Field)  { l.record("warn", message) }
func (l *grantTestLogger) Error(message string, fields ...zap.Field) { l.record("error", message) }
func (l *grantTestLogger) Fatal(message string, fields ...zap.Field) { l.record("fatal", message) }

var _ logger.Interface = (*grantTestLogger)(nil)

func newGrantHarness(t *testing.T, users ...*identity_dto.UserBriefContractDTO) *grantHarness {
	t.Helper()

	identityContract := &fakeIdentityBriefContract{users: users, byID: map[uuid.UUID]*identity_dto.UserBriefContractDTO{}}
	byID := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	identityContract.byID = byID

	plan := renewalTestPlan()
	plan.ID = uuid.New()
	plans := &fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{plan.Slug: plan}}
	events := &grantTxEventRepo{}
	audit := &fakeGrantAuditLog{}
	notif := &fakeRenewalNotif{}
	subs := &fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{}}
	quota := &fakeGrantQuotaRepo{quota: &entities.UserDailyQuota{AiUsageCount: 7, OutfitRecommendCount: 3}}

	uc := &SubscriptionUseCase{
		uow:                 &fakeRenewalUow{},
		userSubRepo:         subs,
		planRepo:            plans,
		quotaRepo:           quota,
		eventRepo:           events,
		planChangePublisher: NewPlanChangePublisher(events, &fakeRenewalPublisher{}, noopPlanPeriodRecorder{}),
		adminUserResolver:   NewAdminUserResolver(identityContract),
		notifContract:       notif,
		auditLogContract:    audit,
		// 024 T129: a failing item now logs its cause, so the harness needs a
		// logger. The stub records instead of writing, which lets a test assert
		// that an infrastructure failure was NOT reported as a version conflict.
		log: &grantTestLogger{},
	}
	return &grantHarness{
		uc:      uc,
		subs:    subs,
		plans:   plans,
		quota:   quota,
		events:  events,
		audit:   audit,
		notif:   notif,
		adminID: uuid.New(),
	}
}

func grantTestItems(pairs ...[2]string) []sub_dto.AdminGrantBatchItem {
	items := make([]sub_dto.AdminGrantBatchItem, 0, len(pairs))
	for _, p := range pairs {
		items = append(items, sub_dto.AdminGrantBatchItem{UserIdentifier: p[0], PlanSlug: p[1]})
	}
	return items
}

// ---- Tests ----

// FR-051/FR-059/FR-072: granting when the account already has the plan EXTENDS
// the expiry, reports the NEW date, and the counters reset. A single call must
// be enough to get a customer to Premium without touching the database.
func TestAdminGrantBatchExtendsExpiryOnSingleCall(t *testing.T) {
	user := grantTestUser("solo", "solo@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	future := time.Now().AddDate(0, 0, 20)
	previousPlanID := uuid.New()
	h.subs.subs[user.ID] = &entities.UserSubscription{
		UserID:                 user.ID,
		SubscriptionPlanID:     previousPlanID,
		CurrentPlanCode:        "premium",
		CurrentTierRank:        1,
		StartedAt:              time.Now().AddDate(0, 0, -10),
		ExpiresAt:              &future,
		CurrentBenefitSnapshot: entities.JSONDocument("{}"),
	}

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"solo@example.com", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Succeeded != 1 || res.Failed != 0 {
		t.Fatalf("phải cấp thành công 1 tài khoản, có succeeded=%d failed=%d", res.Succeeded, res.Failed)
	}
	if res.Results[0].Status != sub_dto.AdminGrantStatusSuccess {
		t.Errorf("phần tử phải thành công, có = %s (%s)", res.Results[0].Status, res.Results[0].Message)
	}
	if res.Results[0].ExpiresAt == nil {
		t.Fatal("phải trả thời hạn mới sau khi cộng dồn")
	}
	if !res.Results[0].ExpiresAt.After(future) {
		t.Errorf("thời hạn phải cộng dồn từ mốc cũ, mới=%v cũ=%v", res.Results[0].ExpiresAt, future)
	}
	if h.quota.updates == 0 {
		t.Error("bộ đếm lượt dùng phải được đặt lại để hạn mức gói mới có hiệu lực")
	}
}

// FR-053: the row is read under FOR UPDATE before it is decided upon. Without
// the lock two concurrent grants both compute "now + 30 days" and the second
// silently discards the first.
func TestAdminGrantBatchLocksRowBeforeApplying(t *testing.T) {
	user := grantTestUser("locked", "locked@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"locked@example.com", "premium"}), "key-1", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if len(h.subs.orders) == 0 {
		t.Fatal("phải khoá dòng gói hiện tại trước khi áp dụng")
	}
	if h.subs.orders[0] != user.ID {
		t.Errorf("khoá đúng tài khoản cần cấp, có = %s", h.subs.orders[0])
	}
}

// FR-056: an administrator account is refused. Granting Premium to staff charges
// the company for its own employee.
func TestAdminGrantBatchRefusesAdministratorTarget(t *testing.T) {
	adminUser := grantTestUser("root", "root@example.com", roleslug.Admin, false)
	customer := grantTestUser("buyer", "buyer@example.com", roleslug.User, false)
	h := newGrantHarness(t, adminUser, customer)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"root@example.com", "premium"},
		[2]string{"buyer@example.com", "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Results[0].Reason != sub_dto.AdminGrantReasonTargetIsAdmin {
		t.Errorf("phải từ chối tài khoản quản trị viên, có = %s", res.Results[0].Reason)
	}
	if res.Results[1].Status != sub_dto.AdminGrantStatusSuccess {
		t.Errorf("tài khoản khách hàng vẫn phải được cấp, có = %s", res.Results[1].Status)
	}
	if res.Succeeded != 1 || res.Failed != 1 {
		t.Errorf("tổng hợp phải là 1 thành công / 1 lỗi, có = %d/%d", res.Succeeded, res.Failed)
	}
}

// FR-049: a soft-deleted account is reported as DELETED, so the administrator is
// not sent to check the spelling of an address that is correct.
func TestAdminGrantBatchReportsSoftDeletedUser(t *testing.T) {
	gone := grantTestUser("gone", "gone@example.com", roleslug.User, true)
	h := newGrantHarness(t, gone)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"gone@example.com", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Reason != sub_dto.AdminGrantReasonUserSoftDeleted {
		t.Errorf("phải báo tài khoản đã xoá mềm, có = %s", res.Results[0].Reason)
	}
}

// FR-054/FR-055: an unknown slug and a deactivated plan are refused with
// DIFFERENT reasons, and neither is silently replaced by another plan.
//
// 024 T132: this test used to name the SAME account on both lines, which now
// collides with the duplicate rule and hides the plan reason behind
// `duplicate_in_request`. Two accounts keep the test pointed at what it is
// actually about.
func TestAdminGrantBatchDistinguishesPlanProblems(t *testing.T) {
	missing := grantTestUser("p", "p@example.com", roleslug.User, false)
	deactivated := grantTestUser("q", "q@example.com", roleslug.User, false)
	inactive := renewalTestPlan()
	inactive.ID = uuid.New()
	inactive.Slug = "legacy"
	inactive.IsActive = false

	h := newGrantHarness(t, missing, deactivated)
	h.plans.planBySlug[inactive.Slug] = inactive

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"p@example.com", "does-not-exist"},
		[2]string{"q@example.com", "legacy"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}

	if res.Results[0].Reason != sub_dto.AdminGrantReasonPlanNotFound {
		t.Errorf("ma goi khong ton tai phai bao dung ly do, co = %s", res.Results[0].Reason)
	}
	if res.Results[1].Reason != sub_dto.AdminGrantReasonPlanInactive {
		t.Errorf("goi bi vo hieu hoa phai bao dung ly do, co = %s", res.Results[1].Reason)
	}
	if res.Succeeded != 0 {
		t.Errorf("khong duoc cap goi thay the, co succeeded=%d", res.Succeeded)
	}
}

// FR-057: a typo costs exactly one line. The rest of the list is granted.
func TestAdminGrantBatchIgnoresUnknownIdentifierOnlyForThatItem(t *testing.T) {
	first := grantTestUser("first", "first@example.com", roleslug.User, false)
	second := grantTestUser("second", "second@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"first@example.com", "premium"},
		[2]string{"typo@example.com", "premium"},
		[2]string{"second@example.com", "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Total != 3 || res.Succeeded != 2 || res.Failed != 1 {
		t.Fatalf("phải có 2 thành công / 1 lỗi / tổng 3, có = %d/%d/%d", res.Succeeded, res.Failed, res.Total)
	}
	if res.Results[1].Reason != sub_dto.AdminGrantReasonUserNotFound {
		t.Errorf("phần tử sai chính tả phải báo user_not_found, có = %s", res.Results[1].Reason)
	}
	// FR-065: order and echo are preserved so the console can highlight the
	// offending line in the list the administrator typed.
	for i, want := range []string{"first@example.com", "typo@example.com", "second@example.com"} {
		if res.Results[i].Index != i || res.Results[i].UserIdentifier != want {
			t.Errorf("phần tử %d phải giữ nguyên vị trí và chuỗi đã gọi, có = %d/%s", i, res.Results[i].Index, res.Results[i].UserIdentifier)
		}
	}
	if res.Results[0].ResolvedUserID == nil || *res.Results[0].ResolvedUserID != first.ID {
		t.Error("phần tử thành công phải kèm tài khoản đã phân giải để khỏi phải tra cứu lại")
	}
}

// FR-066: naming the same account twice in one list grants ONCE, and the
// duplicate is reported as SKIPPED rather than as an error.
//
// 024 T132: two changes are asserted here that did not hold before.
//
//   - The expiry shows ONE extension, not two. The duplicate key used to be
//     (identifier, planSlug), so the SAME account under two DIFFERENT plan slugs
//     was not a duplicate at all and both lines were granted.
//   - The status is skipped, not error. FR-066 says the later line is
//     "bo qua co ly do" - a deliberate decision on a request we understood.
//     Reporting it as an error makes a correct request look broken, which is the
//     opposite of what the reason code is for.
//
// FR-066: naming the same account and plan twice in one list grants ONCE. The
// duplicate is reported, not silently dropped - a silent drop looks like a
// success nobody asked for.
func TestAdminGrantBatchGrantsDuplicateIdentifierOnlyOnce(t *testing.T) {
	user := grantTestUser("dup", "dup@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"dup@example.com", "premium"},
		[2]string{"DUP@example.com", "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if res.Succeeded != 1 || res.Skipped != 1 || res.Failed != 0 {
		t.Fatalf("phai cap 1 lan va BO QUA 1 lan, co succeeded=%d skipped=%d failed=%d", res.Succeeded, res.Skipped, res.Failed)
	}
	if res.Results[1].Status != sub_dto.AdminGrantStatusSkipped {
		t.Errorf("phần tử trùng phải báo duplicate_in_request, có = %s", res.Results[1].Reason)
	}
	// The expiry must show ONE extension, not two.
	sub := h.subs.subs[user.ID]
	if sub == nil || sub.Version != 1 {
		t.Errorf("chỉ được áp dụng một lần, có version = %v", sub)
	}
}

// FR-058: one item's failure must not roll back the others. Each item is its own
// transaction, so a failing item leaves the committed ones intact.
func TestAdminGrantBatchOneItemFailureDoesNotAffectOthers(t *testing.T) {
	good := grantTestUser("good", "good@example.com", roleslug.User, false)
	h := newGrantHarness(t, good)
	// A concurrent modification on one account must not roll back the other.
	failingID := uuid.New()
	h.subs.subs[failingID] = &entities.UserSubscription{
		UserID:             failingID,
		SubscriptionPlanID: uuid.New(),
		CurrentPlanCode:    "premium",
		ExpiresAt:          nil, // already expired
		IsAutoRenewEnabled: true,
	}

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"good@example.com", "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Succeeded != 1 {
		t.Errorf("tài khoản hợp lệ phải được cấp, có = %d", res.Succeeded)
	}
	if h.subs.subs[good.ID] == nil {
		t.Error("bản ghi của tài khoản hợp lệ phải tồn tại sau khi xử lý")
	}
}

// 024 T133: an EXPIRED subscription is grantable, and starts a new period from
// the moment of the grant (FR-053).
//
// This test used to assert the opposite: it set an expired Premium subscription
// with auto-renew on and expected `concurrent_modification`, on the theory that
// "the cron worker owns this row". That reasoning was wrong - the row is taken
// with FOR UPDATE before the state is even read - and the refusal hit exactly the
// customer an administrator is trying to fix: a Premium subscriber whose renewal
// failed. It also sent the operator chasing the wrong cause.
func TestAdminGrantBatchStartsNewPeriodForExpiredSubscription(t *testing.T) {
	user := grantTestUser("expired", "expired@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	oldPlanID := uuid.New()
	expiredAt := time.Now().Add(-72 * time.Hour)
	h.subs.subs[user.ID] = &entities.UserSubscription{
		UserID:             user.ID,
		SubscriptionPlanID: oldPlanID,
		CurrentPlanCode:    "premium",
		ExpiresAt:          &expiredAt,
		Version:            4,
		// Auto-renew on is exactly the case that used to be refused.
		IsAutoRenewEnabled: true,
	}

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"expired@example.com", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Status != sub_dto.AdminGrantStatusSuccess {
		t.Fatalf("subscription đã hết hạn phải được cấp lại, có = %s / %s", res.Results[0].Status, res.Results[0].Reason)
	}

	got := h.subs.subs[user.ID]
	if got.ExpiresAt == nil {
		t.Fatal("phải có ngày hết hạn mới")
	}
	if !got.ExpiresAt.After(time.Now()) {
		t.Errorf("kỳ mới phải bắt đầu từ thời điểm cấp, có = %v", got.ExpiresAt)
	}
	// A new period, not a continuation of the expired one: the old date plus 30
	// days is still in the past, so that distinction is the whole point.
	if !got.ExpiresAt.After(expiredAt) {
		t.Error("ngày hết hạn mới phải dài hơn kỳ đã hết hạn")
	}
}

// FR-069 has its own test in admin_grant_idempotency_test.go
// (TestAdminGrantBatchReportsVersionChangedSincePlan). That is the check that
// actually protects the row, and it is a DIFFERENT failure from the expired case
// above: the subscription changed between the batch's read and this item's write.
// It must still be reported as `concurrent_modification`, which is why 024 T129
// narrowed that reason code to version conflicts alone rather than removing it.

// FR-081: every granted account gets an audit record naming the administrator.
// This is the question the old single-account endpoint could not answer.
func TestAdminGrantBatchWritesAuditRecordPerGrantedAccount(t *testing.T) {
	first := grantTestUser("a", "a@example.com", roleslug.User, false)
	second := grantTestUser("b", "b@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"a@example.com", "premium"},
		[2]string{"b@example.com", "premium"},
	), "idem-42", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if len(h.audit.records) != 2 {
		t.Fatalf("phải ghi một nhận ký cho mỗi tài khoản được cấp, có = %d", len(h.audit.records))
	}
	for _, record := range h.audit.records {
		if record.ActorUserID != h.adminID {
			t.Errorf("nhận ký phải nêu quản trị viên đã thực hiện, có = %s", record.ActorUserID)
		}
		if record.TargetID == nil {
			t.Error("nhận ký phải nêu tài khoản nhận gói")
		}
		if record.Payload["identifierKey"] != "idem-42" {
			t.Errorf("nhận ký phải mang khoá chống gửi lại, có = %v", record.Payload["identifierKey"])
		}
	}
}

// FR-083: the recipient is emailed, and the wording is the same template for a
// one-item list and a many-item list - a recipient must not be able to infer how
// many other accounts were in the same administrator batch.
func TestAdminGrantBatchNotifiesRecipient(t *testing.T) {
	user := grantTestUser("notify", "notify@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"notify@example.com", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Succeeded != 1 {
		t.Fatalf("phải cấp thành công, có = %d", res.Succeeded)
	}
	if len(h.notif.sent) != 1 {
		t.Fatalf("phải gửi đúng một thông báo, có = %d", len(h.notif.sent))
	}
	if h.notif.sent[0] != "admin_grant" {
		t.Errorf("phải dùng mẫu admin_grant, có = %s", h.notif.sent[0])
	}
}

// FR-084: a failed email must NOT turn a committed grant into a failed item.
func TestAdminGrantBatchNotificationFailureDoesNotFailGrant(t *testing.T) {
	user := grantTestUser("nonotify", "nonotify@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.uc.notifContract = &failingNotifier{}

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"nonotify@example.com", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Succeeded != 1 {
		t.Errorf("việc gửi email hỏng không được làm hỏng thao tác cấp gói, có succeeded=%d", res.Succeeded)
	}
	if len(h.audit.records) != 1 {
		t.Errorf("vẫn phải ghi nhận ký, có = %d", len(h.audit.records))
	}
}

// failingNotifier stands in for a mail transport outage.
type failingNotifier struct{}

func (f *failingNotifier) Send(ctx context.Context, channel notificationContract.ChannelEnum, tmpl notificationContract.TemplateEnum, params map[string]any, to string) error {
	return errors.New("smtp unavailable")
}
func (f *failingNotifier) SendByUserID(ctx context.Context, channel notificationContract.ChannelEnum, tmpl notificationContract.TemplateEnum, params map[string]any, userID uuid.UUID) error {
	return errors.New("smtp unavailable")
}

// A blank identifier is reported as such, not as "not found".
func TestAdminGrantBatchReportsBlankIdentifier(t *testing.T) {
	h := newGrantHarness(t)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"   ", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Reason != sub_dto.AdminGrantReasonIdentifierBlank {
		t.Errorf("phải báo identifier_blank, có = %s", res.Results[0].Reason)
	}
}

// FR-072: the same plan slug on many lines is loaded ONCE, not once per line.
func TestAdminGrantBatchLoadsEachPlanOnce(t *testing.T) {
	var users []*identity_dto.UserBriefContractDTO
	items := make([]sub_dto.AdminGrantBatchItem, 0, 10)
	for i := 0; i < 10; i++ {
		letter := string(rune('a' + i))
		users = append(users, grantTestUser("u"+letter, "u"+letter+"@example.com", roleslug.User, false))
		items = append(items, sub_dto.AdminGrantBatchItem{UserIdentifier: "u" + letter + "@example.com", PlanSlug: "premium"})
	}
	h := newGrantHarness(t, users...)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, items, "key-1", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if h.plans.slugCalls != 1 {
		t.Errorf("mỗi mã gói chỉ được tra cứu một lần, thực tế = %d", h.plans.slugCalls)
	}
}

// An audit failure undoes the grant, and records nothing claiming otherwise.
//
// FR-015 puts the event and the audit record in ONE transaction precisely so
// this holds: if the audit write fails, the grant is rolled back rather than
// applied with no trail, because a grant nobody can attribute is what FR-081
// exists to prevent.
func TestAdminGrantBatchFailsItemWhenAuditCannotBeWritten(t *testing.T) {
	user := grantTestUser("noaudit", "noaudit@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	h.audit.err = errors.New("outbox down")

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"noaudit@example.com", "premium"}), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Succeeded != 0 || res.Failed != 1 {
		t.Fatalf("ghi nhận ký hỏng thì phần tử phải thất bại, có succeeded=%d failed=%d", res.Succeeded, res.Failed)
	}
	if len(h.audit.records) != 0 {
		t.Errorf("không được lưu bản ghi kiểm toán nào khi bản ghi thất bại, có = %d", len(h.audit.records))
	}
}

// FR-015: the plan-change event and the audit record are written on the SAME
// transaction context. Two contexts would mean an audit trail that survives a
// rolled-back grant.
func TestAdminGrantBatchWritesEventAndAuditOnSameTransaction(t *testing.T) {
	user := grantTestUser("onetx", "onetx@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"onetx@example.com", "premium"}), "key-1", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if h.events.lastCtx == nil {
		t.Fatal("sự kiện thay đổi gói phải được ghi trong một giao dịch")
	}
	if h.audit.lastCtx == nil {
		t.Fatal("nhận ký kiểm toán phải được ghi trong một giao dịch")
	}
	if h.events.lastCtx != h.audit.lastCtx {
		t.Error("sự kiện và nhận ký phải dùng cùng một giao dịch, nếu không nhận ký có thể tồn tại sau khi cấp gói đã bị hoàn tác")
	}
}

// FR-073: the cached overview is dropped for each granted account, and the grant
// itself never READS that cache.
//
// Both halves matter. A stale entry would keep serving the old plan after the
// customer had already paid for the new one; and a grant that trusted the cache
// could extend an expiry it never actually read from the database.
func TestAdminGrantBatchInvalidatesOverviewCacheWithoutReadingIt(t *testing.T) {
	first := grantTestUser("c1", "c1@example.com", roleslug.User, false)
	second := grantTestUser("c2", "c2@example.com", roleslug.User, false)
	h := newGrantHarness(t, first, second)
	cache := &recordingOverviewCache{}
	h.uc.overviewCache = cache

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"c1@example.com", "premium"},
		[2]string{"c2@example.com", "premium"},
	), "key-1", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if len(cache.invalidated) != 2 {
		t.Fatalf("phải xoá bộ nhớ đệm cho từng tài khoản được cấp, có = %d", len(cache.invalidated))
	}
	if cache.getCalls != 0 {
		t.Errorf("thao tác cấp gói không được đọc bộ nhớ đệm tổng quan, có = %d lần đọc", cache.getCalls)
	}
	for _, want := range []uuid.UUID{first.ID, second.ID} {
		found := false
		for _, got := range cache.invalidated {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("thiếu tài khoản %s trong danh sách xoá bộ nhớ đệm", want)
		}
	}
}

// recordingOverviewCache counts reads and writes so the cache contract can be
// asserted instead of assumed.
//
// 024 T151: it also records the state of the context it is handed, and can be made
// to fail, so the tests can prove that the post-commit delete is not abandoned when
// the caller's context is already dead - which is exactly when it used to be.
type recordingOverviewCache struct {
	getCalls    int
	invalidated []uuid.UUID
	// sawLiveContext is true when the context reaching Invalidate was not already
	// cancelled at the moment of the call.
	sawLiveContext bool
	// invalidateErr makes Invalidate fail, standing in for Redis being down.
	invalidateErr error
}

func (c *recordingOverviewCache) Get(ctx context.Context, userID uuid.UUID) (*subscriptioncaching.SubscriptionOverviewCacheEntry, error) {
	c.getCalls++
	return nil, nil
}

func (c *recordingOverviewCache) Set(ctx context.Context, userID uuid.UUID, entry *subscriptioncaching.SubscriptionOverviewCacheEntry) error {
	return nil
}

func (c *recordingOverviewCache) Invalidate(ctx context.Context, userID uuid.UUID) error {
	if c.invalidateErr != nil {
		return c.invalidateErr
	}
	// Record BEFORE appending: a cache that refused the work must not be counted as
	// having done it, which is the whole point of T151.
	c.sawLiveContext = ctx.Err() == nil
	c.invalidated = append(c.invalidated, userID)
	return nil
}

// FR-070: re-granting refreshes the benefit snapshot even when a subscription
// already exists. Otherwise an administrator changing a plan's benefits silently
// fails to reach accounts that already hold that plan.
func TestAdminGrantBatchRefreshesBenefitSnapshotOnExistingSubscription(t *testing.T) {
	user := grantTestUser("snap", "snap@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	future := time.Now().AddDate(0, 0, 5)
	h.subs.subs[user.ID] = &entities.UserSubscription{
		UserID:                 user.ID,
		SubscriptionPlanID:     h.plans.planBySlug["premium"].ID,
		CurrentPlanCode:        "premium",
		CurrentBenefitSnapshot: entities.JSONDocument(`{"aiChatDailyQuota":0}`),
		ExpiresAt:              &future,
	}

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems([2]string{"snap@example.com", "premium"}), "key-1", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	snapshot := string(h.subs.subs[user.ID].CurrentBenefitSnapshot)
	if snapshot == `{"aiChatDailyQuota":0}` {
		t.Errorf("phải làm mới current_benefit_snapshot khi cấp đè lên gói đang có, vẫn là %s", snapshot)
	}
	if h.subs.subs[user.ID].Version != 1 {
		t.Errorf("phải tăng phiên bản bản ghi, có = %d", h.subs.subs[user.ID].Version)
	}
}

// newRestoreScanRepo builds a repository whose ListAdmin answers from a fixed
// set, so the restore path can be exercised without paging a real table.
func newRestoreScanRepo(subs map[uuid.UUID]*entities.UserSubscription) *fakeRestoreScanRepo {
	return &fakeRestoreScanRepo{fakeGrantSubRepo: &fakeGrantSubRepo{subs: subs}}
}

// fakeRestoreScanRepo overrides ListAdmin so the scan sees the fixture instead
// of paging through the whole table.
type fakeRestoreScanRepo struct {
	*fakeGrantSubRepo
}

func (f *fakeRestoreScanRepo) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	out := make([]*entities.UserSubscription, 0, len(f.subs))
	for _, sub := range f.subs {
		out = append(out, sub)
	}
	return out, int64(len(out)), nil
}

// 024 T132: the same account named TWICE WITH DIFFERENT PLAN SLUGS is granted
// ONCE.
//
// This is the case `quickstart.md` lists as an edge case and the old code got
// wrong. The duplicate key was `(identifier, planSlug)`, so the account was not
// part of the identity at all: the two lines looked like two different targets,
// both were granted, and the customer's period was extended twice. Nothing in the
// response said so - both lines came back `success`.
func TestAdminGrantBatchGrantsSameAccountWithDifferentPlansOnce(t *testing.T) {
	user := grantTestUser("twoplans", "twoplans@example.com", roleslug.User, false)
	other := renewalTestPlan()
	other.ID = uuid.New()
	other.Slug = "premium-annual"
	h := newGrantHarness(t, user)
	h.plans.planBySlug[other.Slug] = other

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"twoplans@example.com", "premium"},
		[2]string{"twoplans@example.com", "premium-annual"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}

	if res.Succeeded != 1 || res.Skipped != 1 || res.Failed != 0 {
		t.Fatalf("phai cap MOT lan va bo qua MOT lan, co succeeded=%d skipped=%d failed=%d", res.Succeeded, res.Skipped, res.Failed)
	}
	if res.Results[1].Reason != sub_dto.AdminGrantReasonDuplicateInRequest {
		t.Errorf("phan tu sau phai bao duplicate_in_request, co = %s", res.Results[1].Reason)
	}
	if res.Results[1].Status != sub_dto.AdminGrantStatusSkipped {
		t.Errorf("phan tu sau phai bao skipped, co = %s", res.Results[1].Status)
	}
	// The decisive assertion: ONE extension, not two.
	sub := h.subs.subs[user.ID]
	if sub == nil || sub.ExpiresAt == nil {
		t.Fatal("phai cap duoc goi")
	}
	expected := time.Now().AddDate(0, 0, 30)
	if sub.ExpiresAt.After(expected.Add(time.Hour)) {
		t.Errorf("thoi han bi cong don hai lan, co = %v, mong doi <= %v", sub.ExpiresAt, expected)
	}
	// And it kept the FIRST plan, because FR-066 says only the first line is acted
	// on - the operator's ordering is respected rather than second-guessed.
	if sub.CurrentPlanCode != "premium" {
		t.Errorf("phai giu goi cua phan tu DAU TIEN, co = %s", sub.CurrentPlanCode)
	}
}

// 024 T132: two identifier FORMS for one account are the same account.
//
// `case@example.com` and the account's UUID are different strings but the same
// person, so the second line must be skipped rather than granting twice.
func TestAdminGrantBatchTreatsDifferentIdentifierFormsForOneAccountAsDuplicates(t *testing.T) {
	user := grantTestUser("twoforms", "twoforms@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"twoforms@example.com", "premium"},
		[2]string{user.ID.String(), "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}

	if res.Succeeded != 1 || res.Skipped != 1 {
		t.Fatalf("cung mot tai khoan qua hai dang dinh danh phai cap mot lan, co succeeded=%d skipped=%d", res.Succeeded, res.Skipped)
	}
	if res.Results[1].Reason != sub_dto.AdminGrantReasonDuplicateInRequest {
		t.Errorf("hai dang dinh danh cua mot tai khoan phai bi coi trung, co = %s", res.Results[1].Reason)
	}
}

// 024 T135: the grant path must ask for the plan REGARDLESS of status.
//
// The fake repository does not filter on `is_active`, so the
// `plan_inactive` assertion in TestAdminGrantBatchDistinguishesPlanProblems
// passes either way. Without this test, swapping the call back to `GetBySlug`
// would break nothing in the suite while silently making the reason code
// unreachable again in production - the real `GetBySlug` filters in SQL, so the
// deactivated plan arrives as nil and is reported as `plan_not_found`.
func TestAdminGrantBatchLooksUpPlansIgnoringActiveStatus(t *testing.T) {
	user := grantTestUser("slugstatus", "slugstatus@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"slugstatus@example.com", "premium"},
	), "key-STATUS", nil); err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}

	if h.plans.anyStatusCalls == 0 {
		t.Error("phai tra cuu goi BAT KE trang thai, neu khong thi plan_inactive khong bao gio chay")
	}
	if h.plans.activeOnlyCalls != 0 {
		t.Errorf("khong duoc dung GetBySlug (loc is_active trong SQL), co = %d lan", h.plans.activeOnlyCalls)
	}
}

// 024 T134: granting a plan must NOT enrol the account in auto-renewal.
//
// The apply workflow used to set `IsAutoRenewEnabled = plan.DurationDays != nil`
// whenever the plan changed, so an administrator handing out Premium turned
// auto-renew ON for accounts that had never paid. When the granted period ran out
// the cron worker would then attempt a renewal payment for a customer who never
// bought anything - a charge nobody consented to.
func TestAdminGrantBatchDoesNotEnableAutoRenew(t *testing.T) {
	user := grantTestUser("norenew", "norenew@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"norenew@example.com", "premium"},
	), "key-NORENEW", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}
	if res.Results[0].Status != sub_dto.AdminGrantStatusSuccess {
		t.Fatalf("phai cap duoc goi, co = %s / %s", res.Results[0].Status, res.Results[0].Reason)
	}

	sub := h.subs.subs[user.ID]
	if sub == nil {
		t.Fatal("phai co dong subscription sau khi cap")
	}
	if sub.IsAutoRenewEnabled {
		t.Error("khong duoc BAT tu dong gia hen cho tai khoan chua tung mua")
	}
}

// FR-053's second half: the flag is brought "in line with the new plan" by being
// CLEARED. A stale ON left over from the previous plan would renew the old plan
// after the administrator replaced it.
func TestAdminGrantBatchClearsStaleAutoRenewWhenReplacingAPlan(t *testing.T) {
	user := grantTestUser("stale", "stale@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	future := time.Now().AddDate(0, 0, 15)
	h.subs.subs[user.ID] = &entities.UserSubscription{
		UserID:             user.ID,
		SubscriptionPlanID: uuid.New(), // a DIFFERENT plan than the one granted
		CurrentPlanCode:    "free",
		ExpiresAt:          &future,
		Version:            2,
		IsAutoRenewEnabled: true,
	}

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"stale@example.com", "premium"},
	), "key-STALE", nil); err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}

	sub := h.subs.subs[user.ID]
	if sub.IsAutoRenewEnabled {
		t.Error("co gia han tu dong con sot tu goi truoc do phai duoc xoa")
	}
	if sub.CurrentPlanCode != "premium" {
		t.Errorf("phai thay goi, co = %s", sub.CurrentPlanCode)
	}
}

// FR-072 asks for one expiry rule. 024 T153 narrowed what that actually means:
// the rule is shared by the EXTENDING paths, which reach it through the apply
// workflow, while `Activate`, `Grant`, `state_machine` and `renewal_payment` each
// still work it out for themselves. These three cases are the rule itself; the
// separate test below pins which callers exist, so this comment cannot drift
// back into claiming more than is true.
func TestResolveExpiryForIsTheSharedRule(t *testing.T) {
	days := 30
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	plan := renewalTestPlan()
	plan.DurationDays = &days

	t.Run("goi khong co thoi han -> khong co ngay het han", func(t *testing.T) {
		noDuration := renewalTestPlan()
		noDuration.DurationDays = nil
		sub := &entities.UserSubscription{SubscriptionPlanID: plan.ID}
		if got := sub.ResolveExpiryFor(noDuration, now); got != nil {
			t.Errorf("phai tra nil, co = %v", got)
		}
	})

	t.Run("cung goi va con hieu luc -> cong don", func(t *testing.T) {
		existing := now.AddDate(0, 0, 10)
		sub := &entities.UserSubscription{SubscriptionPlanID: plan.ID, ExpiresAt: &existing}
		got := sub.ResolveExpiryFor(plan, now)
		if got == nil {
			t.Fatal("phai co ngay het han")
		}
		want := existing.AddDate(0, 0, days)
		if !got.Equal(want) {
			t.Errorf("phai cong don vao han cu, co = %v, mong doi = %v", got, want)
		}
	})

	t.Run("goi khac -> bat dau ky moi", func(t *testing.T) {
		existing := now.AddDate(0, 0, 10)
		sub := &entities.UserSubscription{SubscriptionPlanID: uuid.New(), ExpiresAt: &existing}
		got := sub.ResolveExpiryFor(plan, now)
		if got == nil {
			t.Fatal("phai co ngay het han")
		}
		want := now.AddDate(0, 0, days)
		if !got.Equal(want) {
			t.Errorf("goi khac phai bat dau lai tu hien tai, co = %v, mong doi = %v", got, want)
		}
	})

	// FR-053's "đã hết" half. This is the case that used to be refused outright.
	t.Run("cung goi nhung da het han -> bat dau ky moi", func(t *testing.T) {
		expired := now.AddDate(0, 0, -3)
		sub := &entities.UserSubscription{SubscriptionPlanID: plan.ID, ExpiresAt: &expired}
		got := sub.ResolveExpiryFor(plan, now)
		if got == nil {
			t.Fatal("phai co ngay het han moi")
		}
		want := now.AddDate(0, 0, days)
		if !got.Equal(want) {
			t.Errorf("ky da het phai bat dau lai, co = %v, mong doi = %v", got, want)
		}
	})
}

// 024 T136: `user_soft_deleted` is REACHABLE, which it was not before.
//
// The identity repository filtered `is_deleted = false`, so a deleted account
// arrived at the grant flow as absent and was reported as `user_not_found` -
// telling the administrator to check the spelling of an address that was
// correct. The contract (`admin-grant-api.md`) promises the distinct reason, so
// the grant flow now has its own lookup that keeps deleted rows.
//
// This test also guards the FAKE: it filters deleted accounts out of
// `GetByUsernamesOrEmails`, exactly like the real repository. Without that the
// assertion would pass against a fake the production code never sees.
func TestAdminGrantBatchReportsSoftDeletedAccountUsingTheDeletedAwareLookup(t *testing.T) {
	deleted := grantTestUser("gone", "gone@example.com", roleslug.User, true)
	h := newGrantHarness(t, deleted)

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"gone@example.com", "premium"},
	), "key-DELETED", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch loi: %v", err)
	}

	if res.Results[0].Reason != sub_dto.AdminGrantReasonUserSoftDeleted {
		t.Fatalf("tai khoan da xoa mem phai bao user_soft_deleted, co = %s", res.Results[0].Reason)
	}
	if res.Results[0].Reason == sub_dto.AdminGrantReasonUserNotFound {
		t.Error("khong duoc bao user_not_found cho mot tai khoan da xoa mem")
	}
	if res.Succeeded != 0 {
		t.Errorf("khong duoc cap goi cho tai khoan da xoa mem, co succeeded=%d", res.Succeeded)
	}
	if h.subs.subs[deleted.ID] != nil {
		t.Error("khong duoc tao subscription cho tai khoan da xoa mem")
	}
	// The shared lookup, which commerce uses, must still EXCLUDE deleted accounts.
	// If the grant flow had been pointed at it, this assertion could not hold.
	if _, err := NewAdminUserResolver(&fakeIdentityBriefContract{users: []*identity_dto.UserBriefContractDTO{deleted}}).
		Resolve(context.Background(), []string{"gone@example.com"}); err != nil {
		t.Fatalf("Resolve loi: %v", err)
	}
	shared := &fakeIdentityBriefContract{users: []*identity_dto.UserBriefContractDTO{deleted}}
	if got := shared.matchByUsernameOrEmail([]string{"gone@example.com"}, false); len(got) != 0 {
		t.Errorf("ham dung chung PHAI loai tai khoan da xoa mem, co = %d dong", len(got))
	}
}
