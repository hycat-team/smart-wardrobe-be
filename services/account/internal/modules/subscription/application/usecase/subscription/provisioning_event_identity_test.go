package subscription

import (
	"context"
	"encoding/json"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// ---- 024 T203: the event must describe the plan the row actually carries ----
//
// `ProvisionDefault` is `ON CONFLICT (user_id) DO NOTHING`, so the insert can
// quietly do nothing. The existence check that precedes it is a plain read taken
// BEFORE the write, and the two are not in the same snapshot: on the lazy door
// the read runs outside the unit of work (subscription_state_helper.go:72 vs
// :89), and on the eager door it runs on the caller's transaction before the
// insert (:134 vs :150).
//
// A row committed in that window - the grant path creates one for an account
// that has none, see admin_grant_batch_uc.go:401 and
// ApplySubscriptionPlanWithEvent:45-47 - is therefore kept as-is by the
// conflicted insert. The body then re-read the row and published it with
// `defaultPlan`, so the history row said the account moved to whatever it now
// carries (say `premium`) while `BuildPlanChangedPayload` filled `toPlanId` and
// `toPlanCode` from the FREE plan it had tried to insert.
//
// FR-019 requires the event identity to come from the history row. The
// receiver writes an `ai_policy_grants` row from `toPlanId`, so this mismatch is
// not cosmetic: a premium customer would be handed the free plan's policy.

// racedProvisionSubRepo models the window: the first read finds nothing, the
// insert is refused by the primary key, and the re-read returns the row another
// writer committed in between.
type racedProvisionSubRepo struct {
	fakeGrantSubRepo
	// winner is the row that appeared between the check and the insert.
	winner *entities.UserSubscription
	// reads counts GetByUserID calls, so the fake only hides the row from the
	// FIRST one - the existence check - and not from the re-read.
	reads int
}

func (r *racedProvisionSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	// ON CONFLICT (user_id) DO NOTHING: the row another writer committed wins.
	return nil
}

func (r *racedProvisionSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	r.reads++
	if r.reads == 1 {
		return nil, nil
	}
	return r.winner, nil
}

func newRacedProvisionRepo(winner *entities.UserSubscription) *racedProvisionSubRepo {
	return &racedProvisionSubRepo{
		fakeGrantSubRepo: fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{}},
		winner:           winner,
	}
}

// planChangedFields is the wire view of the published payload. The publisher's
// own type is unexported in the mapper package, and the JSON is the contract
// (see the comment on `planChangedEventPayload`), so the receiver's spelling is
// what this asserts on.
type planChangedFields struct {
	ToPlanID   string `json:"toPlanId"`
	ToPlanCode string `json:"toPlanCode"`
	EventType  string `json:"eventType"`
}

func publishedPlanChanged(t *testing.T, publisher *recordingPublisher, index int) planChangedFields {
	t.Helper()
	if len(publisher.events) <= index {
		t.Fatalf("expected at least %d published events, got %d", index+1, len(publisher.events))
	}
	raw, err := json.Marshal(publisher.events[index])
	if err != nil {
		t.Fatalf("the published payload must be serialisable: %v", err)
	}
	var fields planChangedFields
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("the published payload must be JSON: %v (raw = %s)", err, raw)
	}
	return fields
}

func historyPlanCode(t *testing.T, eventRepo *eagerEventRepo, key string) string {
	t.Helper()
	event, ok := eventRepo.byKey[key]
	if !ok {
		t.Fatalf("no history row was written for %q, keys = %v", key, eventRepo.keys)
	}
	if event.ToPlanCode == nil {
		t.Fatalf("the history row must name the plan it moved to, %q has none", key)
	}
	return *event.ToPlanCode
}

func TestProvisioningEventCarriesThePlanOfTheRowThatWasKept(t *testing.T) {
	userID := uuid.New()
	premium := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "premium",
		Name:            "Premium",
		IsActive:        true,
		PlanKind:        plankind.Lifetime,
		TierRank:        2,
	}
	// What the grant committed: the account is on premium, with the plan row
	// preloaded - `GetByUserID` preloads `SubscriptionPlan` (see
	// NewUserSubscriptionRepository relations).
	winner := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: premium.ID,
		CurrentPlanCode:    premium.Slug,
		CurrentTierRank:    premium.TierRank,
		CurrentPlanKind:    premium.PlanKind,
		Version:            3,
		SubscriptionPlan:   premium,
	}

	subRepo := newRacedProvisionRepo(winner)
	eventRepo := newEagerEventRepo()
	publisher := &recordingPublisher{}
	free := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "free",
		Name:            "Free",
		IsActive:        true,
		PlanKind:        plankind.DefaultFree,
	}
	support := NewSubscriptionStateSupport(
		subRepo,
		&provisionPlanRepo{
			fakeGrantPlanRepo: fakeGrantPlanRepo{planBySlug: map[string]*entities.SubscriptionPlan{"free": free, "premium": premium}},
			plan:              free,
		},
		&provisionQuotaRepo{},
		NewPlanChangePublisher(eventRepo, publisher, noopPlanPeriodRecorder{}),
		&eagerUow{},
	)

	if err := support.EnsureDefaultSubscription(context.Background(), userID); err != nil {
		t.Fatalf("provisioning must not fail when the insert was refused: %v", err)
	}

	// FR-019: the identity of the event is the history row's, and the payload
	// must agree with it. Before the fix the payload said "free" for an account
	// the history row recorded as premium.
	key := "DEFAULT_PROVISION:" + userID.String() + ":3"
	if got := historyPlanCode(t, eventRepo, key); got != premium.Slug {
		t.Errorf("the history row must record the plan the account ends up on, got %q", got)
	}
	fields := publishedPlanChanged(t, publisher, 0)
	if fields.ToPlanCode != premium.Slug {
		t.Errorf("the published payload must name the plan the row carries, got %q (want %q)", fields.ToPlanCode, premium.Slug)
	}
	if fields.ToPlanID != premium.ID.String() {
		t.Errorf("the published payload must carry that plan's id, got %q (want %q)", fields.ToPlanID, premium.ID)
	}
}

// The ordinary path is unchanged: when the insert really happens, the event is
// the default plan - the one the row was built from.
func TestProvisioningEventCarriesTheDefaultPlanWhenTheInsertHappened(t *testing.T) {
	h := newEagerHarness(t)

	if err := h.support.EnsureDefaultSubscription(context.Background(), h.userID); err != nil {
		t.Fatalf("EnsureDefaultSubscription failed: %v", err)
	}

	fields := publishedPlanChanged(t, h.publisher, 0)
	if fields.ToPlanCode != h.plan.Slug {
		t.Errorf("a freshly provisioned account must publish the default plan, got %q (want %q)", fields.ToPlanCode, h.plan.Slug)
	}
	if fields.ToPlanID != h.plan.ID.String() {
		t.Errorf("the published payload must carry the default plan's id, got %q (want %q)", fields.ToPlanID, h.plan.ID)
	}
}
