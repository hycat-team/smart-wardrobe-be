package mapper

import (
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 024 T155: "absent" has two spellings, and checking only one of them lets the other
// through.
//
// An empty string is the obvious case - no plan row at all. The nil UUID is the more
// dangerous one: it stringifies to a non-empty string, so a plain emptiness check
// accepts it, and it then reaches the AI receiver looking exactly like a real plan
// identifier. That is how an unusable event becomes a corrupt grant row instead of a
// clean rejection with a reason.
func TestIsUnsetUUIDTreatsBothSpellingsOfAbsent(t *testing.T) {
	cases := []struct {
		name      string
		value     string
		wantIsSet bool
	}{
		{name: "chuỗi rỗng", value: "", wantIsSet: false},
		{name: "nil UUID", value: uuid.Nil.String(), wantIsSet: false},
		{name: "không phải UUID", value: "free-plan", wantIsSet: false},
		{name: "UUID thật", value: uuid.New().String(), wantIsSet: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := !isUnsetUUID(tc.value); got != tc.wantIsSet {
				t.Errorf("isUnsetUUID(%q) => co định danh = %v, mong đợi %v", tc.value, got, tc.wantIsSet)
			}
		})
	}
}

// A payload carrying real ids must pass, or the guard would refuse every legitimate
// event.
func TestValidateAcceptsAWellFormedPayload(t *testing.T) {
	plan := premiumPlan()
	plan.ID = uuid.New()
	event := &entities.UserSubscriptionEvent{
		UserID:      uuid.New(),
		OccurredAt:  time.Now().UTC(),
		EffectiveAt: time.Now().UTC(),
	}
	payload := BuildPlanChangedPayload(event, &entities.UserSubscription{}, plan)
	payload.SourceEventID = uuid.New().String()

	if err := ValidatePlanChangedPayload(payload); err != nil {
		t.Errorf("payload hợp lệ phải được chấp nhận, nhận lỗi: %v", err)
	}
}

// The event id is assigned by GORM during insert. Before that it is the nil UUID, and
// the receiver cannot de-duplicate without it.
func TestValidateRejectsPayloadWithoutSourceEventID(t *testing.T) {
	plan := premiumPlan()
	plan.ID = uuid.New()
	event := &entities.UserSubscriptionEvent{
		UserID:      uuid.New(),
		OccurredAt:  time.Now().UTC(),
		EffectiveAt: time.Now().UTC(),
	}
	payload := BuildPlanChangedPayload(event, &entities.UserSubscription{}, plan)

	if err := ValidatePlanChangedPayload(payload); err == nil {
		t.Error("payload chưa qua insert (chưa có id sự kiện) phải bị từ chối")
	}
}

// The exact payload the old code published: a plan row of nil. It must be rejected, and
// the reason has to name the missing field so an operator reading the log knows where
// to look.
func TestValidateRejectsPayloadBuiltFromNilPlan(t *testing.T) {
	event := &entities.UserSubscriptionEvent{
		UserID:      uuid.New(),
		OccurredAt:  time.Now().UTC(),
		EffectiveAt: time.Now().UTC(),
	}
	payload := BuildPlanChangedPayload(event, &entities.UserSubscription{}, nil)

	err := ValidatePlanChangedPayload(payload)
	if err == nil {
		t.Fatal("payload dựng từ plan nil phải bị từ chối")
	}
	if !strings.Contains(err.Error(), "toPlanId") {
		t.Errorf("lý do phải nêu toPlanId, nhận: %q", err.Error())
	}
}
