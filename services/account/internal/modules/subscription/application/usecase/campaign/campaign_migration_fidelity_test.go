package campaign

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
)

// 026 T060 (FR-050, FR-090, FR-091, FR-092) — quickstart V4's fidelity comparison:
// moving `launch-2026-10` from configuration into the registry must not change ONE
// number the four read endpoints report.
//
// THE "BEFORE" SIDE NO LONGER EXISTS IN CODE, and that is the problem this file
// solves. 026 removed the `campaigns` block from both YAML files (FR-002), so there is
// no longer a configuration-driven code path to compare the registry-driven one
// against. A test that ran both would therefore be testing nothing.
//
// So the comparison is made against a FIXTURE: the values that block carried, written
// out here as literals, and — the part that keeps them honest — checked against the
// migration file that ships them. A fixture nobody re-verifies is a fixture that rots
// quietly, and this is precisely the migration where a silent divergence moves a live
// promotion's budget by hours or by slots.
//
// WHY A UNIT TEST AND NOT A DATABASE ONE. The four read endpoints are driven through
// their real use case with in-memory collaborators, which is the only way to compare
// their OUTPUT: a test that reached the repositories would live in the wrong layer
// (constitution §IV forbids `application` importing `infrastructure`, and the read use
// case is application code). The database half of V4 — that the seed really inserted
// those six values, and that the rollback really keeps the ledger — is
// `campaign_seed_integration_test.go` (T059), where a database is available.
//
// WHAT IS COMPARED, and why these three: the granted counts, `watermarkAt` and
// `status`. They are the only fields derived from state the migration does not carry —
// every other number on the response comes straight out of the seeded row, so it cannot
// differ. These three come from the CLAIM LEDGER, which the migration deliberately does
// not touch (FR-091), and they are what an operator uses to decide whether a promotion
// still has room in it.

// launchCampaignCode is the campaign the migration seeds (FR-090).
const launchCampaignCode = "launch-2026-10"

// formerConfigurationCampaign is what `configs/config.production.yaml` carried for
// `launch-2026-10` before 026 deleted the `campaigns` block (T013).
//
// IT IS A FIXTURE, AND IT IS VERIFIED AGAINST THE MIGRATION in the first test below.
// The values are the ones quickstart §1.2 states as the expected seed row, and they are
// the ones production is running: changing any of them here without changing the
// migration is what would make this feature lie.
func formerConfigurationCampaign() *entities.Campaign {
	startsAt := time.Date(2026, 10, 2, 0, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	return &entities.Campaign{
		Code:     launchCampaignCode,
		PlanSlug: "premium-monthly",
		Quota:    50,
		Reserve:  5,
		StartsAt: startsAt,
		EndsAt:   nil,
		Version:  1,
	}
}

// seedMigrationPath locates the seed migration from this package.
//
// SIX LEVELS UP, because the package sits at
// `services/account/internal/modules/subscription/application/usecase/campaign` and the
// migrations live at `services/account/database/migrations`. The glob is used rather
// than the exact file name because a rename is a legitimate thing for a human to do, and
// the assertion that matters is that a seed migration for this campaign EXISTS.
const seedMigrationPath = "../../../../../../database/migrations/*_seed_launch_campaign.sql"

// TestTheSeedMigrationStatesTheValuesTheConfigurationUsedToHold is the "before and
// after" identity, at the only place it can be checked without a database.
//
// The seed's `INSERT` is READ OUT OF THE FILE that ships, and its six values are
// compared with the fixture above. If somebody retyped a quota, moved a start date or
// pointed the seed at a different plan, this fails — which is the defect that would
// otherwise reach production as "the promotion now grants a different number of plans",
// discovered by customers rather than by a test.
func TestTheSeedMigrationStatesTheValuesTheConfigurationUsedToHold(t *testing.T) {
	insert := seedMigrationInsert(t)

	want := formerConfigurationCampaign()
	seeded := seedValues(t, insert)

	if seeded[0] != launchCampaignCode {
		t.Errorf("FR-090: the seed must insert %s, got %q", launchCampaignCode, seeded[0])
	}
	if seeded[1] != want.PlanSlug {
		t.Errorf("FR-090: the seed must insert plan_slug %q, got %q", want.PlanSlug, seeded[1])
	}

	// The two integers are parsed as NUMBERS rather than as text, so `"50"` and `"050"`
	// cannot both pass and a stray non-digit is reported instead of silently comparing
	// equal as a string.
	for i, spec := range []struct {
		column string
		want   int
	}{
		{"quota", want.Quota},
		{"reserve", want.Reserve},
	} {
		got, err := strconv.Atoi(seeded[2+i])
		if err != nil {
			t.Fatalf("the seeded %s %q is not a number: %v", spec.column, seeded[2+i], err)
		}
		if got != spec.want {
			t.Errorf("FR-090: the seed states %s = %d, but the configuration this campaign replaced "+
				"carried %d. The registry must not change the meaning of a running promotion",
				spec.column, got, spec.want)
		}
	}

	// `starts_at` IS COMPARED AS THE LITERAL THAT SHIPS, offset included, because a
	// TIMESTAMPTZ stores an instant and the operator's intent was 00:00 in +07:00. A seed
	// written in UTC would store a different instant and open the campaign seven hours
	// late — the whole launch-day feature failing quietly (FR-015).
	if seeded[4] != "2026-10-02T00:00:00+07:00" {
		t.Errorf("FR-090/FR-015: the seed must insert starts_at 2026-10-02T00:00:00+07:00 as an "+
			"explicit offset, got %q", seeded[4])
	}
	// And the campaign is open-ended: `ends_at` must be NULL, not a zero time, or the
	// campaign would read as `expired` the moment it started.
	if seeded[5] != "NULL" {
		t.Errorf("FR-090: the seed must leave ends_at NULL (the campaign is open-ended), got %q",
			seeded[5])
	}
	// And the whole INSERT must be idempotent, because a service migrates on every boot.
	if !strings.Contains(insert, "ON CONFLICT") {
		t.Errorf("FR-090/US6.2: the seed INSERT must be idempotent, got:\n%s", insert)
	}
}

// seedValues splits the seed's `VALUES (…)` tuple into its six elements, unquoted.
//
// IT PARSES THE TUPLE RATHER THAN SEARCHING FOR EACH VALUE, and the reason is that a
// search is what makes this kind of test rot: `quota` and the number `50` are on
// DIFFERENT LINES of the statement, so a regex pairing the column name with a digit
// silently matches nothing and the assertion passes vacuously. Splitting the tuple makes
// a missing element a hard failure instead.
func seedValues(t *testing.T, insert string) []string {
	t.Helper()

	marker := "VALUES ("
	start := strings.Index(insert, marker)
	if start < 0 {
		t.Fatalf("the seed must insert with a VALUES tuple, got:\n%s", insert)
	}
	tuple := insert[start+len(marker):]
	if end := strings.Index(tuple, ")"); end >= 0 {
		tuple = tuple[:end]
	}

	parts := strings.Split(tuple, ",")
	if len(parts) != 6 {
		t.Fatalf("the seed must state exactly six values (data-model.md §2.5), got %d in %q",
			len(parts), tuple)
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, strings.Trim(strings.TrimSpace(part), "'"))
	}
	return values
}

// TestTheFourReadEndpointsAnswerTheSameBeforeAndAfterTheSwitch is quickstart V4's
// comparison, run against the fixture.
//
// THE LEDGER IS SEEDED TO PRODUCTION'S SHAPE: 50 grants from the main bucket — the
// fiftieth carrying `fills_main_quota`, which is what freezes the watermark — and 5 from
// the reserve. That is 55 claims against a hard cap of 55, so the campaign is
// `exhausted`: the state an operator would have been looking at before the migration and
// must still be looking at after it.
//
// If the four endpoints answered anything else, the switch would have changed what a
// promotion is doing — which is the entire risk US6 exists to retire (spec.md Câu 5).
func TestTheFourReadEndpointsAnswerTheSameBeforeAndAfterTheSwitch(t *testing.T) {
	campaign := formerConfigurationCampaign()
	claims := newFakeClaimRepo()
	grantedUser := uuid.New()
	grantedAccount := &identitydto.UserBriefContractDTO{
		ID:        grantedUser,
		Username:  "verify-026-migrated",
		CreatedAt: baseInstant().Add(-96 * time.Hour),
	}

	// THE WATERMARK IS CAPTURED FROM THE SEEDING LOOP rather than written out, because
	// it is not a number anybody chose: it is the `registered_at` of the claim that
	// brought the main count to the quota. Asserting it against a hand-written constant
	// would make the expectation a second, unrelated claim about the same instant — and
	// the instant IS the number an operator reads.
	var watermark time.Time

	// The fiftieth main grant carries the watermark, exactly as the grant path writes
	// it: the claim that brought the main count TO the quota.
	for i := 0; i < campaign.Quota; i++ {
		registeredAt := baseInstant().Add(-time.Duration(campaign.Quota-i) * time.Minute)
		fillsMain := i == campaign.Quota-1
		if fillsMain {
			watermark = registeredAt
		}
		userID := uuid.New()
		if i == 0 {
			// One known account, so the eligibility endpoint below can be asked about
			// somebody who really was granted.
			userID = grantedUser
		}
		claims.seed(launchCampaignCode, userID, registeredAt, entities.GrantBucketMain, fillsMain)
	}
	for i := 0; i < campaign.Reserve; i++ {
		claims.seed(launchCampaignCode, uuid.New(), baseInstant().Add(-time.Duration(i+1)*time.Minute),
			entities.GrantBucketReserve, false)
	}

	h := newTestQueryUC(t, []*entities.Campaign{campaign}, claims,
		newFakeSweepRunRepo(), newFakeUserBriefContract(grantedAccount))
	ctx := context.Background()

	// ---- 1. GET /admin/campaigns ----
	list, total, err := h.uc.ListCampaigns(ctx, 1, 20)
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected exactly the one seeded campaign, got total=%d len=%d", total, len(list))
	}
	assertFidelity(t, "ListCampaigns", list[0], campaign, watermark, sub_dto.CampaignStatusExhausted)

	// ---- 2. GET /admin/campaigns/{code} ----
	single, err := h.uc.GetCampaign(ctx, launchCampaignCode)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	assertFidelity(t, "GetCampaign", single, campaign, watermark, sub_dto.CampaignStatusExhausted)

	// ---- 3. GET /admin/campaigns/{code}/claims ----
	page, claimTotal, err := h.uc.ListClaims(ctx, launchCampaignCode,
		repositories.CampaignClaimsFilter{}, 1, 100)
	if err != nil {
		t.Fatalf("ListClaims: %v", err)
	}
	if claimTotal != 55 {
		t.Errorf("the ledger holds 55 grants, and 55 is what the migration left untouched; "+
			"the read path reported %d", claimTotal)
	}
	if len(page) != 55 {
		t.Fatalf("expected a full page of 55 claims, got %d", len(page))
	}
	// Every claim names the plan the campaign was configured to grant, because the
	// registry row still says `premium-monthly` — a mismatch here would mean the seed
	// moved the campaign onto another plan without anybody noticing.
	for _, claim := range page {
		if claim.PlanSlug != campaign.PlanSlug {
			t.Fatalf("claim for user %s was granted %q, want %q", claim.UserID, claim.PlanSlug, campaign.PlanSlug)
		}
	}
	// And the bucket split is the one the budget arithmetic depends on: 50 from `quota`
	// and 5 from the ceiling above it.
	assertClaimSources(t, page, 50, 5)

	// ---- 4. GET /admin/campaigns/{code}/eligibility/{userId} ----
	//
	// Asked about an account that WAS granted, because that is the answer the migration
	// must not alter: this is the number a support agent reads when a customer asks why
	// they have Premium.
	verdict, err := h.uc.GetAccountEligibility(ctx, launchCampaignCode, grantedUser)
	if err != nil {
		t.Fatalf("GetAccountEligibility: %v", err)
	}
	if verdict.Eligibility != sub_dto.CampaignEligibilityGranted || !verdict.Granted {
		t.Errorf("a granted account must read as granted, got %q (granted=%v)",
			verdict.Eligibility, verdict.Granted)
	}
	// The eligibility response carries the SAME watermark, so the boundary every later
	// priority comparison is measured against is visible on both endpoints and both show
	// the campaign's history unchanged.
	if verdict.WatermarkAt == nil || !verdict.WatermarkAt.Equal(watermark) {
		t.Errorf("watermarkAt = %v, want the frozen watermark %s", verdict.WatermarkAt, watermark)
	}
}

// assertFidelity is the expectation table, stated once because three of the four
// endpoints must agree on it.
//
// The granted counts, the remainders and the ceiling are computed HERE, from the
// fixture, rather than read from the response and compared with itself: a test that
// derived its expectation from the code under test would pass against any behaviour,
// which is the failure mode that lets a migration quietly change a live promotion's
// numbers.
func assertFidelity(
	t *testing.T,
	endpoint string,
	got *sub_dto.CampaignSummaryRes,
	want *entities.Campaign,
	wantWatermark time.Time,
	wantStatus string,
) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s returned no summary", endpoint)
	}
	if got.Code != want.Code || got.PlanSlug != want.PlanSlug {
		t.Errorf("%s: identity changed: %q/%q, want %q/%q",
			endpoint, got.Code, got.PlanSlug, want.Code, want.PlanSlug)
	}
	if got.Quota != want.Quota || got.Reserve != want.Reserve {
		t.Errorf("%s: budget changed: %d/%d, want %d/%d",
			endpoint, got.Quota, got.Reserve, want.Quota, want.Reserve)
	}

	// The three fields quickstart V4 names, and the reason they are the ones that could
	// have moved: the migration does not carry them, the ledger does, and every one of
	// them is computed at query time.
	if got.GrantedMainCount != int64(want.Quota) {
		t.Errorf("%s: grantedMainCount = %d, want %d — the ledger was not touched by the migration",
			endpoint, got.GrantedMainCount, want.Quota)
	}
	if got.GrantedCompensationCount != int64(want.Reserve) {
		t.Errorf("%s: grantedCompensationCount = %d, want %d",
			endpoint, got.GrantedCompensationCount, want.Reserve)
	}
	if got.GrantedTotalCount != int64(want.Quota+want.Reserve) {
		t.Errorf("%s: grantedTotalCount = %d, want %d",
			endpoint, got.GrantedTotalCount, want.Quota+want.Reserve)
	}
	if got.RemainingMain != 0 || got.RemainingCompensation != 0 {
		t.Errorf("%s: the budget is spent, got remaining %d/%d, want 0/0",
			endpoint, got.RemainingMain, got.RemainingCompensation)
	}
	if got.HardCap != int64(want.Quota+want.Reserve) {
		t.Errorf("%s: hardCap = %d, want %d", endpoint, got.HardCap, want.Quota+want.Reserve)
	}
	if got.WatermarkAt == nil || !got.WatermarkAt.Equal(wantWatermark) {
		t.Errorf("%s: watermarkAt = %v, want the frozen watermark %s", endpoint, got.WatermarkAt, wantWatermark)
	}
	if got.Status != wantStatus {
		t.Errorf("%s: status = %q, want %q — the state an operator saw before the migration",
			endpoint, got.Status, wantStatus)
	}

	// FR-050's compatibility rule, read off the same response: 026 ADDED three keys and
	// changed nothing else. `starts_at` must still be the instant the configuration
	// declared, compared as an instant rather than as a formatted string.
	if !got.StartsAt.Equal(want.StartsAt) {
		t.Errorf("%s: startsAt = %s, want %s", endpoint, got.StartsAt, want.StartsAt)
	}
	if got.EndsAt != nil {
		t.Errorf("%s: endsAt must stay absent — the campaign is open-ended, got %v", endpoint, got.EndsAt)
	}
	if got.ClosedAt != nil {
		t.Errorf("%s: a seeded campaign is not force-closed, got closedAt=%v", endpoint, got.ClosedAt)
	}
}

// TestTheSeededCampaignIsBudgetLocked is FR-020 arriving through the read endpoints.
//
// `launch-2026-10` has 55 real grants, so `budgetLocked` MUST be true — and it is
// derived from the counts rather than stored, which is why asserting it here is a
// statement about the migration having left the ledger intact. An operator whose screen
// showed `budgetLocked: false` for a campaign that has handed out 55 plans would be
// invited to edit the budget of a live promotion.
func TestTheSeededCampaignIsBudgetLocked(t *testing.T) {
	campaign := formerConfigurationCampaign()
	claims := newFakeClaimRepo()
	claims.seed(launchCampaignCode, uuid.New(), baseInstant().Add(-time.Hour), entities.GrantBucketMain, false)

	h := newTestQueryUC(t, []*entities.Campaign{campaign}, claims,
		newFakeSweepRunRepo(), newFakeUserBriefContract())
	ctx := context.Background()

	// One grant is enough: FR-020's condition is "at least one", and an operator must not
	// be able to edit the budget of a campaign that has already given anything away.
	got, err := h.uc.GetCampaign(ctx, launchCampaignCode)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	if !got.BudgetLocked {
		t.Error("FR-020/FR-051: a campaign with a claim must report budgetLocked = true")
	}
	if got.Version != 1 {
		t.Errorf("FR-051: a seeded campaign must report the version it is stored at, got %d", got.Version)
	}
}

// assertClaimSources counts the two `grant_bucket` values in a page of claims.
//
// SEEN AS THE TWO COLUMN VALUES rather than as the STATUS names, because the bucket is
// what the budget arithmetic reasons about: deriving "how many drew on the reserve" from
// a derived status would let a status change silently alter which grants are counted
// against which ceiling (research.md R2b).
func assertClaimSources(t *testing.T, claims []*sub_dto.CampaignClaimRes, wantMain, wantReserve int) {
	t.Helper()

	main, reserve := 0, 0
	for _, claim := range claims {
		switch claim.Source {
		case sub_dto.CampaignClaimSourceMain:
			main++
		case sub_dto.CampaignClaimSourceReserve:
			reserve++
		default:
			t.Fatalf("claim for user %s has source %q, which is neither %q nor %q",
				claim.UserID, claim.Source,
				sub_dto.CampaignClaimSourceMain, sub_dto.CampaignClaimSourceReserve)
		}
	}
	if main != wantMain || reserve != wantReserve {
		t.Errorf("bucket split = %d main / %d reserve, want %d / %d", main, reserve, wantMain, wantReserve)
	}
}

// seedMigrationInsert returns the `INSERT` statement of the seed migration.
//
// The `Up` section is cut at the first `-- +goose` marker after it, which is where the
// statement ends — so what is compared below is the SQL that actually runs, and not the
// file's own prose about it.
func seedMigrationInsert(t *testing.T) string {
	t.Helper()

	matches, err := filepath.Glob(seedMigrationPath)
	if err != nil {
		t.Fatalf("look for the seed migration (%s): %v", seedMigrationPath, err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one seed migration matching %s, found %d", seedMigrationPath, len(matches))
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	up := string(body)
	start := strings.Index(up, "-- +goose Up")
	if start < 0 {
		t.Fatalf("%s has no `-- +goose Up` section; FR-090 requires one", matches[0])
	}
	rest := up[start+len("-- +goose Up"):]
	if end := strings.Index(rest, "-- +goose StatementEnd"); end >= 0 {
		rest = rest[:end]
	}
	if end := strings.Index(rest, "-- +goose Down"); end >= 0 {
		rest = rest[:end]
	}
	if !strings.Contains(rest, "INSERT INTO subscription.campaigns") {
		t.Fatalf("%s must insert into subscription.campaigns, got:\n%s", matches[0], rest)
	}
	return rest
}
