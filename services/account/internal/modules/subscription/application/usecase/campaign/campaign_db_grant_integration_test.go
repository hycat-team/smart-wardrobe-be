package campaign

import (
	"context"
	"os"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 026 T038 — quickstart V1, the OPEN half, against a real Postgres.
//
// WHAT THIS PROVES THAT NOTHING ABOVE IT CAN. Everything before this file runs against
// fakes, and a fake registry holds rows in a slice. The claim that 026 exists to make
// is that opening a campaign is a DATA operation: write a row through the real
// repository, and a signup message arriving afterwards reaches the grant path through
// the real `ListOpen`, the real advisory lock, the real `ON CONFLICT` claim insert and
// the real `EXISTS`. Every one of those is a SQL statement, and every one of them can
// be replaced by something that compiles and answers nothing in production.
//
// SPECIFICALLY IT PINS FOUR THINGS A FAKE WOULD NOT:
//
//   - THE CAMPAIGN IS FOUND BY THE GRANT PATH. `ListOpen` filters `closed_at IS NULL`
//     and orders by `starts_at DESC, code DESC`; a campaign that was inserted but not
//     returned grants nobody, and the failure is silent — the account simply reports
//     `campaign_not_configured`.
//   - A `starts_at` IN THE PAST IS ENOUGH. US1's whole promise is that a campaign
//     opened retroactively covers accounts registering afterwards, with no deploy.
//   - THE CAMPAIGN CLAIMED THE SLOT. A row in `campaign_claims` is the ledger's own
//     record, and `ExistsClaims` reading it back is FR-020's precondition.
//   - THE PLAN MUST EXIST AND BE ACTIVE. FR-014 keeps that a RUN-time check, so a
//     campaign naming a plan nobody sells grants nobody — and the reason must be
//     `plan_unavailable`, not silence.
//
// THE HARNESS IS THE REPO'S, NOT A NEW ONE. `TEST_DATABASE_URL` plus a `t.Skip` is how
// every Postgres-backed test in this service already behaves, and the schema comes from
// the EMBEDDED GOOSE MIGRATIONS rather than from `AutoMigrate` — which is the point
// that matters, because `AutoMigrate` builds tables from the Go structs and would
// therefore not create the CHECK constraints 026 relies on.
//
// It needs a database already provisioned with the base `identity` schema: the claim
// ledger's `user_id` carries a foreign key onto `identity.users(id)` and no migration in
// this repository creates that table.

// dbCampaignPrefix namespaces every campaign this file writes, so a run against a shared
// test database cannot collide with a real one or delete it.
const dbCampaignPrefix = "verify-026-open-"

// dbAccountMarker prefixes the accounts this file inserts into `identity.users`, for the
// same reason and because that table carries unique indexes on `lower(email)` and
// `lower(username)`.
const dbAccountMarker = "verify-026-open-"

// openGrantTestDB connects and migrates, skipping when no database is provisioned.
func openGrantTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping campaign open/grant integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// The migrations, not `AutoMigrate`: the CHECK constraints and the immutability
	// trigger are part of what 026 changed, and a schema built from the structs would
	// not have them (see the header).
	if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS, "goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() {
		// Claims first: the ledger names the campaign as text and has no foreign key,
		// so the order is tidiness rather than a constraint (FR-091).
		dbConn.Exec("DELETE FROM subscription.campaign_claims WHERE campaign_code LIKE ?", dbCampaignPrefix+"%")
		dbConn.Exec("DELETE FROM subscription.campaigns WHERE code LIKE ?", dbCampaignPrefix+"%")
	})
	return dbConn
}

// newTestAccount inserts an `identity.users` row and removes it afterwards.
//
// The account is inserted RAW rather than through the identity module, because this
// test is about the CAMPAIGN side of the grant and the identity module's whole surface
// is heavier than the one row this needs. The columns named are the ones an already
// migrated `identity.users` declares NOT NULL (`id`, `username`, `email`, `role_slug`)
// with the rest defaulted — the same list 024's eager-provisioning test uses, so a
// database that genuinely lacked one of them fails loudly with the database's own error
// instead of being papered over.
func newTestAccount(t *testing.T, dbConn *gorm.DB, registeredAt time.Time) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	handle := dbAccountMarker + userID.String()
	if err := dbConn.Exec(
		`INSERT INTO identity.users (id, username, email, role_slug, status, is_deleted, provider, email_verified, created_at)
		 VALUES (?, ?, ?, 'user', 0, false, 'local', true, ?)`,
		userID, handle, handle+"@example.test", registeredAt,
	).Error; err != nil {
		t.Fatalf("insert account row: %v", err)
	}
	t.Cleanup(func() {
		// The subscription row cascades from the account in production, but the ledger
		// is deleted explicitly first so a run leaves nothing behind even if a future
		// migration changes the cascade.
		dbConn.Exec("DELETE FROM subscription.user_subscriptions WHERE user_id = ?", userID)
		dbConn.Exec("DELETE FROM subscription.user_subscription_events WHERE user_id = ?", userID)
		dbConn.Exec("DELETE FROM identity.users WHERE id = ?", userID)
	})
	return userID
}

// insertTestPlan writes a real plan row and removes it afterwards.
//
// The plan has to be REAL for FR-014 to be exercised at all: the grant path reads it
// with `GetBySlug`, and an inactive or missing plan is skipped with
// `plan_unavailable`. A stubbed plan repository would make that branch unreachable.
func insertTestPlan(t *testing.T, dbConn *gorm.DB, slug string) *entities.SubscriptionPlan {
	t.Helper()
	days := 30
	plan := &entities.SubscriptionPlan{
		Slug:               slug,
		Name:               "Premium Tháng (verify-026)",
		IsActive:           true,
		DurationDays:       &days,
		PlanKind:           1,
		TierRank:           1,
		AICostPolicyID:     uuid.New(),
		MaxWardrobeItems:   100,
		MaxOutfits:         20,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
	}
	if err := persistence.NewSubscriptionPlanRepository(dbConn).Create(context.Background(), plan); err != nil {
		t.Fatalf("insert plan %s: %v", slug, err)
	}
	t.Cleanup(func() {
		dbConn.Exec("DELETE FROM subscription.subscription_plans WHERE slug = ?", slug)
	})
	return plan
}

// dbGrantFixture holds the two use cases and the repositories a test reads back
// through.
//
// The repositories are returned rather than reached for again, because a test that
// constructs its own would be free to use a different one than the path under test —
// and "the ledger says so" is only evidence when the ledger is the one that was
// written.
type dbGrantFixture struct {
	grant    *CampaignUseCase
	registry *CampaignRegistryUseCase
	campaign repositories.ICampaignRepository
	claims   repositories.ICampaignClaimRepository
	subs     repositories.IUserSubscriptionRepository
}

// newDBGrantFixture wires the REAL registry writer and the REAL grant path over one
// database.
//
// The collaborators that are faked are the ones whose subject is not this test: the
// event repository and the publisher (the plan-change event goes to the AI service over
// a queue, and this test is not about the AI service), the audit contract (FR-060's
// outbox is proven by the outbox's own tests and by quickstart V7), and the
// notification contract (this test must not send an email to anybody).
//
// The registry, the claim ledger, the plan catalogue, the subscription row and the unit
// of work are ALL real, because every claim in the header is a statement about them.
func newDBGrantFixture(t *testing.T, dbConn *gorm.DB) *dbGrantFixture {
	t.Helper()

	campaignRepo := persistence.NewCampaignRepository(dbConn)
	claimRepo := persistence.NewCampaignClaimRepository(dbConn)
	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	subRepo := persistence.NewUserSubscriptionRepository(dbConn)
	events := newFakeEventRepo()
	publisher := &fakePublisher{}
	uow := shareddb.NewGormUnitOfWork(dbConn)

	grantUC := NewCampaignUseCase(
		testConfig(),
		campaignRepo,
		claimRepo,
		subRepo,
		planRepo,
		subscription.NewPlanChangePublisher(events, publisher),
		&fakeAuditContract{},
		&fakeNotifContract{},
		nil,
		uow,
		&recordingLogger{},
	).(*CampaignUseCase)
	// Pin the clock inside the campaign's window. A test that used the real clock would
	// pass or fail depending on the day the suite ran, and this file's subject is
	// precisely a window that opens in the PAST.
	grantUC.now = func() time.Time { return baseInstant() }

	registryUC := NewCampaignRegistryUseCase(
		testConfig(),
		campaignRepo,
		&fakeAuditContract{},
		uow,
		&recordingLogger{},
	).(*CampaignRegistryUseCase)
	registryUC.now = func() time.Time { return baseInstant() }

	return &dbGrantFixture{
		grant:    grantUC,
		registry: registryUC,
		campaign: campaignRepo,
		claims:   claimRepo,
		subs:     subRepo,
	}
}

// TestIntegrationOpeningACampaignLetsANewAccountReceiveThePlan is T038 / quickstart V1
// steps 1-2 / US1's Independent Test.
//
// The order is the feature: create through the registry API's use case, THEN deliver the
// signup message. Nothing is restarted, nothing is re-deployed, and no file changes —
// which is the whole point of FR-001 and SC-001. Reversing the two steps would prove
// only that a pre-existing campaign grants, which 025 already established.
func TestIntegrationOpeningACampaignLetsANewAccountReceiveThePlan(t *testing.T) {
	dbConn := openGrantTestDB(t)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	const (
		campaignCode = dbCampaignPrefix + "grant"
		planSlug     = dbCampaignPrefix + "premium-monthly"
	)
	plan := insertTestPlan(t, dbConn, planSlug)

	// STEP 1 — open the campaign with `starts_at` IN THE PAST. That is the shape
	// quickstart V1 uses and it is the shape US1 promises: accounts registering
	// afterwards are covered immediately, with no deploy.
	planSlugField := planSlug
	quota := 2
	startsAt := baseInstant().Add(-24 * time.Hour)
	summary, err := f.registry.Create(ctx, uuid.New(), &sub_dto.CampaignCreateReq{
		Code:     campaignCode,
		PlanSlug: &planSlugField,
		Quota:    &quota,
		StartsAt: startsAt.Format(time.RFC3339),
	}, nil)
	if err != nil {
		t.Fatalf("create the campaign: %v", err)
	}
	if summary.Code != campaignCode {
		t.Fatalf("the create response describes %q, want %q", summary.Code, campaignCode)
	}

	// The row really is in the registry, read back through the same method the grant
	// path uses. Asserting the response alone would pass even if the write were
	// silently discarded.
	stored, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || stored == nil {
		t.Fatalf("the campaign row must be readable from the registry, got err=%v row=%v", err, stored)
	}
	if stored.Quota != 2 || stored.Version != 1 {
		t.Errorf("the stored row lost its values: quota=%d version=%d", stored.Quota, stored.Version)
	}

	// STEP 2 — a new account registers and the message arrives.
	registeredAt := baseInstant()
	userID := newTestAccount(t, dbConn, registeredAt)
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:         userID,
		RegisteredAt:   registeredAt,
		CreatedByAdmin: false,
	}); err != nil {
		t.Fatalf("process the signup message: %v", err)
	}

	// THE GRANT HAPPENED, and the ledger is the evidence rather than a log line or a
	// response body: `campaign_claims` is the durable record every operator question
	// ("was I given a plan", "which part of the budget did it come from") is answered
	// from.
	claim, found, err := f.claims.GetByUser(ctx, userID)
	if err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	}
	if !found || claim == nil {
		t.Fatalf("US1: the account %s must hold a claim from %s; quickstart V1 step 2 is the acceptance condition", userID, campaignCode)
	}
	if claim.CampaignCode != campaignCode {
		t.Errorf("the claim names campaign %q, want %q", claim.CampaignCode, campaignCode)
	}
	if claim.PlanSlug != planSlug {
		t.Errorf("the claim names plan %q, want %q", claim.PlanSlug, planSlug)
	}
	if claim.GrantBucket != entities.GrantBucketMain {
		t.Errorf("the first grant must come from the main budget, got %q", claim.GrantBucket)
	}

	// FR-020's precondition, read through the very method the budget-locked check uses:
	// "has granted" is what makes `quota`/`reserve`/`plan_slug` immutable, and it has to
	// be answerable or that immutability would be enforced on a guess.
	granted, err := f.campaign.ExistsClaims(ctx, campaignCode)
	if err != nil {
		t.Fatalf("ExistsClaims: %v", err)
	}
	if !granted {
		t.Error("FR-020: the registry must be able to see that this campaign has granted")
	}

	// And the account holds the plan, which is the part a CUSTOMER would notice.
	sub, err := f.subs.GetByUserID(ctx, userID)
	if err != nil || sub == nil {
		t.Fatalf("reload the subscription: err=%v sub=%v", err, sub)
	}
	if sub.CurrentPlanCode != planSlug {
		t.Errorf("current_plan_code = %q, want %q", sub.CurrentPlanCode, planSlug)
	}
	if sub.CurrentTierRank != plan.TierRank {
		t.Errorf("tier_rank = %d, want the granted plan's %d", sub.CurrentTierRank, plan.TierRank)
	}
}

// TestIntegrationACampaignNamingAMissingPlanGrantsNobodyWithPlanUnavailable is FR-014's
// whole content, and it is the decision 025 kept.
//
// The slug is NOT validated at write time and NOT at startup: loading configuration and
// writing the registry must not treat the plan catalogue as a precondition (FR-003). So
// the row is allowed to exist, and every grant against it is skipped with the reason
// code an operator reads on the account look-up. If this test were removed, the failure
// it guards — a write-time existence check, or worse a startup check — would look like a
// small convenience and would make a paused plan able to break campaign opening.
func TestIntegrationACampaignNamingAMissingPlanGrantsNobodyWithPlanUnavailable(t *testing.T) {
	dbConn := openGrantTestDB(t)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	const campaignCode = dbCampaignPrefix + "no-plan"
	planSlug := dbCampaignPrefix + "never-created"
	quota := 2

	if _, err := f.registry.Create(ctx, uuid.New(), &sub_dto.CampaignCreateReq{
		Code:     campaignCode,
		PlanSlug: &planSlug,
		Quota:    &quota,
		StartsAt: baseInstant().Add(-24 * time.Hour).Format(time.RFC3339),
	}, nil); err != nil {
		t.Fatalf("a campaign naming a missing plan must still be writable: %v", err)
	}

	registeredAt := baseInstant()
	userID := newTestAccount(t, dbConn, registeredAt)
	// A message that cannot be decided is retried; this one is a DECISION ("skip"), so
	// it returns nil. A non-nil error here would mean the account is queued for redelivery
	// over a campaign whose plan will never appear.
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       userID,
		RegisteredAt: registeredAt,
	}); err != nil {
		t.Fatalf("a missing plan must be a skip, not a retryable failure: %v", err)
	}

	if _, found, err := f.claims.GetByUser(ctx, userID); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if found {
		t.Errorf("FR-014: an account must not be granted a plan that does not exist")
	}
}
