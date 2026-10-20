package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/messaging"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- 024 T198: the eager provisioning door, proven against a real Postgres ----
//
// Everything that protects eager provisioning's non-duplication property is
// enforced by the DATABASE, not by the use case:
//
//   - `user_subscriptions.user_id` is the primary key, and `ProvisionDefault` is
//     `ON CONFLICT (user_id) DO NOTHING` (user_subscription_repo_impl.go:135).
//   - `user_subscription_events.event_key` is UNIQUE, and `CreateIfAbsent` refuses
//     the loser instead of failing it (subscription_audit_repo_impl.go:60).
//   - `user_subscriptions.user_id` is a FOREIGN KEY onto `identity.users(id)`.
//
// A fake that models those three mechanisms models them by hand, which means
// removing the `clause.OnConflict` clause from the repository, or the UNIQUE index
// from the schema, leaves every existing assertion green. These tests close that
// hole: they read the constraints off a real database, so the mechanisms under test
// are the ones production runs on.

// eagerUserFK is the constraint name used here for the
// `user_subscriptions.user_id -> identity.users(id)` foreign key.
//
// It is added and dropped inside a single test rather than left in place. The other
// integration tests in this package seed `user_subscriptions` rows for accounts
// that never existed, which a permanent constraint would reject for a reason
// unrelated to what they are testing.
//
// 024 T211: it is now LOAD-BEARING. It used to be installed and never asserted
// against - the accounts these tests create are committed (see `newAccount`), so
// every `INSERT user_subscriptions` satisfied it and deleting the whole helper
// left both tests green. DDL with no assertion is not a fixture, it is residue.
// `TestIntegrationEagerUserForeignKeyRejectsAnUnknownAccount` is what gives it a
// job, and therefore what gives the argument at the top of this file its
// protection.
const eagerUserFK = "fk_user_subscriptions_user_eager_provision_test"

// eagerAccountTable is the parent of that foreign key.
//
// Production declares it as `identity.users`
// (`database/scripts/closy_db-schema.sql:818`) and points the constraint at it
// with `ON DELETE CASCADE` (:1329). The table below reproduces that definition
// column for column rather than inventing a shape of its own: the point of the
// assertion is that a row whose account is not visible to the writing transaction
// is refused, and that behaviour is the constraint's, not the parent's.
//
// It is created only when it is ABSENT, and dropped only when this test created
// it. A `TEST_DATABASE_URL` pointing at an already-migrated database therefore
// reuses the real `identity.users` and never removes it - and, because the insert
// below names only columns this definition and production both make NOT NULL
// (`id`, `username`, `email`, `role_slug`, everything else defaulted), that reuse
// cannot hit a missing column. A table that genuinely lacked one of those four
// fails the test loudly with the database's own error instead of being papered
// over.
const eagerAccountTable = "identity.users"

// eagerAccountMarker prefixes every row these tests insert, so cleanup can delete
// exactly what was created without touching anything else in the database. It is
// also what keeps the insert valid against a migrated `identity.users`, which
// carries unique indexes on `lower(email)` and `lower(username)`.
const eagerAccountMarker = "eager-t198-"

// TestIntegrationEagerProvisioningRollsBackWithTheCallersTransaction is the
// argument T196 rests on, and it cannot be made with a fake at all.
//
// `user_subscriptions.user_id` is a foreign key onto `identity.users(id)`, and the
// account row only exists inside the transaction that created it. That is why
// `EnsureDefaultSubscription` runs on the CALLER's transaction instead of opening
// one of its own: a second `uow.Execute` would run on a different connection, where
// the uncommitted account does not exist.
//
// This test proves it behaviourally rather than by argument. Provision inside a
// real transaction, roll it back, and assert that not one of the three rows
// survives. The counterfactual at the end is what makes it a real proof: the very
// same call on a bare context commits all three, so "nothing survived" cannot be an
// artefact of the fixture writing nothing in the first place.
func TestIntegrationEagerProvisioningRollsBackWithTheCallersTransaction(t *testing.T) {
	f := newEagerProvisionFixture(t)
	uow := db.NewGormUnitOfWork(f.dbConn)
	ctx := context.Background()

	// Positive control. Without it a passing rollback assertion is worth nothing:
	// an empty fixture satisfies "no rows survived" trivially. The control account
	// is a DIFFERENT account from the one below, because provisioning is a no-op
	// for an account that already has a subscription.
	committed := f.newAccount(t)
	if err := uow.Execute(ctx, func(txCtx context.Context) error {
		return f.support.EnsureDefaultSubscription(txCtx, committed)
	}); err != nil {
		t.Fatalf("provisioning inside a transaction failed: %v", err)
	}
	if got := f.countRows(t, committed); got != 1 {
		t.Fatalf("the committed control account must have exactly 1 subscription row, got %d - the fixture is not proving anything", got)
	}
	if got := f.countEvents(t, committed); got != 1 {
		t.Fatalf("the committed control account must have exactly 1 history row, got %d", got)
	}
	if got := f.countOutbox(t); got != 1 {
		t.Fatalf("the committed control account must have produced exactly 1 outbox record, got %d", got)
	}

	// The subject: same flow, then the caller fails and the transaction rolls back.
	rolledBack := f.newAccount(t)
	callerFailure := errors.New("caller failed after provisioning")
	err := uow.Execute(ctx, func(txCtx context.Context) error {
		if err := f.support.EnsureDefaultSubscription(txCtx, rolledBack); err != nil {
			return err
		}
		return callerFailure
	})
	if !errors.Is(err, callerFailure) {
		t.Fatalf("the unit of work must surface the caller's own error, got %v", err)
	}

	if got := f.countRows(t, rolledBack); got != 0 {
		t.Errorf("an ORPHAN subscription row survived the rollback (%d rows): the writes did not join the caller's transaction", got)
	}
	if got := f.countEvents(t, rolledBack); got != 0 {
		t.Errorf("an orphan history row survived the rollback (%d rows)", got)
	}
	if got := f.countOutbox(t); got != 1 {
		t.Errorf("expected the outbox to still hold only the control account's record, got %d", got)
	}

	// The counterfactual that gives the assertions above their meaning: on a bare
	// context every write falls back to the base connection and commits on its own,
	// so the rows DO stay. This is the exact failure 024 T126 removed on the lazy
	// door, and it is what the rollback assertions detect.
	//
	// It runs through `SubscriptionStateSupport` directly, not through the module
	// contract, because the contract now refuses a transaction-less context by
	// design (024 T202) - and this test needs to reach the body to show what the
	// guard is protecting against.
	untransacted := f.newAccount(t)
	if err := f.support.EnsureDefaultSubscription(ctx, untransacted); err != nil {
		t.Fatalf("untransacted provisioning failed: %v", err)
	}
	if got := f.countRows(t, untransacted); got != 1 {
		t.Fatalf("a bare context must still commit, got %d subscription rows - if this fails the rollback assertions above prove nothing", got)
	}
}

// TestIntegrationConcurrentEagerAndLazyProvisioningCreateOneRecord is the race the
// two doors actually have.
//
// The window is a real lock, not a scheduling coincidence: while the eager
// transaction is still open, its `user_subscriptions` row is invisible to everyone
// else, so a lazy `GetOrCreateUserSubscription` reads "absent" and then blocks
// inside `INSERT ... ON CONFLICT DO NOTHING` until the eager transaction commits.
// A fake has no unique index and no uncommitted row, so it cannot produce this at
// all.
//
// Twelve goroutines - half the eager door, half the lazy one - all racing for one
// brand-new account must end with exactly one subscription row, one history row and
// one outbox record. The outbox count is the assertion that carries the most
// weight: refusing the duplicate INSERT is only half the guarantee, because if the
// loser published anyway the AI service would hear about one grant twice.
func TestIntegrationConcurrentEagerAndLazyProvisioningCreateOneRecord(t *testing.T) {
	f := newEagerProvisionFixture(t)
	userID := f.newAccount(t)

	type result struct {
		sub *entities.UserSubscription
		err error
	}
	results := make([]result, concurrencyRacers)
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < concurrencyRacers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			if idx%2 == 0 {
				// The eager door, exactly as the three account-creation doors call
				// it: inside a transaction the caller opened.
				results[idx].err = db.NewGormUnitOfWork(f.dbConn).Execute(context.Background(),
					func(txCtx context.Context) error {
						return f.support.EnsureDefaultSubscription(txCtx, userID)
					})
				return
			}
			// The lazy door: the first subscription read for an account that was
			// created some other way.
			results[idx].sub, results[idx].err = f.support.GetOrCreateUserSubscription(context.Background(), userID)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, res := range results {
		if res.err != nil {
			t.Fatalf("racer %d failed: %v", i, res.err)
		}
	}

	if got := f.countRows(t, userID); got != 1 {
		t.Errorf("expected exactly 1 subscription row after %d racers, got %d - two entitlements mean two bills", concurrencyRacers, got)
	}
	if got := f.countEvents(t, userID); got != 1 {
		t.Errorf("expected exactly 1 history row after %d racers, got %d", concurrencyRacers, got)
	}
	if got := f.countOutbox(t); got != 1 {
		t.Errorf("expected exactly 1 published outbox record after %d racers, got %d - the AI service would hear about the same grant twice", concurrencyRacers, got)
	}

	// Every lazy racer must observe the SAME row, not merely avoid erroring. A
	// per-caller in-memory struct would satisfy the counts above and still be a
	// bug: the caller would carry a subscription that is not the stored one.
	// `user_subscriptions` is keyed by `user_id`, so `StartedAt` stands in for row
	// identity - a per-caller struct would carry each goroutine's own timestamp.
	var stored entities.UserSubscription
	if err := f.dbConn.Where("user_id = ?", userID).First(&stored).Error; err != nil {
		t.Fatalf("reload the stored subscription: %v", err)
	}
	for i, res := range results {
		if res.sub == nil {
			continue
		}
		if res.sub.UserID != stored.UserID || res.sub.CurrentPlanCode != stored.CurrentPlanCode {
			t.Errorf("racer %d observed subscription %s/%q, the stored row is %s/%q",
				i, res.sub.UserID, res.sub.CurrentPlanCode, stored.UserID, stored.CurrentPlanCode)
		}
		if !res.sub.StartedAt.Equal(stored.StartedAt) {
			t.Errorf("racer %d observed started_at %s, the stored row has %s - it is not reading the committed row back",
				i, res.sub.StartedAt, stored.StartedAt)
		}
	}
}

// TestIntegrationEagerUserForeignKeyRejectsAnUnknownAccount is 024 T211, and it
// is the assertion that makes the temporary foreign key worth installing.
//
// Everything else in this file creates its accounts with `newAccount`, which
// COMMITS them, so the constraint was satisfied by construction and could not
// reject anything: deleting `installEagerUserForeignKey` outright left both tests
// green. That is not a fixture, that is leftover DDL - and it left the argument
// the file exists to support unguarded, because the argument is precisely that
// `user_subscriptions.user_id` is a foreign key onto `identity.users(id)` and
// therefore cannot be written from another connection.
//
// So the constraint is now asked to do that job directly. The control comes
// first: an insert for an account that DOES exist must succeed, which is what
// proves the refusal below is the constraint and not a column or type problem.
// Then the same statement, for an account that does not exist, must be refused.
//
// Both run outside any transaction on purpose. A statement-level check is the
// weakest form the constraint has and is the one that must still hold; if it is
// not enforced here it is not enforced at all.
func TestIntegrationEagerUserForeignKeyRejectsAnUnknownAccount(t *testing.T) {
	f := newEagerProvisionFixture(t)
	ctx := context.Background()

	insert := func(userID uuid.UUID) error {
		return f.dbConn.WithContext(ctx).Exec(
			`INSERT INTO user_subscriptions (user_id, subscription_plan_id, current_plan_code, current_tier_rank, current_plan_kind)
			 VALUES (?, ?, ?, ?, ?)`,
			userID, f.defaultPlan.ID, f.defaultPlan.Slug, f.defaultPlan.TierRank, int16(f.defaultPlan.PlanKind),
		).Error
	}

	existing := f.newAccount(t)
	if err := insert(existing); err != nil {
		t.Fatalf("the constraint must accept a subscription for an account that exists: %v", err)
	}

	unknown := uuid.New()
	if err := insert(unknown); err == nil {
		t.Fatalf("the constraint must refuse a subscription for account %s, which does not exist - "+
			"without it the account and its entitlement could be committed by different connections", unknown)
	}

	// The refusal must be the constraint, and it must have written nothing.
	if got := f.countRows(t, unknown); got != 0 {
		t.Errorf("the refused insert left %d rows behind", got)
	}
}

// --- fixture ----------------------------------------------------------------

// eagerOutboxPublisher is `messaging.RabbitMQClient.Publish` without the broker.
//
// It marshals the payload and writes the outbox record through the repository, on
// the caller's transaction when there is one - which is the whole of what the
// account service does at publish time. The AMQP hop happens later, in a background
// worker, so counting the rows it writes here counts exactly the events production
// would have produced.
type eagerOutboxPublisher struct {
	outbox messaging.OutboxRepository

	mu  sync.Mutex
	ids []uuid.UUID
}

func (p *eagerOutboxPublisher) Publish(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}
	headers, err := json.Marshal(map[string]any{
		"correlationId": uuid.NewString(),
		"eventVersion":  "v1",
	})
	if err != nil {
		return fmt.Errorf("marshal outbox headers: %w", err)
	}
	record := &messaging.OutboxRecord{
		Topic:   topic,
		Payload: body,
		Headers: headers,
		Status:  messaging.OutboxStatusPending,
	}
	if err := p.outbox.Create(ctx, record); err != nil {
		return fmt.Errorf("create outbox record: %w", err)
	}
	p.mu.Lock()
	p.ids = append(p.ids, record.ID)
	p.mu.Unlock()
	return nil
}

type eagerProvisionFixture struct {
	dbConn      *gorm.DB
	support     *SubscriptionStateSupport
	publisher   *eagerOutboxPublisher
	defaultPlan *entities.SubscriptionPlan

	mu       sync.Mutex
	accounts []uuid.UUID
}

func newEagerProvisionFixture(t *testing.T) *eagerProvisionFixture {
	t.Helper()

	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)
	installEagerUserForeignKey(t, dbConn)

	publisher := &eagerOutboxPublisher{outbox: messaging.NewOutboxRepository(dbConn)}
	uow := db.NewGormUnitOfWork(dbConn)

	f := &eagerProvisionFixture{dbConn: dbConn, publisher: publisher}
	f.support = NewSubscriptionStateSupport(
		persistence.NewUserSubscriptionRepository(dbConn),
		persistence.NewSubscriptionPlanRepository(dbConn),
		persistence.NewUserDailyQuotaRepository(dbConn),
		NewPlanChangePublisher(persistence.NewUserSubscriptionEventRepository(dbConn), publisher),
		uow,
	)

	// One default plan serves every account in the test, and it must be the row
	// `GetDefaultPlan` finds: `plan_kind = DefaultFree AND is_active`. That row is
	// seeded by database/init/03-seed.sql and the migrated schema carries
	// ux_single_active_default_free_plan, so a SECOND active free plan cannot
	// exist — reuse the seeded one instead of inserting a duplicate.
	defaultPlan, err := persistence.NewSubscriptionPlanRepository(dbConn).GetDefaultPlan(context.Background())
	if err != nil || defaultPlan == nil {
		t.Fatalf("default free plan missing from test database (apply database/init/03-seed.sql): %v", err)
	}
	f.defaultPlan = defaultPlan
	t.Cleanup(func() { cleanupSubscriptionTables(t, dbConn) })

	return f
}

// newAccount inserts a real `identity.users` row and returns its id.
//
// The account is COMMITTED, unlike the ones the account-creation doors insert, so
// the foreign key is satisfied for every racer regardless of which transaction
// ends up winning. What is under test is the entitlement, not the account insert -
// and the foreign key's own enforcement is asserted separately, by
// `TestIntegrationEagerUserForeignKeyRejectsAnUnknownAccount`.
func (f *eagerProvisionFixture) newAccount(t *testing.T) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := eagerAccountMarker + userID.String() + "@example.test"
	if err := f.dbConn.Exec(fmt.Sprintf(
		`INSERT INTO %s (id, username, email, role_slug, status, is_deleted, provider, email_verified)
		 VALUES (?, ?, ?, 'user', 0, false, 'local', true)`,
		eagerAccountTable,
	), userID, eagerAccountMarker+userID.String(), email).Error; err != nil {
		t.Fatalf("insert account row: %v", err)
	}

	f.mu.Lock()
	f.accounts = append(f.accounts, userID)
	f.mu.Unlock()

	t.Cleanup(func() {
		f.deleteAccountRows(t, userID)
	})
	return userID
}

// deleteAccountRows removes exactly what this test created for one account.
func (f *eagerProvisionFixture) deleteAccountRows(t *testing.T, userID uuid.UUID) {
	t.Helper()
	var outboxIDs []uuid.UUID
	if f.publisher != nil {
		f.publisher.mu.Lock()
		outboxIDs = append(outboxIDs, f.publisher.ids...)
		f.publisher.mu.Unlock()
	}
	for _, stmt := range []struct {
		query string
		args  []any
	}{
		{`DELETE FROM user_subscriptions WHERE user_id = ?`, []any{userID}},
		{`DELETE FROM user_subscription_events WHERE user_id = ?`, []any{userID}},
		{fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, eagerAccountTable), []any{userID}},
	} {
		if err := f.dbConn.Exec(stmt.query, stmt.args...).Error; err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}
	if len(outboxIDs) > 0 {
		if err := f.dbConn.Exec(`DELETE FROM outbox_records WHERE id IN ?`, outboxIDs).Error; err != nil {
			t.Errorf("cleanup outbox records: %v", err)
		}
	}
}

func (f *eagerProvisionFixture) countRows(t *testing.T, userID uuid.UUID) int64 {
	t.Helper()
	var count int64
	if err := f.dbConn.Model(&entities.UserSubscription{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count subscription rows: %v", err)
	}
	return count
}

func (f *eagerProvisionFixture) countEvents(t *testing.T, userID uuid.UUID) int64 {
	t.Helper()
	var count int64
	if err := f.dbConn.Model(&entities.UserSubscriptionEvent{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count history rows: %v", err)
	}
	return count
}

func (f *eagerProvisionFixture) countOutbox(t *testing.T) int64 {
	t.Helper()
	f.publisher.mu.Lock()
	ids := append([]uuid.UUID(nil), f.publisher.ids...)
	f.publisher.mu.Unlock()

	var count int64
	if len(ids) == 0 {
		return 0
	}
	if err := f.dbConn.Model(&messaging.OutboxRecord{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		t.Fatalf("count outbox records: %v", err)
	}
	return count
}

// installEagerUserForeignKey puts the production FK in place for the duration of
// one test.
//
// `identity.users` is created with plain DDL rather than through AutoMigrate:
// the test only exercises the foreign key and the row insert, not the full
// identity schema. The columns below are the ones the foreign key and the row
// insert need, and they are the NOT NULL set of the production definition.
func installEagerUserForeignKey(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	createdUsersTable := ensureEagerAccountTable(t, dbConn)

	if err := dbConn.Exec(fmt.Sprintf(`ALTER TABLE user_subscriptions DROP CONSTRAINT IF EXISTS %s`, eagerUserFK)).Error; err != nil {
		t.Fatalf("drop a leftover user foreign key: %v", err)
	}
	if err := dbConn.Exec(fmt.Sprintf(
		`ALTER TABLE user_subscriptions ADD CONSTRAINT %s FOREIGN KEY (user_id) REFERENCES %s(id) ON DELETE CASCADE`,
		eagerUserFK, eagerAccountTable,
	)).Error; err != nil {
		t.Fatalf("add the user foreign key: %v", err)
	}
	t.Cleanup(func() {
		if err := dbConn.Exec(fmt.Sprintf(`ALTER TABLE user_subscriptions DROP CONSTRAINT IF EXISTS %s`, eagerUserFK)).Error; err != nil {
			t.Errorf("drop the user foreign key: %v", err)
		}
		// Only ever drop what this test made. On a migrated database the table
		// belongs to the identity module and dropping it would destroy it.
		if !createdUsersTable {
			return
		}
		if err := dbConn.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, eagerAccountTable)).Error; err != nil {
			t.Errorf("drop the account table: %v", err)
		}
	})
}

// ensureEagerAccountTable creates `identity.users` when it is missing and reports
// whether THIS call is what created it.
//
// The existence probe rather than `CREATE TABLE IF NOT EXISTS` is deliberate: the
// old helper used `IF NOT EXISTS` and then left the table behind unconditionally,
// so every run against an unmigrated test database left a `public.users` behind
// and the next run silently reused it. What the caller needs is not "a table
// exists" but "did I make this one", because only the latter may be dropped.
//
// The `identity` schema is left in place on purpose. Dropping it would mean
// `DROP SCHEMA ... CASCADE`, and this database is shared with every other
// integration test in the repository - including packages this one cannot see -
// so a cascade here could destroy a table another test created seconds earlier. An
// empty schema is inert; a cascade is not.
func ensureEagerAccountTable(t *testing.T, dbConn *gorm.DB) bool {
	t.Helper()
	if err := dbConn.Exec(`CREATE SCHEMA IF NOT EXISTS identity`).Error; err != nil {
		t.Fatalf("create the identity schema: %v", err)
	}
	if eagerTableExists(t, dbConn, eagerAccountTable) {
		return false
	}
	if err := dbConn.Exec(fmt.Sprintf(`CREATE TABLE %s (
		id uuid PRIMARY KEY,
		username character varying(255) NOT NULL,
		email character varying(255) NOT NULL,
		role_slug character varying(50) NOT NULL,
		status smallint NOT NULL DEFAULT 0,
		is_deleted boolean NOT NULL DEFAULT false,
		provider character varying(50) NOT NULL DEFAULT 'local',
		email_verified boolean NOT NULL DEFAULT false,
		created_at timestamp with time zone NOT NULL DEFAULT now(),
		updated_at timestamp with time zone NOT NULL DEFAULT now()
	)`, eagerAccountTable)).Error; err != nil {
		t.Fatalf("create the account table: %v", err)
	}
	return true
}

// eagerTableExists reports whether a schema-qualified table is present.
func eagerTableExists(t *testing.T, dbConn *gorm.DB, qualifiedName string) bool {
	t.Helper()
	schema, table, found := strings.Cut(qualifiedName, ".")
	if !found {
		t.Fatalf("%q is not schema-qualified", qualifiedName)
	}
	var count int64
	if err := dbConn.Raw(
		`SELECT count(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?`,
		schema, table,
	).Scan(&count).Error; err != nil {
		t.Fatalf("look up %s: %v", qualifiedName, err)
	}
	return count > 0
}
