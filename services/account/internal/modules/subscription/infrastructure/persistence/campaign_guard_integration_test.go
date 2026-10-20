package persistence

import (
	"context"
	"errors"
	"testing"
)

// 026 T057 (FR-020, FR-021, FR-022, FR-023, FR-062) — quickstart V3 on a real
// Postgres: the immutability TRIGGER, which is the half of FR-021 no application code
// can provide.
//
// WHY A REAL DATABASE AND NOT A FAKE. FR-062 says the campaign table must not be
// writable by direct SQL in operation, and the trigger is the only thing that makes
// that true — application discipline does not bind a hand-written `UPDATE`. A fake
// repository has no trigger, so this file is where the guarantee is actually proved or
// lost; everything in `registry_uc_update_test.go` is about the layer ABOVE it.
//
// THE `TEST_DATABASE_URL` + `t.Skip` HARNESS IS THE REPO'S, reused unchanged from
// `campaign_repo_integration_test.go`: the schema comes from the EMBEDDED GOOSE
// MIGRATIONS rather than from `AutoMigrate`, and `AutoMigrate` would build the table
// from the Go struct — without the trigger, and therefore testing a schema production
// does not have. Without a provisioned database these tests skip cleanly rather than
// failing, which is the same contract every Postgres-backed test in this service has.
//
// WHAT IS DELIBERATELY NOT HERE: the `422`. Asserting it would mean reaching the
// application use case from `infrastructure`, which skips a layer in the wrong
// direction (constitution §IV) and would make the data-layer test depend on the rule
// layer. The API half of quickstart V3 step 2 lives where the API is:
// `registry_uc_update_test.go` (T056) proves the application layer answers 422, and
// `campaign_registry_update_router_test.go` (T058) proves it reaches the client.

// postgresCheckViolation is SQLSTATE 23514.
//
// The trigger raises with this code explicitly (data-model.md §2.2), and it is the
// right code: the condition it enforces is a rule about the row's budget, which is what
// a CHECK means to a client. Asserting the STATE and not merely "an error" is what makes
// this a test of the intended refusal — a foreign-key or not-null failure arriving from
// the same statement would satisfy "err != nil" and mean nothing.
const postgresCheckViolation = "23514"

// TestIntegrationTriggerRefusesABudgetChangeOnceGrantsExist is quickstart V3 step 3,
// asserted the way quickstart states it: the hand-written UPDATE raises, and the value
// does not move.
//
// THE STATEMENT IS VERBATIM FROM QUICKSTART. `UPDATE subscription.campaigns SET quota =
// quota + 1 WHERE code = '...'` is what an operator with a psql prompt actually types,
// and it is deliberately the crudest possible form: no version predicate, no audit, no
// application layer. If the trigger refuses THIS, it refuses every spelling.
func TestIntegrationTriggerRefusesABudgetChangeOnceGrantsExist(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "guard-quota")
	seedRegistryClaim(t, dbConn, seeded.Code)

	before, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || before == nil {
		t.Fatalf("read the campaign before the attempt: %v", err)
	}

	err = dbConn.Exec("UPDATE subscription.campaigns SET quota = quota + 1 WHERE code = ?", seeded.Code).Error
	if err == nil {
		t.Fatal("FR-021/FR-062: the trigger MUST refuse a hand-written quota change once grants exist")
	}
	if state := sqlStateOf(err); state != postgresCheckViolation {
		t.Errorf("the trigger must raise check_violation (%s), got SQLSTATE %q from %v",
			postgresCheckViolation, state, err)
	}

	// AND THE VALUE IS UNCHANGED, which quickstart V3 asks for explicitly. An exception
	// alone would not be enough: a trigger that raised AFTER the row had been written,
	// or one inside a statement the caller could catch and continue from, would leave
	// the budget moved and the ledger wrong.
	after, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || after == nil {
		t.Fatalf("read the campaign after the refused attempt: %v", err)
	}
	if after.Quota != before.Quota {
		t.Errorf("quota moved from %d to %d despite the refusal — the row must be untouched",
			before.Quota, after.Quota)
	}
}

// TestIntegrationTriggerRefusesPlanSlugAndReserveToo is FR-020's full column list.
//
// `plan_slug` is the one that is easy to forget and the most damaging to lose: each
// claim froze the plan it was given, so a campaign whose `plan_slug` changed mid-flight
// would have a ledger saying accounts received Premium Monthly from a row that now
// promises Premium Yearly.
func TestIntegrationTriggerRefusesPlanSlugAndReserveToo(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{"plan-slug", "UPDATE subscription.campaigns SET plan_slug = 'premium-yearly' WHERE code = ?"},
		{"reserve", "UPDATE subscription.campaigns SET reserve = reserve + 5 WHERE code = ?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbConn := openRegistryTestDB(t)
			repo := NewCampaignRepository(dbConn)
			seeded := integrationCampaign(t, repo, "guard-"+tc.name)
			seedRegistryClaim(t, dbConn, seeded.Code)

			before, err := repo.GetByCode(context.Background(), seeded.Code)
			if err != nil || before == nil {
				t.Fatalf("read the campaign: %v", err)
			}

			if err := dbConn.Exec(tc.sql, seeded.Code).Error; err == nil {
				t.Errorf("FR-020: the trigger MUST refuse a hand-written %s change once grants exist", tc.name)
			} else if state := sqlStateOf(err); state != postgresCheckViolation {
				t.Errorf("%s: expected check_violation (%s), got %q from %v",
					tc.name, postgresCheckViolation, state, err)
			}

			after, err := repo.GetByCode(context.Background(), seeded.Code)
			if err != nil || after == nil {
				t.Fatalf("read the campaign after the refused attempt: %v", err)
			}
			if after.PlanSlug != before.PlanSlug || after.Reserve != before.Reserve {
				t.Errorf("FR-020: %s moved despite the refusal: %q/%d → %q/%d",
					tc.name, before.PlanSlug, before.Reserve, after.PlanSlug, after.Reserve)
			}
		})
	}
}

// TestIntegrationTriggerAllowsABudgetChangeBeforeAnyGrant is FR-022's half at the data
// layer, and it is the CONTROL the refusal tests need.
//
// A trigger that refused unconditionally would pass both tests above and make the whole
// registry read-only: "open a campaign, then raise its budget" is the ordinary way to
// size a promotion nobody has been granted from yet, and it is how quickstart V3 step 1
// is meant to succeed.
func TestIntegrationTriggerAllowsABudgetChangeBeforeAnyGrant(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "guard-open-budget")

	if err := dbConn.Exec("UPDATE subscription.campaigns SET quota = quota + 1 WHERE code = ?", seeded.Code).Error; err != nil {
		t.Fatalf("FR-022: with no claims the budget MUST be editable, got %v", err)
	}

	after, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || after == nil {
		t.Fatalf("read the campaign: %v", err)
	}
	if after.Quota != seeded.Quota+1 {
		t.Errorf("quota = %d, want %d — the permitted change did not take effect",
			after.Quota, seeded.Quota+1)
	}
}

// TestIntegrationTriggerLeavesTheWindowEditable is FR-023 at the data layer, and it is
// the second most valuable assertion in this file.
//
// A trigger written as "any UPDATE on this table is refused" would satisfy every
// immutability test above and silently break the one thing operators need most: moving
// `ends_at` on a campaign that is already running. The trigger's predicate is the three
// budget columns and NOTHING else, and this is what proves the predicate is that narrow.
func TestIntegrationTriggerLeavesTheWindowEditable(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "guard-window")
	seedRegistryClaim(t, dbConn, seeded.Code)

	later := seeded.StartsAt.AddDate(0, 2, 0)
	if err := dbConn.Exec("UPDATE subscription.campaigns SET ends_at = ? WHERE code = ?", later, seeded.Code).Error; err != nil {
		t.Fatalf("FR-023: the window must stay editable once grants exist, got %v", err)
	}

	after, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || after == nil {
		t.Fatalf("read the campaign: %v", err)
	}
	if after.EndsAt == nil {
		t.Fatal("ends_at must have been written")
	}
	if !after.EndsAt.Equal(later) {
		t.Errorf("ends_at = %s, want %s", after.EndsAt, later)
	}
}

// TestIntegrationTriggerDoesNotFireOnARepositoryUpdate is the two layers agreeing.
//
// The application's own path goes through `CampaignRepository.Update`, which passes
// `plan_slug`, `quota`, `reserve`, `starts_at` and `ends_at` on EVERY statement. If the
// trigger compared the whole row instead of the three budget columns, every legitimate
// edit would raise once a campaign had granted — and the failure would look like "the
// edit endpoint is broken" rather than like a trigger that is too broad. Asserting that
// a WINDOW edit goes through the repository proves the two layers are compatible.
func TestIntegrationTriggerDoesNotFireOnARepositoryUpdate(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "guard-repo-window")
	seedRegistryClaim(t, dbConn, seeded.Code)

	current, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || current == nil {
		t.Fatalf("read the campaign: %v", err)
	}

	candidate := *current
	candidate.StartsAt = current.StartsAt.AddDate(0, 1, 0)
	updated, err := repo.Update(context.Background(), &candidate, current.Version)
	if err != nil {
		t.Fatalf("FR-021/FR-023: the two layers must agree — a window edit through the repository "+
			"raised from the trigger: %v", err)
	}
	if !updated {
		t.Error("the window edit must change exactly one row")
	}
}

// TestIntegrationTriggerExistsOnTheCampaignsTable is the presence check, kept separate
// from the behaviour above.
//
// Every test in this file would also pass if `trg_campaigns_immutable_budget` had been
// DROPPED and the refusals came from something else — the CHECK constraints would refuse
// `quota = 0` but nothing here sends that. So the trigger is looked up by name: an
// absent trigger is then a failure rather than an absence of evidence.
func TestIntegrationTriggerExistsOnTheCampaignsTable(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	var triggerName string
	err := dbConn.Raw(
		`SELECT tgname FROM pg_trigger
		  WHERE tgrelid = 'subscription.campaigns'::regclass AND NOT tgisinternal AND tgname = ?`,
		"trg_campaigns_immutable_budget",
	).Scan(&triggerName).Error
	if err != nil {
		t.Fatalf("read the trigger list: %v", err)
	}
	if triggerName != "trg_campaigns_immutable_budget" {
		t.Errorf("FR-021: `trg_campaigns_immutable_budget` must exist on subscription.campaigns, found %q",
			triggerName)
	}
}

// sqlStateOf reads the SQLSTATE out of a driver's error.
//
// IT ASSERTS ON THE METHOD RATHER THAN THE TYPE, for the reason
// `campaign_repo_impl.go`'s own `sqlStateCarrier` gives: naming `*pgconn.PgError` in
// application-adjacent code would turn a transitive dependency of the GORM postgres
// driver into a direct one. `SQLState() string` is part of SQL's standard error
// reporting, so the interface assertion is both smaller and correct.
func sqlStateOf(err error) string {
	var carrier sqlStateCarrier
	if !errors.As(err, &carrier) {
		return ""
	}
	return carrier.SQLState()
}
