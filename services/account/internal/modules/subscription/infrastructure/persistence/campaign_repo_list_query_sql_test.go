package persistence

import (
	"context"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 026 T088 (FR-040, FR-041, FR-070) — the SQL that `CampaignRepository.ListOpen` and
// `ListAll` ACTUALLY build.
//
// THE GAP THIS CLOSES, STATED BY THE PRODUCTION CODE ITSELF.
// `campaign_repo_impl.go:63`-`:67` says a copy of `listOrdered` that dropped `code DESC`
// "would keep every behavioural test green while making the result non-deterministic". That
// was true when it was written, and convergence made it truer: the two assertions about
// these queries are `TestIntegrationListOpenExcludesClosedCampaigns` and
// `TestIntegrationListAllOrdersNewestFirstWithCodeTieBreak`, both of which open with
// `openRegistryTestDB` and therefore `t.Skip` without `TEST_DATABASE_URL` (D-01). T084
// pinned the filtering of the FAKE repository, not of this one. So the real queries had no
// protection anywhere.
//
// THE WEIGHTS ARE NOT EQUAL, and the assertions are ordered that way.
//
//	`closed_at IS NULL` (FR-070) — losing it does not make a query look wrong, it makes a
//	  CLOSED CAMPAIGN GRANT AGAIN. That is the very thing FR-031 was created to stop, and
//	  every behavioural test stays green because they all go through a fake.
//	`code DESC` (FR-040) — losing it makes the operator's list, and the registry itself,
//	  non-deterministic between two identical calls on the same rows.
//
// THE TECHNIQUE IS THE REPOSITORY'S, NOT NEW. `admin_audit_log_campaign_test.go` runs a
// repository against an UNREACHABLE dialector with a `gormlogger.Interface` that captures
// the statement: `gorm.Open` with `DisableAutomaticPing` connects nothing, the call fails,
// and GORM reports the statement to its logger before and after the attempt — so the SQL
// the driver would have been handed is readable without a server.
//
// IT IS EASIER HERE THAN THERE, and the reason is worth recording: that method counts the
// page before selecting it, so without a server only the count is ever BUILT and the
// ORDER BY — which lives on the page statement — was unreachable. `listOrdered` issues a
// SINGLE `Find` with no prior count, so one captured statement carries the WHERE and the
// ORDER BY together, and both halves of the contract are pinned by the same capture.
//
// WHAT IT STILL DOES NOT PROVE. No server means no proof that PostgreSQL accepts the
// statement, that the rows come back in that order, or that a closed row is really
// excluded from a result set. It proves the repository BUILDS the query FR-040 and FR-070
// describe. The behavioural half remains D-01.

// unreachableRegistryDSN points at a port nothing listens on. With
// `DisableAutomaticPing` the dialect initialises and the statement is built, which is all
// these assertions need — and the connection attempt failing is what proves the call
// reached the driver rather than short-circuiting somewhere.
const unreachableRegistryDSN = "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable"

// registrySQLCapture records every statement GORM reports.
//
// IT EMBEDS A BASE LOGGER rather than implementing all four methods, so the only behaviour
// it changes is the one it exists for. It is this package's own copy rather than the one
// in `modules/admin/infrastructure/persistence`: sharing a test helper across modules is
// exactly the import `constitution` §II forbids, and a duplicated twenty lines is the
// cheaper of the two costs.
type registrySQLCapture struct {
	gormlogger.Interface
	mu         sync.Mutex
	statements []string
}

func newRegistrySQLCapture() *registrySQLCapture {
	base := gormlogger.New(log.New(io.Discard, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})
	return &registrySQLCapture{Interface: base}
}

// ParamsFilter keeps the statement's PLACEHOLDERS rather than substituting values in.
//
// `closed_at IS NULL` and the ordering carry no bound parameters, so this hook does not
// change what these assertions read — it is here so the capture is honest about what it
// recorded, exactly as `admin_audit_log_campaign_test.go:93` leaves it, and so the same
// helper can be pointed at a query that binds one later.
func (l *registrySQLCapture) ParamsFilter(
	_ context.Context,
	sql string,
	_ ...any,
) (string, []any) {
	return sql, nil
}

func (l *registrySQLCapture) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (string, int64),
	err error,
) {
	statement, _ := fc()
	l.mu.Lock()
	l.statements = append(l.statements, statement)
	l.mu.Unlock()
	l.Interface.Trace(ctx, begin, fc, err)
}

func (l *registrySQLCapture) captured() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.statements))
	copy(out, l.statements)
	return out
}

var _ gormlogger.Interface = (*registrySQLCapture)(nil)

// registryListStatement runs one of the two listings against an unreachable dialector and
// returns the statement it built.
//
// THE STATEMENT IS `SELECT * FROM "campaigns" …` WITH NO SCHEMA QUALIFIER, because the
// schema comes from the connection's `search_path` rather than from the statement: the
// entity's `TableName()` is the bare name (`campaign.go`) and the DSN supplies the schema.
// That is why these assertions name the table and not `subscription.campaigns` — asserting
// a qualifier the repository never writes would fail on a correct query.
func registryListStatement(
	t *testing.T,
	list func(repositories.ICampaignRepository) ([]*entities.Campaign, error),
) string {
	t.Helper()

	capture := newRegistrySQLCapture()
	dbConn, err := gorm.Open(postgres.Open(unreachableRegistryDSN), &gorm.Config{
		Logger:               capture,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("initialise the dialector without connecting: %v", err)
	}
	repo := NewCampaignRepository(dbConn)

	// A failure is the EXPECTED outcome, and it is what proves the statement was handed to
	// the driver instead of the repository giving up before that.
	if _, err := list(repo); err == nil {
		t.Fatal("expected the call to fail without a server; a success would mean it never reached " +
			"the driver, and these assertions would pass without a statement having been built")
	}

	statements := capture.captured()
	if len(statements) == 0 {
		t.Fatal("no statement was captured; the assertions would pass vacuously")
	}
	// EXACTLY ONE, and that is the property that makes a single capture sufficient: the
	// count that defeats `admin_audit_log_campaign_test.go`'s capture does not exist here.
	// If a count is ever added ahead of the `Find`, these assertions would silently start
	// reading the wrong statement, so the count is checked rather than assumed.
	if len(statements) != 1 {
		t.Fatalf("the listing must issue ONE statement so one capture carries both the WHERE and the "+
			"ORDER BY, got %d: %v", len(statements), statements)
	}
	return statements[0]
}

// TestTheListOpenQueryFiltersOnClosedAtInSQL is FR-070/FR-031 read off the statement the
// repository builds, and it is the heavier of the two because of what its absence costs.
//
// A `ListOpen` that lost `closed_at IS NULL` returns closed campaigns to BOTH the grant
// path and the compensation sweep, so closing a campaign early would stop meaning anything
// until someone noticed — while every unit test stayed green, since
// `fakeCampaignRepo.ListOpen` filters for itself.
//
// WHAT IS DELIBERATELY NOT ASSERTED: that a closed campaign should appear. That question
// was settled (`spec.md` Câu 6, research.md R9) — `closed_at IS NULL` is the ONLY term in
// SQL and the window stays in the pure evaluator — and this test records the decision
// rather than reopening it. The counterpart is that `ListAll` must still show closed
// campaigns for the operator (FR-032), asserted in the test below.
func TestTheListOpenQueryFiltersOnClosedAtInSQL(t *testing.T) {
	statement := registryListStatement(t,
		func(repo repositories.ICampaignRepository) ([]*entities.Campaign, error) {
			return repo.ListOpen(context.Background())
		})

	if !strings.Contains(statement, `FROM "campaigns"`) {
		t.Errorf("FR-070: the query must read the registry, got:\n%s", statement)
	}
	if !strings.Contains(statement, "closed_at IS NULL") {
		t.Errorf("FR-070/FR-031: ListOpen MUST filter on `closed_at IS NULL`. Without it a closed "+
			"campaign keeps granting plans and keeps being swept — which is the thing closing it "+
			"early is supposed to stop. Got:\n%s", statement)
	}
	// `IS NULL`, never `= NULL`: the second is not a comparison in SQL, it is an unknown,
	// so it would silently match no rows and turn every campaign into
	// `campaign_not_configured` instead of filtering anything.
	if strings.Contains(statement, "closed_at = NULL") {
		t.Errorf("FR-070: the predicate must be `closed_at IS NULL`; `= NULL` is UNKNOWN in SQL and "+
			"matches nothing. Got:\n%s", statement)
	}

	assertNoWindowPredicateInSQL(t, statement)
	assertOrderedNewestFirstThenCode(t, statement)
}

// TestTheListAllQueryOrdersNewestFirstWithCodeTieBreak is FR-040/FR-041, and the ORDER BY
// is the whole content of it.
//
// TWO CAMPAIGNS MAY SHARE A `starts_at` — an operator can create the second from any
// screen at any moment — and without a second sort key PostgreSQL returns whichever order
// the plan happens to produce, which may differ between two identical calls. On the admin
// list that is rows moving under the cursor; in the registry it is an evaluator whose
// choice of campaign is not reproducible. `code DESC` is the total order that removes it.
//
// `ListAll` is ALSO the operator list, so it must NOT carry `closed_at`: FR-032 adds
// `status = closed` precisely so an operator can see a campaign that has stopped granting.
// That is asserted here rather than only its absence above, because a repository that
// gained the filter would not look broken from the grant path.
func TestTheListAllQueryOrdersNewestFirstWithCodeTieBreak(t *testing.T) {
	statement := registryListStatement(t,
		func(repo repositories.ICampaignRepository) ([]*entities.Campaign, error) {
			return repo.ListAll(context.Background())
		})

	if !strings.Contains(statement, `FROM "campaigns"`) {
		t.Errorf("FR-040: the query must read the registry, got:\n%s", statement)
	}
	if strings.Contains(statement, "closed_at") {
		t.Errorf("FR-032: ListAll is the operator list and MUST still return closed campaigns — "+
			"`status = closed` exists so one can be seen, not so it can be hidden. Got:\n%s", statement)
	}

	assertOrderedNewestFirstThenCode(t, statement)
	assertNoWindowPredicateInSQL(t, statement)
}

// assertOrderedNewestFirstThenCode is the ORDER BY half both listings share through
// `listOrdered`, asserted as the exact clause rather than as two substrings.
//
// EXACT, because a partial match cannot tell a correct order from a wrong one: a query
// reading `ORDER BY code DESC, starts_at DESC` satisfies "contains code DESC" and "contains
// starts_at DESC" while ordering by code first — which is the opposite of FR-040.
func assertOrderedNewestFirstThenCode(t *testing.T, statement string) {
	t.Helper()

	const want = "ORDER BY starts_at DESC, code DESC"
	if !strings.Contains(statement, want) {
		t.Errorf("FR-040/FR-041: the listing must be ordered `%s` — newest `starts_at` first, and "+
			"`code DESC` as the total tie-break that makes the result reproducible when two "+
			"campaigns share an instant. Got:\n%s", want, statement)
	}
}

// assertNoWindowPredicateInSQL is `spec.md` Câu 6's division of responsibility, asserted.
//
// THE WINDOW (`starts_at`/`ends_at` AGAINST `now`) BELONGS TO THE PURE EVALUATOR, and
// moving it into SQL is not a refactor: `campaignEvaluator.isAwake` is specified to be the
// same predicate as `data-model.md` §4.1, so a SQL window and an evaluator window can
// disagree at the boundary, and the reason code `campaign_sleeping` disappears in favour of
// `campaign_not_configured` (FR-042). A second filtering term in the query would therefore
// be a silent loss of a documented outcome, not a duplicate filter.
//
// `starts_at DESC` IN THE ORDER BY IS NOT A WINDOW PREDICATE, which is why the terms below
// are comparison shapes rather than the bare column name.
func assertNoWindowPredicateInSQL(t *testing.T, statement string) {
	t.Helper()

	for _, term := range []string{"ends_at", "now()", "starts_at >", "starts_at <", "starts_at IS", "starts_at BETWEEN"} {
		if strings.Contains(statement, term) {
			t.Errorf("FR-042/spec.md Câu 6: the listing must not decide the WINDOW in SQL — %q "+
				"appears in it. `isAwake` is specified to be the same predicate as data-model.md "+
				"§4.1, and two implementations of one predicate disagree at the boundary, which "+
				"turns `campaign_sleeping` into `campaign_not_configured`. Got:\n%s", term, statement)
		}
	}
}
