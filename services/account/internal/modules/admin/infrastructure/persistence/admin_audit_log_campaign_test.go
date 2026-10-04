package persistence

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	"smart-wardrobe-be/services/account/internal/modules/admin/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 026 T070 (FR-062, FR-063, research.md R8) — `ListByCampaignCode`, the query the
// per-campaign audit trail is built on.
//
// FOUR THINGS ARE ASSERTED, AND THE LAST IS THE ONE NOBODY WOULD THINK TO CHECK:
//
//  1. THE PREDICATE — both terms, with the code BOUND rather than interpolated.
//  2. THE ORDERING — newest first, with a total tie-break, or two rows written in the same
//     millisecond could swap places between two identical calls.
//  3. THE MAPPING — a campaign row comes back with a NIL `TargetID`, because FR-063
//     stores NULL there for every campaign row.
//  4. THE BOUNDARY — no task in the `admin` module reads `subscription.campaigns`.
//
// THE PREDICATE IS ASSERTED WITHOUT A SERVER, and it goes THROUGH THE REPOSITORY rather
// than through a restatement of its clause chain. `gorm.Open` with
// `DisableAutomaticPing` connects nothing, so the call fails — but GORM reports every
// statement to its logger before and after the attempt, and that report is the SQL the
// driver would have been handed. A restated chain would pass while the repository drifted,
// which is exactly the failure this assertion exists to catch.
//
// A REAL ROUND TRIP IS HERE TOO, guarded by `TEST_DATABASE_URL` as every other
// Postgres-backed test in this service behaves: the captured SQL proves the statement, and
// only a server can prove that `payload->>'campaignCode'` is a legal expression against a
// `jsonb` column, that `->>` returns text rather than JSON, and that another campaign's
// rows — and another TARGET TYPE's rows carrying the same key — really are excluded.

// campaignAuditCode is the code the fixtures file their rows under. It matches
// `chk_campaigns_code_format`'s alphabet, so the fixture is honest about the shape of the
// value it binds.
const campaignAuditCode = "launch-2026-10"

// unrouteableDSN points at a port nothing listens on. Combined with
// `DisableAutomaticPing` it lets the dialect initialise and the statement be built, which
// is all the predicate assertions need.
const unrouteableDSN = "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable"

// sqlCapturingLogger records every statement GORM reports.
//
// IT EXISTS BECAUSE GORM DOES NOT HAND THE STATEMENT BACK. A repository method returns
// rows and an error, so the SQL it built is visible only through the logger — which is
// called on the failure path too, and that is the point: there is no server here to
// succeed against.
//
// IT EMBEDS A BASE LOGGER rather than implementing all four methods, so the only behaviour
// it changes is the one it exists for.
type sqlCapturingLogger struct {
	gormlogger.Interface
	mu         sync.Mutex
	statements []string
}

func newSQLCapturingLogger() *sqlCapturingLogger {
	base := gormlogger.New(log.New(io.Discard, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})
	return &sqlCapturingLogger{Interface: base}
}

// ParamsFilter keeps the statement's PLACEHOLDERS instead of substituting the values.
//
// GORM offers every logger this hook and defaults to substituting the variables in, so the
// displayed statement cannot distinguish a bound value from an interpolated one — and the
// campaign code would appear in the text either way. Returning the SQL unchanged is what
// makes "the code is BOUND" an assertable fact rather than an assumption.
func (l *sqlCapturingLogger) ParamsFilter(
	_ context.Context,
	sql string,
	_ ...any,
) (string, []any) {
	return sql, nil
}

func (l *sqlCapturingLogger) Trace(
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

func (l *sqlCapturingLogger) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.statements))
	copy(out, l.statements)
	return out
}

var _ gormlogger.Interface = (*sqlCapturingLogger)(nil)

// campaignAuditStatements runs the repository against an unrouteable dialector and
// returns the statements it tried to execute.
//
// ONE STATEMENT IS CAPTURED, AND THAT IS CORRECT RATHER THAN A SHORTCOMING: the method
// totals the trail first and returns on failure, so without a server the page SELECT is
// never BUILT. The `WHERE` clause is identical in both statements — GORM assembles the
// conditions before the count and reuses them — so the captured statement pins the
// predicate exactly. The ORDER BY lives only on the page statement and is therefore
// asserted behaviourally, on a real database, further down.
func campaignAuditStatements(t *testing.T) string {
	t.Helper()
	capture := newSQLCapturingLogger()
	dbConn, err := gorm.Open(postgres.Open(unrouteableDSN), &gorm.Config{
		Logger:               capture,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("initialise the dialector without connecting: %v", err)
	}
	repo := NewAdminAuditLogRepository(dbConn)

	// A failure is the expected outcome — there is no server — and it is what proves the
	// call reached the driver rather than short-circuiting somewhere.
	if _, _, err := repo.ListByCampaignCode(context.Background(), campaignAuditCode, 1, 20); err == nil {
		t.Fatal("expected the call to fail without a server; a success would mean it never reached the driver")
	}

	statements := capture.all()
	if len(statements) == 0 {
		t.Fatal("no statement was captured; the assertion would pass vacuously")
	}
	return statements[0]
}

// TestListByCampaignCodeFiltersOnBothTerms is the predicate, read off the statement the
// repository actually builds.
//
// BOTH TERMS ARE PINNED SEPARATELY, because each alone produces a page that looks right.
// A query with only `payload->>'campaignCode' = ?` returns another target type's row that
// happened to carry the same key — a plan catalogue change filed with the same field name
// would appear in a campaign's trail. A query with only `target_type = 'campaign'` returns
// EVERY campaign's trail for every code, which is precisely what FR-063's dedicated
// endpoint exists to prevent.
//
// THE VALUE MUST BE BOUND, and that is asserted rather than assumed: `payload` reaches
// this table from an HTTP body, and a code containing a quote would otherwise change the
// query's meaning. The statement is captured with `ParameterizedQueries`, so a placeholder
// appears where the value would be interpolated.
func TestListByCampaignCodeFiltersOnBothTerms(t *testing.T) {
	statement := campaignAuditStatements(t)

	if !strings.Contains(statement, "target_type") {
		t.Errorf("FR-063: the query must filter on target_type, got %s", statement)
	}
	if !strings.Contains(statement, "payload->>'campaignCode'") {
		t.Errorf("FR-063/contract §6.4: the code must be read from the payload with ->>, got %s", statement)
	}
	// `->>` and not `->`: the comparison is against text, and `->` yields jsonb, which a
	// string parameter would silently never equal.
	if strings.Contains(statement, "payload->'campaignCode'") {
		t.Errorf("the selector must use ->> (text) rather than -> (json): %s", statement)
	}
	// BOUND, NOT INTERPOLATED. With `ParameterizedQueries` the code cannot appear here at
	// all, which is the point: a query that interpolated it would show it.
	if strings.Contains(statement, campaignAuditCode) {
		t.Errorf("the campaign code must be a bound parameter rather than part of the SQL: %s", statement)
	}
	// And nothing but the audit table is read.
	if !strings.Contains(statement, `"shared"."admin_audit_logs"`) {
		t.Errorf("the query must read shared.admin_audit_logs and nothing else: %s", statement)
	}
	if strings.Contains(statement, `"subscription"`) {
		t.Errorf("FR-062/constitution §II: the query must not touch the campaign registry: %s", statement)
	}
}

// TestListByCampaignCodeIsOrderedNewestFirstAndReproducibly is the ordering, asserted
// BEHAVIOURALLY rather than off the SQL text, and that is forced and right.
//
// `ORDER BY` lives on the page statement, which is only built once the total has been
// counted — so it cannot be captured without a server. What a reader actually needs to
// know is not that the clause is spelled a particular way but that two runs over the same
// rows return them in the same order, and this asserts exactly that.
//
// THE FIXTURE SHARES AN INSTANT ON PURPOSE. A campaign's create and its first edit
// routinely land inside one millisecond, and without a second sort key PostgreSQL may
// return those two rows in either order between two identical calls — a screen showing
// the edit above the create it followed, intermittently, which is the worst kind of defect
// to notice. Both rows here carry the SAME `created_at`, so the ordering has to be decided
// by the tie-break or this test fails.
func TestListByCampaignCodeIsOrderedNewestFirstAndReproducibly(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping campaign audit ordering integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS,
		"goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	actorID := insertAuditActor(t, dbConn)
	rows := []*entities.AdminAuditLog{
		newCampaignAuditRow(actorID, adminaudit.CampaignCreate, campaignAuditCode),
		newCampaignAuditRow(actorID, adminaudit.CampaignUpdate, campaignAuditCode),
	}
	for _, row := range rows {
		if err := dbConn.Create(row).Error; err != nil {
			t.Fatalf("insert audit row %s: %v", row.Action, err)
		}
	}
	t.Cleanup(func() {
		for _, row := range rows {
			dbConn.Delete(&entities.AdminAuditLog{}, "id = ?", row.ID)
		}
	})

	repo := NewAdminAuditLogRepository(dbConn)
	first, _, err := repo.ListByCampaignCode(context.Background(), campaignAuditCode, 1, 20)
	if err != nil {
		t.Fatalf("ListByCampaignCode: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("precondition: expected the two fixture rows, got %d", len(first))
	}
	// `id` DESC breaks the tie, and the ids were generated in the order the actions were
	// written, so the UPDATE — created second — must come first.
	if first[0].ID != rows[1].ID || first[1].ID != rows[0].ID {
		t.Errorf("the tie-break must make the order total and reproducible; got %s then %s, want "+
			"%s then %s", first[0].ID, first[1].ID, rows[1].ID, rows[0].ID)
	}

	// THE SAME ORDER TWICE. This is the assertion the whole test exists for: an order that
	// depends on the planner is correct on the day it was written.
	for attempt := 0; attempt < 5; attempt++ {
		again, _, err := repo.ListByCampaignCode(context.Background(), campaignAuditCode, 1, 20)
		if err != nil {
			t.Fatalf("ListByCampaignCode (attempt %d): %v", attempt, err)
		}
		if len(again) != 2 || again[0].ID != first[0].ID || again[1].ID != first[1].ID {
			t.Fatalf("attempt %d returned a different order: %v, want %s then %s", attempt,
				[]uuid.UUID{again[0].ID, again[1].ID}, first[0].ID, first[1].ID)
		}
	}
}

// TestListByCampaignCodeMapsRowsWithANullTargetID is FR-063's invariant on the DTO the
// consumer receives.
//
// THE ASSERTION IS ABOUT WHAT IS *NOT* THERE. A mapper that filled in a UUID would make the
// row look as though it pointed at something, and the consumer's own response type exists
// precisely to SHOW the null rather than hide it — so a nil here is the correct value and
// the test fails only if something starts inventing one.
func TestListByCampaignCodeMapsRowsWithANullTargetID(t *testing.T) {
	actorID := uuid.New()
	row := newCampaignAuditRow(actorID, adminaudit.CampaignUpdate, campaignAuditCode)

	mapped := mapper.MapAdminAuditLogToDTO(row)
	if mapped == nil {
		t.Fatal("the mapper must return a DTO for a row")
	}
	if mapped.TargetID != nil {
		t.Errorf("FR-063: targetId must stay nil for a campaign row, got %v", *mapped.TargetID)
	}
	if mapped.TargetType != string(adminaudit.Campaign) {
		t.Errorf("FR-063: targetType = %q, want %q", mapped.TargetType, adminaudit.Campaign)
	}
	if got, ok := mapped.Payload["campaignCode"]; !ok || got != campaignAuditCode {
		t.Errorf("the code must survive the mapping in payload.campaignCode, got %v", got)
	}
	if mapped.ActorUserID != actorID {
		t.Errorf("actorUserId = %s, want %s", mapped.ActorUserID, actorID)
	}
}

// TestTheAdminModuleNeverReadsTheCampaignRegistry is the boundary assertion (FR-062,
// constitution §II), and it PARSES the module rather than grepping it.
//
// WHY A SCAN AT ALL. There is no behavioural failure to observe: a module that reached
// across would still compile, still pass every test, and still answer correctly. The rule
// is about WHICH TABLE a module may read, and the only way to assert that kind of absence
// is to look at what the module contains. It is also the assertion that makes the
// endpoint's EMPTY PAGE a documented property rather than a bug: this module cannot check
// whether a campaign exists, which is why the caller must (contract §6.4).
//
// WHY THE AST AND NOT `strings.Contains`, which is the part that took the thinking. The
// obvious implementation — grep every `.go` file for `subscription.campaigns` — fails on
// the module's own PROSE: the comments explaining WHY this module must not read the
// registry necessarily name it. A test that fails on the explanation is a test that gets
// deleted instead of fixed. Parsing without `parser.ParseComments` therefore sees only
// code, and three things are then checked:
//
//   - a STRING LITERAL naming the registry's table — how a raw query or a `TableName()`
//     would reach it;
//   - a SELECTOR on the `entities` package for a `Campaign` type — how a GORM model would
//     reach it;
//   - an IDENTIFIER naming the registry's repository interface — how injected access
//     would reach it.
//
// TEST FILES ARE SKIPPED: the requirement is about production code, and a test may
// legitimately NAME the registry in an assertion message.
func TestTheAdminModuleNeverReadsTheCampaignRegistry(t *testing.T) {
	// Two levels up from `persistence/` inside `infrastructure/` is the module root.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve the module root: %v", err)
	}

	scanned := 0
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		scanned++
		fileSet := token.NewFileSet()
		// NOTE: no `parser.ParseComments`. Dropping comments is what makes this scan a
		// check on CODE rather than on prose.
		parsed, parseErr := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return parseErr
		}
		rel, _ := filepath.Rel(root, path)
		for _, finding := range campaignRegistryReach(parsed) {
			t.Errorf("FR-062/constitution §II: %s reaches the campaign registry through %s. The "+
				"`admin` module owns the audit trail and must not read the campaign registry: a "+
				"campaign's existence is answered by the `subscription` module, which is exactly why "+
				"this trail answers an unknown code with an EMPTY PAGE rather than a 404 "+
				"(contract §6.4) — at %s", rel, finding.what, fileSet.Position(finding.position.Pos()))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan the admin module: %v", walkErr)
	}
	if scanned == 0 {
		t.Fatal("no production file was scanned; the boundary assertion would pass vacuously")
	}
}

// registryTable is the registry's table, assembled rather than written out so this test's
// own source does not contain the literal it forbids.
var registryTable = "subscription." + "campaigns"

// TestTheBoundaryScanDetectsAReach is the scan's own self-check, and it exists because a
// scan that silently stopped matching would make the assertion above pass for ever.
//
// THE THREE PLANTED CASES ARE THE THREE ROUTES to the table: a raw query naming it, a
// GORM model selected off the `entities` package, and the repository interface reached
// through a selector. The fourth case is the one that motivated the AST: the same table
// named in a COMMENT, which must NOT be reported — otherwise the module's own explanation
// of the rule would fail the rule's test.
func TestTheBoundaryScanDetectsAReach(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantReport bool
	}{
		{"a raw query", `package p
func f() { _ = ` + "`" + `SELECT * FROM subscription.campaigns` + "`" + ` }`, true},
		{"a quoted table name", `package p
func f() { _ = ` + "`" + `SELECT * FROM "subscription"."campaigns"` + "`" + ` }`, true},
		{"the GORM model", `package p
import "x/entities"
var _ = entities.Campaign{}`, true},
		{"the repository interface", `package p
import "x/repositories"
var _ repositories.ICampaignRepository`, true},
		{"the same table named in a comment", "package p\n// this module must not read subscription.campaigns\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fileSet := token.NewFileSet()
			parsed, err := parser.ParseFile(fileSet, "planted.go", tc.source,
				parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse the planted source: %v", err)
			}
			reported := len(campaignRegistryReach(parsed)) > 0
			if reported != tc.wantReport {
				t.Errorf("campaignRegistryReach reported %v, want %v, for:\n%s",
					reported, tc.wantReport, tc.source)
			}
		})
	}
}

// registryReach is one place where a file reaches for the campaign registry.
type registryReach struct {
	// what names the kind of reach, so the failure message says which of the three it is.
	what string
	// position is the AST node to point at.
	position ast.Node
}

// campaignRegistryReach reports every way `file` could read the campaign registry.
func campaignRegistryReach(file *ast.File) []registryReach {
	var found []registryReach
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			ident, isIdent := typed.X.(*ast.Ident)
			if !isIdent {
				return true
			}
			// `entities.Campaign`, `entities.CampaignClaim`: the GORM model behind the table.
			if ident.Name == "entities" && strings.HasPrefix(typed.Sel.Name, "Campaign") {
				found = append(found, registryReach{
					what:     fmt.Sprintf("the entity %s.%s", ident.Name, typed.Sel.Name),
					position: typed,
				})
			}
			// `repositories.ICampaignRepository`: injected access to the registry.
			if ident.Name == "repositories" && strings.Contains(typed.Sel.Name, "CampaignRepository") {
				found = append(found, registryReach{
					what:     fmt.Sprintf("the interface %s.%s", ident.Name, typed.Sel.Name),
					position: typed,
				})
			}
		case *ast.BasicLit:
			// A raw query, or a `TableName()` that points a model at the table.
			//
			// QUOTES ARE STRIPPED BEFORE COMPARING, because the same identifier can be
			// written `subscription.campaigns` in a hand-written query and
			// `"subscription"."campaigns"` in a quoted one, and both reach the table.
			if typed.Kind == token.STRING {
				normalised := strings.ReplaceAll(typed.Value, `"`, "")
				if strings.Contains(normalised, registryTable) {
					found = append(found, registryReach{
						what:     "a string literal naming the registry's table",
						position: typed,
					})
				}
			}
		}
		return true
	})
	return found
}

// TestListByCampaignCodeReadsOnlyItsOwnCampaignOnARealDatabase is the round trip the
// captured SQL cannot make: that `->>` on a `jsonb` column returns text, that another
// campaign's rows are excluded, and that a row of ANOTHER TARGET TYPE carrying the same
// payload key is excluded too — the case the `target_type` term exists for.
//
// IT SKIPS WITHOUT A DATABASE, and it says which one. `TEST_DATABASE_URL` plus `t.Skip` is
// how every Postgres-backed test in this service behaves, and a suite that FAILED instead
// would make `go test ./...` red on any machine without one.
func TestListByCampaignCodeReadsOnlyItsOwnCampaignOnARealDatabase(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping campaign audit repository integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// The migrations, not `AutoMigrate`: the audit table's `jsonb` column is what the
	// `->>` selector is written against, and a schema built from the Go structs would not
	// be the one production runs.
	if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS,
		"goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	actorID := insertAuditActor(t, dbConn)
	otherCode := campaignAuditCode + "-other"
	unknownCode := "verify-026-audit-khong-ton-tai"

	rows := []*entities.AdminAuditLog{
		newCampaignAuditRow(actorID, adminaudit.CampaignCreate, campaignAuditCode),
		newCampaignAuditRow(actorID, adminaudit.CampaignUpdate, campaignAuditCode),
		newCampaignAuditRow(actorID, adminaudit.CampaignCreate, otherCode),
		{
			// A row of ANOTHER TARGET TYPE carrying the same payload key. Without the
			// `target_type` term this query would return it under a campaign's trail, and
			// an operator would read a plan-catalogue change as a promotion change.
			ID:          uuid.New(),
			ActorUserID: actorID,
			Action:      string(adminaudit.SubscriptionPlanCreate),
			TargetType:  string(adminaudit.SubscriptionPlan),
			TargetID:    nil,
			Payload:     entities.JSONDocument{"campaignCode": campaignAuditCode},
			CreatedAt:   auditInstant,
		},
	}
	// The two campaign rows share an instant, so the tie-break is exercised rather than
	// assumed: without `id DESC` their order is whatever the planner feels like.
	for _, row := range rows {
		if err := dbConn.Create(row).Error; err != nil {
			t.Fatalf("insert audit row %s: %v", row.Action, err)
		}
	}
	t.Cleanup(func() {
		for _, row := range rows {
			dbConn.Delete(&entities.AdminAuditLog{}, "id = ?", row.ID)
		}
	})

	repo := NewAdminAuditLogRepository(dbConn)
	found, total, err := repo.ListByCampaignCode(context.Background(), campaignAuditCode, 1, 20)
	if err != nil {
		t.Fatalf("ListByCampaignCode: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2 — another campaign's row and another target type's row must "+
			"both be excluded", total)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 rows on the page, got %d", len(found))
	}
	for _, row := range found {
		if row.TargetType != string(adminaudit.Campaign) {
			t.Errorf("FR-063: a foreign target type leaked into the trail: %q", row.TargetType)
		}
		if row.TargetID != nil {
			t.Errorf("FR-063: target_id must be NULL for every campaign row, got %v", *row.TargetID)
		}
		if got, _ := row.Payload["campaignCode"].(string); got != campaignAuditCode {
			t.Errorf("payload.campaignCode = %q, want %q", got, campaignAuditCode)
		}
	}

	// An UNKNOWN CODE IS AN EMPTY PAGE, which is not a failure mode to hide but the reason
	// the caller checks the registry first (contract §6.4).
	found, total, err = repo.ListByCampaignCode(context.Background(), unknownCode, 1, 20)
	if err != nil {
		t.Fatalf("ListByCampaignCode for an unknown code: %v", err)
	}
	if total != 0 || len(found) != 0 {
		t.Errorf("an unknown code must answer an empty page, got %d rows and a total of %d",
			len(found), total)
	}
}

// auditInstant is a FIXED instant, so the two campaign rows really do share a
// `created_at` and the tie-break is exercised rather than assumed.
var auditInstant = time.Date(2026, 10, 3, 10, 15, 0, 0, time.UTC)

// newCampaignAuditRow builds one campaign audit row, with FR-063's three facts already
// applied: `target_type = campaign`, `target_id` nil, the code in the payload.
func newCampaignAuditRow(actorID uuid.UUID, action adminaudit.AuditAction, code string) *entities.AdminAuditLog {
	return &entities.AdminAuditLog{
		ID:          uuid.New(),
		ActorUserID: actorID,
		Action:      string(action),
		TargetType:  string(adminaudit.Campaign),
		// Intentionally nil: FR-063. The code is text and the audit filter takes a UUID.
		TargetID:  nil,
		Payload:   entities.JSONDocument{"campaignCode": code, "version": float64(1)},
		CreatedAt: auditInstant,
	}
}

// insertAuditActor writes the `identity.users` row the audit trail's actor column requires
// and removes it afterwards.
//
// IT IS A RAW INSERT BECAUSE THIS MODULE DOES NOT OWN THAT TABLE, and a test may not
// borrow another module's repository to create it: doing so would be the cross-module
// dependency this very file asserts against. The columns named are the ones an already
// migrated `identity.users` declares NOT NULL, so a database genuinely lacking one of them
// fails with the database's own error rather than being papered over.
func insertAuditActor(t *testing.T, dbConn *gorm.DB) uuid.UUID {
	t.Helper()
	actorID := uuid.New()
	handle := "verify-026-audit-" + actorID.String()
	err := dbConn.Exec(
		`INSERT INTO identity.users (id, username, email, role_slug, status, is_deleted, provider, email_verified, created_at)
		 VALUES (?, ?, ?, 'admin', 0, false, 'local', true, ?)`,
		actorID, handle, handle+"@example.test", auditInstant,
	).Error
	if err != nil {
		t.Fatalf("%v: insert the actor account: %v", errAuditActorMissing, err)
	}
	t.Cleanup(func() {
		dbConn.Exec("DELETE FROM identity.users WHERE id = ?", actorID)
	})
	return actorID
}

// errAuditActorMissing wraps the one failure in this file that is a fixture problem rather
// than a repository fault, so the assertion message is not swallowed into a wrapped error.
var errAuditActorMissing = errors.New("audit integration: could not create the actor account")
