package campaign

import (
	"context"
	"sort"
	"sync"
	"time"

	admindto "smart-wardrobe-be/services/account/dto/admin"
	notificationcontract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
)

// ---- 025 T036-T038: fakes for the quota rules ----
//
// The claim ledger's three guarantees are all DATA-LAYER mechanisms: a UNIQUE
// constraint, an advisory lock, and a partial unique index. A hand-written fake
// models them by hand, which is the wrong tool - removing `ON CONFLICT` from the
// repository would leave every assertion here green.
//
// So these fakes exist only to drive the use case's control flow, and the
// assertions that matter are about WHICH BUCKET the use case chose and what it
// wrote, not about whether the database would have stopped it. The database-level
// guarantees are covered by the repository's own tests and by the quickstart
// scenarios K2.1-K2.3 (quickstart.md §2).
//
// What the fakes DO reproduce faithfully, because the use case depends on it:
// `Create` refuses a second claim for one account (FR-025), `GetWatermark`
// reports the single `fills_main_quota` row, and `LockClaimWindow` serialises
// concurrent callers the way `pg_advisory_xact_lock` does.

// fakeClaimRepo is an in-memory stand-in for the claim ledger.
type fakeClaimRepo struct {
	// advisory is the transaction-scoped lock. Real mutual exclusion, because
	// `pg_advisory_xact_lock` blocks the second caller until the first commits.
	advisory   sync.Mutex
	mu         sync.Mutex
	claims     map[uuid.UUID]*entities.CampaignClaim
	everLocked bool
	// failNextCount makes the count fail once, so the "a failed count must not be
	// read as zero" rule can be exercised.
	failNextCount bool
	// watermarkMissing simulates the FR-030 anomaly: the main budget is full but
	// no row carries the flag.
	watermarkMissing bool
	// countHold delays a bucket count AFTER its value has been read.
	//
	// It is zero everywhere except the concurrent sweep test, and it exists to
	// make one specific mistake deterministically visible: reading the budget
	// OUTSIDE the advisory lock. A count is a snapshot, and a caller that is not
	// serialised will act on the snapshot it took rather than on the number the
	// last writer left behind. Sleeping after the read guarantees that two such
	// callers hold the same stale number at the same moment, instead of the test
	// depending on the scheduler to interleave them. With the lock taken, the
	// second caller is still waiting and re-reads a fresh number.
	countHold time.Duration
}

func newFakeClaimRepo() *fakeClaimRepo {
	return &fakeClaimRepo{claims: map[uuid.UUID]*entities.CampaignClaim{}}
}

// LockClaimWindow models `pg_advisory_xact_lock`: it is taken for the duration of
// the transaction and released at COMMIT or ROLLBACK.
//
// It is a REAL mutual exclusion, not a counter. That is the point of the fake:
// `pg_advisory_xact_lock` blocks the second caller until the first commits, and a
// fake that only recorded "a lock was taken" would let a use case that counts
// outside the lock pass every test here while still overrunning the ceiling in
// production.
func (f *fakeClaimRepo) LockClaimWindow(ctx context.Context, campaignCode string) error {
	f.advisory.Lock()
	f.mu.Lock()
	f.everLocked = true
	f.mu.Unlock()
	return nil
}

// releaseLock models the COMMIT — and the ROLLBACK, which releases it too.
func (f *fakeClaimRepo) releaseLock() {
	f.mu.Lock()
	held := f.everLocked
	f.mu.Unlock()
	if held {
		f.advisory.Unlock()
	}
}

func (f *fakeClaimRepo) lockWasTaken() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.everLocked
}

func (f *fakeClaimRepo) CountByBucket(ctx context.Context, campaignCode string, bucket entities.GrantBucket) (int64, error) {
	f.mu.Lock()
	failNext := f.failNextCount
	f.failNextCount = false
	f.mu.Unlock()
	if failNext {
		return 0, context.DeadlineExceeded
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, claim := range f.claims {
		if claim.CampaignCode == campaignCode && claim.GrantBucket == bucket {
			count++
		}
	}
	// The value is already a snapshot. The wait happens after the read on
	// purpose — see `countHold`.
	hold := f.countHold
	f.mu.Unlock()
	if hold > 0 {
		time.Sleep(hold)
	}
	f.mu.Lock()
	return count, nil
}

func (f *fakeClaimRepo) GetWatermark(ctx context.Context, campaignCode string) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.watermarkMissing {
		return time.Time{}, false, nil
	}
	for _, claim := range f.claims {
		if claim.CampaignCode == campaignCode && claim.FillsMainQuota {
			return claim.RegisteredAt, true, nil
		}
	}
	return time.Time{}, false, nil
}

func (f *fakeClaimRepo) Create(ctx context.Context, claim *entities.CampaignClaim) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.claims[claim.UserID]; exists {
		return false, nil
	}
	copied := *claim
	if copied.ID == uuid.Nil {
		copied.ID = uuid.New()
	}
	f.claims[claim.UserID] = &copied
	return true, nil
}

func (f *fakeClaimRepo) ListByCampaign(ctx context.Context, campaignCode string, limit int) ([]*entities.CampaignClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*entities.CampaignClaim
	for _, claim := range f.claims {
		if claim.CampaignCode == campaignCode {
			out = append(out, claim)
		}
	}
	return out, nil
}

// ListPageByCampaign serves the operator look-up's paginated, filtered read.
//
// The filter and the ordering are reproduced rather than stubbed, because both are
// what FR-067 is about: a fake that ignored `From`/`To` would let a filter
// regression pass, and a fake that did not sort would hand the test rows in map
// order and prove nothing about the page an operator sees.
func (f *fakeClaimRepo) ListPageByCampaign(
	ctx context.Context,
	campaignCode string,
	filter repositories.CampaignClaimsFilter,
	page, limit int,
) ([]*entities.CampaignClaim, int64, error) {
	f.mu.Lock()
	matched := make([]*entities.CampaignClaim, 0, len(f.claims))
	for _, claim := range f.claims {
		if claim.CampaignCode != campaignCode {
			continue
		}
		// Half-open [From, To), matching the repository and the sweep's own window
		// query: From inclusive, To exclusive.
		if filter.From != nil && claim.GrantedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && !claim.GrantedAt.Before(*filter.To) {
			continue
		}
		copied := *claim
		matched = append(matched, &copied)
	}
	f.mu.Unlock()

	// Newest-first, with the id breaking a tie so two grants written in the same
	// instant keep a stable order between calls.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].GrantedAt.Equal(matched[j].GrantedAt) {
			return matched[i].ID.String() < matched[j].ID.String()
		}
		return matched[i].GrantedAt.After(matched[j].GrantedAt)
	})

	total := int64(len(matched))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	start := (page - 1) * limit
	if start >= len(matched) {
		return nil, total, nil
	}
	end := start + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

// GetByUser returns the account's claim whatever campaign wrote it.
//
// The "whatever campaign" part is the point of the method, so the fake searches
// the whole map rather than a per-campaign index — a fake that scoped itself to
// one campaign could not represent FR-012 at all.
func (f *fakeClaimRepo) GetByUser(ctx context.Context, userID uuid.UUID) (*entities.CampaignClaim, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	claim, ok := f.claims[userID]
	if !ok {
		return nil, false, nil
	}
	copied := *claim
	return &copied, true, nil
}

// seed inserts a claim directly, bypassing the use case, so a test can start from
// an arbitrary ledger state.
func (f *fakeClaimRepo) seed(campaignCode string, userID uuid.UUID, registeredAt time.Time, bucket entities.GrantBucket, fills bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claims[userID] = &entities.CampaignClaim{
		ID:             uuid.New(),
		UserID:         userID,
		CampaignCode:   campaignCode,
		PlanSlug:       "premium-monthly",
		PlanID:         uuid.New(),
		RegisteredAt:   registeredAt,
		ExpiresAt:      registeredAt.AddDate(0, 0, 30),
		GrantedAt:      registeredAt,
		GrantBucket:    bucket,
		FillsMainQuota: fills,
	}
}

func (f *fakeClaimRepo) claimFor(userID uuid.UUID) *entities.CampaignClaim {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claims[userID]
}

func (f *fakeClaimRepo) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.claims)
}

func (f *fakeClaimRepo) countBucket(campaignCode string, bucket entities.GrantBucket) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int
	for _, claim := range f.claims {
		if claim.CampaignCode == campaignCode && claim.GrantBucket == bucket {
			count++
		}
	}
	return count
}

func (f *fakeClaimRepo) watermarkCount(campaignCode string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int
	for _, claim := range f.claims {
		if claim.CampaignCode == campaignCode && claim.FillsMainQuota {
			count++
		}
	}
	return count
}

// ---- subscription fakes ----

type fakeSubRepo struct {
	mu   sync.Mutex
	subs map[uuid.UUID]*entities.UserSubscription
}

func newFakeSubRepo() *fakeSubRepo {
	return &fakeSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{}}
}

func (f *fakeSubRepo) put(sub *entities.UserSubscription) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subs[sub.UserID] = sub
}

func (f *fakeSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sub, ok := f.subs[userID]
	if !ok {
		return nil, nil
	}
	copied := *sub
	return &copied, nil
}

func (f *fakeSubRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.GetByUserID(ctx, userID)
}

func (f *fakeSubRepo) GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*entities.UserSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*entities.UserSubscription, 0, len(userIDs))
	for _, id := range userIDs {
		if sub, ok := f.subs[id]; ok {
			copied := *sub
			out = append(out, &copied)
		}
	}
	return out, nil
}

// GetByIDs satisfies the generic repository contract; the campaign path never
// calls it, but a fake that omits it does not satisfy the interface at all.
func (f *fakeSubRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return f.GetByUserIDs(ctx, ids)
}

func (f *fakeSubRepo) Create(ctx context.Context, sub *entities.UserSubscription) error {
	f.put(sub)
	return nil
}

func (f *fakeSubRepo) Update(ctx context.Context, sub *entities.UserSubscription) error {
	f.put(sub)
	return nil
}

func (f *fakeSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	return nil
}

func (f *fakeSubRepo) GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}

func (f *fakeSubRepo) GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error) {
	return nil, nil
}

func (f *fakeSubRepo) GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error) {
	return f.GetByUserID(ctx, userID)
}

func (f *fakeSubRepo) GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}

func (f *fakeSubRepo) GetDistributionByPlanCode(ctx context.Context) ([]repositories.SubscriptionDistributionDTO, error) {
	return nil, nil
}

func (f *fakeSubRepo) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	return nil, 0, nil
}

func (f *fakeSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserSubscription, error) {
	return nil, nil
}

func (f *fakeSubRepo) GetAll(ctx context.Context) ([]*entities.UserSubscription, error) {
	return nil, nil
}

func (f *fakeSubRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func (f *fakeSubRepo) BatchUpdate(ctx context.Context, es []*entities.UserSubscription) error {
	return nil
}

func (f *fakeSubRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// ---- plan fake ----

type fakePlanRepo struct {
	mu    sync.Mutex
	plans map[string]*entities.SubscriptionPlan
}

func newFakePlanRepo() *fakePlanRepo {
	return &fakePlanRepo{plans: map[string]*entities.SubscriptionPlan{}}
}

func (f *fakePlanRepo) put(plan *entities.SubscriptionPlan) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.plans[plan.Slug] = plan
}

func (f *fakePlanRepo) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	plan, ok := f.plans[slug]
	if !ok || !plan.IsActive {
		return nil, nil
	}
	return plan, nil
}

func (f *fakePlanRepo) GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.plans[slug], nil
}

func (f *fakePlanRepo) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakePlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakePlanRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakePlanRepo) GetBySlugsAnyStatus(ctx context.Context, slugs []string) (map[string]*entities.SubscriptionPlan, error) {
	return map[string]*entities.SubscriptionPlan{}, nil
}

func (f *fakePlanRepo) ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakePlanRepo) GetAll(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakePlanRepo) Create(ctx context.Context, e *entities.SubscriptionPlan) error { return nil }
func (f *fakePlanRepo) Update(ctx context.Context, e *entities.SubscriptionPlan) error { return nil }
func (f *fakePlanRepo) Delete(ctx context.Context, id uuid.UUID) error                 { return nil }
func (f *fakePlanRepo) BatchUpdate(ctx context.Context, es []*entities.SubscriptionPlan) error {
	return nil
}

func (f *fakePlanRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// ---- event / audit / notification fakes ----

type recordedEvent struct {
	EventKey string
	Event    *entities.UserSubscriptionEvent
	Sub      *entities.UserSubscription
	Plan     *entities.SubscriptionPlan
}

type fakeEventRepo struct {
	mu     sync.Mutex
	byKey  map[string]*entities.UserSubscriptionEvent
	events []recordedEvent
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{byKey: map[string]*entities.UserSubscriptionEvent{}}
}

func (f *fakeEventRepo) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, recordedEvent{Event: event})
	return nil
}

func (f *fakeEventRepo) CreateIfAbsent(ctx context.Context, event *entities.UserSubscriptionEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.byKey[event.EventKey]; exists {
		return false, nil
	}
	// GORM assigns the primary key during INSERT, and the receiver needs it as
	// `sourceEventId` to de-duplicate on. A fake that left it nil would make every
	// publish fail validation for a reason that has nothing to do with the code
	// under test.
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	f.byKey[event.EventKey] = event
	f.events = append(f.events, recordedEvent{Event: event})
	return true, nil
}

func (f *fakeEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.byKey[key], nil
}

func (f *fakeEventRepo) published() []recordedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedEvent, len(f.events))
	copy(out, f.events)
	return out
}

type fakePublisher struct {
	mu        sync.Mutex
	published []publishedEvent
	// failNext makes the NEXT Publish fail, so the atomicity of a grant can be
	// exercised: FR-039 requires that a plan which cannot publish its event leaves
	// nothing behind.
	failNext bool
}

type publishedEvent struct {
	Topic   string
	Payload any
}

func (f *fakePublisher) Publish(ctx context.Context, topic string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return context.DeadlineExceeded
	}
	f.published = append(f.published, publishedEvent{Topic: topic, Payload: payload})
	return nil
}

func (f *fakePublisher) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.published)
}

// payloads returns the published message bodies, for assertions on what the AI
// service would receive.
func (f *fakePublisher) payloads() []publishedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]publishedEvent, len(f.published))
	copy(out, f.published)
	return out
}

type recordedAudit struct {
	actor      uuid.UUID
	Action     string
	TargetType string
	TargetID   *uuid.UUID
	Payload    map[string]any
}

// ActorID is the recorded actor.
func (r recordedAudit) ActorID() uuid.UUID { return r.actor }

// ActorIsZero reports whether NO administrator performed the action, which is
// what a campaign grant records: the column is NOT NULL, and a sentinel is what
// says "nobody" rather than "unknown".
func (r recordedAudit) ActorIsZero() bool { return r.actor == uuid.Nil }

// fakeAuditContract records the audit commands a use case issues.
//
// WHY `fail` RECORDS FIRST AND FAILS AFTERWARDS, rather than refusing to record. FR-061
// — "a failed write MUST NOT leave an audit record" — is a statement about the
// ROLLBACK, and a fake that simply refuses would satisfy every assertion about it
// without the rollback doing any work: the record would never have existed. Recording
// and then failing is the shape that actually exercises the guarantee, and it is a real
// one: the outbox row is inserted inside the caller's transaction, so anything that
// makes the TRANSACTION fail after that point — a later step in the same use case, or a
// COMMIT that does not land — leaves the row to be rolled back with it. That is exactly
// the situation 026's T036 rollback test and 024's grant atomicity rule care about.
type fakeAuditContract struct {
	mu      sync.Mutex
	records []recordedAudit
	// issued counts EVERY command `Record` was handed, including the ones a rollback
	// then removed, and is therefore the only way a test can tell "the rollback undid the
	// record" apart from "nothing was ever written".
	issued int
	fail   bool
}

func (f *fakeAuditContract) Record(ctx context.Context, input admindto.RecordAuditLogInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issued++
	f.records = append(f.records, recordedAudit{
		actor:      input.ActorUserID,
		Action:     input.Action,
		TargetType: input.TargetType,
		TargetID:   input.TargetID,
		Payload:    input.Payload,
	})
	if f.fail {
		return context.DeadlineExceeded
	}
	return nil
}

// issuedCount is how many audit commands were handed to the fake, rolled back or not.
func (f *fakeAuditContract) issuedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.issued
}

func (f *fakeAuditContract) all() []recordedAudit {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedAudit, len(f.records))
	copy(out, f.records)
	return out
}

// truncateTo drops every record recorded after the given count.
//
// 026 T036: it models the ROLLBACK of an outbox insert. `Record` appends the moment it
// is called, because that is what publishing to the outbox looks like from inside the
// transaction — and in production the row disappears with the transaction if it rolls
// back. Without this, a use case that issued its audit command AFTER the transaction
// would leave the record behind and pass every assertion in the grant tests, while
// FR-061 ("a failed write leaves no audit record") was broken.
func (f *fakeAuditContract) truncateTo(count int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if count < 0 || count > len(f.records) {
		return
	}
	f.records = f.records[:count]
}

type sentEmail struct {
	Template string
	Params   map[string]any
	UserID   uuid.UUID
}

type fakeNotifContract struct {
	mu   sync.Mutex
	sent []sentEmail
	fail bool
}

func (f *fakeNotifContract) Send(ctx context.Context, channel notificationcontract.ChannelEnum, tmpl notificationcontract.TemplateEnum, params map[string]any, to string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return context.DeadlineExceeded
	}
	f.sent = append(f.sent, sentEmail{Template: string(tmpl), Params: params})
	return nil
}

func (f *fakeNotifContract) SendByUserID(ctx context.Context, channel notificationcontract.ChannelEnum, tmpl notificationcontract.TemplateEnum, params map[string]any, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return context.DeadlineExceeded
	}
	f.sent = append(f.sent, sentEmail{Template: string(tmpl), Params: params, UserID: userID})
	return nil
}

func (f *fakeNotifContract) emails() []sentEmail {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sentEmail, len(f.sent))
	copy(out, f.sent)
	return out
}

// passThroughUow runs the closure against the caller's context.
//
// The campaign use case opens exactly ONE transaction (R3), so a test that only
// cares about decisions does not need a database. The claim ledger is the only
// component that requires a real transaction, and its concurrency behaviour is
// covered by the repository tests and quickstart K2.3.
type passThroughUow struct{}

func (passThroughUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
