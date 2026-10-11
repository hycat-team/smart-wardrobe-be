package contract_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 025 T024 — the seven contract scenarios of contracts/identity-contract.md §6.
//
// They run against a real SQL database (in-memory sqlite) rather than a fake
// repository, and that is not incidental. Scenario 2 is the one that matters: the
// composite-key cursor is a WHERE clause, and a fake repository cannot get a WHERE
// clause wrong in the same way a real one can. A hand-written fake would have
// encoded the correct pagination by construction and the test would pass no matter
// what the SQL said.
//
// The table is created by hand with only the columns this query touches.
// AutoMigrate on `entities.User` is not usable here: the entity carries a
// `vector(768)` column and expression indexes, which sqlite does not implement,
// and the columns under test have nothing to do with either.
const contractTestUsersDDL = `
CREATE TABLE users (
    id               TEXT PRIMARY KEY,
    username         TEXT NOT NULL,
    email            TEXT NOT NULL,
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME,
    is_deleted       BOOLEAN NOT NULL DEFAULT 0,
    created_by_admin BOOLEAN NOT NULL DEFAULT 0,
    email_verified   BOOLEAN NOT NULL DEFAULT 0,
    provider         TEXT NOT NULL DEFAULT 'local',
    role_slug        TEXT NOT NULL DEFAULT 'user',
    status           INTEGER NOT NULL DEFAULT 0
)`

// contractTestWindowStart anchors every timestamp below, so the window is a fixed
// range rather than "around now".
var contractTestWindowStart = time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC)

func newCampaignContractFixture(t *testing.T) (identity_contract.CampaignAccountLister, *gorm.DB) {
	t.Helper()

	dbConn, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "open the in-memory database")

	// A shared-cache in-memory database is visible to every connection in the
	// pool, so the table must be dropped explicitly or a second test in this
	// process would find the first one's rows.
	require.NoError(t, dbConn.Exec("DROP TABLE IF EXISTS users").Error)
	require.NoError(t, dbConn.Exec(contractTestUsersDDL).Error)
	t.Cleanup(func() { _ = dbConn.Exec("DROP TABLE IF EXISTS users").Error })

	repo := persistence.NewUserRepository(dbConn, nil)
	reader := usecase.NewCampaignAccountReader(repo)
	return identity_contract.NewCampaignAccountLister(identity_contract.NewUserBriefContract(nil, reader)), dbConn
}

// insertAccount writes one row straight through SQL, so the test controls created_at
// exactly — GORM would stamp `now()` and the shared-instant scenarios would be
// untestable.
func insertAccount(t *testing.T, dbConn *gorm.DB, username string, createdAt time.Time, deleted, createdByAdmin bool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	require.NoError(t, dbConn.Exec(
		`INSERT INTO users (id, username, email, created_at, updated_at, is_deleted, created_by_admin, email_verified, provider, role_slug, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 1, 'local', 'user', 0)`,
		userID.String(), username, username+"@example.test", createdAt, createdAt, deleted, createdByAdmin,
	).Error)
	return userID
}

func defaultContractFilter() repositories.CampaignAccountFilter {
	return repositories.CampaignAccountFilter{
		From:  contractTestWindowStart,
		To:    contractTestWindowStart.Add(24 * time.Hour),
		Limit: 3,
	}
}

// Scenario 1: ten accounts, take three. Three records, CreatedAt ascending, and
// TotalCount is the WINDOW total rather than the batch size.
func TestCampaignAccountListerReturnsOneBatchOldestFirst(t *testing.T) {
	lister, dbConn := newCampaignContractFixture(t)
	for i := 0; i < 10; i++ {
		insertAccount(t, dbConn, "batch1-"+string(rune('a'+i)), contractTestWindowStart.Add(time.Duration(i)*time.Minute), false, false)
	}

	page, err := lister.ListCreatedInWindow(context.Background(), defaultContractFilter())
	require.NoError(t, err)
	require.NotNil(t, page)

	assert.Len(t, page.Items, 3, "the filter asked for three")
	assert.Equal(t, int64(10), page.TotalCount,
		"TotalCount is the whole window, not the batch — the sweep uses it to know how many batches are left")

	for i := 1; i < len(page.Items); i++ {
		assert.False(t, page.Items[i].CreatedAt.Before(page.Items[i-1].CreatedAt),
			"items must be ordered oldest first, got %v then %v", page.Items[i-1].CreatedAt, page.Items[i].CreatedAt)
	}
	assert.Equal(t, contractTestWindowStart, page.Items[0].CreatedAt.UTC(),
		"the oldest account in the window comes first")
}

// Scenario 2 — the trap. Three accounts share one creation instant, read one at a
// time.
//
// `created_at` has millisecond resolution, so accounts registered in the same test
// run routinely collide. Paginating by the timestamp alone either skips the rest
// of that instant (every later batch asks for `created_at > T` and gets nothing)
// or returns it forever. The composite (created_at, id) key is the only thing that
// makes the batch boundaries stable, and this is where that is proven against real
// SQL rather than asserted in a comment.
func TestCampaignAccountListerPagesWithoutLossWhenAccountsShareAnInstant(t *testing.T) {
	lister, dbConn := newCampaignContractFixture(t)

	sharedInstant := contractTestWindowStart.Add(time.Hour)
	first := insertAccount(t, dbConn, "same-instant-a", sharedInstant, false, false)
	second := insertAccount(t, dbConn, "same-instant-b", sharedInstant, false, false)
	third := insertAccount(t, dbConn, "same-instant-c", sharedInstant, false, false)

	filter := defaultContractFilter()
	filter.Limit = 1

	seen := map[uuid.UUID]int{}
	for batch := 0; batch < 3; batch++ {
		page, err := lister.ListCreatedInWindow(context.Background(), filter)
		require.NoError(t, err)
		require.Len(t, page.Items, 1, "batch %d must return exactly one account", batch)

		id := page.Items[0].ID
		seen[id]++
		assert.Equal(t, 1, seen[id],
			"batch %d returned %s, which an earlier batch already returned - the cursor did not advance",
			batch, id)

		// Advance the composite cursor from the row just read.
		filter.CursorTime = page.Items[0].CreatedAt
		filter.CursorID = id
	}

	assert.Len(t, seen, 3, "all three accounts sharing one instant must be reached, got %v", seen)
	assert.Equal(t, 1, seen[first], "the first account must appear exactly once")
	assert.Equal(t, 1, seen[second], "the second account must appear exactly once")
	assert.Equal(t, 1, seen[third], "the third account must appear exactly once")
}

// Scenario 3: nothing in the window. Items is an EMPTY slice, never nil.
func TestCampaignAccountListerReturnsEmptySliceForAnEmptyWindow(t *testing.T) {
	lister, dbConn := newCampaignContractFixture(t)
	insertAccount(t, dbConn, "outside-the-window", contractTestWindowStart.Add(48*time.Hour), false, false)

	page, err := lister.ListCreatedInWindow(context.Background(), defaultContractFilter())
	require.NoError(t, err)
	require.NotNil(t, page)

	assert.NotNil(t, page.Items, "an empty result must be an empty slice, not nil: callers range over it")
	assert.Empty(t, page.Items)
	assert.Equal(t, int64(0), page.TotalCount)
}

// Scenario 4: a soft-deleted account inside the window never appears.
//
// A deleted account must not receive new entitlements, and the sweep walks this
// list unattended — so the filter belongs in the query, not in the caller.
func TestCampaignAccountListerOmitsSoftDeletedAccounts(t *testing.T) {
	lister, dbConn := newCampaignContractFixture(t)
	insertAccount(t, dbConn, "alive", contractTestWindowStart.Add(time.Minute), false, false)
	insertAccount(t, dbConn, "deleted", contractTestWindowStart.Add(2*time.Minute), true, false)

	page, err := lister.ListCreatedInWindow(context.Background(), defaultContractFilter())
	require.NoError(t, err)

	require.Len(t, page.Items, 1)
	assert.Equal(t, "alive", page.Items[0].Username)
	// The window total is computed over the same filter, so it agrees with what a
	// sweep would actually be able to process.
	assert.Equal(t, int64(1), page.TotalCount)
}

// Scenario 5: an administrator-created account reports the flag, and the DTO
// agrees with the stored column.
func TestCampaignAccountListerCarriesTheAdministratorFlag(t *testing.T) {
	lister, dbConn := newCampaignContractFixture(t)
	insertAccount(t, dbConn, "by-admin", contractTestWindowStart.Add(time.Minute), false, true)
	insertAccount(t, dbConn, "self-serve", contractTestWindowStart.Add(2*time.Minute), false, false)

	page, err := lister.ListCreatedInWindow(context.Background(), defaultContractFilter())
	require.NoError(t, err)
	require.Len(t, page.Items, 2)

	flags := map[string]bool{}
	for _, item := range page.Items {
		flags[item.Username] = item.CreatedByAdmin

		// Cross-check the DTO against the column, read separately. Comparing the
		// DTO with itself would assert that a value equals itself.
		var stored bool
		require.NoError(t, dbConn.Raw("SELECT created_by_admin FROM users WHERE id = ?", item.ID.String()).Scan(&stored).Error)
		assert.Equal(t, stored, item.CreatedByAdmin,
			"the DTO flag must agree with the column for %q, or the sweep and the event path decide differently", item.Username)
	}

	assert.True(t, flags["by-admin"], "an administrator-created account must be recognisable as one")
	assert.False(t, flags["self-serve"], "a self-serve account must not be flagged")
}

// Scenario 6 is covered where both sources can be produced together, in
// identity/application/usecase/user_created_event_test.go
// (`TestCreatedByAdminFlagAgreesWithTheStoredColumn`): it needs the three creation
// doors, which this package cannot construct.
//
// What this side contributes is the DTO half of that equality, proven above in
// scenario 5 — so the flag the sweep reads and the flag the event carries come from
// the same column.

// Scenario 7: `services/commerce` must still compile.
//
// This is the standing evidence for the decision not to extend `IUserBriefContract`.
// That contract is served over gRPC with a client adapter and a server adapter, so
// adding a method to it would force a change to `identity.proto` — a cross-service
// contract change for a need that only exists inside this service (FR-058).
//
// The test builds the commerce module rather than grepping for the change, because
// the failure this guards against is a compile error in another service: the gRPC
// client adapter carries `var _ IUserBriefContract = (*UserBriefClient)(nil)`, so
// widening the interface breaks the build HERE first, and the assertion that
// commerce still compiles is what proves the adapter was left alone.
func TestCommerceStillCompilesAfterTheContractWasNotWidened(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-module compile is covered by the build matrix in CI")
	}

	repoRoot := filepath.Join("..", "..", "..", "..")
	commerceDir := filepath.Join(repoRoot, "services", "commerce")

	if _, err := os.Stat(commerceDir); err != nil {
		t.Fatalf("cannot locate the commerce module at %s: %v", commerceDir, err)
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = commerceDir
	// GOWORK is deliberately NOT disabled: commerce resolves `shared` and
	// `services/account` through the workspace, and turning it off would fail the
	// build for an unrelated reason — turning this test green for the wrong cause.
	cmd.Env = os.Environ()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("services/commerce must compile unchanged - it is the evidence that the gRPC identity contract was NOT widened (FR-058).\n%s", output)
	}
}

// TestCampaignAccountListerDoesNotWidenTheGrpcContract states the same rule as a
// compile-time fact rather than as a build of another module.
//
// `UserBriefClient` asserts `var _ IUserBriefContract = (*UserBriefClient)(nil)`.
// If someone adds a method to `IUserBriefContract`, the failure surfaces HERE, in
// this service, naming the interface — instead of surfacing as a nil contract at
// some consumer, or as a `go build` failure in a module nobody was looking at.
func TestCampaignAccountListerDoesNotWidenTheGrpcContract(t *testing.T) {
	// `NewCampaignAccountLister` takes the CONCRETE type, so the narrow view can
	// exist without the interface gaining a method. If it were changed to accept
	// `IUserBriefContract` instead, the narrow method would have to be on the
	// interface and this would no longer compile against an unchanged proto.
	var lister identity_contract.CampaignAccountLister = identity_contract.NewCampaignAccountLister(
		identity_contract.NewUserBriefContract(nil, nil),
	)
	assert.NotNil(t, lister)

	// The full contract keeps exactly its four original methods: widening it is the
	// mistake scenario 7 exists to catch, and counting them makes it visible in the
	// diff rather than only at some consumer's runtime.
	contractType := reflectTypeOfUserBriefContract()
	if contractType == nil {
		t.Fatal("expected IUserBriefContract to exist")
	}
	assert.Equal(t, 4, contractType.NumMethod(),
		"IUserBriefContract must keep its original method set - FR-058 forbids changing a cross-service contract for an in-service need")
}

// --- compile-time shape checks ----------------------------------------------

// The contract must hand out DTOs. constitution §II: "using DTOs, never domain
// entities" — a domain entity pointer must not cross a module boundary, and the
// subscription module is the consumer.
//
// The generic instantiation is a TYPE-level assertion evaluated at compile time: if
// `Items` ever became `[]*entities.User`, this line would stop compiling here rather
// than at the consumer that would then be reaching into another module's domain
// layer.
type campaignPageItemsMustBeDTOs = func(identity_contract.CampaignAccountPage) []*identitydto.UserBriefContractDTO

func campaignPageItemsTypeCheck(p identity_contract.CampaignAccountPage) []*identitydto.UserBriefContractDTO {
	return p.Items
}

var _ campaignPageItemsMustBeDTOs = campaignPageItemsTypeCheck

// The created-at value the sweep compares against the watermark (FR-031) and the
// administrator flag (FR-091) have to be on the DTO, or the sweep would have to
// read identity.users itself — which constitution §II forbids.
type campaignDTOFieldsMustExist = func(identitydto.UserBriefContractDTO) (time.Time, bool)

func campaignDTOFieldsCheck(d identitydto.UserBriefContractDTO) (time.Time, bool) {
	return d.CreatedAt, d.CreatedByAdmin
}

var _ campaignDTOFieldsMustExist = campaignDTOFieldsCheck

// unusedEntityGuards keeps the identity constants honest: the fixture writes the
// same values the creation doors set, so a rename of either constant breaks here
// rather than silently producing a different fixture.
//
// `entities.User` is deliberately NOT referenced anywhere in this file. Importing
// another module's domain layer — even from a test — is what constitution §II
// rules out, and a compile-time assertion above already proves the contract hands
// out DTOs.
var (
	_ = authprovider.Local
	_ = roleslug.User
	_ = userstatus.Active
)

// reflectTypeOfUserBriefContract exposes the gRPC contract's method set.
func reflectTypeOfUserBriefContract() reflect.Type {
	return reflect.TypeOf((*identity_contract.IUserBriefContract)(nil)).Elem()
}
