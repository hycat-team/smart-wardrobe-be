package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	shrepos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/messaging"

	"gorm.io/gorm"
)

// TestIntegrationCancelledCallerContextStopsEveryTxRepository is the regression
// test for 024 T149.
//
// A unit of work used to hand repositories a transaction handle that was live but
// detached: GORM clones the handle with `Session{Context: db.Statement.Context}`,
// and on a `gorm.Open` handle that context is `context.Background()`. So
// `txCtx.Err()` reported the caller's deadline while every query issued straight
// off `GetTx(txCtx)` ran with no deadline at all.
//
// The visible symptom is the worst kind: the caller abandons a request at its
// 60 second deadline, reports a timeout, and the abandoned database work keeps
// going and commits anyway.
//
// `GenericRepository.GetDB` already re-attached the caller with
// `tx.WithContext(ctx)`. The five repositories that take the transaction from the
// context and use it directly did not. This test walks all five, because "the
// shared fix is in place" and "each repository actually honours cancellation" are
// different claims, and only the second one is worth anything to a caller.
//
// The context is cancelled INSIDE the unit of work, after the transaction has
// opened. Cancelling beforehand would be a weaker test: it would fail at Begin and
// never reach a repository.
func TestIntegrationCancelledCallerContextStopsEveryTxRepository(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	uow := db.NewGormUnitOfWork(dbConn)
	set := contextProbeRepos(dbConn)

	for _, probe := range contextProbeList(set) {
		t.Run(probe.name, func(t *testing.T) {
			// Positive control first: with a live context the same call must not
			// report cancellation. Without it, a repository failing for some
			// unrelated reason could satisfy the assertion below for the wrong
			// reason.
			liveErr := probe.run(context.Background())
			if errors.Is(liveErr, context.Canceled) || errors.Is(liveErr, context.DeadlineExceeded) {
				t.Fatalf("live context reported cancellation, probe is not isolating the bug: %v", liveErr)
			}

			ctx, cancel := context.WithCancel(context.Background())
			err := uow.Execute(ctx, func(txCtx context.Context) error {
				// The caller gives up the moment the transaction opens. Everything
				// the callback does from here on must observe that.
				cancel()
				return probe.run(txCtx)
			})

			if !errors.Is(err, context.Canceled) {
				t.Fatalf("query survived a cancelled caller context: err=%v", err)
			}
		})
	}
}

// TestIntegrationTxHandleCarriesCallerDeadline pins the mechanism rather than the
// symptom: the handle inside the unit of work must BE the caller's context.
//
// It needs no query, so it fails for exactly one reason and points straight at
// InjectTx. The behavioural test above is what proves the consequence; this is what
// stops a future refactor from quietly reverting it.
func TestIntegrationTxHandleCarriesCallerDeadline(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	uow := db.NewGormUnitOfWork(dbConn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var handleIsCaller, hasDeadline bool
	if err := uow.Execute(ctx, func(txCtx context.Context) error {
		tx := db.GetTx(txCtx)
		if tx == nil {
			t.Fatal("no transaction in context inside Execute")
		}
		handleIsCaller = tx.Statement.Context == ctx
		_, hasDeadline = tx.Statement.Context.Deadline()
		return nil
	}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !hasDeadline {
		t.Fatal("transaction handle carries no deadline: the caller's context never reached it")
	}
	if !handleIsCaller {
		t.Fatal("transaction handle is not the caller's context")
	}
}

// contextProbeRepos holds the five repositories that take the transaction out of
// the context and use it directly.
type contextProbeRepoSet struct {
	outbox  messaging.OutboxRepository
	audit   repositories.IUserSubscriptionEventRepository
	attempt repositories.ISubscriptionRenewalAttemptRepository
	payment repositories.IPaymentEventRepository
	inbox   repositories.IWebhookInboxRepository
}

func contextProbeRepos(dbConn *gorm.DB) contextProbeRepoSet {
	return contextProbeRepoSet{
		// Create is the outbox method that joins the caller's transaction. The
		// others deliberately use baseDB.WithContext instead: they are background
		// worker operations that must not run inside a business transaction, and
		// they already attach a context of their own.
		outbox:  messaging.NewOutboxRepository(dbConn),
		audit:   persistence.NewUserSubscriptionEventRepository(dbConn),
		attempt: persistence.NewSubscriptionRenewalAttemptRepository(dbConn),
		payment: persistence.NewPaymentEventRepository(dbConn),
		inbox:   persistence.NewWebhookInboxRepository(dbConn),
	}
}

type contextProbe struct {
	name string
	run  func(ctx context.Context) error
}

func contextProbeList(set contextProbeRepoSet) []contextProbe {
	return []contextProbe{
		{
			name: "outbox.Create",
			run: func(ctx context.Context) error {
				return set.outbox.Create(ctx, &shrepos.OutboxRecord{
					Topic:   "subscription.plan.changed",
					Payload: shrepos.JSONField(`{}`),
				})
			},
		},
		{
			name: "UserSubscriptionEventRepository.GetByKey",
			run: func(ctx context.Context) error {
				_, err := set.audit.GetByKey(ctx, "PROBE:no-such-key")
				return err
			},
		},
		{
			name: "SubscriptionRenewalAttemptRepository.GetByKey",
			run: func(ctx context.Context) error {
				_, err := set.attempt.GetByKey(ctx, "PROBE:no-such-key")
				return err
			},
		},
		{
			name: "PaymentEventRepository.GetByReference",
			run: func(ctx context.Context) error {
				_, err := set.payment.GetByReference(ctx, "PAYOS", "PROBE-no-such-reference")
				return err
			},
		},
		{
			name: "WebhookInboxRepository.GetByHash",
			run: func(ctx context.Context) error {
				_, err := set.inbox.GetByHash(ctx, "PAYOS", "PROBE-no-such-hash")
				return err
			},
		},
	}
}
