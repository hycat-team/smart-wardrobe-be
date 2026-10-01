package contract

import (
	"context"
	"errors"
	"net/http"
	"testing"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ---- 024 T202: the adapter refuses to provision outside a transaction ----
//
// `EnsureDefaultSubscription` is documented as having to run on the caller's
// transaction, and nothing checked. A caller that got the context wrong still got
// a nil error and still got three writes - each on the base connection, each in
// its own implicit transaction. That is the failure 024 T126 fixed on the lazy
// door, reintroduced on the eager one, and it is silent: the account ends up
// holding a plan whose history row and outbox record are independent commits.
//
// The guard belongs HERE, in the adapter, and not in `SubscriptionStateSupport`.
// All three account-creation doors receive the same instance from the Wire graph
// and pass their transaction context straight through, so one check at this layer
// covers every production path; a check inside the use case would cover less and
// would also reject the callers that legitimately reach it without a transaction.

type fakeSubscriptionUseCase struct {
	uc_interfaces.ISubscriptionUseCase
	calls   []uuid.UUID
	lastCtx context.Context
	err     error
}

func (f *fakeSubscriptionUseCase) EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error {
	f.calls = append(f.calls, userID)
	f.lastCtx = ctx
	return f.err
}

// A bare context must be refused outright, and nothing downstream may run. The
// assertion that matters is the second one: an implementation that validated and
// then proceeded anyway would leave the three orphaned writes this exists to
// prevent, so "the use case was never called" is the property, not "it returned
// an error".
func TestEnsureDefaultSubscriptionRefusesACtxWithoutTransaction(t *testing.T) {
	subUC := &fakeSubscriptionUseCase{}
	contract := NewUserSubscriptionProvisioningContract(subUC)
	userID := uuid.New()

	err := contract.EnsureDefaultSubscription(context.Background(), userID)
	if err == nil {
		t.Fatal("must fail when the context carries no transaction")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("must return an apperror, got %T: %v", err, err)
	}
	if appErr.Status != http.StatusInternalServerError {
		t.Errorf("a programming error must not be reported as a client error, got status %d", appErr.Status)
	}
	if appErr.Message == "" {
		t.Error("the error must carry a message a developer can act on")
	}
	if !errors.Is(err, subscriptionerrors.ErrProvisioningRequiresCallerTransaction()) {
		t.Errorf("must return the documented error, got %q", appErr.Message)
	}

	if len(subUC.calls) != 0 {
		t.Fatalf("the provisioning body must not run at all, it was called %d times with %v", len(subUC.calls), subUC.calls)
	}
}

// The positive control: with a transaction present the same call must reach the
// use case untouched. Without this, "refuses a bare context" would also be
// satisfied by a contract that refuses everything.
func TestEnsureDefaultSubscriptionRunsWhenTheContextCarriesATransaction(t *testing.T) {
	subUC := &fakeSubscriptionUseCase{}
	contract := NewUserSubscriptionProvisioningContract(subUC)
	userID := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	txCtx := db.InjectTx(ctx, newDiscardTransaction(t))

	if err := contract.EnsureDefaultSubscription(txCtx, userID); err != nil {
		t.Fatalf("a transaction context must be accepted, got %v", err)
	}
	if len(subUC.calls) != 1 || subUC.calls[0] != userID {
		t.Fatalf("expected the use case to be called once for %s, got %v", userID, subUC.calls)
	}
	if subUC.lastCtx != txCtx {
		t.Error("the caller's context must be handed to the use case unchanged - rewrapping it would detach the transaction")
	}
}

// A nil transaction is what `db.InjectTx(ctx, nil)` produces, and the
// repositories treat it as "no transaction here". A contract that accepted it
// would be checking one thing and letting another through.
func TestEnsureDefaultSubscriptionRefusesANilTransaction(t *testing.T) {
	subUC := &fakeSubscriptionUseCase{}
	contract := NewUserSubscriptionProvisioningContract(subUC)

	txCtx := db.InjectTx(context.Background(), nil)
	if err := contract.EnsureDefaultSubscription(txCtx, uuid.New()); err == nil {
		t.Fatal("must fail when the context carries a nil transaction")
	}
	if len(subUC.calls) != 0 {
		t.Fatalf("the provisioning body must not run, it was called %d times", len(subUC.calls))
	}
}

// An upstream failure must pass through unchanged - the account-creation doors
// translate it into "could not create the account", and a guard error here would
// be indistinguishable from a broken plan catalogue.
func TestEnsureDefaultSubscriptionPropagatesTheUseCaseError(t *testing.T) {
	cause := errors.New("outbox insert failed")
	subUC := &fakeSubscriptionUseCase{err: cause}
	contract := NewUserSubscriptionProvisioningContract(subUC)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	txCtx := db.InjectTx(ctx, newDiscardTransaction(t))

	err := contract.EnsureDefaultSubscription(txCtx, uuid.New())
	if !errors.Is(err, cause) {
		t.Fatalf("the use case error must reach the caller unchanged, got %v", err)
	}
}

// newDiscardTransaction returns a transaction handle that never talks to a server.
//
// The guard reads only the transaction's PRESENCE - `db.GetTx` is a context lookup
// - and the use case above is a fake that only records the context it was handed.
// But `db.InjectTx` itself calls `tx.WithContext(ctx)`, so the handle has to be a
// real, initialised `*gorm.DB`. GORM opens lazily and automatic ping is disabled
// here, so nothing connects and no live Postgres is required.
func newDiscardTransaction(t *testing.T) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(
		postgres.New(postgres.Config{DSN: "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable"}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	if err != nil {
		t.Fatalf("build discard transaction handle: %v", err)
	}
	return conn
}
