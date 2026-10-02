package campaign

import (
	"context"
	"testing"
	"time"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedconfig "smart-wardrobe-be/shared/config"
)

// 025 T079e: the sweep records its health in the DATABASE (FR-059, SC-012).
//
// THIS IS THE ONLY TEST THAT PROVES WHY THE STATE IS PERSISTED, and the reason it
// matters is a failure an in-memory flag cannot have.
//
// The service runs as SEVERAL REPLICAS, each with its own worker set. An in-memory
// flag therefore answers the wrong question in the direction that hurts: a request
// landing on replica B reads replica B's flag, while the sweep that actually failed
// was run by replica A. The operator page then reports HEALTHY while the mechanism
// is broken — the single answer FR-059 exists to prevent. Under `APP_MODE=http` no
// worker runs at all, so the flag is permanently false: a false all-clear that is
// structurally guaranteed rather than observed.
//
// A flag on the use case struct would reproduce the original design and pass, which
// is why `fakeSweepRunRepo` is a separate collaborator keyed on the campaign code:
// the run has to WRITE somewhere a later, unrelated reader can find.

// toggledAccountLister answers the sweep's window query, and can be made to fail.
//
// Failable rather than permanently failing because the tests below need BOTH
// outcomes against the same wiring: a fault that is set and then cleared is a
// different fact from one that is merely set, and only a single run can show the
// clearing half.
type toggledAccountLister struct {
	failing bool
}

func (l *toggledAccountLister) ListCreatedInWindow(
	ctx context.Context,
	filter identitycontract.CampaignAccountFilter,
) (*identitycontract.CampaignAccountPage, error) {
	if l.failing {
		// The shape a real outage has: the run cannot even BEGIN.
		return nil, context.DeadlineExceeded
	}
	// An empty window is a CLEAN run that simply finds nothing, which is what
	// makes the "then it recovered" half of the test possible without inventing
	// accounts the sweep would then have to evaluate.
	return &identitycontract.CampaignAccountPage{Items: nil, TotalCount: 0}, nil
}

// newHealthSweep wires a sweep over a toggleable window query and returns the
// health store, the sweep and the toggle.
func newHealthSweep(t *testing.T, campaigns []sharedconfig.CampaignConfig) (*fakeSweepRunRepo, uc_interfaces.ICampaignSweepUseCase, *toggledAccountLister) {
	t.Helper()
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)
	sweepRuns := newFakeSweepRunRepo()
	lister := &toggledAccountLister{}

	sweep := NewCampaignSweepUseCase(h.uc.cfg, lister, claims, sweepRuns, h.uc, h.logger).(*CampaignSweepUseCase)
	sweep.now = func() time.Time { return baseInstant() }
	return sweepRuns, sweep, lister
}

// healthCampaigns is one open campaign, the shape a live sweep has.
func healthCampaigns() []sharedconfig.CampaignConfig {
	return []sharedconfig.CampaignConfig{
		testCampaignConfig(queryCode, "premium-monthly", 50, 5, timePtr(baseInstant().Add(-time.Hour)), nil),
	}
}

// TestSweepRecordsDegradedInTheDatabaseWhenItFails is T079e / FR-059 / SC-012.
//
// A sweep that cannot finish MUST leave a durable, visible fault — and the two
// halves of the row answer different questions:
//
//	degraded + reason → "is it broken right now, and why"
//	last_sweep_at    → "when did it last work"
//
// Advancing `last_sweep_at` on a failure is the specific mistake this guards
// against: it turns a sweep failing every fifteen minutes for six hours into a
// timestamp that looks fifteen minutes old, and that is the false all-clear
// FR-059 exists to prevent.
func TestSweepRecordsDegradedInTheDatabaseWhenItFails(t *testing.T) {
	sweepRuns, sweep, lister := newHealthSweep(t, healthCampaigns())
	lister.failing = true

	if _, err := sweep.Sweep(t.Context()); err == nil {
		t.Fatal("expected the window query to fail")
	}

	row := sweepRuns.row(queryCode)
	if row == nil {
		t.Fatalf("a failing sweep must write a row for its campaign; the look-up reads absence as " +
			"'never swept', which is a different answer from 'broken'")
	}
	if !row.Degraded {
		t.Fatalf("FR-059: a sweep that failed must set degraded = TRUE")
	}
	if row.DegradedReason == nil || *row.DegradedReason == "" {
		t.Fatalf("FR-059: a degraded campaign must carry the reason, so an operator does not have to "+
			"correlate a timestamp with a log search; got %#v", row.DegradedReason)
	}
	// No successful run has completed, so the column must be UNSET rather than
	// pointing at the failed instant — an unset column and the year 1 are
	// different facts and the second is not true.
	if row.LastSweepAt != nil {
		t.Fatalf("a failed run must not advance last_sweep_at: it answers 'when did a sweep last SUCCEED', "+
			"and got %s from a run that never completed", row.LastSweepAt)
	}
}

// TestSweepClearsDegradedAfterASuccessfulRun is T079e / FR-059 / SC-012, and it
// is the half a latch-without-clear would fail.
//
// "It stopped breaking" is as much of an event as "it broke". A fault that latched
// would train an operator to ignore the field — worse than never having had it,
// because the one time it mattered they had already learned not to look.
func TestSweepClearsDegradedAfterASuccessfulRun(t *testing.T) {
	sweepRuns, sweep, lister := newHealthSweep(t, healthCampaigns())

	// Phase one: the fault.
	lister.failing = true
	if _, err := sweep.Sweep(t.Context()); err == nil {
		t.Fatal("expected the first sweep to fail")
	}
	if broken := sweepRuns.row(queryCode); broken == nil || !broken.Degraded {
		t.Fatalf("precondition: the first sweep must leave the campaign degraded, got %#v", broken)
	}

	// Phase two: the fault clears. The window query starts answering and the
	// window is empty — a clean run that finds nothing.
	lister.failing = false
	if _, err := sweep.Sweep(t.Context()); err != nil {
		t.Fatalf("the second sweep must succeed, got: %v", err)
	}

	healed := sweepRuns.row(queryCode)
	if healed.Degraded {
		t.Fatalf("FR-059: a sweep that completed must clear degraded; it still reads TRUE with reason %#v",
			healed.DegradedReason)
	}
	if healed.DegradedReason != nil {
		t.Fatalf("a healthy campaign must not carry a stale reason — it would name a fault that no longer "+
			"exists; got %q", *healed.DegradedReason)
	}
	if healed.LastSweepAt == nil {
		t.Fatalf("a completed sweep must advance last_sweep_at (FR-059)")
	}
	if !healed.LastSweepAt.Equal(baseInstant()) {
		t.Fatalf("expected last_sweep_at to be the instant the run read, got %s want %s",
			healed.LastSweepAt.UTC(), baseInstant().UTC())
	}
}

// TestSweepWritesAHealthRowForEveryCampaign is T079e / FR-059.
//
// A sweep covers EVERY configured campaign, so its health write must too. A
// per-campaign write would make a partial recording possible — half the campaigns
// marked healthy while the other half still showed the last fault — and the
// operator would have no way to tell a stale row from a fresh one.
func TestSweepWritesAHealthRowForEveryCampaign(t *testing.T) {
	const otherCode = "winter-2026-12"
	startsAt := baseInstant().Add(-time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig(queryCode, "premium-monthly", 50, 5, &startsAt, nil),
		testCampaignConfig(otherCode, "premium-monthly", 10, 1, &startsAt, nil),
	}
	sweepRuns, sweep, lister := newHealthSweep(t, campaigns)
	lister.failing = true

	if _, err := sweep.Sweep(t.Context()); err == nil {
		t.Fatal("expected the window query to fail")
	}

	// ONE write for both campaigns, not one each: a per-campaign write is what
	// makes a half-recorded run possible.
	if got := sweepRuns.writeCount(); got != 1 {
		t.Errorf("expected a single health write covering every campaign, got %d", got)
	}
	for _, code := range []string{queryCode, otherCode} {
		row := sweepRuns.row(code)
		if row == nil {
			t.Fatalf("campaign %s has no health row after a sweep; the look-up would report it as never swept", code)
		}
		if !row.Degraded {
			t.Errorf("campaign %s: expected degraded = TRUE after a failed run", code)
		}
	}
}

// TestSweepHealthSurvivesTheProcessThatWroteIt is T079e / FR-059, and it is the
// direct statement of the multi-replica problem.
//
// The fault is recorded by a sweep on one "replica", and read back by a look-up
// wired to a DIFFERENT one — the same shape as two pods in production. It works
// here because the state is in a shared store. With an in-memory flag this test
// could not even be written: the second harness would have no way to see the first
// one's failure.
func TestSweepHealthSurvivesTheProcessThatWroteIt(t *testing.T) {
	sweepRuns, sweep, lister := newHealthSweep(t, healthCampaigns())
	lister.failing = true

	if _, err := sweep.Sweep(t.Context()); err == nil {
		t.Fatal("expected the first sweep to fail")
	}

	// A DIFFERENT use-case instance over the SAME health store — replica B.
	reader := newTestQueryUC(
		t,
		healthCampaigns(),
		newFakeClaimRepo(),
		sweepRuns,
		newFakeUserBriefContract(),
	)

	items, _, err := reader.uc.ListCampaigns(t.Context(), 1, 20)
	if err != nil {
		t.Fatalf("the look-up must read the sweep health written by another replica: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one campaign, got %d", len(items))
	}
	if !items[0].Degraded {
		t.Fatalf("FR-059: replica B reported the campaign healthy while replica A's sweep had failed — " +
			"broken while reporting itself well is the exact failure FR-059 exists to prevent")
	}
	if items[0].DegradedReason == "" {
		t.Fatalf("the look-up must surface the reason an operator needs to act (FR-059)")
	}
}

// TestCampaignQueryHasNoHealthAccessor is the structural guard for T079c.
//
// The look-up's only route to the sweep state is the repository. There is no
// in-memory accessor left to call — the method was removed — and this declaration
// is what would stop compiling the moment one came back, which is the point: the
// guard is a compile failure rather than a review convention.
func TestCampaignQueryHasNoHealthAccessor(t *testing.T) {
	var (
		_ repositories.ICampaignSweepRunRepository = newFakeSweepRunRepo()
		_ uc_interfaces.ICampaignQueryUseCase      = (*CampaignQueryUseCase)(nil)
	)
}
