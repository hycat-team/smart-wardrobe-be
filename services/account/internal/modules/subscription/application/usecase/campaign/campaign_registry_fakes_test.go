package campaign

import (
	"context"
	"sort"
	"sync"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedentities "smart-wardrobe-be/shared/domain/entities"

	"github.com/google/uuid"
)

// 026 T021: the campaign REGISTRY's test scaffolding, plus the two row builders
// every campaign test uses.
//
// It is a separate file from `campaign_fakes_test.go` because the ledger and the
// registry are two different collaborators: the ledger is append-only and its
// guarantees live in database constraints, while the registry is a mutable row that
// a version guards. Splitting them keeps each file's header able to say what kind
// of fake it is, which is the first thing a reader needs to know before deciding
// whether to trust an assertion made through it.
// ---- helpers ----

// testCampaign builds one row of the registry, the shape the production code now
// receives (026).
//
// A POINTER and an `*entities.Campaign`, not a configuration value: the list these
// tests build used to be `[]sharedconfig.CampaignConfig` handed straight to the
// use case, and after T015/T016 the same rows arrive as
// `[]*entities.Campaign` from `ICampaignRepository.ListOpen`. Keeping the helper's
// shape identical to the production one is what stops a test from passing against a
// type the system no longer uses.
//
// `endsAt` is nil for an open-ended campaign, which is NOT "the zero time" â€” that
// distinction is condition 2's whole subject.
func testCampaign(code string, planSlug string, quota, reserve int, startsAt, endsAt *time.Time) *entities.Campaign {
	return &entities.Campaign{
		Code:     code,
		PlanSlug: planSlug,
		Quota:    quota,
		Reserve:  reserve,
		StartsAt: *startsAt,
		EndsAt:   endsAt,
		Version:  1,
	}
}

// closedCampaign is a registry row an operator has force-closed (026, FR-031).
//
// Its window is deliberately still OPEN, so a test that uses it is testing the
// closed column and nothing else: if the answer were "asleep because the window
// closed", the test would pass for the wrong reason.
func closedCampaign(code string, planSlug string, quota, reserve int, startsAt, endsAt *time.Time) *entities.Campaign {
	campaign := testCampaign(code, planSlug, quota, reserve, startsAt, endsAt)
	closedAt := startsAt.Add(24 * time.Hour)
	campaign.ClosedAt = &closedAt
	return campaign
}

// fakeCampaignRepo is an in-memory registry for the use-case tests.
//
// It reproduces `ListOpen`'s THREE terms â€” not closed, inside the window â€” rather
// than returning whatever the test built, because those terms are the FR-031 rule
// and a fake that ignored them would let a use case keep using a closed campaign
// while every test stayed green (research.md R7's argument, applied to the read).
//
// It also carries the ONE bug this fake must be able to express: a registry that
// fails to read. The grant path has to answer that with a retryable error rather
// than with "no campaign configured", and a fake that cannot fail cannot prove it.
type fakeCampaignRepo struct {
	mu        sync.Mutex
	campaigns []*entities.Campaign
	// grantedCodes drives ExistsClaims. The fake holds no ledger of its own, so a
	// test states which campaign has granted; the real EXISTS query is covered by
	// the repository integration test rather than by a unit test.
	grantedCodes map[string]bool
	failReads    bool
	listCalls    int
	openCalls    int
	getCalls     int
	shareCalls   int
	// closeOnShareCode and budgetOnShare stage an OPERATOR'S CHANGE at the one
	// instant a fake can make it land between two reads: the moment the LOCKED read
	// is served.
	//
	// FR-045 (026 T082/T083) is exactly a statement about that gap. Conditions 1-7
	// read the campaign outside the grant's transaction, and the decider re-reads the
	// row under a share lock inside it; a close or a budget edit that lands between
	// the two is invisible to the first read and decisive for the second. Without a
	// way to place a change there, both halves of FR-045 are unreachable in a unit
	// test: `campaign_db_close_integration_test.go` needs a second connection to
	// block, and this fake has only one.
	closeOnShareCode string
	budgetOnShare    *budgetChangeOnShare
}

// budgetChangeOnShare is the budget an operator's edit installs at the locked read.
type budgetChangeOnShare struct {
	code    string
	quota   int
	reserve int
}

func newFakeCampaignRepo(campaigns ...*entities.Campaign) *fakeCampaignRepo {
	return &fakeCampaignRepo{campaigns: campaigns, grantedCodes: map[string]bool{}}
}

// markGranted tells the fake that a campaign has at least one claim (FR-020).
func (f *fakeCampaignRepo) markGranted(campaignCode string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.grantedCodes[campaignCode] = true
}

// put adds or replaces a row by code, the way an operator's API call would.
func (f *fakeCampaignRepo) put(campaign *entities.Campaign) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.campaigns {
		if f.campaigns[i].Code == campaign.Code {
			f.campaigns[i] = campaign
			return
		}
	}
	f.campaigns = append(f.campaigns, campaign)
}

// ListAll returns every row, newest-starting first with `code DESC` breaking a tie
// (FR-040).
func (f *fakeCampaignRepo) ListAll(ctx context.Context) ([]*entities.Campaign, error) {
	f.mu.Lock()
	f.listCalls++
	fail := f.failReads
	f.mu.Unlock()
	if fail {
		return nil, context.DeadlineExceeded
	}
	return f.orderedRows(), nil
}

// ListOpen returns the not-closed subset, matching the repository's WHERE clause.
//
// The WINDOW is deliberately not applied here: `campaignEvaluator.isAwake` is the
// single owner of that predicate (data-model.md §4.3), and a fake that filtered it
// out would make condition 2's `campaign_sleeping` and `campaign_not_started`
// answers unreachable â€” which is exactly the ambiguity this package had to resolve.
func (f *fakeCampaignRepo) ListOpen(ctx context.Context) ([]*entities.Campaign, error) {
	f.mu.Lock()
	f.openCalls++
	fail := f.failReads
	f.mu.Unlock()
	if fail {
		return nil, context.DeadlineExceeded
	}
	open := make([]*entities.Campaign, 0, len(f.rows()))
	for _, campaign := range f.orderedRows() {
		if campaign.ClosedAt == nil {
			open = append(open, campaign)
		}
	}
	return open, nil
}

func (f *fakeCampaignRepo) GetByCode(ctx context.Context, campaignCode string) (*entities.Campaign, error) {
	f.mu.Lock()
	f.getCalls++
	fail := f.failReads
	f.mu.Unlock()
	if fail {
		return nil, context.DeadlineExceeded
	}
	return f.byCode(campaignCode), nil
}

// GetByCodeForShare is a plain read in the fake: the real one takes a row lock, and
// a unit test has no second connection to make that observable. `shareCalls` is
// what proves the PRODUCTION path asks for the locked read, which is the part a
// unit test can check (FR-045).
func (f *fakeCampaignRepo) GetByCodeForShare(ctx context.Context, campaignCode string) (*entities.Campaign, error) {
	f.mu.Lock()
	f.shareCalls++
	fail := f.failReads
	f.mu.Unlock()
	if fail {
		return nil, context.DeadlineExceeded
	}
	// THE CHANGE LANDS HERE, and it is the only place a test can put one. The
	// production sequence is "evaluate from the rows we already read" (conditions 1-7)
	// and then, inside the transaction, "re-read under a share lock" (FR-045); an edit
	// committed between those two reads is what the re-read exists to catch. Applying
	// it here reproduces exactly that ordering, because every call after this one in
	// the grant path sees the NEW row.
	f.applyShareChange(campaignCode)
	return f.byCode(campaignCode), nil
}

// applyShareChange stages the closure or the budget edit the test armed for this code.
//
// THE ROW IS REPLACED, NOT MUTATED IN PLACE, and that is load-bearing rather than
// stylistic. The pointer the evaluator chose came out of `ListOpen`, so mutating the
// row behind it would change that "already read" copy too — and a test then observing a
// skip could not tell a decision made from the re-read row from one made from a copy
// that had been rewritten underneath it. Replacing the slice element leaves the
// evaluator holding the values conditions 1-7 actually saw, which is the whole state
// FR-045 is about. `Close` and `Update` replace for the same reason.
func (f *fakeCampaignRepo) applyShareChange(campaignCode string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := f.byCodeLocked(campaignCode)
	if stored == nil {
		return
	}
	copied := *stored
	changed := false
	if f.closeOnShareCode == campaignCode && copied.ClosedAt == nil {
		closedAt := copied.StartsAt.Add(48 * time.Hour)
		copied.ClosedAt = &closedAt
		copied.Version++
		changed = true
	}
	if change := f.budgetOnShare; change != nil && change.code == campaignCode {
		copied.Quota = change.quota
		copied.Reserve = change.reserve
		changed = true
	}
	if !changed {
		return
	}
	f.campaigns[f.indexOfLocked(campaignCode)] = &copied
}

func (f *fakeCampaignRepo) Create(ctx context.Context, campaign *entities.Campaign) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byCodeLocked(campaign.Code) != nil {
		return context.DeadlineExceeded
	}
	f.campaigns = append(f.campaigns, campaign)
	return nil
}

// Update mirrors the repository's conditional UPDATE and reports whether a row
// changed, which is how FR-055's 412 is produced at the use-case level.
func (f *fakeCampaignRepo) Update(ctx context.Context, campaign *entities.Campaign, expectedVersion int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := f.byCodeLocked(campaign.Code)
	if stored == nil || stored.Version != expectedVersion {
		return false, nil
	}
	copied := *campaign
	copied.Version = stored.Version + 1
	f.campaigns[f.indexOfLocked(campaign.Code)] = &copied
	return true, nil
}

func (f *fakeCampaignRepo) Close(ctx context.Context, campaignCode string, expectedVersion int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := f.byCodeLocked(campaignCode)
	if stored == nil || stored.Version != expectedVersion {
		return false, nil
	}
	copied := *stored
	closedAt := stored.StartsAt.Add(48 * time.Hour)
	copied.ClosedAt = &closedAt
	copied.Version = stored.Version + 1
	f.campaigns[f.indexOfLocked(campaignCode)] = &copied
	return true, nil
}

func (f *fakeCampaignRepo) ExistsClaims(ctx context.Context, campaignCode string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// The fake has no ledger of its own, so this is driven by the caller through
	// `seedClaims`; a repository test covers the real EXISTS query.
	return f.grantedCodes[campaignCode], nil
}

// rows returns a snapshot under the lock.
func (f *fakeCampaignRepo) rows() []*entities.Campaign {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*entities.Campaign, len(f.campaigns))
	copy(out, f.campaigns)
	return out
}

// orderedRows applies FR-040's ordering: newest `starts_at` first, `code DESC` as
// the deterministic tie-break.
func (f *fakeCampaignRepo) orderedRows() []*entities.Campaign {
	rows := f.rows()
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].StartsAt.Equal(rows[j].StartsAt) {
			return rows[i].Code > rows[j].Code
		}
		return rows[i].StartsAt.After(rows[j].StartsAt)
	})
	return rows
}

func (f *fakeCampaignRepo) byCode(campaignCode string) *entities.Campaign {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.byCodeLocked(campaignCode)
}

func (f *fakeCampaignRepo) byCodeLocked(campaignCode string) *entities.Campaign {
	for i := range f.campaigns {
		if f.campaigns[i].Code == campaignCode {
			return f.campaigns[i]
		}
	}
	return nil
}

func (f *fakeCampaignRepo) indexOfLocked(campaignCode string) int {
	for i := range f.campaigns {
		if f.campaigns[i].Code == campaignCode {
			return i
		}
	}
	return -1
}

func (f *fakeCampaignRepo) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.campaigns)
}

func (f *fakeCampaignRepo) readCalls() (all, open, get int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listCalls, f.openCalls, f.getCalls
}

var _ repositories.ICampaignRepository = (*fakeCampaignRepo)(nil)

// isCampaignAwake is the fake registry's copy of data-model.md §4.3, kept here
// rather than calling the evaluator so a test can assert on the two independently:
// if the fake borrowed `campaignEvaluator.isAwake`, a bug in that function would
// hide inside the fake and the test would pass. It is what `TestTheFakeRegistry
// AndTheEvaluatorAgreeOnAwake` compares the evaluator against.
func isCampaignAwake(campaign *entities.Campaign, now time.Time) bool {
	if campaign.ClosedAt != nil {
		return false
	}
	if now.Before(campaign.StartsAt) {
		return false
	}
	return campaign.EndsAt == nil || now.Before(*campaign.EndsAt)
}

func testPlan(slug string, durationDays int) *entities.SubscriptionPlan {
	days := durationDays
	return &entities.SubscriptionPlan{
		// A real id, not uuid.Nil: `ValidatePlanChangedPayload` refuses a payload
		// whose `toPlanId` is unset, and it is right to â€” a nil UUID looks like a
		// valid identifier all the way into the receiver's grant table, where it
		// would be written as a real policy reference.
		AuditableEntity: sharedentities.AuditableEntity{
			BaseEntity: sharedentities.BaseEntity{ID: uuid.New()},
		},
		Slug:               slug,
		Name:               "Premium Tháng",
		IsActive:           true,
		DurationDays:       &days,
		PlanKind:           1,
		TierRank:           1,
		AICostPolicyID:     uuid.New(),
		MaxWardrobeItems:   100,
		MaxOutfits:         20,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
	}
}
