package campaign

import (
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// 026 T084 — FR-070/FR-072 and spec.md Câu 9: "Chiến dịch đã đóng thôi được quét; hàng sức
// khoẻ và sổ lượt cấp của nó không bị xoá".
//
// WHY THIS FILE IS NOT A RESTATEMENT OF `campaign_sweep_reads_registry_each_run_test.go`.
// That file proves the sweep RE-READS the registry every run, which is FR-070's first
// half. Its second half — "và chỉ xét chiến dịch `closed_at IS NULL`" — had nothing, and
// the silence was structural rather than accidental: `fakeCampaignRepo.ListOpen` filters
// `closed_at` itself, so a fixture holding a closed campaign would watch it vanish from
// the sweep list with nothing asserting WHY. A test that cannot fail cannot be read as
// evidence of a decision, so the decision has to be pinned here.
//
// WHAT "ONLY STOPPED" MEANS, and why all four halves are asserted. A close must change
// exactly ONE thing: which campaigns the next run walks. The health row and the claim
// ledger are the two records an operator answers questions from, and a close that
// reached either of them would be a defect with a long fuse:
//
//	a rewritten `campaign_sweep_runs` row loses "when did a sweep last succeed" for the
//	    campaign an operator just stopped;
//	a deleted claim loses which part of the budget a real customer drew from, and
//	    `grant_bucket` is the one thing about that grant nobody kept anywhere else.
//
// THE ARRANGEMENT. Both runs go over a registry that also holds a campaign nobody closed,
// so "the closed campaign dropped out" is distinguishable from "the sweep did nothing" —
// without the survivor, a run that scanned nothing would satisfy every assertion below.
//
// THE SECOND RUN HAPPENS LATER. `Sweep.now` is moved forward between the runs, because a
// health row written at the same instant as the one it replaced would compare equal and
// the central assertion would be vacuous.

// The three campaigns these tests use, one per role.
const (
	// sweepSurvivorCode stays open throughout and is what proves the second run really
	// ran rather than having found nothing to do.
	sweepSurvivorCode = "sweep-survivor"
	// sweepClosedCode is swept once and then closed between the runs.
	sweepClosedCode = "sweep-closed-midway"
	// sweepNeverCode is closed before it has EVER been swept, which is the edge case
	// FR-072's `lastSweepAt = null` is written for.
	sweepNeverCode = "sweep-closed-before-first-run"
)

// newClosedCampaignSweep wires a sweep and hands back the health store.
//
// Built here rather than through `newTestSweep` for the reason
// `TestASweepOverAnEmptyRegistryWritesNoHealthRow` builds its own: `newTestSweep` does
// not return `fakeSweepRunRepo`, and every assertion below is about what that store
// holds. Changing that helper's signature would touch every sweep test in the package
// for no benefit.
func newClosedCampaignSweep(
	t *testing.T,
	campaigns []*entities.Campaign,
	accounts []*identitydto.UserBriefContractDTO,
) (*testHarness, *fakeSweepRunRepo, *CampaignSweepUseCase) {
	t.Helper()
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)
	runs := newFakeSweepRunRepo()
	sweep := NewCampaignSweepUseCase(
		h.uc.cfg, newFakeAccountLister(accounts...), h.campaigns, claims, runs, h.uc, h.logger,
	).(*CampaignSweepUseCase)
	sweep.now = func() time.Time { return baseInstant() }
	return h, runs, sweep
}

// assertClaimUntouched compares a claim with the copy taken before, field by field.
//
// FIELD BY FIELD rather than with `reflect.DeepEqual`, so a failure names the column
// that moved. The plan id and the expiry are in the list because a close that rewrote
// the row would keep `grant_bucket` (the one field a reader is likely to check) and lose
// the term, and the term is the customer's paid month.
func assertClaimUntouched(t *testing.T, before, after *entities.CampaignClaim) {
	t.Helper()
	if after == nil {
		t.Fatalf("FR-033/FR-091: the claim row must survive the close, got nothing (before %+v)", before)
	}
	if after.UserID != before.UserID || after.CampaignCode != before.CampaignCode ||
		after.PlanSlug != before.PlanSlug || after.PlanID != before.PlanID ||
		after.GrantBucket != before.GrantBucket || after.FillsMainQuota != before.FillsMainQuota ||
		!after.RegisteredAt.Equal(before.RegisteredAt) || !after.ExpiresAt.Equal(before.ExpiresAt) {
		t.Errorf("FR-033/FR-043: the claim row changed across the close:\n before %+v\n after  %+v",
			before, after)
	}
}

// TestASweepStopsScanningACampaignThatWasClosedBetweenRuns is T084 (a)/(b)/(c) and the
// edge case "Chiến dịch bị đóng giữa hai lần quét".
//
// THE ARRANGEMENT IS FR-040's own ordering, used as the lever: the campaign that is
// closed later starts later, so it is the one that takes the first sweep's grant. The
// survivor is the campaign that starts earlier and therefore grants nothing while the
// other one is live — which is what leaves it with an untouched budget to grant from once
// the other is out of the way.
// closedCampaignState is what the first run left behind, captured so the assertions after
// the close compare against it rather than against a re-read of the same rows.
//
// `claimBefore` and `watermarkBefore` are VALUES rather than pointers for the same reason
// the fake replaces a row instead of editing it: a pointer into the ledger would be a
// reference to whatever is there now, and "nothing changed" would be unprovable.
type closedCampaignState struct {
	claim     *entities.CampaignClaim
	watermark *entities.CampaignClaim
	health    *entities.CampaignSweepRun
	grants    int
}

// sweepOnceThenClose runs the first pass, asserts its premises and closes the campaign
// that pass was granting from.
//
// The premises live here because each one is a precondition of a specific assertion in
// `assertClosedCampaignLeftAlone`, and stating them at the point where they are produced
// is what keeps a later failure about the CLOSE from actually being about a fixture that
// never worked.
func sweepOnceThenClose(
	t *testing.T,
	h *testHarness,
	runs *fakeSweepRunRepo,
	sweep *CampaignSweepUseCase,
	early, lateArrival []*identitydto.UserBriefContractDTO,
) closedCampaignState {
	t.Helper()
	if _, err := sweep.Sweep(t.Context()); err != nil {
		t.Fatalf("the first sweep: %v", err)
	}

	firstClaim := h.claims.claimFor(early[0].ID)
	if firstClaim == nil || firstClaim.CampaignCode != sweepClosedCode {
		t.Fatalf("precondition: the first sweep must grant %s from %s (the later `starts_at` wins FR-040), "+
			"got %+v", early[0].ID, sweepClosedCode, firstClaim)
	}
	grants := h.claims.countBucket(sweepClosedCode, entities.GrantBucketMain)
	if grants != len(early) {
		t.Fatalf("precondition: every account must be granted from %s on the first run, got %d grants",
			sweepClosedCode, grants)
	}
	// The budget is exactly full, so there IS a priority watermark to protect — the one
	// row whose `registered_at` every later reserve decision is compared against.
	watermark := h.claims.flaggedClaim(sweepClosedCode)
	if watermark == nil {
		t.Fatalf("precondition: %d grants against a quota of 3 must have filled the main budget and written "+
			"the FR-030 watermark", grants)
	}
	health := runs.row(sweepClosedCode)
	if health == nil || health.LastSweepAt == nil {
		t.Fatalf("precondition: the first sweep must have recorded health for %s", sweepClosedCode)
	}
	if survivor := h.claims.countBucket(sweepSurvivorCode, entities.GrantBucketMain); survivor != 0 {
		t.Fatalf("precondition: %s must have granted nobody yet, got %d grants", sweepSurvivorCode, survivor)
	}
	if late := h.claims.claimFor(lateArrival[0].ID); late != nil {
		t.Fatalf("precondition: the late arrival registers after the first run and must hold no claim, got %+v", late)
	}

	current := h.campaigns.byCode(sweepClosedCode)
	if current == nil {
		t.Fatalf("read %s before closing it", sweepClosedCode)
	}
	if closed, err := h.campaigns.Close(t.Context(), sweepClosedCode, current.Version); err != nil || !closed {
		t.Fatalf("close %s: closed=%v err=%v", sweepClosedCode, closed, err)
	}
	copied := *firstClaim
	return closedCampaignState{claim: &copied, watermark: watermark, health: health, grants: grants}
}

// assertClosedCampaignLeftAlone is T084's (a)/(b)/(c) after the second run.
//
// `secondInstant` is the clock that run read, and it is passed in rather than recomputed
// because it is the value a rewritten health row would carry: the assertion is "the row
// still says the FIRST run's instant", and the reader needs both instants to check it.
func assertClosedCampaignLeftAlone(
	t *testing.T,
	h *testHarness,
	runs *fakeSweepRunRepo,
	before closedCampaignState,
	early, lateArrival []*identitydto.UserBriefContractDTO,
	secondInstant time.Time,
) {
	t.Helper()

	// (a) THE RUN IS ALIVE AND SCANNED. Without this, "the closed campaign was not
	// scanned" would also be true of a run that scanned nothing at all.
	survivorClaim := h.claims.claimFor(lateArrival[0].ID)
	if survivorClaim == nil || survivorClaim.CampaignCode != sweepSurvivorCode {
		t.Errorf("(a) the second run must have granted the late arrival from the campaign nobody closed, got "+
			"%+v — without it, every assertion below would also hold for a run that did nothing", survivorClaim)
	}
	// AND THE CLOSED CAMPAIGN IS NOT A CANDIDATE, read through the very predicate FR-070
	// names.
	open, err := h.campaigns.ListOpen(t.Context())
	if err != nil {
		t.Fatalf("ListOpen after the close: %v", err)
	}
	for _, row := range open {
		if row != nil && row.Code == sweepClosedCode {
			t.Errorf("FR-070: %s was closed and must not be a candidate for any run", sweepClosedCode)
		}
	}

	// (b) THE HEALTH ROW SURVIVES, UNTOUCHED. `LastSweepAt` is the whole assertion: it
	// must still be the FIRST run's instant, not the second run's.
	healthAfter := runs.row(sweepClosedCode)
	if healthAfter == nil {
		t.Fatalf("FR-070: %s was closed, but its health row must not be deleted — an operator asking when "+
			"the campaign last swept deserves an answer", sweepClosedCode)
	}
	if healthAfter.LastSweepAt == nil || !healthAfter.LastSweepAt.Equal(*before.health.LastSweepAt) {
		t.Errorf("FR-070: the closed campaign's health row was rewritten: last_sweep_at was %s, now %v "+
			"(the second run read %s). A run that stopped walking a campaign must stop RECORDING it too",
			before.health.LastSweepAt.UTC(), healthAfter.LastSweepAt, secondInstant.UTC())
	}
	if healthAfter.Degraded {
		t.Error("FR-059: a closed campaign is not a degraded one; the flag describes the last run, and " +
			"the last run that covered this campaign succeeded")
	}
	// AND THE RUN DID WRITE, for the survivor — so the closed campaign's silence is a
	// decision about that campaign and not a broken health write.
	survivorHealth := runs.row(sweepSurvivorCode)
	if survivorHealth == nil || survivorHealth.LastSweepAt == nil ||
		!survivorHealth.LastSweepAt.Equal(secondInstant) {
		t.Errorf("the second run must record health for the campaign it swept, got %+v", survivorHealth)
	}
	if got := runs.writeCount(); got != 2 {
		t.Errorf("expected exactly 2 health writes (one per run), got %d", got)
	}

	// (c) THE CLAIM LEDGER IS INTACT.
	assertClaimUntouched(t, before.claim, h.claims.claimFor(early[0].ID))
	if got := h.claims.countBucket(sweepClosedCode, entities.GrantBucketMain); got != before.grants {
		t.Errorf("FR-091: the closed campaign's ledger must not change, %d grants before and %d after",
			before.grants, got)
	}
	// AND THE WATERMARK WITH IT, which is the row that matters most: every later reserve
	// decision compares against this `registered_at` (FR-030), so a close that disturbed
	// it would hand the reserve to the wrong accounts weeks later with nothing in the
	// logs to explain it.
	assertClaimUntouched(t, before.watermark, h.claims.flaggedClaim(sweepClosedCode))
	if got := h.claims.watermarkCount(sweepClosedCode); got != 1 {
		t.Errorf("FR-030: the closed campaign must still hold exactly 1 watermark row, got %d", got)
	}
}

func TestASweepStopsScanningACampaignThatWasClosedBetweenRuns(t *testing.T) {
	registeredBase := baseInstant().Add(-8 * time.Hour)
	survivorStart := registeredBase.Add(-2 * time.Hour)
	closedStart := registeredBase.Add(-time.Hour)

	campaigns := []*entities.Campaign{
		testCampaign(sweepSurvivorCode, "premium-monthly", 3, 0, &survivorStart, nil),
		testCampaign(sweepClosedCode, "premium-monthly", 3, 0, &closedStart, nil),
	}
	// THREE ACCOUNTS INSIDE THE FIRST RUN'S WINDOW, plus one that registers DURING THE
	// GAP between the runs. The late arrival is not decoration: the three early accounts
	// all hold a claim after run one and are refused with `already_claimed` in run two
	// (FR-012), so without a fresh account the second run would have nothing to grant and
	// "(a) the run is alive" would be unfalsifiable. It also happens to be FR-071's case —
	// an account created between two sweeps is not missed.
	early := []*identitydto.UserBriefContractDTO{
		sweepAccount(registeredBase, false),
		sweepAccount(registeredBase.Add(time.Minute), false),
		sweepAccount(registeredBase.Add(2*time.Minute), false),
	}
	lateArrival := []*identitydto.UserBriefContractDTO{sweepAccount(baseInstant().Add(5*time.Minute), false)}
	h, runs, sweep := newClosedCampaignSweep(t, campaigns, append(early, lateArrival...))

	// RUN ONE, THEN THE CLOSE.
	before := sweepOnceThenClose(t, h, runs, sweep, early, lateArrival)

	// RUN TWO, A QUARTER OF AN HOUR LATER, so a rewritten health row is visible: both runs
	// would otherwise record the SAME instant and the central assertion would hold either
	// way.
	secondInstant := baseInstant().Add(15 * time.Minute)
	sweep.now = func() time.Time { return secondInstant }
	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("the second sweep: %v", err)
	}
	if report.Scanned != len(early)+len(lateArrival) {
		t.Errorf("(a) the second run must still walk the whole window: scanned %d, want %d", report.Scanned,
			len(early)+len(lateArrival))
	}

	assertClosedCampaignLeftAlone(t, h, runs, before, early, lateArrival, secondInstant)
}

// TestACampaignClosedBeforeItsFirstSweepIsNeverRecorded is T084 (d) / FR-072 / SC-007, and
// it is the half that could be untested by accident.
//
// FR-072 says a campaign nobody has swept reports `lastSweepAt = null`. The read side of
// that promise is covered (`campaign_read_contract_shape_test.go`), and covering it again
// would prove nothing new — the question is what the SWEEP does. A run that wrote a
// speculative row for a campaign it did not walk would give `degraded` and `lastSweepAt`
// something to say about a campaign that was never scanned, and an operator would read it
// as health.
//
// So this asserts BOTH ends: the sweep wrote no row, and the operator's page therefore
// answers `null` and `false` for that campaign. The second half is not redundant — it is
// what makes the absence mean "never swept" rather than "a read that lost a row".
func TestACampaignClosedBeforeItsFirstSweepIsNeverRecorded(t *testing.T) {
	startsAt := baseInstant().Add(-2 * time.Hour)
	survivor := testCampaign(sweepSurvivorCode, "premium-monthly", 3, 0, &startsAt, nil)
	// Closed BEFORE any run, and with its window still open — so if the answer were "asleep
	// because the window closed", this test would pass for the wrong reason.
	never := closedCampaign(sweepNeverCode, "premium-monthly", 3, 0, &startsAt, nil)
	campaigns := []*entities.Campaign{survivor, never}
	accounts := []*identitydto.UserBriefContractDTO{sweepAccount(baseInstant().Add(-time.Hour), false)}

	h, runs, sweep := newClosedCampaignSweep(t, campaigns, accounts)

	if !isCampaignAwake(never, baseInstant()) && never.ClosedAt == nil {
		t.Fatal("precondition: the campaign must be closed")
	}
	if never.ClosedAt == nil || never.EndsAt != nil {
		t.Fatalf("precondition: the fixture must be a CLOSED campaign whose window is still open, got %+v", never)
	}

	// TWO RUNS, because "never swept" has to survive more than the first opportunity: a
	// row written on the second run would still be a row nobody asked for.
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := sweep.Sweep(t.Context()); err != nil {
			t.Fatalf("sweep %d: %v", attempt, err)
		}
	}

	// SWEEP SIDE. No row, ever — for this campaign.
	if row := runs.row(sweepNeverCode); row != nil {
		t.Fatalf("FR-070/FR-072: no run ever walked %s, so it must have NO health row; got %+v. A row here "+
			"would give `degraded` and `lastSweepAt` something to report about a campaign nothing swept",
			sweepNeverCode, row)
	}
	// AND THE RUNS WERE REAL, so the absence above is a decision about this campaign.
	if survivorHealth := runs.row(sweepSurvivorCode); survivorHealth == nil || survivorHealth.LastSweepAt == nil {
		t.Fatalf("precondition: both runs must have recorded health for %s, got %+v", sweepSurvivorCode, survivorHealth)
	}
	if got := runs.writeCount(); got != 2 {
		t.Errorf("expected exactly 2 health writes across the two runs, got %d", got)
	}
	if got := h.claims.countBucket(sweepSurvivorCode, entities.GrantBucketMain); got == 0 {
		t.Error("precondition: the survivor must have granted somebody, or these runs proved nothing about " +
			"which campaigns they walked")
	}

	// READ SIDE — the operator's page, over the SAME health store, which is the only
	// surface `lastSweepAt = null` is observable through.
	reader := newTestQueryUC(t, campaigns, newFakeClaimRepo(), runs, newFakeUserBriefContract())
	items, _, err := reader.uc.ListCampaigns(t.Context(), 1, 20)
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	var neverSweep *sub_dto.CampaignSummaryRes
	for _, item := range items {
		if item.Code == sweepNeverCode {
			neverSweep = item
			if item.DegradedReason != "" {
				t.Errorf("FR-059: a campaign no run ever swept must carry no degradedReason, got %q",
					item.DegradedReason)
			}
		}
	}
	if neverSweep == nil {
		t.Fatalf("FR-032: %s must still appear in the operator list even though it was closed", sweepNeverCode)
	}
	if neverSweep.LastSweepAt != nil {
		t.Errorf("FR-072: lastSweepAt must be null for a campaign no run ever swept, got %s", neverSweep.LastSweepAt)
	}
	if neverSweep.Degraded {
		t.Error("FR-059: degraded must be false for a campaign nobody has swept — that is not a fault")
	}
}
