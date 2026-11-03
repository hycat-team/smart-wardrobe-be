package cost

import (
	"context"
	"errors"
	"testing"
	"time"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"

	"github.com/google/uuid"
)

// admissionCostRepo is a spy for the non-monetary admission path. It records
// every call and mirrors the production cap/upsert contract closely enough for
// the use case tests: a row that already exists is updated in place, and the
// paid attempt cap is only consumed when a new paid row is written.
type admissionCostRepo struct {
	repositories.IAICostRepository

	grant      *aiexecutor_entities.AIPolicyGrant
	policy     *aiexecutor_entities.AICostPolicy
	op         *aiexecutor_entities.AICostPolicyOperation
	resolveErr error

	events                map[uuid.UUID]*aiexecutor_entities.AIUsageEvent
	admitCalls            int
	paidAttempts          int
	lastCap               *int
	reserveCalls          int
	prepareBillableCalls  int
	preparedBillableEvent *aiexecutor_entities.AIUsageEvent
}

func (r *admissionCostRepo) ResolvePolicy(context.Context, uuid.UUID, aioperation.AIOperation, time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, error) {
	return r.grant, r.policy, r.op, r.resolveErr
}

func (r *admissionCostRepo) Reserve(context.Context, *aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, *int, *aiexecutor_entities.AIUsageEvent, time.Time) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error) {
	r.reserveCalls++
	return &aiexecutor_entities.AIUsagePeriodLedger{}, true, nil
}

func (r *admissionCostRepo) CreateEvent(_ context.Context, event *aiexecutor_entities.AIUsageEvent) error {
	if r.events == nil {
		r.events = make(map[uuid.UUID]*aiexecutor_entities.AIUsageEvent)
	}
	r.events[event.RequestID] = event
	return nil
}

func (r *admissionCostRepo) AdmitUnbilledAttempt(_ context.Context, policy *aiexecutor_entities.AICostPolicy, cap *int, event *aiexecutor_entities.AIUsageEvent, _ time.Time) (bool, error) {
	r.admitCalls++
	r.lastCap = cap
	if r.events == nil {
		r.events = make(map[uuid.UUID]*aiexecutor_entities.AIUsageEvent)
	}
	if existing, ok := r.events[event.RequestID]; ok {
		// Upsert: keep the route the first call decided and do not count the cap
		// again.
		existing.Provider = event.Provider
		existing.Model = event.Model
		return true, nil
	}
	if event.LogicalRoute == airoute.Paid && policy != nil && policy.EnforcementMode == aienforcementmode.ObserveOnly {
		if cap == nil || r.paidAttempts >= *cap {
			return false, nil
		}
		r.paidAttempts++
	}
	row := *event
	r.events[event.RequestID] = &row
	return true, nil
}

func (r *admissionCostRepo) PrepareBillableAttempt(_ context.Context, _ *aiexecutor_entities.AIPolicyGrant, _ *aiexecutor_entities.AICostPolicy, event *aiexecutor_entities.AIUsageEvent, _ time.Time) (bool, error) {
	r.prepareBillableCalls++
	r.preparedBillableEvent = event
	return true, nil
}

func admissionFixture(mode aienforcementmode.AIEnforcementMode, cap *int, route airoute.AIRoute) (*admissionCostRepo, *recordingCostRepo) {
	userID := uuid.New()
	op := &aiexecutor_entities.AICostPolicyOperation{
		Operation:             aioperation.Chat,
		MaxPaidAttemptsPerDay: cap,
		NormalRoute:           route,
		FreeRoute:             airoute.Free,
	}
	policy := &aiexecutor_entities.AICostPolicy{
		EnforcementMode: mode,
		PeriodDays:      30,
		Operations:      []*aiexecutor_entities.AICostPolicyOperation{op},
	}
	grant := &aiexecutor_entities.AIPolicyGrant{UserID: userID}
	repo := &admissionCostRepo{grant: grant, policy: policy, op: op, events: make(map[uuid.UUID]*aiexecutor_entities.AIUsageEvent)}
	return repo, &recordingCostRepo{}
}

func admissionPlan(route airoute.AIRoute) *uc_interfaces.ExecutionPlan {
	return &uc_interfaces.ExecutionPlan{
		RequestID:             uuid.New(),
		UserID:                uuid.New(),
		Operation:             aioperation.Chat,
		Provider:              "openai",
		Model:                 "qwen3:4b",
		Route:                 route,
		MaxInputTokens:        3000,
		MaxOutputTokens:       500,
		EstimatedPromptTokens: 120,
	}
}

func intPtr(v int) *int { return &v }

// A provider=openai attempt never touches pricing or the reservation ledger.
func TestAdmitUnbilledNeverConsultsPricingOrReserve(t *testing.T) {
	cap := 3
	repo, recorder := admissionFixture(aienforcementmode.ObserveOnly, &cap, airoute.Paid)
	pricing := &stubPricingUC{err: errors.New("pricing must not be consulted for provider=openai")}
	uc := &CostUseCase{repo: repo, pricingUC: pricing}

	plan := admissionPlan(airoute.Paid)
	decision, err := uc.AdmitUnbilledAttempt(context.Background(), uc_interfaces.UnbilledAttemptAdmission{
		Plan: plan, AttemptIndex: 1, Provider: "openai", Model: "qwen3:4b",
	})
	if err != nil {
		t.Fatalf("AdmitUnbilledAttempt: %v", err)
	}
	if decision.Route != airoute.Paid {
		t.Errorf("route = %q, want %q", decision.Route, airoute.Paid)
	}
	if recorder.reserveCalls != 0 || repo.reserveCalls != 0 {
		t.Error("unbilled admission must not call Reserve")
	}
	if pricing.gotProvider != "" || pricing.gotModel != "" {
		t.Errorf("unbilled admission must not call pricing, got %q/%q", pricing.gotProvider, pricing.gotModel)
	}
	event := repo.events[plan.RequestID]
	if event == nil {
		t.Fatal("unbilled admission must write a usage event")
	}
	if event.LogicalRoute != airoute.Paid {
		t.Errorf("event route = %q, want the plan route paid", event.LogicalRoute)
	}
	if event.ReservedCostMicroVND != 0 || event.ActualCostMicroVND != 0 {
		t.Errorf("unbilled event cost = %d/%d, want 0/0", event.ReservedCostMicroVND, event.ActualCostMicroVND)
	}
}

// observe_only consumes the existing daily paid-attempt cap and degrades to the
// free route when it is exhausted instead of returning a new 429.
func TestAdmitUnbilledObserveOnlyConsumesPaidCapAndDegrades(t *testing.T) {
	cap := 1
	repo, _ := admissionFixture(aienforcementmode.ObserveOnly, &cap, airoute.Paid)
	repo.paidAttempts = 1
	pricing := &stubPricingUC{err: errors.New("pricing must not be consulted")}
	uc := &CostUseCase{repo: repo, pricingUC: pricing}

	plan := admissionPlan(airoute.Paid)
	decision, err := uc.AdmitUnbilledAttempt(context.Background(), uc_interfaces.UnbilledAttemptAdmission{
		Plan: plan, AttemptIndex: 1, Provider: "openai", Model: "qwen3:4b",
	})
	if err != nil {
		t.Fatalf("cap exhaustion must not return an error (no new 429): %v", err)
	}
	if decision.Route != airoute.Free {
		t.Errorf("degraded route = %q, want free", decision.Route)
	}
	if repo.admitCalls != 2 {
		t.Errorf("admission calls = %d, want 2 (paid refused, free admitted)", repo.admitCalls)
	}
	event := repo.events[plan.RequestID]
	if event == nil {
		t.Fatal("degraded admission must still write an event")
	}
	if event.LogicalRoute != airoute.Free {
		t.Errorf("degraded event route = %q, want free", event.LogicalRoute)
	}
	if repo.paidAttempts != 1 {
		t.Errorf("paid attempts = %d, want unchanged 1", repo.paidAttempts)
	}
}

// strict has no attempt cap and always admits the unbilled attempt.
func TestAdmitUnbilledStrictHasNoAttemptCap(t *testing.T) {
	repo, _ := admissionFixture(aienforcementmode.Strict, nil, airoute.Paid)
	uc := &CostUseCase{repo: repo, pricingUC: &stubPricingUC{err: errors.New("pricing must not be consulted")}}

	plan := admissionPlan(airoute.Paid)
	decision, err := uc.AdmitUnbilledAttempt(context.Background(), uc_interfaces.UnbilledAttemptAdmission{
		Plan: plan, AttemptIndex: 1, Provider: "openai", Model: "qwen3:4b",
	})
	if err != nil {
		t.Fatalf("AdmitUnbilledAttempt: %v", err)
	}
	if decision.Route != airoute.Paid {
		t.Errorf("route = %q, want paid", decision.Route)
	}
	if repo.lastCap != nil {
		t.Errorf("strict policy cap = %v, want nil (no attempt cap)", repo.lastCap)
	}
}

// A free/local route keeps its route and never consumes the paid cap.
func TestAdmitUnbilledFreeRouteStaysFree(t *testing.T) {
	cap := 0
	repo, _ := admissionFixture(aienforcementmode.ObserveOnly, &cap, airoute.Free)
	uc := &CostUseCase{repo: repo, pricingUC: &stubPricingUC{err: errors.New("pricing must not be consulted")}}

	plan := admissionPlan(airoute.Free)
	decision, err := uc.AdmitUnbilledAttempt(context.Background(), uc_interfaces.UnbilledAttemptAdmission{
		Plan: plan, AttemptIndex: 1, Provider: "openai", Model: "qwen3:4b",
	})
	if err != nil {
		t.Fatalf("AdmitUnbilledAttempt: %v", err)
	}
	if decision.Route != airoute.Free {
		t.Errorf("route = %q, want free", decision.Route)
	}
	event := repo.events[plan.RequestID]
	if event == nil || event.LogicalRoute != airoute.Free {
		t.Fatalf("event = %+v, want logical route free", event)
	}
	if repo.admitCalls != 1 {
		t.Errorf("admission calls = %d, want 1 (free route admitted immediately)", repo.admitCalls)
	}
}

// A repeated BeforeAttempt is idempotent: one event, one cap increment.
func TestAdmitUnbilledIsIdempotentByRequestID(t *testing.T) {
	cap := 2
	repo, _ := admissionFixture(aienforcementmode.ObserveOnly, &cap, airoute.Paid)
	uc := &CostUseCase{repo: repo, pricingUC: &stubPricingUC{err: errors.New("pricing must not be consulted")}}

	plan := admissionPlan(airoute.Paid)
	admission := uc_interfaces.UnbilledAttemptAdmission{Plan: plan, AttemptIndex: 1, Provider: "openai", Model: "qwen3:4b"}
	if _, err := uc.AdmitUnbilledAttempt(context.Background(), admission); err != nil {
		t.Fatalf("first admission: %v", err)
	}
	if _, err := uc.AdmitUnbilledAttempt(context.Background(), admission); err != nil {
		t.Fatalf("second admission: %v", err)
	}

	if len(repo.events) != 1 {
		t.Errorf("events = %d, want 1 (upsert by request_id)", len(repo.events))
	}
	if repo.paidAttempts != 1 {
		t.Errorf("paid attempts = %d, want 1 (cap not double-counted)", repo.paidAttempts)
	}
}

// Prepare must not write an event for provider=openai; AdmitUnbilledAttempt is
// the only writer on that path.
func TestPrepareOpenAIDoesNotWriteAnEvent(t *testing.T) {
	repo, _ := admissionFixture(aienforcementmode.ObserveOnly, intPtr(5), airoute.Paid)
	uc := &CostUseCase{repo: repo, pricingUC: &stubPricingUC{err: errors.New("pricing must not be consulted")}}

	plan := admissionPlan(airoute.Paid)
	decision, err := uc.Prepare(context.Background(), plan)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if decision.Route != airoute.Paid {
		t.Errorf("route = %q, want paid", decision.Route)
	}
	if len(repo.events) != 0 {
		t.Errorf("prepare wrote %d event(s) for provider=openai, want 0", len(repo.events))
	}
}

// PrepareBillableAttempt prices and reserves the SAME event before Gemini.
func TestPrepareBillableAttemptPricesAndReservesTheSameEvent(t *testing.T) {
	repo, recorder := admissionFixture(aienforcementmode.ObserveOnly, intPtr(5), airoute.Paid)
	pricing := &stubPricingUC{cfg: testPricing()}
	uc := &CostUseCase{repo: repo, pricingUC: pricing}

	plan := admissionPlan(airoute.Paid)
	decision, err := uc.PrepareBillableAttempt(context.Background(), uc_interfaces.BillableAttemptAdmission{
		Plan: plan, AttemptIndex: 2, Provider: "google", Model: "gemini-2.0-flash",
	})
	if err != nil {
		t.Fatalf("PrepareBillableAttempt: %v", err)
	}
	if decision.Route != airoute.Paid {
		t.Errorf("route = %q, want paid", decision.Route)
	}
	if recorder.reserveCalls != 0 {
		t.Error("PrepareBillableAttempt must not use the monetary Reserve path directly")
	}
	if repo.prepareBillableCalls != 1 {
		t.Errorf("PrepareBillableAttempt calls = %d, want 1", repo.prepareBillableCalls)
	}
	if pricing.gotProvider != "google" || pricing.gotModel != "gemini-2.0-flash" {
		t.Errorf("pricing lookup = %q/%q, want google/gemini-2.0-flash", pricing.gotProvider, pricing.gotModel)
	}
	event := repo.preparedBillableEvent
	if event == nil {
		t.Fatal("no billable event was prepared")
	}
	if event.RequestID != plan.RequestID {
		t.Errorf("event request id = %s, want the same %s", event.RequestID, plan.RequestID)
	}
	if event.Provider == nil || *event.Provider != "google" {
		t.Errorf("event provider = %v, want google", event.Provider)
	}
	if event.ReservedCostMicroVND <= 0 {
		t.Errorf("event reserved = %d, want > 0 (Gemini reserved before the call)", event.ReservedCostMicroVND)
	}
	if event.LogicalRoute != airoute.Paid {
		t.Errorf("event route = %q, want paid", event.LogicalRoute)
	}
}
