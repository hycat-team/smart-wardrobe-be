package subscription

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 024 T155.
//
// `BuildPlanChangedPayload` cannot fail - it returns a value - so with no plan row it
// produced a payload whose `toPlanId` was empty and the publisher sent it anyway.
//
// That was not a harmless gap. T137 made the AI receiver REFUSE such messages by
// design (FR-022/FR-023): it will not write a grant row with a blank plan id. So the
// sender was manufacturing messages that could only ever be discarded, and FR-012's
// "every change publishes an event" had quietly become "every change publishes an
// event nobody can act on" - with no log, no metric and no error on either side.
//
// The fix is at the publisher, not at the twelve call sites, so a new caller cannot
// forget.

// A payload with no plan row must never reach the broker.
func TestPublishPlanChangeRefusesPayloadWithoutPlanID(t *testing.T) {
	eventRepo := &grantTxEventRepo{}
	publisher := &fakeRenewalPublisher{}
	changePublisher := NewPlanChangePublisher(eventRepo, publisher, noopPlanPeriodRecorder{})

	userID := uuid.New()
	planCode := "free"
	tierRank := 0
	event := &entities.UserSubscriptionEvent{
		EventKey:    "T155:no-plan",
		UserID:      userID,
		EventType:   "ACTIVATED",
		ToPlanCode:  &planCode,
		ToTierRank:  &tierRank,
		OccurredAt:  time.Now().UTC(),
		EffectiveAt: time.Now().UTC(),
	}

	// plan == nil is the branch that produced the unusable payload.
	_, err := changePublisher.PublishPlanChange(context.Background(), event, &entities.UserSubscription{}, nil)
	if err == nil {
		t.Fatal("publisher phải từ chối payload thiếu toPlanId thay vì gửi thông điệp không ai dùng được")
	}
	if !strings.Contains(err.Error(), "toPlanId") {
		t.Errorf("lỗi phải nói rõ thiếu toPlanId, nhận: %v", err)
	}
	if len(publisher.events) != 0 {
		t.Errorf("không được publish thông điệp không dùng được, đã gửi %d", len(publisher.events))
	}
}

// A plan row whose id was never assigned is the same defect wearing a disguise: the
// nil UUID stringifies to a non-empty string, so an emptiness check alone lets it
// through and it lands in the receiver's grant table looking like a real identifier.
// The validator's own coverage of that case lives in the mapper package, where the
// unexported helper is reachable.
func TestNilPlanProducesPayloadTheValidatorRejects(t *testing.T) {
	event := &entities.UserSubscriptionEvent{UserID: uuid.New()}
	payload := mapper.BuildPlanChangedPayload(event, &entities.UserSubscription{}, nil)

	// This is the exact payload the old code published.
	if err := mapper.ValidatePlanChangedPayload(payload); err == nil {
		t.Error("payload dựng từ plan nil phải bị validator từ chối")
	}
}

// The normal path must be untouched: a plan read from the database carries an id, and
// that event must still publish.
func TestPublishPlanChangeStillPublishesRealPlan(t *testing.T) {
	eventRepo := &grantTxEventRepo{}
	publisher := &fakeRenewalPublisher{}
	changePublisher := NewPlanChangePublisher(eventRepo, publisher, noopPlanPeriodRecorder{})

	plan := renewalTestPlan()
	userID := uuid.New()
	planCode := plan.Slug
	tierRank := plan.TierRank
	event := &entities.UserSubscriptionEvent{
		EventKey:    "T155:real-plan",
		UserID:      userID,
		EventType:   "ACTIVATED",
		ToPlanCode:  &planCode,
		ToTierRank:  &tierRank,
		OccurredAt:  time.Now().UTC(),
		EffectiveAt: time.Now().UTC(),
	}

	if _, err := changePublisher.PublishPlanChange(context.Background(), event, &entities.UserSubscription{}, plan); err != nil {
		t.Fatalf("payload hợp lệ phải publish được: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Errorf("phải publish đúng 1 thông điệp, có %d", len(publisher.events))
	}
}
