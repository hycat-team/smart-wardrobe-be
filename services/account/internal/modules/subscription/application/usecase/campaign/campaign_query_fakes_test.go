package campaign

import (
	"context"
	"sync"
	"testing"
	"time"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedconfig "smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// 025 T079: scaffolding for the READ side of the campaign mechanism.
//
// The grant path has fakes of its own (`campaign_fakes_test.go`). This file adds
// the two the operator look-up needs and nothing else:
//
//   - `fakeSweepRunRepo`, which stands in for the table FR-059 moved the health
//     flag into. It exists as a fake rather than an in-memory field on the use
//     case because THAT is the point of T079c: the state has to survive the
//     process, so a fake that lived on the use case would let the original
//     defect pass.
//   - `fakeUserBriefContract`, so the look-up reads account data through the
//     contract (constitution §II) rather than from `identity.users`.

const (
	// queryQuota and queryReserve keep the budget small enough that a test can
	// seed an exact number of grants by hand. A campaign of fifty would need
	// fifty seeds to reach the same state, and a test that seeds fifty rows to
	// assert a state machine is a test that stops being read.
	queryQuota   = 2
	queryReserve = 1
	// queryCode is the campaign code used by every test in this file.
	queryCode = "launch-2026-10"
)

// fakeSweepRunRepo is an in-memory stand-in for
// `subscription.campaign_sweep_runs`.
type fakeSweepRunRepo struct {
	mu      sync.Mutex
	runs    map[string]*fakeSweepRun
	writes  int
	failAll bool
}

// fakeSweepRun mirrors one row of the table.
type fakeSweepRun struct {
	CampaignCode   string
	LastSweepAt    *time.Time
	Degraded       bool
	DegradedReason *string
}

func newFakeSweepRunRepo() *fakeSweepRunRepo {
	return &fakeSweepRunRepo{runs: map[string]*fakeSweepRun{}}
}

// RecordRun reproduces the upsert, including the one asymmetry the table
// depends on: a FAILED sweep sets the fault and records the reason but does NOT
// move `last_sweep_at`. That column answers "when did a sweep last complete", and
// a run that failed is not evidence that one did — which is what turns a sweep
// failing every fifteen minutes for six hours into a six-hour-old timestamp, the
// alarm an operator actually wants (data-model.md §5.1).
func (f *fakeSweepRunRepo) RecordRun(ctx context.Context, results []repositories.CampaignSweepResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if f.failAll {
		return context.DeadlineExceeded
	}
	for _, result := range results {
		row, ok := f.runs[result.CampaignCode]
		if !ok {
			row = &fakeSweepRun{CampaignCode: result.CampaignCode}
			f.runs[result.CampaignCode] = row
		}
		row.Degraded = !result.Succeeded
		if result.Succeeded {
			row.DegradedReason = nil
			sweptAt := result.SweptAt
			row.LastSweepAt = &sweptAt
			continue
		}
		reason := result.FailureReason
		row.DegradedReason = &reason
	}
	return nil
}

func (f *fakeSweepRunRepo) GetByCampaigns(ctx context.Context, campaignCodes []string) (map[string]*entities.CampaignSweepRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]*entities.CampaignSweepRun, len(campaignCodes))
	for _, code := range campaignCodes {
		row, ok := f.runs[code]
		if !ok {
			continue
		}
		out[code] = &entities.CampaignSweepRun{
			CampaignCode:   row.CampaignCode,
			LastSweepAt:    row.LastSweepAt,
			Degraded:       row.Degraded,
			DegradedReason: row.DegradedReason,
		}
	}
	return out, nil
}

// row returns a COPY of one campaign's row as the production entity, or nil when
// the sweep has never written it.
//
// The conversion is deliberate: a test should assert on the same type the look-up
// reads, so a change to the entity's shape breaks the assertions rather than
// quietly passing against a look-alike. A copy rather than the pointer so a test
// cannot hold a reference the next sweep mutates.
func (f *fakeSweepRunRepo) row(campaignCode string) *entities.CampaignSweepRun {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.runs[campaignCode]
	if !ok {
		return nil
	}
	return &entities.CampaignSweepRun{
		CampaignCode:   row.CampaignCode,
		LastSweepAt:    row.LastSweepAt,
		Degraded:       row.Degraded,
		DegradedReason: row.DegradedReason,
	}
}

func (f *fakeSweepRunRepo) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

// ---- account contract fake ----

// fakeUserBriefContract serves accounts from an in-memory list.
//
// Only `GetByID` and `GetByIDs` are meaningful to the look-up; the other two
// exist because the interface is the module's own contract and a fake that
// omitted them would not satisfy it.
type fakeUserBriefContract struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]*identitydto.UserBriefContractDTO
}

func newFakeUserBriefContract(accounts ...*identitydto.UserBriefContractDTO) *fakeUserBriefContract {
	f := &fakeUserBriefContract{accounts: map[uuid.UUID]*identitydto.UserBriefContractDTO{}}
	for _, account := range accounts {
		f.accounts[account.ID] = account
	}
	return f
}

func (f *fakeUserBriefContract) GetByID(ctx context.Context, userID uuid.UUID) (*identitydto.UserBriefContractDTO, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.accounts[userID], nil
}

func (f *fakeUserBriefContract) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*identitydto.UserBriefContractDTO, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*identitydto.UserBriefContractDTO, 0, len(userIDs))
	for _, id := range userIDs {
		if account, ok := f.accounts[id]; ok {
			out = append(out, account)
		}
	}
	return out, nil
}

func (f *fakeUserBriefContract) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*identitydto.UserBriefContractDTO, error) {
	return nil, nil
}

func (f *fakeUserBriefContract) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*identitydto.UserBriefContractDTO, error) {
	return nil, nil
}

var _ identitycontract.IUserBriefContract = (*fakeUserBriefContract)(nil)

// ---- harness ----

// queryHarness is the look-up use case plus the fakes a test asserts on.
type queryHarness struct {
	uc        *CampaignQueryUseCase
	claims    *fakeClaimRepo
	sweepRuns *fakeSweepRunRepo
	accounts  *fakeUserBriefContract
	subs      *fakeSubRepo
	plans     *fakePlanRepo
}

// newTestQueryUC wires the look-up onto the SAME fakes the grant path uses, so a
// ledger seeded for a grant-path test is readable by the look-up and vice versa.
//
// The clock is pinned to `baseInstant` for the same reason it is in
// `newTestUseCase`: a state machine whose transitions are all clock-driven cannot
// be tested against a clock that moves.
func newTestQueryUC(
	t *testing.T,
	campaigns []sharedconfig.CampaignConfig,
	claims *fakeClaimRepo,
	sweepRuns *fakeSweepRunRepo,
	accounts *fakeUserBriefContract,
) *queryHarness {
	t.Helper()
	cfg := campaignConfig(campaigns)
	subs := newFakeSubRepo()
	plans := newFakePlanRepo()
	plans.put(testPlan("premium-monthly", 30))

	uc := NewCampaignQueryUseCase(cfg, claims, sweepRuns, plans, subs, accounts, quietLogger{}).(*CampaignQueryUseCase)
	uc.now = func() time.Time { return baseInstant() }
	return &queryHarness{
		uc:        uc,
		claims:    claims,
		sweepRuns: sweepRuns,
		accounts:  accounts,
		subs:      subs,
		plans:     plans,
	}
}

// queryAccount builds an account the look-up will read through the contract.
func queryAccount(createdAt time.Time, createdByAdmin bool) *identitydto.UserBriefContractDTO {
	return &identitydto.UserBriefContractDTO{
		ID:             uuid.New(),
		Username:       "user-" + uuid.NewString()[:8],
		CreatedAt:      createdAt,
		CreatedByAdmin: createdByAdmin,
	}
}
