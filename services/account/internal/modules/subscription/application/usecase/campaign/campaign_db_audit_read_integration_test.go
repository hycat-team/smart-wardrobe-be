package campaign

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	admincontract "smart-wardrobe-be/services/account/contract/admin"
	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	admindto "smart-wardrobe-be/services/account/dto/admin"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 026 T072 (quickstart V7, spec.md §11.8, FR-060..FR-063) — the campaign lifecycle and the
// audit trail a reader then sees, against a REAL Postgres.
//
// WHAT THIS FILE IS, AND WHY IT IS BUILT THE WAY IT IS. It is the end-to-end reading of
// US5: create → edit a time field → close, then read the trail for that one code. The
// registry side is entirely real — the real repository, the real unit of work, the real
// transactions — and the trail rows are real rows in `shared.admin_audit_logs`, because
// FR-060's whole point is that the trail is data.
//
// THE ASYMMETRY IS DELIBERATE AND WORTH NAMING. In production the trail reaches that table
// through the OUTBOX and the existing consumer, asynchronously, so a read immediately after
// a write may legitimately show nothing — FR-060 and SC-006 make that delay part of the
// contract. This file stands that consumer in: the audit contract handed to the registry
// writer PERSISTS what it is given, which is exactly what
// `admin/application/usecase.AuditLogUseCase.Record` does when the consumer runs. That is
// the ONLY substitution, and it is what makes everything above it — the payloads, the
// filter, the mapping, the `null` on the wire — real rather than faked.
//
// WHY THE READ SIDE RESTATES THE ADMIN MODULE'S QUERY. The `subscription` module may hold
// the narrow `admincontract.ICampaignAuditLookup` and nothing else: reaching the admin
// module's repository or entity from here would be the cross-module import constitution §II
// forbids, and the admin module's own half of this story is tested where it belongs, in
// `admin_audit_log_campaign_test.go` (T070), against the real repository — including a
// capture of the SQL it builds. So this file implements the CONTRACT, with the same
// two-term predicate, and everything above it is the production code.

// dbAuditPrefix namespaces everything this file writes, so a run against a shared test
// database can collide with nothing and delete nothing belonging to another suite.
const dbAuditPrefix = "verify-026-audit-"

// openAuditTestDB connects, migrates and registers the cleanup for BOTH tables this file
// touches — the campaigns AND the audit rows.
//
// The audit rows are cleaned up by prefix rather than by id because a failed test may leave
// rows whose ids the test never learned, and `shared.admin_audit_logs` has no cascade from
// anything. Leaving them would make a later run's totals wrong and would fill the table
// with rows no campaign refers to.
func openAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping campaign audit read integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// The migrations, not `AutoMigrate`: `payload` is a `jsonb` column and the whole
	// selector is written against that type, which a schema built from Go structs would not
	// be.
	if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS,
		"goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() {
		dbConn.Exec("DELETE FROM subscription.campaign_claims WHERE campaign_code LIKE ?", dbAuditPrefix+"%")
		dbConn.Exec("DELETE FROM subscription.campaigns WHERE code LIKE ?", dbAuditPrefix+"%")
		dbConn.Exec(`DELETE FROM shared.admin_audit_logs
			WHERE target_type = ? AND payload->>'campaignCode' LIKE ?`,
			string(adminaudit.Campaign), dbAuditPrefix+"%")
	})
	return dbConn
}

// auditTrailRow is this file's VIEW of `shared.admin_audit_logs`, and it is declared here
// rather than imported for a reason worth stating plainly.
//
// `entities.AdminAuditLog` belongs to the `admin` module, and importing it from
// `subscription` would be exactly the cross-module reach constitution §II forbids — the
// same rule T070's boundary scan enforces from the other side. So the test declares its own
// mapping and moves the rows with raw SQL. That is a little more code and it is the honest
// shape: the production path crosses this boundary through the narrow CONTRACT, and the
// test's path crosses it the same way, with the module's own model left where it belongs.
type auditTrailRow struct {
	ID          uuid.UUID
	ActorUserID uuid.UUID
	Action      string
	TargetType  string
	TargetID    *uuid.UUID
	// Payload is the raw `jsonb`. It stays bytes here because that is what a jsonb column
	// hands a driver, and decoding it is this file's job rather than a GORM type's.
	Payload   []byte
	RequestIP *string
	CreatedAt time.Time
}

// persistingAuditContract writes each audit command straight into `shared.admin_audit_logs`.
//
// IT STANDS IN FOR THE OUTBOX CONSUMER, and everything it does is what that consumer does:
// build a row from the command and insert it. The only thing it skips is the gate-2
// de-duplication, which no campaign action takes.
//
// `target_id` IS WRITTEN AS IT ARRIVED, NULL included, because that is the invariant the
// whole endpoint exists to show: a campaign row has no target id (FR-063).
type persistingAuditContract struct {
	db *gorm.DB
}

// Record writes one audit row and returns the insert's error, so a failing write rolls the
// caller's transaction back exactly as the real outbox does (FR-060/FR-061).
func (c *persistingAuditContract) Record(ctx context.Context, input admindto.RecordAuditLogInput) error {
	payload := input.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	createdAt := time.Now()
	return c.db.WithContext(ctx).Exec(
		`INSERT INTO shared.admin_audit_logs
		   (id, actor_user_id, action, target_type, target_id, payload, request_ip, created_at)
		 VALUES (?, ?, ?, ?, ?, ?::jsonb, ?, ?)`,
		uuid.New(), input.ActorUserID, input.Action, input.TargetType, input.TargetID,
		string(encoded), input.RequestIP, createdAt,
	).Error
}

var _ admincontract.IAuditLogContract = (*persistingAuditContract)(nil)

// dbCampaignAuditLookup implements the narrow contract over a real database.
//
// THE PREDICATE IS THE ADMIN MODULE'S, RESTATED: `target_type = 'campaign'` AND
// `payload->>'campaignCode' = ?`. Both terms, for the reasons T070 gives — one alone would
// return another target type's row carrying the same key, or every campaign's trail.
type dbCampaignAuditLookup struct {
	db *gorm.DB
}

// trailSelect is the one statement, so the total and the page cannot disagree about which
// rows they are counting. `ORDER BY created_at DESC, id DESC` is newest first with a TOTAL
// order, exactly as the admin repository orders it; T070's ordering test is what makes the
// second key a requirement rather than a preference.
const trailSelect = `SELECT id, actor_user_id, action, target_type, target_id, payload, request_ip, created_at
	FROM shared.admin_audit_logs
	WHERE target_type = ? AND payload->>'campaignCode' = ?`

func (l *dbCampaignAuditLookup) ListByCampaignCode(
	ctx context.Context,
	campaignCode string,
	page, limit int,
) ([]*admindto.AdminAuditLogDTO, int64, error) {
	scoped := l.db.WithContext(ctx).
		Where("target_type = ?", string(adminaudit.Campaign)).
		Where("payload->>'campaignCode' = ?", campaignCode)

	var total int64
	if err := scoped.Model(&auditTrailRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}

	var rows []auditTrailRow
	err := scoped.Raw(trailSelect+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		string(adminaudit.Campaign), campaignCode, limit, (page-1)*limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]*admindto.AdminAuditLogDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, &admindto.AdminAuditLogDTO{
			ID:          row.ID,
			ActorUserID: row.ActorUserID,
			Action:      row.Action,
			TargetType:  row.TargetType,
			TargetID:    row.TargetID,
			Payload:     decodeAuditPayload(row.Payload),
			RequestIP:   row.RequestIP,
			CreatedAt:   row.CreatedAt,
		})
	}
	return items, total, nil
}

// decodeAuditPayload turns the stored `jsonb` back into a document.
//
// A MALFORMED DOCUMENT DECODES TO AN EMPTY ONE rather than failing the read, because this
// is a test double for a query whose real implementation hands the column to its own
// mapper: the column is written by this same file, so a decode failure here would be a bug
// in the test rather than a fact about the trail.
func decodeAuditPayload(raw []byte) map[string]any {
	payload := map[string]any{}
	if len(raw) == 0 {
		return payload
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

var _ admincontract.ICampaignAuditLookup = (*dbCampaignAuditLookup)(nil)

// countingCampaignAuditLookup counts how often the trail is consulted.
//
// IT WRAPS the real implementation rather than replacing it, so the count is of a genuine
// query and observing it changes nothing about what would have been asked.
type countingCampaignAuditLookup struct {
	inner admincontract.ICampaignAuditLookup
	calls int
}

func (l *countingCampaignAuditLookup) ListByCampaignCode(
	ctx context.Context,
	campaignCode string,
	page, limit int,
) ([]*admindto.AdminAuditLogDTO, int64, error) {
	l.calls++
	return l.inner.ListByCampaignCode(ctx, campaignCode, page, limit)
}

// auditFixture is the wired pair a test drives: the registry writer, the look-up, the real
// registry repository and the administrator whose id every audit row must carry.
type auditFixture struct {
	registry *CampaignRegistryUseCase
	lookup   *CampaignQueryUseCase
	campaign repositories.ICampaignRepository
	actor    uuid.UUID
	// counting wraps the trail the look-up holds, so a test can assert that the trail was
	// never consulted.
	counting *countingCampaignAuditLookup
}

// newAuditFixture wires the REAL registry writer and the REAL look-up over one database.
//
// The collaborators that are NOT real are the ones whose subject is not this test: the
// claim ledger, the sweep table, the plan catalogue and the subscription row are the
// package's existing fakes, because reading the audit trail needs none of them; and the
// account contract is a fake for the same reason. The registry, the transactions and the
// audit table are all real, because every claim in this file is a statement about them.
func newAuditFixture(t *testing.T, dbConn *gorm.DB) *auditFixture {
	t.Helper()

	actor := insertAuditActorAccount(t, dbConn)
	campaignRepo := persistence.NewCampaignRepository(dbConn)
	uow := shareddb.NewGormUnitOfWork(dbConn)
	log := &recordingLogger{}

	registry := NewCampaignRegistryUseCase(
		testConfig(), campaignRepo, &persistingAuditContract{db: dbConn}, uow, log,
	).(*CampaignRegistryUseCase)
	// The clock is pinned for the same reason it is everywhere in this package: a window
	// that opens in the PAST cannot be tested against a clock that moves.
	registry.now = func() time.Time { return baseInstant() }

	counting := &countingCampaignAuditLookup{inner: &dbCampaignAuditLookup{db: dbConn}}
	lookup := NewCampaignQueryUseCase(
		testConfig(),
		campaignRepo,
		newFakeClaimRepo(),
		newFakeSweepRunRepo(),
		newFakePlanRepo(),
		newFakeSubRepo(),
		newFakeUserBriefContract(),
		counting,
		log,
	).(*CampaignQueryUseCase)
	lookup.now = func() time.Time { return baseInstant() }

	return &auditFixture{
		registry: registry,
		lookup:   lookup,
		campaign: campaignRepo,
		actor:    actor,
		counting: counting,
	}
}

// insertAuditActorAccount writes the `identity.users` row the audit trail's actor column
// requires, and removes it afterwards.
//
// IT IS A RAW INSERT FOR THE SAME REASON the other integration suites in this package do
// it: this module does not own `identity.users`, and borrowing another module's repository
// to create the row would be the cross-module dependency this feature refuses elsewhere.
func insertAuditActorAccount(t *testing.T, dbConn *gorm.DB) uuid.UUID {
	t.Helper()
	actor := uuid.New()
	handle := dbAuditPrefix + "admin-" + actor.String()
	err := dbConn.Exec(
		`INSERT INTO identity.users (id, username, email, role_slug, status, is_deleted, provider, email_verified, created_at)
		 VALUES (?, ?, ?, 'admin', 0, false, 'local', true, ?)`,
		actor, handle, handle+"@example.test", baseInstant(),
	).Error
	if err != nil {
		t.Fatalf("insert the administrator account: %v", err)
	}
	t.Cleanup(func() {
		dbConn.Exec("DELETE FROM identity.users WHERE id = ?", actor)
	})
	return actor
}

// openAuditCampaign creates a campaign through the registry's own use case — the only way
// a campaign is written in operation (FR-005, FR-062) — with a window already open.
func openAuditCampaign(t *testing.T, f *auditFixture, code string) *sub_dto.CampaignSummaryRes {
	t.Helper()
	planSlug := "premium-monthly"
	quota := 10
	summary, err := f.registry.Create(context.Background(), f.actor, &sub_dto.CampaignCreateReq{
		Code:     code,
		PlanSlug: &planSlug,
		Quota:    &quota,
		StartsAt: baseInstant().Add(-24 * time.Hour).Format(time.RFC3339),
	}, nil)
	if err != nil {
		t.Fatalf("create the campaign %s: %v", code, err)
	}
	return summary
}

// TestIntegrationTheCampaignTrailRecordsTheWholeLifecycle is quickstart V7 in full.
//
// THE THREE OPERATIONS, IN THE ORDER V7 PERFORMS THEM, then the read. Every assertion
// below is something contract §6.3 promises:
//
//   - `campaign.create`, `campaign.update`, `campaign.close` — one row each, so a reader
//     can tell which operation produced a line;
//   - `targetType = "campaign"` and `targetId = NULL` on every row — the code is a string
//     and the audit filter takes a UUID (FR-063);
//   - `payload.campaignCode` on every row — the only thing that makes this endpoint able to
//     find them at all;
//   - `before`/`after` on the edit, `reason` on the closure (contract §6.3).
//
// `targetId` IS ASSERTED NIL AND NOT MERELY UNSET, twice: once on the DTO and once on the
// SERIALISED RESPONSE. The second is the one that matters, because `omitempty` would
// satisfy the first and fail the second — and a client reading this trail is exactly where
// that difference would show.
func TestIntegrationTheCampaignTrailRecordsTheWholeLifecycle(t *testing.T) {
	dbConn := openAuditTestDB(t)
	f := newAuditFixture(t, dbConn)
	ctx := context.Background()

	campaignCode := dbAuditPrefix + "lifecycle"
	const reason = "Hết ngân sách khuyến mãi tháng 11"

	// STEP 1 — open.
	created := openAuditCampaign(t, f, campaignCode)

	// STEP 2 — edit a TIME FIELD, with the version the create returned. That is what an
	// operator's screen does, and it is what FR-055's If-Match is for.
	endsAt := baseInstant().AddDate(0, 1, 0)
	edited, err := f.registry.Update(ctx, f.actor, campaignCode,
		decodeUpdateReq(t, `{"version":`+strconv.Itoa(created.Version)+
			`,"endsAt":"`+endsAt.Format(time.RFC3339)+`"}`), nil)
	if err != nil {
		t.Fatalf("quickstart V7 step 2 (edit endsAt): %v", err)
	}

	// STEP 3 — close, with the version the EDIT returned.
	closeReason := reason
	if _, err := f.registry.Close(ctx, f.actor, campaignCode, &sub_dto.CampaignCloseReq{
		Version: edited.Version,
		Reason:  &closeReason,
	}, nil); err != nil {
		t.Fatalf("quickstart V7 step 3 (close): %v", err)
	}

	// STEP 4 — WAIT FOR THE CONSUMER, then read. Here the consumer is synchronous, so there
	// is nothing to wait for: each row was written inside its own operation's transaction.
	items, total, err := f.lookup.ListCampaignAudit(ctx, campaignCode, 1, 20)
	if err != nil {
		t.Fatalf("read the campaign trail: %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("quickstart V7 step 5: expected exactly ONE row per operation (3), got a total of %d "+
			"and %d rows on the page", total, len(items))
	}

	// THE THREE ACTIONS AS A SET, because the ORDER is the subject of the next assertion and
	// of T070's; sorting here keeps this one about WHICH operations were recorded.
	if got, want := sortedActions(items),
		[]string{string(adminaudit.CampaignClose), string(adminaudit.CampaignCreate), string(adminaudit.CampaignUpdate)}; !equalStrings(got, want) {
		t.Errorf("quickstart V7: recorded actions = %v, want %v", got, want)
	}

	for _, item := range items {
		// FR-063, on every row.
		if item.TargetType != string(adminaudit.Campaign) {
			t.Errorf("FR-063: targetType = %q, want %q", item.TargetType, adminaudit.Campaign)
		}
		if item.TargetID != nil {
			t.Errorf("FR-063: targetId must be NULL for every campaign row, got %v", *item.TargetID)
		}
		if got, _ := item.Payload["campaignCode"].(string); got != campaignCode {
			t.Errorf("FR-063: payload.campaignCode = %q, want %q", got, campaignCode)
		}
		if item.ActorUserID != f.actor {
			t.Errorf("FR-060: actorUserId = %s, want the administrator %s", item.ActorUserID, f.actor)
		}
	}

	// THE NEWEST ROW IS THE CLOSURE, which is the question an operator asks first.
	if items[0].Action != string(adminaudit.CampaignClose) {
		t.Errorf("the trail must be newest first, got %q first", items[0].Action)
	}
	// `after.closedAt` is a timestamp, so it is compared for PRESENCE and shape only; the
	// empty-string comparison is used for the fields whose absence is meaningful.
	assertPayloadSide(t, items[0].Payload, "after", "closedAt", endsAtUTC(baseInstant()))
	if got, _ := items[0].Payload["reason"].(string); got != reason {
		t.Errorf("contract §6.3: the closure must carry the reason that was sent, got %q", got)
	}

	// THE EDIT CARRIES OLD → NEW, and the closure's reason must not have leaked onto it.
	var update *sub_dto.CampaignAuditLogRes
	for _, item := range items {
		if item.Action == string(adminaudit.CampaignUpdate) {
			update = item
		}
	}
	if update == nil {
		t.Fatal("the edit produced no audit row")
	}
	// `before.endsAt` is EMPTY because the campaign was open-ended, and the key is PRESENT
	// anyway — which is the whole argument for rendering an absent window as `""` rather
	// than dropping it: a reader never has to tell "absent" from "empty".
	assertPayloadSide(t, update.Payload, "before", fieldEndsAt, "")
	assertPayloadSide(t, update.Payload, "after", fieldEndsAt, endsAtUTC(endsAt))
	if _, present := update.Payload["reason"]; present {
		t.Errorf("contract §6.3: a closure reason belongs to the close alone, got %#v", update.Payload)
	}

	// AND THE SERIALISED RESPONSE EMITS `targetId: null`.
	//
	// Asserted on the raw JSON because decoding it back into the DTO would prove nothing:
	// Go fills a missing key and an explicit null into the same nil pointer, which is
	// precisely the ambiguity `CampaignAuditLogRes` exists to remove. A client reading this
	// endpoint is where the difference is observable, so that is where it is checked.
	encoded, err := json.Marshal(map[string]any{
		"items": mapper.MapCampaignAuditLogResList(toAdminDTOs(items)),
	})
	if err != nil {
		t.Fatalf("marshal the trail as the handler would: %v", err)
	}
	if !strings.Contains(string(encoded), `"targetId":null`) {
		t.Errorf("FR-063/contract §6.2: the response must EMIT targetId as null, not omit it: %s", encoded)
	}
}

// TestIntegrationAnUnknownCampaignCodeNeverAnswersAnEmptyTrail is FR-072 and contract §6.4,
// and it is the ORDERING assertion the handler test cannot make.
//
// The handler test stands in for the whole USE CASE, so it can only report that the stub
// was consulted once while producing the 404. Here the real `CampaignQueryUseCase` runs over
// the real registry, so `f.counting.calls` answers exactly the question that matters: an
// unknown code must produce 404 with the trail NEVER CONSULTED. A use case that asked the
// trail first would answer with an empty page and a total of zero — indistinguishable from
// "nobody has ever touched this campaign", which is what sends an operator to report a
// broken audit mechanism instead of their own typo.
func TestIntegrationAnUnknownCampaignCodeNeverAnswersAnEmptyTrail(t *testing.T) {
	dbConn := openAuditTestDB(t)
	f := newAuditFixture(t, dbConn)

	const unknownCode = dbAuditPrefix + "khong-ton-tai"
	items, total, err := f.lookup.ListCampaignAudit(context.Background(), unknownCode, 1, 20)
	if err == nil {
		t.Fatal("an unknown campaign code must be refused, got an empty trail")
	}
	if status := httpStatusOf(t, err); status != 404 {
		t.Errorf("FR-054/contract §6.5: an unknown code must be 404, got %d (%v)", status, err)
	}
	if items != nil || total != 0 {
		t.Errorf("a refused read must return nothing, got %d items and a total of %d", len(items), total)
	}
	if f.counting.calls != 0 {
		t.Errorf("contract §6.4: the trail must not be consulted for a code that does not exist, got "+
			"%d calls — that ordering is the whole difference between a 404 and a misleading empty page",
			f.counting.calls)
	}
	// And the message names the code, in Vietnamese: an operator who mistyped it has to know
	// what to correct (constitution: client-facing text is Vietnamese).
	if !strings.Contains(err.Error(), unknownCode) {
		t.Errorf("the 404 must name the code that was asked for, got %q", err.Error())
	}
}

// TestIntegrationTheTrailIsInvisibleToAnotherCode is FR-063's isolation, asserted from the
// CONSUMER side.
//
// The code lives in a `jsonb` payload rather than a typed column, so the filter has to
// exclude other campaigns' rows BY VALUE. Two campaigns written in the same test are the
// only way to show it: a filter that ignored the value would return two rows per campaign
// here, and one that ignored `target_type` would pull in rows from other features.
func TestIntegrationTheTrailIsInvisibleToAnotherCode(t *testing.T) {
	dbConn := openAuditTestDB(t)
	f := newAuditFixture(t, dbConn)
	ctx := context.Background()

	codes := []string{dbAuditPrefix + "one", dbAuditPrefix + "two"}
	for _, code := range codes {
		openAuditCampaign(t, f, code)
	}

	for _, code := range codes {
		items, total, err := f.lookup.ListCampaignAudit(ctx, code, 1, 20)
		if err != nil {
			t.Fatalf("read the trail of %s: %v", code, err)
		}
		if total != 1 || len(items) != 1 {
			t.Fatalf("FR-063: %s must see exactly its own one row, got %d rows and a total of %d",
				code, len(items), total)
		}
		if got, _ := items[0].Payload["campaignCode"].(string); got != code {
			t.Errorf("FR-063: %s's trail contains a row for %q", code, got)
		}
	}
}

// TestIntegrationAFailedWriteLeavesNoTrailEntry is FR-061 on a real database, and it is
// the last of quickstart V7's conditions rather than an extra.
//
// THE FAILURE IS FORCED BY A STALE VERSION, so the operation is refused before the audit
// row is ever written. That is a WEAKER statement than the unit test's — that one makes the
// audit FAIL after recording the command, so the ROLLBACK is what removes the row. What
// this adds is the other half: a refused operation does not even reach the audit table, so
// an operator's trail never shows an attempt that changed nothing.
func TestIntegrationAFailedWriteLeavesNoTrailEntry(t *testing.T) {
	dbConn := openAuditTestDB(t)
	f := newAuditFixture(t, dbConn)
	ctx := context.Background()

	campaignCode := dbAuditPrefix + "refused"
	openAuditCampaign(t, f, campaignCode)

	// A version no row can hold. The campaign exists, so this is 412 and not 404.
	reason := "Kiểm chứng 026"
	_, err := f.registry.Close(ctx, f.actor, campaignCode,
		&sub_dto.CampaignCloseReq{Version: 99, Reason: &reason}, nil)
	if err == nil {
		t.Fatal("a stale version must be refused")
	}
	if status := httpStatusOf(t, err); status != 412 {
		t.Errorf("FR-055: a stale version must be 412, got %d (%v)", status, err)
	}

	items, total, err := f.lookup.ListCampaignAudit(ctx, campaignCode, 1, 20)
	if err != nil {
		t.Fatalf("read the trail after a refused operation: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("FR-061: only the successful create may appear, got %d rows and a total of %d",
			len(items), total)
	}
	if items[0].Action != string(adminaudit.CampaignCreate) {
		t.Errorf("FR-061: the refused close must leave no row, got %q", items[0].Action)
	}
}

// ---- helpers ----

// sortedActions lists the trail's actions alphabetically, so an assertion about WHICH
// operations were recorded does not depend on the ORDER the endpoint returned them in —
// the ordering is asserted separately and behaviourally (T070).
func sortedActions(items []*sub_dto.CampaignAuditLogRes) []string {
	actions := make([]string, 0, len(items))
	for _, item := range items {
		actions = append(actions, item.Action)
	}
	sort.Strings(actions)
	return actions
}

// equalStrings compares two string slices.
//
// IT EXISTS TO KEEP THE ASSERTION READABLE: `reflect.DeepEqual` would work, but the failure
// message would print the two slices with no indication of which was which, and the whole
// point of the comparison is that the reader can see WHICH action was missing.
func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// toAdminDTOs converts the response rows back into the cross-module DTO the mapper takes,
// so the serialisation assertion goes through the SAME mapper the use case uses rather than
// re-implementing the mapping in the test.
func toAdminDTOs(items []*sub_dto.CampaignAuditLogRes) []*admindto.AdminAuditLogDTO {
	out := make([]*admindto.AdminAuditLogDTO, 0, len(items))
	for _, item := range items {
		out = append(out, &admindto.AdminAuditLogDTO{
			ID:          item.ID,
			ActorUserID: item.ActorUserID,
			Action:      item.Action,
			TargetType:  item.TargetType,
			TargetID:    item.TargetID,
			Payload:     item.Payload,
			RequestIP:   item.RequestIP,
			CreatedAt:   item.CreatedAt,
		})
	}
	return out
}

// endsAtUTC renders an instant the way the audit payloads do (UTC RFC3339), for the reason
// `endsAtForAudit` gives: a payload is read weeks later and compared against another row, so
// a value keeping the writer's offset would render differently per administrator.
func endsAtUTC(at time.Time) string { return at.UTC().Format(time.RFC3339) }

// assertPayloadSide reads one `before`/`after` sub-document and one key out of it.
func assertPayloadSide(t *testing.T, payload map[string]any, side, key, want string) {
	t.Helper()
	raw, ok := payload[side]
	if !ok {
		t.Fatalf("payload.%s must be present, got %#v", side, payload)
	}
	values, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("payload.%s must be an object, got %#v", side, raw)
	}
	got, present := values[key]
	if !present {
		t.Fatalf("payload.%s.%s must be present, got %#v", side, key, values)
	}
	text, isText := got.(string)
	if !isText {
		t.Fatalf("payload.%s.%s = %#v, want a string", side, key, got)
	}
	if text != want {
		t.Errorf("payload.%s.%s = %q, want %q", side, key, text, want)
	}
}
