package campaign

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedentities "smart-wardrobe-be/shared/domain/entities"

	"github.com/google/uuid"
)

// 025 T100-T102: scaffolding for the catch-up sweep.
//
// The sweep is the only part of the mechanism that WALKS a list, and walking a
// list is where the two properties under test live: the composite-key cursor
// (contracts/identity-contract.md §6, scenario 2) and the interaction between
// two runs on the same ledger. A fake that returned a whole window in one call
// would let both of them pass without ever exercising either.

// fakeAccountLister serves the window query from an in-memory list.
//
// It reproduces the repository's ORDER and its CURSOR rather than just its
// signature: `ORDER BY created_at ASC, id ASC` filtered on the same composite
// predicate. The composite part is the one that matters — `created_at` has
// millisecond resolution, so accounts in one test run routinely share an instant,
// and a fake that paged on the timestamp alone would silently skip the rest of
// that instant and make a batch-boundary bug unreachable.
type fakeAccountLister struct {
	mu       sync.Mutex
	accounts []*identitydto.UserBriefContractDTO
	queries  int

	// entered is signalled once per `ListCreatedInWindow` call, so a test can
	// wait until the expected number of concurrent runs are inside the query
	// before releasing them. Buffered writes are dropped when nobody is
	// listening, which is what lets the same fake serve a single-run test.
	entered chan struct{}
	// release parks every caller until it is closed. It is how T100 makes two
	// sweeps genuinely concurrent rather than incidentally interleaved: a
	// scheduler that happens to serialise them would pass a broken sweep.
	release chan struct{}
	// hold delays every answer, widening the window during which two runs
	// overlap without needing a second handshaking channel.
	hold time.Duration
	// rotatePerCaller hands each successive caller a window that starts halfway
	// round the list, which is how two runs end up evaluating DIFFERENT accounts
	// at the same moment.
	//
	// This is not decoration. Two replicas of this service each walk the window in
	// the same order, so left to themselves they stay in step: both reach the
	// first account, and the loser is refused by `ux_campaign_claims_user` (the
	// property SC-002 covers). The advisory lock is not what saves the watermark
	// from that collision — the UNIQUE constraint is. What the lock protects is the
	// case where the two walks have DRIFTED, and drift is a timing accident a test
	// cannot wait for. Rotating reproduces it on purpose, so a sweep that evaluates
	// the budget outside the lock is caught here rather than in production.
	rotatePerCaller bool
}

func newFakeAccountLister(accounts ...*identitydto.UserBriefContractDTO) *fakeAccountLister {
	return &fakeAccountLister{accounts: accounts}
}

func (f *fakeAccountLister) ListCreatedInWindow(
	ctx context.Context,
	filter identitycontract.CampaignAccountFilter,
) (*identitycontract.CampaignAccountPage, error) {
	if f.entered != nil {
		select {
		case f.entered <- struct{}{}:
		default:
		}
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.hold > 0 {
		select {
		case <-time.After(f.hold):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	f.mu.Lock()
	f.queries++
	// Copied out under the lock: two concurrent runs must not range over the
	// same backing array while the other sorts it.
	window := make([]*identitydto.UserBriefContractDTO, len(f.accounts))
	copy(window, f.accounts)
	rotation := 0
	if f.rotatePerCaller && f.queries > 1 {
		rotation = len(window) / 2
	}
	f.mu.Unlock()

	// `created_at ASC, id ASC` — the id breaks the tie for accounts that share
	// an instant, and the cursor predicate below orders by the same pair, so a
	// batch boundary cannot drift between two pages.
	sort.Slice(window, func(i, j int) bool {
		if window[i].CreatedAt.Equal(window[j].CreatedAt) {
			return window[i].ID.String() < window[j].ID.String()
		}
		return window[i].CreatedAt.Before(window[j].CreatedAt)
	})
	if rotation > 0 && rotation < len(window) {
		rotated := append(append([]*identitydto.UserBriefContractDTO{}, window[rotation:]...), window[:rotation]...)
		window = rotated
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 1
	}
	items := make([]*identitydto.UserBriefContractDTO, 0, limit)
	for _, account := range window {
		if account == nil {
			continue
		}
		// [From, To) — From inclusive, To exclusive, exactly as the repository.
		if account.CreatedAt.Before(filter.From) || !account.CreatedAt.Before(filter.To) {
			continue
		}
		if !isAfterCampaignCursor(account, filter) {
			continue
		}
		items = append(items, account)
		if len(items) >= limit {
			break
		}
	}
	return &identitycontract.CampaignAccountPage{
		Items:      items,
		TotalCount: int64(len(window)),
	}, nil
}

// isAfterCampaignCursor is the composite cursor predicate:
// `created_at > :cursor` OR (`created_at = :cursor` AND `id > :cursorID`).
func isAfterCampaignCursor(account *identitydto.UserBriefContractDTO, filter identitycontract.CampaignAccountFilter) bool {
	if filter.CursorTime.IsZero() {
		return true
	}
	if account.CreatedAt.After(filter.CursorTime) {
		return true
	}
	if account.CreatedAt.Equal(filter.CursorTime) {
		return account.ID.String() > filter.CursorID.String()
	}
	return false
}

func (f *fakeAccountLister) queryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queries
}

// newTestSweep wires the sweep onto the SAME grant use case the message path
// uses.
//
// That sharing is the whole point of FR-080 and it is what makes this harness
// worth having: a test that built a second, sweep-only rule engine would prove
// nothing about the property the sweep is required to have. The sweep holds the
// interface, the message path holds the concrete type, and both reach the same
// `ProcessUserCreated`.
//
// Both clocks are pinned to `baseInstant` because both are read separately and
// the FR-011 window rule compares the account's registration against them. A
// sweep on one clock and a grant on another would produce a disagreement that
// looks like a rule bug and is not one.
//
// The sweep gets the GRANT harness's registry, not a copy of the slice: reading the
// same rows the message path reads is what makes FR-080's "one set of rules"
// meaningful here, and it is what a later edit that adds a sweep-only campaign
// path would break.
func newTestSweep(
	t *testing.T,
	campaigns []*entities.Campaign,
	accounts []*identitydto.UserBriefContractDTO,
) (*testHarness, *fakeAccountLister, uc_interfaces.ICampaignSweepUseCase) {
	t.Helper()
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)
	lister := newFakeAccountLister(accounts...)
	sweep := NewCampaignSweepUseCase(
		h.uc.cfg,
		lister,
		h.campaigns,
		claims,
		newFakeSweepRunRepo(),
		h.uc,
		h.logger,
	).(*CampaignSweepUseCase)
	sweep.now = func() time.Time { return baseInstant() }
	return h, lister, sweep
}

// sweepAccount builds one row of the window the sweep will read.
func sweepAccount(createdAt time.Time, createdByAdmin bool) *identitydto.UserBriefContractDTO {
	return &identitydto.UserBriefContractDTO{
		ID:             uuid.New(),
		CreatedAt:      createdAt,
		CreatedByAdmin: createdByAdmin,
	}
}

// testFreePlan is the plan the signup transaction provisions before any campaign
// grant happens.
//
// TierRank 0 on purpose: condition 6 reads a subscription sitting on the free
// plan WITH a recorded fallback as "a downgrade is in flight", and a fixture
// with tier rank 1 would clear that check by accident.
func testFreePlan() *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		AuditableEntity: sharedentities.AuditableEntity{
			BaseEntity: sharedentities.BaseEntity{ID: uuid.New()},
		},
		Slug:             "free",
		Name:             "Miễn phí",
		IsActive:         true,
		PlanKind:         0,
		TierRank:         0,
		AICostPolicyID:   uuid.New(),
		MaxWardrobeItems: 20,
		MaxOutfits:       5,
	}
}

// flaggedClaim returns the single claim carrying the FR-030 watermark, or nil.
//
// The method lives beside the sweep tests because only the sweep tests need to
// compare the watermark ACROSS runs: the quota tests assert on the flag at the
// moment it is written, which is a different question.
func (f *fakeClaimRepo) flaggedClaim(campaignCode string) *entities.CampaignClaim {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, claim := range f.claims {
		if claim.CampaignCode == campaignCode && claim.FillsMainQuota {
			copied := *claim
			return &copied
		}
	}
	return nil
}

// occurredAtOf reads the `occurredAt` the AI service orders a published plan
// change by.
//
// The payload type is unexported in the mapper package, so it is read as a map
// for the same reason `asMap` reads it as a map: the receiver decides the
// outcome, and what the test needs is the field the receiver sees.
func occurredAtOf(t *testing.T, payload any) time.Time {
	t.Helper()
	body := asMap(t, payload)
	raw, _ := body["occurredAt"].(string)
	if raw == "" {
		t.Fatalf("published plan change carries no occurredAt: %#v", body)
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatalf("occurredAt %q is not RFC3339: %v", raw, err)
	}
	return parsed.UTC()
}
