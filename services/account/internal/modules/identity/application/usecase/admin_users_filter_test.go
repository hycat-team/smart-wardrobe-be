package usecase_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/dto/identity"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// ---- 034: the admin user list can be narrowed to one exact account id ----
//
// The console needs to look an account up from a UUID pasted out of another
// screen. Free-text search only covers username/email/name, so without an exact
// id filter an administrator has no way to answer "who is this UUID?".
//
// Two behaviours are pinned here because both are easy to regress:
//  1. `userId` wins over `q` when both arrive (AND-ing them hides the account).
//  2. A malformed UUID is a 400, not a silent unfiltered list.

type capturingAdminUserRepo struct {
	repositories.IUserRepository
	filter repositories.UserFilter
	called bool
}

func (r *capturingAdminUserRepo) GetUsersForAdmin(_ context.Context, filter repositories.UserFilter) (*repositories.UserListResult, error) {
	r.filter = filter
	r.called = true
	return &repositories.UserListResult{}, nil
}

type noSubscriptionsContract struct {
	subscription_contract.IUserSubscriptionContract
}

func (noSubscriptionsContract) GetUserSubscriptionOverviews(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO, error) {
	return map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO{}, nil
}

// newAdminListUseCase builds the real use case with only the two dependencies
// GetUsersForAdmin touches. Every other collaborator is nil on purpose: a future
// change that reaches for one of them should panic in these tests instead of
// quietly widening what the list endpoint depends on.
func newAdminListUseCase(repo repositories.IUserRepository) uc_interfaces.IUserUseCase {
	return usecase.NewUserUseCase(
		repo, nil, nil, nil, nil, noSubscriptionsContract{}, nil, nil, nil, nil,
		&config.Config{}, nil, nil,
	)
}

func TestGetUsersForAdminFiltersByExactUserID(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	target := uuid.New()
	query := "someone-else"
	raw := "  " + target.String() + "  "

	res, err := newAdminListUseCase(repo).GetUsersForAdmin(context.Background(), dto.GetUsersQueryReq{
		Query:  &query,
		UserID: &raw,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("want a list result, got nil")
	}
	if !repo.called {
		t.Fatal("the repository must be queried")
	}
	if repo.filter.UserID == nil || *repo.filter.UserID != target {
		t.Fatalf("want the parsed user id %s, got %v", target, repo.filter.UserID)
	}
	if repo.filter.Query != nil {
		// Both filters at once would return an empty page for a valid UUID whose
		// owner's name does not contain the typed text.
		t.Errorf("an exact id filter must clear the free-text query, got %q", *repo.filter.Query)
	}
}

func TestGetUsersForAdminRejectsAMalformedUserID(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	raw := "not-a-uuid"

	_, err := newAdminListUseCase(repo).GetUsersForAdmin(context.Background(), dto.GetUsersQueryReq{
		UserID: &raw,
	})
	if err == nil {
		t.Fatal("a malformed user id must be rejected")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want a 400 apperror, got %T: %v", err, err)
	}
	if !errors.Is(err, identityerrors.ErrInvalidUserIDFilter()) {
		t.Errorf("want the identity module's own factory, got %v", err)
	}
	if repo.called {
		t.Error("a malformed user id must not reach the repository")
	}
}

func TestGetUsersForAdminKeepsFreeTextSearchWithoutUserID(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	query := "duck"

	if _, err := newAdminListUseCase(repo).GetUsersForAdmin(context.Background(), dto.GetUsersQueryReq{
		Query: &query,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.filter.UserID != nil {
		t.Errorf("want no id filter, got %v", repo.filter.UserID)
	}
	if repo.filter.Query == nil || *repo.filter.Query != query {
		t.Errorf("want the free-text query preserved, got %v", repo.filter.Query)
	}
}

// ---- the membership-plan filter ----
//
// Identity owns the account table only, so `planId` is resolved by the
// subscription contract into a set of user ids and then applied as an IN
// filter. Three behaviours matter and are pinned here:
//  1. the resolved ids reach the repository (and the request is NOT dropped);
//  2. a malformed uuid is a 400 that never reaches the repository;
//  3. a contract failure is a failure — answering with the unfiltered list
//     would read as "every account is on this plan".

type planFilterContract struct {
	subscription_contract.IUserSubscriptionPlanFilter
	ids      []uuid.UUID
	err      error
	calls    int
	lastPlan uuid.UUID
}

func (c *planFilterContract) ListUserIDsWithEffectivePlan(_ context.Context, planID uuid.UUID) ([]uuid.UUID, error) {
	c.calls++
	c.lastPlan = planID
	return c.ids, c.err
}

func (planFilterContract) GetUserSubscriptionOverviews(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO, error) {
	return map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO{}, nil
}

func newAdminListUseCaseWithContract(repo repositories.IUserRepository, planFilter subscription_contract.IUserSubscriptionPlanFilter) uc_interfaces.IUserUseCase {
	return usecase.NewUserUseCase(
		repo, nil, nil, nil, nil, noSubscriptionsContract{}, nil, planFilter, nil, nil,
		&config.Config{}, nil, nil,
	)
}

func TestGetUsersForAdminAppliesTheResolvedPlanFilter(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	planID := uuid.New()
	matched := []uuid.UUID{uuid.New(), uuid.New()}
	contract := &planFilterContract{ids: matched}
	raw := planID.String()

	if _, err := newAdminListUseCaseWithContract(repo, contract).GetUsersForAdmin(
		context.Background(), dto.GetUsersQueryReq{PlanID: &raw},
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if contract.calls != 1 || contract.lastPlan != planID {
		t.Fatalf("the contract must resolve plan %s once, got %d calls for %s",
			planID, contract.calls, contract.lastPlan)
	}
	if len(repo.filter.UserIDs) != len(matched) {
		t.Fatalf("want %d resolved ids on the filter, got %v", len(matched), repo.filter.UserIDs)
	}
	for idx, id := range matched {
		if repo.filter.UserIDs[idx] != id {
			t.Errorf("resolved id[%d] = %s, want %s", idx, repo.filter.UserIDs[idx], id)
		}
	}
}

// A plan nobody is on is an EMPTY SET, not "no filter": nil means the caller
// did not filter at all, and conflating the two shows every account.
func TestGetUsersForAdminKeepsAnEmptyPlanResultAsAnEmptySet(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	contract := &planFilterContract{ids: nil}
	raw := uuid.New().String()

	if _, err := newAdminListUseCaseWithContract(repo, contract).GetUsersForAdmin(
		context.Background(), dto.GetUsersQueryReq{PlanID: &raw},
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.filter.UserIDs == nil {
		t.Fatal("an empty contract result must stay a non-nil empty set")
	}
	if len(repo.filter.UserIDs) != 0 {
		t.Errorf("want no resolved ids, got %v", repo.filter.UserIDs)
	}
}

func TestGetUsersForAdminRejectsAMalformedPlanID(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	contract := &planFilterContract{}
	raw := "not-a-uuid"

	_, err := newAdminListUseCaseWithContract(repo, contract).GetUsersForAdmin(
		context.Background(), dto.GetUsersQueryReq{PlanID: &raw},
	)
	if err == nil {
		t.Fatal("a malformed plan id must be rejected")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Fatalf("want a 400 apperror, got %T: %v", err, err)
	}
	if !errors.Is(err, identityerrors.ErrInvalidPlanIDFilter()) {
		t.Errorf("want the identity module's own factory, got %v", err)
	}
	if repo.called {
		t.Error("a malformed plan id must not reach the repository")
	}
	if contract.calls != 0 {
		t.Error("a malformed plan id must not reach the subscription contract")
	}
}

func TestGetUsersForAdminPropagatesAPlanLookupFailure(t *testing.T) {
	repo := &capturingAdminUserRepo{}
	contract := &planFilterContract{err: errors.New("subscription down")}
	raw := uuid.New().String()

	_, err := newAdminListUseCaseWithContract(repo, contract).GetUsersForAdmin(
		context.Background(), dto.GetUsersQueryReq{PlanID: &raw},
	)
	if err == nil {
		t.Fatal("a failing plan lookup must not degrade into an unfiltered list")
	}
	if repo.called {
		t.Error("the repository must not run when the plan filter could not be resolved")
	}
}
