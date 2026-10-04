package persistence

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
)

// 026 T085 / T086 (FR-010, FR-011, FR-012, FR-013, FR-014, FR-021, FR-091) — the two 026
// migrations read as TEXT.
//
// WHY A TEXTUAL FORM WHEN A SCHEMA GUARD ALREADY EXISTS. `campaign_schema_guard_integration_test.go`
// asks the same two questions — no foreign key, and the shape 025 left behind — but it
// asks them of a MIGRATED DATABASE, so every one of its assertions opens with
// `openRegistryTestDB(t)` and therefore `t.Skip`s wherever `TEST_DATABASE_URL` is unset.
// That is every developer machine and the CI profile this repository has, which is exactly
// what `deferred.md` D-01 records: ten ticked tasks whose evidence has never been
// executed. Reading the migration asks a different and complementary question — "what
// would these two files DO to a database" — and it is answered on every ordinary run.
//
// IT READS THE EMBEDDED FILES rather than the directory, for the reason `migrationSection`
// already documents: the embedded set is what the binary ships and what `RunMigrations`
// applies. Reading the directory would let a migration left on disk but excluded from the
// embed be asserted here and never applied.
//
// COMMENTS ARE STRIPPED BEFORE ANY SEARCH, and that is the whole subtlety of this file.
// Both migrations explain at length WHY the safety holds — "no foreign key from the
// ledger to this table", "it never touches subscription.campaign_claims" — so a search
// over raw text fails on the very sentences documenting the property being asserted. A
// guard that fails on its own documentation is a guard that gets deleted.
//
// WHAT THIS DOES NOT PROVE, so nobody reads a green run as more than it is. It reads the
// migration, not a database: it cannot show that PostgreSQL ACCEPTS the statements, that
// the constraints mean what their names say, that the trigger fires, or that the 025
// tables still carry the columns they carried before. Those remain `deferred.md` D-01.

// ---- FR-091: what the two 026 migrations must not contain ----

// registryForeignKeyPattern matches the two ways SQL spells a reference, and BOTH are
// forbidden: an outgoing `FOREIGN KEY` on the registry and an incoming one aimed at it
// are the same defect seen from two tables (FR-091, spec.md Câu 5).
var registryForeignKeyPattern = regexp.MustCompile(`(?i)\b(FOREIGN\s+KEY|REFERENCES)\b`)

// tablesOwnedBy025 are the tables feature 025 created and 026 must leave exactly as they
// are: the grant ledger and the one-row-per-campaign sweep health table.
var tablesOwnedBy025 = []string{"campaign_claims", "campaign_sweep_runs"}

// reach is one way a statement could reach outside `subscription.campaigns`.
type reach struct {
	// what names the reach in the failure message, so a reader is told which shape was
	// found rather than being shown a regular expression.
	what string
	// re matches the statement shape against whitespace-collapsed, comment-free SQL.
	re *regexp.Regexp
}

// reachesOn is every statement shape that would change a table 025 owns, for that table.
//
// ANCHORED ON THE TABLE NAME rather than on the verb, because the verb alone is not the
// property: `CREATE TRIGGER … BEFORE UPDATE ON subscription.campaigns` and
// `CREATE INDEX … ON subscription.campaigns` are both `CREATE … ` statements in the same
// file, and they are the intended work. What FR-091 forbids is the same spellings aimed at
// a table that is not this feature's.
//
// The schema qualifier is optional because the qualified and bare spellings are the same
// statement to the database, and a migration that dropped the schema would be the more
// dangerous variant, not a harmless one.
func reachesOn(table string) []reach {
	owned := `(\w+\.)?` + regexp.QuoteMeta(table) + `\b`
	return []reach{
		{"an ALTER of it", regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(ONLY\s+|IF\s+EXISTS\s+)*` + owned)},
		{"a DROP of it", regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+(IF\s+EXISTS\s+)*` + owned)},
		{"a TRUNCATE of it", regexp.MustCompile(`(?i)\bTRUNCATE\s+(TABLE\s+)?` + owned)},
		{"a RENAME onto it", regexp.MustCompile(`(?i)\bRENAME\s+TO\s+` + owned)},
		{"a SET SCHEMA onto it", regexp.MustCompile(`(?i)\bSET\s+SCHEMA\s+` + owned)},
		{"an index on it", regexp.MustCompile(`(?i)\bCREATE\s+(UNIQUE\s+)?INDEX\s+(CONCURRENTLY\s+)?(IF\s+NOT\s+EXISTS\s+)?\S+\s+ON\s+` + owned)},
		{"a trigger on it", regexp.MustCompile(`(?i)\bCREATE\s+(OR\s+REPLACE\s+)?TRIGGER\b[^;]*?\bON\s+` + owned)},
		{"a data write to it", regexp.MustCompile(`(?i)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM|MERGE\s+INTO)\s+` + owned)},
	}
}

// TestTheRegistryUpSectionsDeclareNoForeignKeyAndLeaveThe025TablesAlone is FR-091
// stated over the EXECUTABLE SQL of both 026 migrations.
//
// WHY THE CLAIM MATTERS MORE THAN ITS SIMPLICITY SUGGESTS. A
// `FOREIGN KEY subscription.campaign_claims.campaign_code -> campaigns.code` compiles,
// runs and reports success on an empty database. It only fails on real data — where the
// ledger legitimately holds a code that is not in the registry (`spec.md` §10), because
// those columns are historical text keys. That turns the cut-over 026 exists to perform
// ("no meaning changed", US6/SC-004) into the riskiest step in the release: the migration
// refuses to run, and an operator reaches for a remedy on a ledger.
//
// WHY IT IS WORTH A TEST THAT RUNS WITHOUT A DATABASE. The five assertions in
// `campaign_schema_guard_integration_test.go` already say this, and none of them has ever
// executed. Between them, the database would catch a foreign key that a REVIEW missed;
// this file catches one that the migration file itself introduced, on every run, including
// on the machine where the review happens.
//
// SCOPE IS THE `Up` SECTIONS, and that is deliberate: the `Down` sections are already
// covered by `TestIntegrationRegistryDownSectionsDoNotTouchTheClaimLedger`, which runs.
// Adding them here would duplicate a working guard rather than close a gap.
//
// THE TRIGGER BODY IS THE PART THAT NEEDS THE CAREFUL WORDING. `Up` legitimately READS
// `subscription.campaign_claims` — that `EXISTS` is what makes the budget immutable
// (FR-020/FR-021). Reading a 025 table is the feature working; altering one is the defect.
// So the reach patterns below are anchored on MUTATING verbs, and a `SELECT 1 FROM
// subscription.campaign_claims` inside the trigger function passes.
func TestTheRegistryUpSectionsDeclareNoForeignKeyAndLeaveThe025TablesAlone(t *testing.T) {
	for _, name := range []string{registrySchemaMigration, registrySeedMigration} {
		up := registryUpSection(t, name)

		// Comments gone, and whitespace collapsed so the search cannot be defeated by
		// re-aligning the file — the mistake a line-by-line match invites.
		executable := collapseWhitespace(stripSQLComments(up))

		if match := registryForeignKeyPattern.FindString(executable); match != "" {
			t.Errorf("FR-091: the executable Up section of %s contains %q. `subscription.campaigns` "+
				"MUST carry no foreign key in either direction: the ledger's `campaign_code` is "+
				"historical text that may name a code which is not in the registry, and a constraint "+
				"would block this migration on live data (spec.md Câu 5, research.md R5):\n%s",
				name, match, executable)
		}

		for _, table := range tablesOwnedBy025 {
			for _, reach := range reachesOn(table) {
				if reach.re.MatchString(executable) {
					t.Errorf("FR-091: the executable Up section of %s declares %s, so 026 would change "+
						"the schema of `subscription.%s` — a table 025 owns and this feature must leave "+
						"exactly as it found it. 026 adds the registry and nothing else:\n%s",
						name, reach.what, table, executable)
				}
			}
		}
	}
}

// ---- FR-010..FR-014 / FR-021: what the schema migration declares ----

// The ten columns FR-010 requires, and the five CHECK constraints FR-011..FR-014 name.
// `quickstart.md` §1.1 lists both by name, so a rename is visible in the same diff as the
// documentation that says the name.
var (
	registryRequiredColumns = []string{
		"closed_at",
		"code",
		"created_at",
		"ends_at",
		"plan_slug",
		"quota",
		"reserve",
		"starts_at",
		"updated_at",
		"version",
	}
	registryRequiredConstraints = []string{
		"chk_campaigns_code_format",
		"chk_campaigns_plan_slug_not_blank",
		"chk_campaigns_quota",
		"chk_campaigns_reserve",
		"chk_campaigns_window",
	}
)

// TestTheRegistryUpSectionDeclaresExactlyTheRequiredColumnsAndConstraints pins FR-010's
// column list and the five constraint names, and it is EQUALITY rather than containment.
//
// WHY SET EQUALITY AND NOT A LINE MATCH. The migration aligns its columns with whitespace
// (`code        VARCHAR(64)  PRIMARY KEY,`), so anything that compares positions would
// break the day someone realigns the file — and would break on nothing else. The column
// list is therefore extracted and compared as a SET, the way `seedValues` already handles
// the seed tuple.
//
// WHY AN ADDED COLUMN IS A FAILURE AND NOT A PASS. Containment would wave through the
// change FR-091 forbids. A column that is not in `registryRequiredColumns` is either a
// mistake or a decision that belongs in `quickstart.md` with the other nine, and this test
// is what makes that decision get made.
//
// THE FAILURE THIS GUARDS IS SPECIFIC. `campaign_repo_impl.go:176`-`:177` says why the
// `Update` column list is written out by hand, and `listOrdered` selects through
// `entities.Campaign`. If a column were RENAMED in the migration and the entity had not
// caught up, nothing fails to compile: GORM silently omits an unknown field from every
// SELECT and INSERT it builds. That is a registry whose budget column stopped being read
// and no error anywhere — the kind only a live database or a textual assertion catches.
func TestTheRegistryUpSectionDeclaresExactlyTheRequiredColumnsAndConstraints(t *testing.T) {
	body := createTableBody(t, stripSQLComments(registryUpSection(t, registrySchemaMigration)))

	columns, constraints := splitColumnAndConstraintDefinitions(body)

	assertSameNames(t, "FR-010", "columns declared for subscription.campaigns",
		columns, registryRequiredColumns)
	// By name, because a CHECK that loses its name is a CHECK nobody can cite: the
	// trigger's error message, quickstart V3 and every audit conversation about a refused
	// write all name these five, and a renamed constraint is a renamed fact.
	assertSameNames(t, "FR-011/FR-012/FR-013/FR-014",
		"named CHECK constraints declared for subscription.campaigns",
		constraints, registryRequiredConstraints)
}

// TestTheRegistryUpSectionDeclaresTheSortIndexAndTheBudgetTrigger pins the two objects
// whose NAMES are part of the contract rather than implementation detail.
//
// `ix_campaigns_starts_at` is what FR-040's ordering is served from, and
// `trg_campaigns_immutable_budget` is what FR-021/FR-062 rest on. `campaign_guard_integration_test.go`
// looks the trigger up in `pg_trigger` by name, and every behavioural test in that file
// would pass just as happily if the refusals came from something else — the trigger has to
// exist under THAT name, or the constraint that names it means nothing.
func TestTheRegistryUpSectionDeclaresTheSortIndexAndTheBudgetTrigger(t *testing.T) {
	executable := collapseWhitespace(stripSQLComments(registryUpSection(t, registrySchemaMigration)))

	required := []struct {
		what    string
		clause  string
		because string
	}{
		{
			what:    "the sort index",
			clause:  "CREATE INDEX IF NOT EXISTS ix_campaigns_starts_at ON subscription.campaigns (starts_at DESC)",
			because: "FR-040 orders by starts_at DESC, and this is the only index 026 declares",
		},
		{
			what:   "the budget trigger",
			clause: "CREATE TRIGGER trg_campaigns_immutable_budget BEFORE UPDATE ON subscription.campaigns",
			because: "FR-021/FR-062: application discipline does not bind a hand-written UPDATE, so the " +
				"immutability has to be enforced in the data layer",
		},
		{
			what:    "the trigger function",
			clause:  "CREATE OR REPLACE FUNCTION subscription.campaigns_guard_immutable_budget()",
			because: "FR-021: the trigger is only as good as the function it executes",
		},
	}
	for _, item := range required {
		if !strings.Contains(executable, item.clause) {
			t.Errorf("FR-010/FR-021: the schema migration must declare %s — %s. Expected to find:\n  %s\nIn:\n%s",
				item.what, item.because, item.clause, executable)
		}
	}
}

// ---- helpers ----

// registryUpSection returns ONLY the `-- +goose Up` half of the migration whose file name
// contains `nameFragment`.
//
// IT IS NOT `migrationSection(t, name, "Up")`, and the difference is load-bearing for
// FR-091. That helper closes on an empty marker, which means it runs to the END OF FILE —
// it returns the `Down` section too. For the claim-ledger question ("does this Down delete
// anything?") reading past the boundary is harmless. For THIS question ("does the `Up`
// section of 026 alter a table 025 owns?") it is not: the schema migration's own Down
// section drops `subscription.campaigns`, and a helper that leaked it in would one day
// report an `Up` section for containing a DROP it never contained. So this one bounds the
// section at the next marker.
//
// `betweenMarkers` is reused rather than re-implemented; only the bounds differ. The
// helper is not changed at the call site that needs the other bounds because a ticked task
// depends on today's behaviour there — see the report.
func registryUpSection(t *testing.T, nameFragment string) string {
	t.Helper()

	body := embeddedMigrationBody(t, nameFragment)
	section := betweenMarkers(body, "-- +goose Up", "-- +goose Down")
	if strings.TrimSpace(section) == "" {
		t.Fatalf("FR-090: the embedded migration matching %q carries no executable `-- +goose Up` "+
			"section, so it would do nothing", nameFragment)
	}
	return section
}

// embeddedMigrationBody returns the whole text of the embedded migration whose file name
// contains `nameFragment`.
func embeddedMigrationBody(t *testing.T, nameFragment string) string {
	t.Helper()

	entries, err := fs.ReadDir(accountmigrations.EmbedFS, ".")
	if err != nil {
		t.Fatalf("read the embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.Contains(entry.Name(), nameFragment) {
			continue
		}
		body, readErr := fs.ReadFile(accountmigrations.EmbedFS, entry.Name())
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}
		return string(body)
	}
	t.Fatalf("no embedded migration matches %q; the 026 registry migrations must be present", nameFragment)
	return ""
}

// collapseWhitespace reduces every run of whitespace to a single space, so an assertion
// about a multi-line clause survives someone realigning the migration file.
func collapseWhitespace(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

// createTableBody returns what is between the parentheses of the `CREATE TABLE` statement
// in an already comment-free `Up` section.
//
// IT BALANCES PARENTHESES rather than cutting at the first `)`, because the body contains
// `CHECK (code ~ '…')` and a `CHECK` whose expression opens a `btrim(` call — a cut at the
// first `)` would return three columns and look like a migration that lost seven.
//
// IT IS NOT A SQL PARSER. It would be fooled by an unbalanced parenthesis inside a string
// literal; none of the 026 migrations has one, and a body it cannot find is a loud failure
// rather than a silent pass.
func createTableBody(t *testing.T, up string) string {
	t.Helper()

	start := strings.Index(up, "CREATE TABLE")
	if start < 0 {
		t.Fatalf("FR-010: the schema migration must create the registry table")
	}
	open := strings.Index(up[start:], "(")
	if open < 0 {
		t.Fatalf("FR-010: `CREATE TABLE` in the schema migration has no column list")
	}

	depth := 0
	for i := start + open; i < len(up); i++ {
		switch up[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return up[start+open+1 : i]
			}
		}
	}
	t.Fatalf("FR-010: the registry table's column list in the schema migration is never closed")
	return ""
}

// tableDefinitionKeywords open a definition that declares a CONSTRAINT rather than a
// column. `CONSTRAINT` is handled separately because it also carries the name.
var tableDefinitionKeywords = map[string]bool{
	"CHECK":   true,
	"EXCLUDE": true,
	"FOREIGN": true,
	"LIKE":    true,
	"PRIMARY": true,
	"UNIQUE":  true,
}

// splitColumnAndConstraintDefinitions reads a `CREATE TABLE` body and returns the column
// names and the named constraints.
//
// SPLIT ON TOP-LEVEL COMMAS, NOT ON LINES. The body is reformatable — someone may align
// it, wrap it or join two definitions onto one line — and a definition list that survives
// that is the whole point of comparing sets instead of matching lines.
//
// Parentheses and string literals are tracked while splitting, so the comma inside a
// `CHECK` expression or a `'…'` default cannot invent a definition that does not exist.
func splitColumnAndConstraintDefinitions(body string) (columns, constraints []string) {
	for _, definition := range splitTopLevel(body) {
		fields := strings.Fields(definition)
		if len(fields) == 0 {
			continue
		}
		head := strings.ToUpper(strings.TrimSuffix(fields[0], ","))
		if head == "CONSTRAINT" {
			if len(fields) > 1 {
				constraints = append(constraints, strings.ToLower(fields[1]))
			}
			continue
		}
		if tableDefinitionKeywords[head] {
			continue
		}
		columns = append(columns, strings.ToLower(strings.TrimSuffix(fields[0], ",")))
	}
	return columns, constraints
}

// splitTopLevel splits a table body on the commas that separate definitions, skipping the
// ones nested in parentheses or inside a `'…'` literal. A doubled quote is an escaped
// quote inside a literal.
func splitTopLevel(body string) []string {
	var (
		definitions []string
		current     strings.Builder
		depth       int
		inLiteral   bool
	)
	for i := 0; i < len(body); i++ {
		char := body[i]
		switch {
		case char == '\'':
			// A literal opens, closes, or escapes itself with a doubled quote. Deciding
			// which needs the NEXT character, so the lookahead is bounded and never reads
			// past the end.
			if inLiteral && i+1 < len(body) && body[i+1] == '\'' {
				current.WriteByte(char)
				i++
				current.WriteByte(body[i])
				continue
			}
			inLiteral = !inLiteral
		case inLiteral:
			// Punctuation inside a literal is data, not structure.
		case char == '(':
			depth++
		case char == ')':
			depth--
		case char == ',' && depth == 0:
			definitions = append(definitions, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(char)
	}
	if strings.TrimSpace(current.String()) != "" {
		definitions = append(definitions, current.String())
	}
	return definitions
}

// 026 phase 11, found by reviewing the Phase 11 helpers (FR-090, US6.2).
//
// THE BUG THIS PINS. `migrationSection(t, name, "Up")` used to pass an empty closing
// marker to `betweenMarkers`, so an `Up` section ran to end-of-file and swallowed the
// `Down` body. For the seed migration that body is
// `DELETE FROM subscription.campaigns WHERE code = 'launch-2026-10'`, and
// `TestIntegrationSeedInsertIsIdempotent` executes the string it is given. It would
// therefore have inserted the row, deleted it again with the section it did not mean to
// include, counted zero, and failed on an assertion about idempotency — a test that
// cannot pass on any database, planted by a helper that looked correct.
//
// WHY A TEXTUAL TEST IS THE RIGHT SHAPE. The defect is entirely in text extraction, so
// it is fully decidable without a database; no `openRegistryTestDB`, no `t.Skip`, and
// therefore no chance of it surviving unnoticed for the same reason D-01 describes.
func TestTheUpSectionStopsAtTheDownMarker(t *testing.T) {
	up, ok := migrationSection(t, registrySeedMigration, "Up")
	if !ok {
		return
	}
	stripped := stripSQLComments(up)

	// The specific statement the seeded `Down` would have leaked in.
	if strings.Contains(stripped, "DELETE") {
		t.Fatalf("FR-090: the `Up` section must stop at the `Down` marker, but it carries "+
			"a DELETE with it, so any caller that executes this string also rolls the row back:\n%s", stripped)
	}

	// The general form of the same mistake: nothing from the `Down` half may appear.
	down, downOK := migrationSection(t, registrySeedMigration, "Down")
	if !downOK {
		return
	}
	for _, line := range strings.Split(stripSQLComments(down), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(up, trimmed) {
			t.Errorf("FR-090: statement %q belongs to the `Down` section but leaked into `Up`", trimmed)
		}
	}

	// And the section must still carry the statement it exists for, or the check above
	// would pass on an empty string.
	if !strings.Contains(stripped, "INSERT") {
		t.Errorf("FR-090: the `Up` section lost its INSERT, so the check above is vacuous:\n%s", stripped)
	}
}
