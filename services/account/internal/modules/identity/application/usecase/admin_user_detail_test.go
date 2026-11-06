package usecase_test

import (
	"context"
	"encoding/json"
	"testing"

	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// ---- 034: the admin detail endpoint answers with ONE subscription shape ----
//
// `GET /admin/users/{id}` used to wrap the user in AdminUserDetailRes and repeat
// the subscription beside it as an untyped `any`. The two copies could disagree
// (the envelope carried the full contract DTO, the user carried the identity
// projection), and every reader had to choose one.
//
// The endpoint now returns UserRes, whose embedded `subscription` is the same
// projection the list endpoint uses. These assertions pin that shape:
//   - the JSON has no `user` envelope,
//   - `subscription` sits directly on the returned object,
//   - auto-renew and personalization survived the consolidation.

type detailUserRepo struct {
	repositories.IUserRepository
	user *entities.User
}

func (r *detailUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*entities.User, error) {
	return r.user, nil
}

type detailSubscriptionContract struct {
	subscription_contract.IUserSubscriptionContract
	overview *subscriptiondto.UserSubscriptionContractDTO
}

func (c detailSubscriptionContract) GetUserSubscriptionOverview(_ context.Context, _ uuid.UUID) (*subscriptiondto.UserSubscriptionContractDTO, error) {
	return c.overview, nil
}

func newDetailUseCase(repo repositories.IUserRepository, contract subscription_contract.IUserSubscriptionContract) uc_interfaces.IUserUseCase {
	return usecase.NewUserUseCase(
		repo, nil, nil, nil, nil, contract, nil, nil, nil, nil,
		&config.Config{}, nil, nil,
	)
}

func TestGetUserDetailReturnsUserResWithEmbeddedSubscription(t *testing.T) {
	userID := uuid.New()
	repo := &detailUserRepo{user: &entities.User{
		Username: "duck.nguyen",
		Email:    "duck.nguyen@example.com",
		RoleSlug: roleslug.User,
		Status:   userstatus.Active,
	}}
	repo.user.ID = userID

	contract := detailSubscriptionContract{overview: &subscriptiondto.UserSubscriptionContractDTO{
		PlanID:                 uuid.New(),
		PlanName:               "Premium Monthly",
		PlanSlug:               "premium-monthly",
		IsAutoRenewEnabled:     true,
		MaxWardrobeItems:       300,
		MaxOutfits:             300,
		AiOutfitDailyQuota:     30,
		AiChatDailyQuota:       10,
		PersonalizationEnabled: true,
	}}

	res, err := newDetailUseCase(repo, contract).GetUserDetail(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("want a user detail, got nil")
	}

	if res.ID != userID {
		t.Errorf("want the requested user, got %s", res.ID)
	}
	if res.Subscription.PlanName != "Premium Monthly" {
		t.Errorf("want the effective plan inside user.subscription, got %q", res.Subscription.PlanName)
	}
	if !res.Subscription.IsAutoRenewEnabled {
		t.Error("auto-renew must survive the consolidation")
	}
	if !res.Subscription.PersonalizationEnabled {
		t.Error("personalization entitlement must survive the consolidation")
	}
	if res.Subscription.AiOutfitDailyQuota != 30 {
		t.Errorf("want the plan quota, got %d", res.Subscription.AiOutfitDailyQuota)
	}

	encoded, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal detail: %v", err)
	}
	if _, hasEnvelope := payload["user"]; hasEnvelope {
		t.Error("the { user, subscription } envelope must be gone")
	}
	if _, hasSubscription := payload["subscription"]; !hasSubscription {
		t.Error("subscription must sit directly on UserRes")
	}
}
