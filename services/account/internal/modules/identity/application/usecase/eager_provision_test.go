package usecase_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/dto/identity"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	identityiface "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/identity"
	appsecurity "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ---- 024 T196: the three account-creation doors provision eagerly ----
//
// FR-013 says provisioning MUST produce a history row and MUST apply equally to
// every account, "including accounts created by an administrator". SC-006 puts a
// deadline on it: a valid grant within 60 seconds of creating the account, with
// NO read of the subscription in between.
//
// These tests assert the three properties that deadline depends on, none of
// which held before:
//
//  1. The entitlement is provisioned WITHOUT any subscription read.
//  2. The user row and the entitlement are ONE transaction, so a failure
//     half-way leaves no orphan grant pointing at an account that does not
//     exist.
//  3. The user insert happens BEFORE provisioning, inside the same transaction,
//     because `user_subscriptions.user_id` is a foreign key onto
//     `identity.users(id)`.
//
// The double-provisioning question (eager door and lazy door both running for
// one account) is answered where the persistence details live, in
// subscription/eager_default_provision_test.go.

// eagerIdentityUow stages the writes its closure makes and only makes them
// visible when the closure succeeds. That is what lets these tests prove the
// rollback without a database.
//
// Accounts and entitlements live in SEPARATE stores even though one real
// database holds both. They are keyed by the same account id, so a single map
// would let the entitlement overwrite the account row and hide exactly the
// mismatch these tests exist to catch.
type eagerIdentityUow struct {
	// accounts holds the committed account rows.
	accounts map[uuid.UUID]*entities.User
	// entitlements holds the committed entitlement markers, one per provisioned
	// account.
	entitlements map[uuid.UUID]*entities.User
	// stagedAccounts and stagedEntitlements are the same two stores before
	// commit.
	stagedAccounts     map[uuid.UUID]*entities.User
	stagedEntitlements map[uuid.UUID]*entities.User
	commits            int
	rollbacks          int
	// order records the calls made inside the transaction, so a test can assert
	// the account insert came first.
	order []string
}

func newEagerIdentityUow() *eagerIdentityUow {
	return &eagerIdentityUow{
		accounts:           map[uuid.UUID]*entities.User{},
		entitlements:       map[uuid.UUID]*entities.User{},
		stagedAccounts:     map[uuid.UUID]*entities.User{},
		stagedEntitlements: map[uuid.UUID]*entities.User{},
	}
}

func (u *eagerIdentityUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := fn(context.WithValue(ctx, eagerTxKey{}, true)); err != nil {
		u.rollbacks++
		u.stagedAccounts = map[uuid.UUID]*entities.User{}
		u.stagedEntitlements = map[uuid.UUID]*entities.User{}
		return err
	}
	u.commits++
	for id, row := range u.stagedAccounts {
		u.accounts[id] = row
	}
	for id, row := range u.stagedEntitlements {
		u.entitlements[id] = row
	}
	u.stagedAccounts = map[uuid.UUID]*entities.User{}
	u.stagedEntitlements = map[uuid.UUID]*entities.User{}
	return nil
}

// rowsAtCall is how many account rows exist at the moment provisioning runs,
// staged ones included. It proves the account insert came first.
func (u *eagerIdentityUow) rowsAtCall() int {
	return len(u.accounts) + len(u.stagedAccounts)
}

type eagerTxKey struct{}

func inEagerTx(ctx context.Context) bool {
	v, _ := ctx.Value(eagerTxKey{}).(bool)
	return v
}

// eagerIdentityRepo is the part of IUserRepository the creation paths use.
// Anything else panics through the embedded nil interface, which fails the test
// rather than silently doing nothing.
type eagerIdentityRepo struct {
	repositories.IUserRepository
	uow *eagerIdentityUow
	// byEmail lets a test stage an account that already exists.
	byEmail map[string]*entities.User
	// byProviderID plays the same role for the Google door.
	byProviderID map[string]*entities.User
}

func newEagerIdentityRepo(uow *eagerIdentityUow) *eagerIdentityRepo {
	return &eagerIdentityRepo{
		uow:          uow,
		byEmail:      map[string]*entities.User{},
		byProviderID: map[string]*entities.User{},
	}
}

func (r *eagerIdentityRepo) GetByUsernameOrEmail(ctx context.Context, value string) (*entities.User, error) {
	return nil, nil
}

func (r *eagerIdentityRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	return r.byEmail[email], nil
}

func (r *eagerIdentityRepo) IsEmailExists(ctx context.Context, email string) (bool, error) {
	_, exists := r.byEmail[email]
	return exists, nil
}

func (r *eagerIdentityRepo) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	return false, nil
}

func (r *eagerIdentityRepo) GetByProviderID(ctx context.Context, provider string, providerID string) (*entities.User, error) {
	return r.byProviderID[providerID], nil
}

func (r *eagerIdentityRepo) Update(ctx context.Context, user *entities.User) error {
	return nil
}

func (r *eagerIdentityRepo) Create(ctx context.Context, user *entities.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	r.uow.order = append(r.uow.order, "create-account")
	if inEagerTx(ctx) {
		r.uow.stagedAccounts[user.ID] = user
		return nil
	}
	r.uow.accounts[user.ID] = user
	return nil
}

// eagerProvisioningSpy records what the identity module asks of the subscription
// module.
//
// It implements BOTH halves of what the subscription module exposes to identity:
// the narrow provisioning contract and the read-only `IUserSubscriptionContract`
// the administrator use case already holds. Counting the reads for real - rather
// than leaving a counter nothing increments - is the difference between an
// assertion and a comment: with the read contract wired in, a future change that
// looks up the subscription to decide what to grant increments `reads` and the
// SC-006 assertions below fail.
type eagerProvisioningSpy struct {
	subscription_contract.IUserSubscriptionContract
	subscription_contract.IUserSubscriptionProvisioningContract
	uow *eagerIdentityUow
	// calls are the accounts provisioning was asked about.
	calls []uuid.UUID
	// reads counts every subscription read the creation paths performed. Nothing
	// may increment it.
	reads   int
	lastCtx context.Context
	err     error
	// rowsAtCall counts the account rows already written when provisioning ran,
	// so a test can prove the account insert came first.
	rowsAtCall int
}

func (s *eagerProvisioningSpy) EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error {
	s.calls = append(s.calls, userID)
	s.lastCtx = ctx
	s.rowsAtCall = s.uow.rowsAtCall()
	if s.err != nil {
		return s.err
	}
	// A real subscription row is invisible from here, so the grant is recorded as
	// a marker row in a second store. That is enough to prove it commits and rolls
	// back with the account.
	s.uow.stagedEntitlements[userID] = entitlementRow(userID)
	return nil
}

// GetUserSubscriptionOverview and GetUserSubscriptionOverviews are the ONLY ways
// the identity module can read subscription state, and both are counted. They
// return "absent" rather than a plan, because reaching either of them at all is
// already the defect SC-006 forbids: at account creation there is nothing to read.
func (s *eagerProvisioningSpy) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*subscriptiondto.UserSubscriptionContractDTO, error) {
	s.reads++
	return nil, nil
}

func (s *eagerProvisioningSpy) GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO, error) {
	s.reads++
	return map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO{}, nil
}

// entitlementRow builds the marker row a provisioned entitlement leaves behind.
func entitlementRow(userID uuid.UUID) *entities.User {
	row := &entities.User{Username: entitlementMarker}
	row.ID = userID
	return row
}

// entitlementMarker distinguishes a provisioned entitlement from an account row.
const entitlementMarker = "<entitlement>"

// subscriptionReadFields are the field names that would hand a use case the only
// way to READ subscription state from the subscription module.
var subscriptionReadFields = []string{"subscriptionContract", "subContract", "subscriptionUC", "subUC"}

// assertNoSubscriptionReadDependency states the SC-006 requirement for the two
// doors that have NO subscription read contract at all: `RegisterUseCase` and
// `GoogleAuthUseCase` are not given `IUserSubscriptionContract`, so no amount of
// new code inside them can read a plan before provisioning.
//
// `UserUseCase` is different - it legitimately holds the contract for the profile
// and role-update paths - so for that door the property is asserted by COUNTING
// real calls instead, which is what `spy.reads` does.
//
// This check fails the moment somebody adds the dependency, which is the point: at
// that moment the count-based assertion stops covering that door and a test
// decision has to be made deliberately instead of by accident.
func assertNoSubscriptionReadDependency(t *testing.T, uc any) {
	t.Helper()
	v := reflect.ValueOf(uc)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			t.Fatal("expected a non-nil use case to inspect")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		t.Fatalf("expected a struct use case, got %s", v.Kind())
	}
	typ := v.Type()
	for _, name := range subscriptionReadFields {
		if _, found := typ.FieldByName(name); found {
			t.Errorf("%s now holds %q: it can read subscription state, so SC-006 needs a counting spy here too, not an absent dependency", typ.Name(), name)
		}
	}
}

// --- builders ---------------------------------------------------------------

func newEagerUserUseCase(repo *eagerIdentityRepo, spy *eagerProvisioningSpy, uow *eagerIdentityUow) uc_interfaces.IUserUseCase {
	// The spy is passed as the read contract too. It was `nil` here before, which
	// is why the "no subscription READ" assertions could never fail: there was no
	// object in the use case that a read could reach.
	return usecase.NewUserUseCase(
		repo, eagerPasswordHasher{}, nil, nil, spy,
		spy, uow,
		nil, &config.Config{}, nil, nil,
	)
}

// eagerPasswordHasher stands in for bcrypt: the point of these tests is the
// transaction, not the hash.
type eagerPasswordHasher struct{}

func (eagerPasswordHasher) HashPassword(plain string) (string, error) { return "hashed", nil }
func (eagerPasswordHasher) VerifyPassword(plain, hashed string) bool {
	return true
}

func newEagerRegisterUseCase(repo *eagerIdentityRepo, spy *eagerProvisioningSpy, uow *eagerIdentityUow) (uc_interfaces.IRegisterUseCase, *eagerOtpService) {
	otpSvc := &eagerOtpService{}
	return usecase.NewRegisterUseCase(
		repo,
		otpSvc,
		&eagerNotifContract{},
		eagerPasswordHasher{},
		&config.Config{},
		spy,
		uow,
	), otpSvc
}

type eagerOtpService struct {
	identityiface.IOtpService
	data string
}

func (o *eagerOtpService) VerifyOtp(ctx context.Context, email, code, purpose string) (string, error) {
	return o.data, nil
}

func (o *eagerOtpService) IsInResendCooldown(ctx context.Context, email, purpose string) (bool, error) {
	return false, nil
}

func (o *eagerOtpService) GenerateOtp(ctx context.Context, email, data, purpose string) (string, error) {
	return "123456", nil
}

func (o *eagerOtpService) GetData(ctx context.Context, email, purpose string) (string, error) {
	return o.data, nil
}

type eagerNotifContract struct {
	notificationContract.INotificationContract
}

func (eagerNotifContract) Send(ctx context.Context, channel notificationContract.ChannelEnum, template notificationContract.TemplateEnum, params map[string]any, to string) error {
	return nil
}

func newEagerGoogleUC(repo *eagerIdentityRepo, spy *eagerProvisioningSpy, uow *eagerIdentityUow, ident *appsecurity.GoogleIdentity) uc_interfaces.IGoogleAuthUseCase {
	return usecase.NewGoogleAuthUseCase(
		repo,
		&fakeSession{},
		&eagerVerifier{ident: ident},
		&fakeStateStore{},
		&fakeBlacklist{},
		&config.Config{},
		uow,
		spy,
	)
}

type eagerVerifier struct {
	appsecurity.IGoogleTokenVerifier
	ident *appsecurity.GoogleIdentity
}

func (v *eagerVerifier) VerifyIDToken(ctx context.Context, idToken string) (*appsecurity.GoogleIdentity, error) {
	return v.ident, nil
}

// --- the administrator door, the one SC-006 names explicitly ----------------

func TestCreateUserByAdminProvisionsWithoutAnySubscriptionRead(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}

	uc := newEagerUserUseCase(repo, spy, uow)
	res, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
		Username:  "duck.nguyen",
		Email:     "duck.nguyen@example.com",
		Password:  "Str0ng-Passw0rd!",
		FirstName: "Duck",
	})
	if err != nil {
		t.Fatalf("CreateUserByAdmin failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected the created account")
	}

	if len(spy.calls) != 1 {
		t.Fatalf("provisioning must run exactly once, ran %d times", len(spy.calls))
	}
	if spy.calls[0] != res.ID {
		t.Errorf("provisioning targeted %s, want the new account %s", spy.calls[0], res.ID)
	}
	if spy.reads != 0 {
		t.Errorf("no subscription READ may be involved (SC-006), got %d", spy.reads)
	}
	if uow.commits != 1 {
		t.Errorf("the account and its entitlement must be ONE transaction, commits = %d", uow.commits)
	}
	if len(uow.accounts) != 1 || len(uow.entitlements) != 1 {
		t.Errorf("expected 1 account and 1 entitlement, got %d and %d", len(uow.accounts), len(uow.entitlements))
	}
}

// The atomicity requirement: a failure after the account row was inserted must
// leave NO entitlement for an account that does not exist. An orphan grant is
// worse than a missing one - the AI service would honour it for an account
// nobody can look up.
func TestCreateUserByAdminRollsBackTheEntitlementWhenProvisioningFails(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow, err: errors.New("outbox insert failed")}

	uc := newEagerUserUseCase(repo, spy, uow)
	res, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
		Username:  "duck.nguyen",
		Email:     "duck.nguyen@example.com",
		Password:  "Str0ng-Passw0rd!",
		FirstName: "Duck",
	})
	if err == nil {
		t.Fatal("must return an error when provisioning fails")
	}
	if res != nil {
		t.Error("must not return an account when the transaction rolled back")
	}
	if uow.rollbacks != 1 {
		t.Errorf("the transaction must roll back, rollbacks = %d", uow.rollbacks)
	}
	if uow.commits != 0 {
		t.Errorf("nothing may commit, commits = %d", uow.commits)
	}
	if len(uow.accounts) != 0 || len(uow.entitlements) != 0 {
		t.Errorf("no row may survive the rollback, got %d accounts and %d entitlements - an entitlement for a nonexistent account is the defect",
			len(uow.accounts), len(uow.entitlements))
	}
	if len(spy.calls) != 1 {
		t.Errorf("provisioning must have been attempted, calls = %d", len(spy.calls))
	}
}

// The account insert must come first: `user_subscriptions.user_id` is a foreign
// key onto `identity.users(id)`, so the reverse order would fail the FK check.
func TestCreateUserByAdminInsertsTheAccountBeforeProvisioning(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}

	uc := newEagerUserUseCase(repo, spy, uow)
	if _, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
		Username:  "duck.nguyen",
		Email:     "duck.nguyen@example.com",
		Password:  "Str0ng-Passw0rd!",
		FirstName: "Duck",
	}); err != nil {
		t.Fatalf("CreateUserByAdmin failed: %v", err)
	}

	if len(uow.order) == 0 || uow.order[0] != "create-account" {
		t.Errorf("the account must be inserted before provisioning is attempted, order = %v", uow.order)
	}
	if spy.rowsAtCall != 1 {
		t.Errorf("provisioning must see the staged account row, saw %d", spy.rowsAtCall)
	}
	if !inEagerTx(spy.lastCtx) {
		t.Error("provisioning must run inside the transaction that inserted the account")
	}
}

// --- the email sign-up door -------------------------------------------------

const eagerRegisterPayload = `{"username":"duck.nguyen","email":"duck.nguyen@example.com","passwordHash":"h","firstName":"Duck","lastName":"Nguyen","dateOfBirth":"1990-01-02"}`

func TestConfirmRegisterOtpProvisionsWithoutAnySubscriptionRead(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	uc, otpSvc := newEagerRegisterUseCase(repo, spy, uow)
	otpSvc.data = eagerRegisterPayload

	ok, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
		Email:   "duck.nguyen@example.com",
		OtpCode: "123456",
	})
	if err != nil {
		t.Fatalf("ConfirmRegisterOtp failed: %v", err)
	}
	if !ok {
		t.Fatal("expected the registration to succeed")
	}
	if len(spy.calls) != 1 {
		t.Fatalf("provisioning must run exactly once, ran %d times", len(spy.calls))
	}
	// This door holds no subscription read contract at all, so no read is possible.
	// The admin door, which does hold one, asserts the same thing by counting real
	// calls - see TestCreateUserByAdminProvisionsWithoutAnySubscriptionRead.
	assertNoSubscriptionReadDependency(t, uc)
	if spy.reads != 0 {
		t.Errorf("no subscription READ may be involved (SC-006), got %d", spy.reads)
	}
	if uow.commits != 1 || len(uow.accounts) != 1 || len(uow.entitlements) != 1 {
		t.Errorf("the account and its entitlement must commit together, commits=%d accounts=%d entitlements=%d",
			uow.commits, len(uow.accounts), len(uow.entitlements))
	}
}

func TestConfirmRegisterOtpRollsBackTheEntitlementWhenProvisioningFails(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow, err: errors.New("outbox insert failed")}
	uc, otpSvc := newEagerRegisterUseCase(repo, spy, uow)
	otpSvc.data = eagerRegisterPayload

	ok, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
		Email:   "duck.nguyen@example.com",
		OtpCode: "123456",
	})
	if err == nil {
		t.Fatal("must return an error when provisioning fails")
	}
	if ok {
		t.Error("must not report success when the transaction rolled back")
	}
	if len(uow.accounts) != 0 || len(uow.entitlements) != 0 {
		t.Errorf("no row may survive the rollback, got %d accounts and %d entitlements", len(uow.accounts), len(uow.entitlements))
	}
}

// --- the Google door, which already had a unit of work around the insert -----

func TestGoogleFirstLoginProvisionsWithoutAnySubscriptionRead(t *testing.T) {
	ctx := context.Background()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}

	uc := newEagerGoogleUC(repo, spy, uow, identityClaim())

	res, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	if len(spy.calls) != 1 {
		t.Fatalf("provisioning must run exactly once, ran %d times", len(spy.calls))
	}
	// See TestConfirmRegisterOtpProvisionsWithoutAnySubscriptionRead: this door has
	// no read contract, and that absence is the assertion.
	assertNoSubscriptionReadDependency(t, uc)
	if spy.reads != 0 {
		t.Errorf("no subscription READ may be involved (SC-006), got %d", spy.reads)
	}
	if uow.commits != 1 || len(uow.accounts) != 1 || len(uow.entitlements) != 1 {
		t.Errorf("the account and its entitlement must commit together, commits=%d accounts=%d entitlements=%d",
			uow.commits, len(uow.accounts), len(uow.entitlements))
	}
	if !inEagerTx(spy.lastCtx) {
		t.Error("provisioning must run inside the transaction that inserted the account")
	}
}

func TestGoogleFirstLoginRollsBackTheEntitlementWhenProvisioningFails(t *testing.T) {
	ctx := context.Background()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow, err: errors.New("outbox insert failed")}

	uc := newEagerGoogleUC(repo, spy, uow, identityClaim())

	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	if err == nil {
		t.Fatal("must return an error when provisioning fails")
	}
	if uow.commits != 0 {
		t.Errorf("nothing may commit, commits = %d", uow.commits)
	}
	if len(uow.accounts) != 0 || len(uow.entitlements) != 0 {
		t.Errorf("no row may survive the rollback, got %d accounts and %d entitlements", len(uow.accounts), len(uow.entitlements))
	}
}

// The auto-link door does NOT create an account, so it must not provision:
// there is already an entitlement, and a second write would race the one that
// granted it.
func TestGoogleAutoLinkDoesNotReprovision(t *testing.T) {
	ctx := context.Background()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}

	existing := &entities.User{
		Username:      "duck",
		Email:         "duck@gmail.com",
		Provider:      authprovider.Local,
		EmailVerified: true,
		RoleSlug:      roleslug.User,
		Status:        userstatus.Active,
	}
	existing.ID = uuid.New()
	repo.byEmail[existing.Email] = existing

	uc := newEagerGoogleUC(repo, spy, uow, identityClaim())

	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.NoError(t, err)
	if len(spy.calls) != 0 {
		t.Errorf("an existing account must not be re-provisioned, calls = %d", len(spy.calls))
	}
	if uow.commits != 0 {
		t.Errorf("linking a Google identity writes no account, commits = %d", uow.commits)
	}
}
