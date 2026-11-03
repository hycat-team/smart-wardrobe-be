package admin

import (
	"context"
	"errors"
	"net/http"
	"testing"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// fakeDepositRepo embeds the full interface and overrides only GetByID: the
// admin use case touches nothing else for these cases.
type fakeDepositRepo struct {
	repositories.IDepositTransactionRepository
	byID *entities.DepositTransaction
	err  error
}

func (f fakeDepositRepo) GetByID(_ context.Context, _ uuid.UUID) (*entities.DepositTransaction, error) {
	return f.byID, f.err
}

// fakeUserBrief embeds the identity contract and overrides the single method the
// enrichment reads. Embedding the interface (not hand-writing four methods) is
// what keeps this fake compiling when the contract grows — and the contract must
// stay untouched by this feature.
type fakeUserBrief struct {
	identity_contract.IUserBriefContract
	user  *identity_dto.UserBriefContractDTO
	err   error
	calls int
}

func (f *fakeUserBrief) GetByID(_ context.Context, _ uuid.UUID) (*identity_dto.UserBriefContractDTO, error) {
	f.calls++
	return f.user, f.err
}

// fakePlanRepo records whether the plan lookup happened at all: a wallet top-up
// names no plan, and paying for a second query nobody can use is the cost this
// check exists to prevent.
type fakePlanRepo struct {
	repositories.ISubscriptionPlanRepository
	plans []*entities.SubscriptionPlan
	err   error
	calls int
	ids   []uuid.UUID
}

func (f *fakePlanRepo) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	f.calls++
	f.ids = ids
	return f.plans, f.err
}

func newTestUseCase(deposit *fakeDepositRepo, user *fakeUserBrief, plan *fakePlanRepo) *AdminBillingUseCase {
	return &AdminBillingUseCase{
		depositRepo: *deposit,
		userBrief:   user,
		planRepo:    plan,
		log:         logger.New("test", "", "info", false),
	}
}

// The generic repository reports a miss as (nil, nil); the admin contract
// promises 404 (029 data-model §8). Without the use-case check the handler
// answered 200 with an empty body.
func TestGetTransactionDetailMissMapsToNotFound(t *testing.T) {
	uc := newTestUseCase(&fakeDepositRepo{byID: nil}, &fakeUserBrief{}, &fakePlanRepo{})

	detail, err := uc.GetTransactionDetail(context.Background(), uuid.New())
	if detail != nil {
		t.Fatalf("expected nil detail, got %+v", detail)
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %T: %v", err, err)
	}
	if appErr.Status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", appErr.Status)
	}
}

// TestGetTransactionDetailReturnsTheEntityWithItsLabels is the happy path: the
// transaction stays the ENTITY the mapper expects, and both labels are resolved
// from the ids the row already carries.
func TestGetTransactionDetailReturnsTheEntityWithItsLabels(t *testing.T) {
	planID := uuid.New()
	want := &entities.DepositTransaction{OrderCode: 4242, SubscriptionPlanID: &planID}
	user := &identity_dto.UserBriefContractDTO{ID: want.UserID, Username: "khachhang"}
	plan := &entities.SubscriptionPlan{Slug: "premium-monthly"}
	plan.ID = planID

	userRepo := &fakeUserBrief{user: user}
	planRepo := &fakePlanRepo{plans: []*entities.SubscriptionPlan{plan}}
	uc := newTestUseCase(&fakeDepositRepo{byID: want}, userRepo, planRepo)

	got, err := uc.GetTransactionDetail(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.Transaction != want {
		t.Fatalf("expected the repository entity back, got %+v", got)
	}
	if got.User != user {
		t.Errorf("user = %+v, want %+v", got.User, user)
	}
	if got.Plan != plan {
		t.Errorf("plan = %+v, want %+v", got.Plan, plan)
	}
	if userRepo.calls != 1 {
		t.Errorf("the buyer must be looked up exactly once, got %d calls", userRepo.calls)
	}
	if planRepo.calls != 1 {
		t.Errorf("the plan must be looked up exactly once, got %d calls", planRepo.calls)
	}
	if len(planRepo.ids) != 1 || planRepo.ids[0] != planID {
		t.Errorf("the plan lookup must receive the transaction's targetPlanId %s, got %v", planID, planRepo.ids)
	}
}

// TestGetTransactionDetailSkipsThePlanLookupForAWalletTopUp: a top-up has no
// target plan, so the second query must not happen at all.
func TestGetTransactionDetailSkipsThePlanLookupForAWalletTopUp(t *testing.T) {
	planRepo := &fakePlanRepo{}
	uc := newTestUseCase(&fakeDepositRepo{byID: &entities.DepositTransaction{OrderCode: 7}}, &fakeUserBrief{}, planRepo)

	got, err := uc.GetTransactionDetail(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Plan != nil {
		t.Errorf("a wallet top-up must have no plan label, got %+v", got.Plan)
	}
	if planRepo.calls != 0 {
		t.Errorf("the plan lookup must be skipped without a targetPlanId, got %d calls", planRepo.calls)
	}
}

// TestGetTransactionDetailDegradesWhenALabelLookupFails pins the decision that
// the reconciliation screen wins over label completeness: a failing lookup is
// logged and the label comes back nil, but the transaction is still answered.
// Failing here would hide the very transaction the operator came to fix.
func TestGetTransactionDetailDegradesWhenALabelLookupFails(t *testing.T) {
	planID := uuid.New()
	tx := &entities.DepositTransaction{OrderCode: 9, SubscriptionPlanID: &planID}

	userRepo := &fakeUserBrief{err: errors.New("identity unavailable")}
	planRepo := &fakePlanRepo{err: errors.New("plan catalogue unavailable")}
	uc := newTestUseCase(&fakeDepositRepo{byID: tx}, userRepo, planRepo)

	got, err := uc.GetTransactionDetail(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("a label lookup failure must not fail the detail, got %v", err)
	}
	if got == nil || got.Transaction != tx {
		t.Fatalf("the transaction must still be returned, got %+v", got)
	}
	if got.User != nil {
		t.Errorf("a failed buyer lookup must answer nil, got %+v", got.User)
	}
	if got.Plan != nil {
		t.Errorf("a failed plan lookup must answer nil, got %+v", got.Plan)
	}
}

// TestGetTransactionDetailReportsASoftDeletedBuyerAsAbsent: the identity contract
// answers (nil, nil) for a soft-deleted account. That is a missing label, not an
// error, and the id stays available on the transaction for the operator.
func TestGetTransactionDetailReportsASoftDeletedBuyerAsAbsent(t *testing.T) {
	uc := newTestUseCase(&fakeDepositRepo{byID: &entities.DepositTransaction{OrderCode: 11}}, &fakeUserBrief{user: nil}, &fakePlanRepo{})

	got, err := uc.GetTransactionDetail(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.User != nil {
		t.Errorf("a soft-deleted buyer must answer nil, got %+v", got.User)
	}
	if got.Transaction == nil {
		t.Fatal("the transaction must still be returned")
	}
}

// TestGetTransactionDetailReportsARemovedPlanAsAbsent: a plan deleted from the
// catalogue is absent from GetByIDs' result, which must not be an error either.
func TestGetTransactionDetailReportsARemovedPlanAsAbsent(t *testing.T) {
	planID := uuid.New()
	tx := &entities.DepositTransaction{OrderCode: 13, SubscriptionPlanID: &planID}
	uc := newTestUseCase(&fakeDepositRepo{byID: tx}, &fakeUserBrief{}, &fakePlanRepo{plans: nil})

	got, err := uc.GetTransactionDetail(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Plan != nil {
		t.Errorf("a removed plan must answer nil, got %+v", got.Plan)
	}
}

// TestGetTransactionDetailPropagatesARepositoryFailure keeps the ONE failure
// that must still fail: the transaction read itself. Degrading that would answer
// 200 for a row nobody read.
func TestGetTransactionDetailPropagatesARepositoryFailure(t *testing.T) {
	boom := errors.New("database down")
	uc := newTestUseCase(&fakeDepositRepo{err: boom}, &fakeUserBrief{}, &fakePlanRepo{})

	if _, err := uc.GetTransactionDetail(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Fatalf("expected the repository error back, got %v", err)
	}
}
