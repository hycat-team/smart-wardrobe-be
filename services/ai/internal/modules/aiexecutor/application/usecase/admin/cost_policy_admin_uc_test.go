package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Test doubles. The project has no database test harness (no sqlmock, no
// testcontainers, no sqlite anywhere), so the use case is driven through
// hand-written interface fakes. That is exactly why the rejection paths are
// testable at all: they are pure decisions over repository output.
// ---------------------------------------------------------------------------

type fakeAICostRepository struct {
	repositories.IAICostRepository

	policies   []*aiexecutor_entities.AICostPolicy
	operations map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation
	grantTotal int64
	grantOpen  int64
}

func (f *fakeAICostRepository) ListPolicies(context.Context) ([]*aiexecutor_entities.AICostPolicy, error) {
	return f.policies, nil
}

func (f *fakeAICostRepository) GetPolicyOperationByID(_ context.Context, id uuid.UUID) (*aiexecutor_entities.AICostPolicyOperation, *aiexecutor_entities.AICostPolicy, error) {
	op, ok := f.operations[id]
	if !ok {
		return nil, nil, nil
	}
	policy := f.policyForOperation(op.PolicyID)
	if policy == nil {
		return nil, nil, nil
	}
	return &op, policy, nil
}

func (f *fakeAICostRepository) policyForOperation(policyID uuid.UUID) *aiexecutor_entities.AICostPolicy {
	for _, p := range f.policies {
		if p.ID == policyID {
			return p
		}
	}
	return nil
}

// UpdatePolicyOperation records the write.
//
// 024 T173: it used to be a bare `return nil`, which made every "the column was not
// written" assertion in this file vacuous — the fake dropped the write, so the value
// could not change whether the use case called it or refused. A test double that
// silently discards the operation under test is worse than no double: it turns a real
// check into a check that always passes.
func (f *fakeAICostRepository) UpdatePolicyOperation(_ context.Context, op *aiexecutor_entities.AICostPolicyOperation) error {
	f.operations[op.ID] = *op
	return nil
}

func (f *fakeAICostRepository) CountGrants(context.Context) (int64, int64, error) {
	return f.grantTotal, f.grantOpen, nil
}

type fakeAuditRepository struct {
	created []*aiexecutor_entities.AICostPolicyAuditLog
}

func (f *fakeAuditRepository) Create(_ context.Context, log *aiexecutor_entities.AICostPolicyAuditLog) error {
	f.created = append(f.created, log)
	return nil
}

// 024 T178: the fake's `CountByActorAndTime` is removed with the interface method it
// satisfied. A fake that still implements a method nothing declares is dead weight that
// hides how much of the interface is actually exercised.

type publishedEvent struct {
	topic   string
	payload any
}

type fakeEventPublisher struct {
	events []publishedEvent
}

func (f *fakeEventPublisher) Publish(_ context.Context, topic string, payload any) error {
	f.events = append(f.events, publishedEvent{topic: topic, payload: payload})
	return nil
}

type fakeUnitOfWork struct {
	shared_repos.IUnitOfWork
	executed bool
}

func (f *fakeUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	f.executed = true
	return fn(ctx)
}

func newPolicy(code string, mode aienforcementmode.AIEnforcementMode, hardCap *int64) *aiexecutor_entities.AICostPolicy {
	return &aiexecutor_entities.AICostPolicy{
		AuditableEntity:  aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		Code:             code,
		Name:             code + " policy",
		Version:          1,
		EnforcementMode:  mode,
		PeriodDays:       30,
		HardCostMicroVND: hardCap,
		IsActive:         true,
	}
}

func newOperation(policyID uuid.UUID, op aioperation.AIOperation, limit *int) aiexecutor_entities.AICostPolicyOperation {
	return aiexecutor_entities.AICostPolicyOperation{
		AuditableEntity:       aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		PolicyID:              policyID,
		Operation:             op,
		NormalRoute:           airoute.Paid,
		ReducedRoute:          airoute.Paid,
		FreeRoute:             airoute.Free,
		MaxPaidAttemptsPerDay: limit,
		IsEnabled:             true,
	}
}

func newUseCase(
	policies []*aiexecutor_entities.AICostPolicy,
	ops map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation,
	audit *fakeAuditRepository,
	publisher *fakeEventPublisher,
	uow shared_repos.IUnitOfWork,
) *AICostPolicyAdminUseCase {
	_, uc := newUseCaseWithRepo(policies, ops, audit, publisher, uow)
	return uc
}

// newUseCaseWithRepo also hands back the repository, so a test can assert on what was
// actually written. Without it the only observable is the response, which cannot
// distinguish "refused" from "wrote something and reported it" for the paths that
// return no value — and an assertion on the local copy of the operation is vacuous,
// because the fake serves a copy out of the map and the local variable is never
// updated.
func newUseCaseWithRepo(
	policies []*aiexecutor_entities.AICostPolicy,
	ops map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation,
	audit *fakeAuditRepository,
	publisher *fakeEventPublisher,
	uow shared_repos.IUnitOfWork,
) (*fakeAICostRepository, *AICostPolicyAdminUseCase) {
	repo := &fakeAICostRepository{policies: policies, operations: ops, grantTotal: 16, grantOpen: 5}
	return repo, NewAICostPolicyAdminUseCase(repo, audit, publisher, uow, logger.New("test", "", "info", false)).(*AICostPolicyAdminUseCase)
}

func intPtr(v int) *int       { return &v }
func int64Ptr(v int64) *int64 { return &v }

var testActor = CostPolicyActor{
	UserID:    uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11"),
	RequestIP: "203.0.113.42",
}

func requireStatus(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error with status %d, got nil", want)
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %T: %v", err, err)
	}
	if appErr.Status != want {
		t.Fatalf("expected HTTP %d, got %d (%v)", want, appErr.Status, err)
	}
}

// ---------------------------------------------------------------------------
// Rejection 1 — a value AND the clear flag in one body is a contradiction.
// ---------------------------------------------------------------------------

func TestUpdateRejectsValueTogetherWithClearFlag(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(4), ClearMaxPaidAttempts: true}, testActor)

	requireStatus(t, err, 400)
	if !strings.Contains(err.Error(), "clearMaxPaidAttempts") {
		t.Fatalf("the message should name the offending field, got: %v", err)
	}
}

func TestUpdateRejectsEmptyBody(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{}, testActor)

	requireStatus(t, err, 400)
}

// ---------------------------------------------------------------------------
// Rejection 2 — NULL in observe_only would switch the only bound off.
// ---------------------------------------------------------------------------

func TestUpdateRejectsNullInObserveOnly(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{ClearMaxPaidAttempts: true}, testActor)

	requireStatus(t, err, 409)
	if !strings.Contains(err.Error(), "observe_only") {
		t.Fatalf("the message must explain which mode forbids NULL, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Rejection 3 — strict with no hard cap enforces nothing at all.
// ---------------------------------------------------------------------------

func TestUpdateRejectsNullInStrictWithoutHardCap(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(30))
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{ClearMaxPaidAttempts: true}, testActor)

	requireStatus(t, err, 409)
	if !strings.Contains(err.Error(), "hard_cost_micro_vnd") {
		t.Fatalf("the message must name the missing cap column, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Rejection 4 — unknown id, or an id that belongs to a different operation.
// ---------------------------------------------------------------------------

func TestUpdateRejectsUnknownOperationID(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, int64Ptr(30000000000))
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), uuid.New(), aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(4)}, testActor)

	requireStatus(t, err, 404)
}

func TestUpdateRejectsOperationMismatch(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, int64Ptr(30000000000))
	op := newOperation(policy.ID, aioperation.Chat, intPtr(10))
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	// The URL says outfit; the row is chat. Accepting it would edit chat while
	// the administrator believes they edited outfit.
	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(4)}, testActor)

	requireStatus(t, err, 404)
}

// ---------------------------------------------------------------------------
// Happy path: one transaction, one audit row, one outbox event.
// ---------------------------------------------------------------------------

func TestUpdateSucceedsAndWritesBothRecordsInOneTransaction(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	audit := &fakeAuditRepository{}
	publisher := &fakeEventPublisher{}
	uow := &fakeUnitOfWork{}
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		audit, publisher, uow,
	)

	resp, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(4)}, testActor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !uow.executed {
		t.Fatal("the write and the audit record must share one transaction")
	}
	if resp.MaxPaidAttemptsPerDay == nil || *resp.MaxPaidAttemptsPerDay != 4 {
		t.Fatalf("response should echo the new limit, got %+v", resp.MaxPaidAttemptsPerDay)
	}
	if len(audit.created) != 1 {
		t.Fatalf("expected exactly one audit row, got %d", len(audit.created))
	}
	entry := audit.created[0]
	if entry.ActorUserID != testActor.UserID || entry.RequestIP != testActor.RequestIP {
		t.Fatalf("audit row lost the actor context: %+v", entry)
	}
	if entry.OldValue == nil || *entry.OldValue != "3" {
		t.Fatalf("audit row lost the old value: %+v", entry.OldValue)
	}
	if entry.NewValue == nil || *entry.NewValue != "4" {
		t.Fatalf("audit row lost the new value: %+v", entry.NewValue)
	}
	if entry.PolicyCode != "free-default" || entry.Operation != string(aioperation.Outfit) {
		t.Fatalf("audit row lost the target: %+v", entry)
	}
	if entry.FieldName != costPolicyFieldMaxPaidAttemptsPerDay {
		t.Fatalf("audit row must name the changed field, got %q", entry.FieldName)
	}
	if entry.Action != string(eventconstants.TopicAuditLog) && entry.Action == "" {
		t.Fatal("audit row must carry an action from the catalogue")
	}

	if len(publisher.events) != 1 {
		t.Fatalf("expected exactly one outbox event, got %d", len(publisher.events))
	}
	if publisher.events[0].topic != eventconstants.TopicAuditLog {
		t.Fatalf("wrong topic: %q", publisher.events[0].topic)
	}
}

// TestAuditEventPayloadMatchesConsumerContract guards the field names against
// services/account/dto/admin.RecordAuditLogInput. RegisterTypedConsumer
// unmarshals straight into that struct, so a renamed key is a PermanentError
// and the message lands in the dead-letter queue with no retry.
func TestAuditEventPayloadMatchesConsumerContract(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	publisher := &fakeEventPublisher{}
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, publisher, &fakeUnitOfWork{},
	)

	if _, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(4)}, testActor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := json.Marshal(publisher.events[0].payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	for _, key := range []string{"actorUserId", "action", "targetType", "targetId", "requestIp", "payload"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("payload is missing the %q key the consumer requires: %s", key, raw)
		}
	}
	if decoded["action"] != "ai.cost_policy_update" {
		t.Fatalf("action must come from the catalogue constant, got %v", decoded["action"])
	}
	if decoded["targetType"] != "ai_cost_policy" {
		t.Fatalf("targetType must come from the catalogue constant, got %v", decoded["targetType"])
	}

	inner, ok := decoded["payload"].(map[string]any)
	if !ok {
		t.Fatalf("inner payload must be an object, got %T", decoded["payload"])
	}
	for _, key := range []string{"policyCode", "operation", "fieldName", "oldValue", "newValue"} {
		if _, ok := inner[key]; !ok {
			t.Fatalf("inner payload is missing %q: %s", key, raw)
		}
	}
}

// TestSettingNullInStrictIsAccepted covers the case the API contract calls out
// as legitimate: a strict policy never reads this column, so NULL is the honest
// value rather than missing data.
func TestSettingNullInStrictIsAccepted(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, int64Ptr(30000000000))
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(30))
	audit := &fakeAuditRepository{}
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		audit, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	resp, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{ClearMaxPaidAttempts: true}, testActor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MaxPaidAttemptsPerDay != nil {
		t.Fatalf("expected NULL in the response, got %d", *resp.MaxPaidAttemptsPerDay)
	}
	if audit.created[0].NewValue != nil {
		t.Fatalf("a cleared value must stay nil, not become an empty string, got %q", *audit.created[0].NewValue)
	}
}

// TestNoOpUpdateIsRejected keeps a silent no-op from looking like a success.
func TestNoOpUpdateIsRejected(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, int64Ptr(30000000000))
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(4))
	audit := &fakeAuditRepository{}
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		audit, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(4)}, testActor)

	requireStatus(t, err, 409)
	if len(audit.created) != 0 {
		t.Fatal("a no-op must not write an audit row")
	}
}

// TestWarningNamesBothCounts covers FR-004b(2): the administrator must be told
// how many rows are frozen and, separately, how many of those will never
// refresh on their own.
func TestWarningNamesBothCounts(t *testing.T) {
	warning := buildReplayWarning(16, 5)
	if !warning.ReplayRequired {
		t.Fatal("replayRequired must always be true for this field")
	}
	if warning.FrozenGrantCount != 16 || warning.OpenGrantCount != 5 {
		t.Fatalf("counts must be reported separately, got %+v", warning)
	}
	if !strings.Contains(warning.Message, "16") || !strings.Contains(warning.Message, "5") {
		t.Fatalf("the message must carry both numbers, got %q", warning.Message)
	}
	if !strings.Contains(warning.Message, "FR-029c") {
		t.Fatalf("the message must point at the replay procedure, got %q", warning.Message)
	}
}

// TestListCostPoliciesMapsOperations covers the read path used by the admin
// screen, including the NULL round-trip for a strict policy.
func TestListCostPoliciesMapsOperations(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, int64Ptr(30000000000))
	policy.Operations = []*aiexecutor_entities.AICostPolicyOperation{
		{AuditableEntity: aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}}, Operation: aioperation.Outfit, NormalRoute: airoute.Paid, IsEnabled: true},
	}
	uc := newUseCase(
		[]*aiexecutor_entities.AICostPolicy{policy}, nil,
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	pols, err := uc.ListCostPolicies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pols) != 1 || len(pols[0].Operations) != 1 {
		t.Fatalf("unexpected shape: %+v", pols)
	}
	if pols[0].HardCostMicroVND == nil || *pols[0].HardCostMicroVND != 30000000000 {
		t.Fatalf("hard cap must survive the mapping, got %+v", pols[0].HardCostMicroVND)
	}
	// NULL must stay NULL in the payload: it is a correct, meaningful state in
	// strict mode, and turning it into 0 would read as "no paid attempts".
	if pols[0].Operations[0].MaxPaidAttemptsPerDay != nil {
		t.Fatalf("NULL limit must stay NULL, got %d", *pols[0].Operations[0].MaxPaidAttemptsPerDay)
	}
}

// ---------------------------------------------------------------------------
// Rejection 1 (024 T173) — a value on a strict policy, where the column is
// never read.
// ---------------------------------------------------------------------------

// FR-005e calls max_paid_attempts_per_day DEAD CONFIG for a strict policy and settled
// on NULL as the honest encoding, specifically so nobody reading the configuration
// table concludes Premium is limited to N chats a day.
//
// The endpoint accepted the value anyway: guardMaxPaidAttemptsWrite returned nil for
// any non-NULL desired value, before it looked at the mode at all. So the number was
// stored, looked authoritative, and enforced nothing — worse than a no-op, because
// the operator now believes a limit is in force.
func TestUpdateRejectsValueOnStrictPolicy(t *testing.T) {
	policy := newPolicy("premium-default", aienforcementmode.Strict, int64Ptr(30000000000))
	op := newOperation(policy.ID, aioperation.Outfit, nil)
	repo, uc := newUseCaseWithRepo(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	_, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(10)}, testActor)

	requireStatus(t, err, 409)
	if !strings.Contains(err.Error(), "strict") {
		t.Errorf("the message must name the strict mode, got: %v", err)
	}
	if !strings.Contains(err.Error(), "observe_only") {
		t.Errorf("the message must point at the way out (switch to observe_only), got: %v", err)
	}
	// The decisive assertion: nothing was written. A warning that still stored the
	// number would leave the misleading value in the table, which is the failure.
	//
	// Read the value the fake actually stored, not the local copy: GetPolicyOperationByID
	// serves a copy out of the map, so `op` is never updated and asserting on it would
	// pass whether or not the write happened.
	if stored := repo.operations[op.ID].MaxPaidAttemptsPerDay; stored != nil {
		t.Errorf("no value may be written to a strict policy, column = %d", *stored)
	}
}

// observe_only is the one mode that really enforces a daily count, so a value there is
// legitimate. This guards against the new rule refusing valid work.
func TestUpdateAcceptsValueOnObserveOnlyPolicy(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	repo, uc := newUseCaseWithRepo(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		&fakeAuditRepository{}, &fakeEventPublisher{}, &fakeUnitOfWork{},
	)

	if _, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(7)}, testActor); err != nil {
		t.Fatalf("observe_only is the mode that enforces a daily count, so a value must be accepted: %v", err)
	}
	if stored := repo.operations[op.ID].MaxPaidAttemptsPerDay; stored == nil || *stored != 7 {
		t.Errorf("the column must hold 7, got = %v", stored)
	}
}

// ---------------------------------------------------------------------------
// 024 T176: a failure to REPORT must not undo or deny a committed write.
// ---------------------------------------------------------------------------

// countFailingRepo fails only CountGrants, so the write path runs to completion and
// the failure lands exactly where it did in production: after the commit.
type countFailingRepo struct {
	*fakeAICostRepository
	countErr error
}

func (r *countFailingRepo) CountGrants(context.Context) (int64, int64, error) {
	return 0, 0, r.countErr
}

// The change is committed by the time the counts are read, and FR-004c puts both
// required records inside that same transaction. Returning the error used to produce a
// 5xx for an applied change — telling the administrator to retry a write that had
// already succeeded, and leaving them unsure whether the audit trail recorded it.
func TestUpdateStillSucceedsWhenTheGrantCountCannotBeRead(t *testing.T) {
	policy := newPolicy("free-default", aienforcementmode.ObserveOnly, nil)
	op := newOperation(policy.ID, aioperation.Outfit, intPtr(3))
	audit := &fakeAuditRepository{}
	publisher := &fakeEventPublisher{}

	base, uc := newUseCaseWithRepo(
		[]*aiexecutor_entities.AICostPolicy{policy},
		map[uuid.UUID]aiexecutor_entities.AICostPolicyOperation{op.ID: op},
		audit, publisher, &fakeUnitOfWork{},
	)
	repo := &countFailingRepo{fakeAICostRepository: base, countErr: errCountUnavailable}
	uc.aiCostRepo = repo

	resp, err := uc.UpdateCostPolicyOperation(context.Background(), op.ID, aioperation.Outfit,
		aiexecutor_dto.UpdateCostPolicyOperationReq{MaxPaidAttemptsPerDay: intPtr(9)}, testActor)

	if err != nil {
		t.Fatalf("after the commit a counting failure must not become an error for the administrator, got: %v", err)
	}
	if resp == nil || resp.MaxPaidAttemptsPerDay == nil || *resp.MaxPaidAttemptsPerDay != 9 {
		t.Fatalf("must return the applied result, got = %+v", resp)
	}

	// FR-004c: both records must still exist. A 200 that quietly dropped the audit row
	// or the outbox event would be worse than the 5xx it replaced.
	if len(audit.created) != 1 {
		t.Errorf("the audit row must be written, got = %d", len(audit.created))
	}
	if len(publisher.events) != 1 {
		t.Errorf("the outbox row must be written, got = %d", len(publisher.events))
	}
	if got := base.operations[op.ID].MaxPaidAttemptsPerDay; got == nil || *got != 9 {
		t.Errorf("the value must be written to the column, got = %v", got)
	}
}

// The warning must still say a replay is needed. Reporting success while implying no
// replay is necessary would trade one wrong answer for another — and the worse one,
// because it looks like "you are done".
func TestWarningStillDemandsAReplayWhenTheCountsAreUnavailable(t *testing.T) {
	warning := buildUncountedReplayWarning()

	if warning == nil || !warning.ReplayRequired {
		t.Fatalf("must still demand an event replay, got = %+v", warning)
	}
	if warning.FrozenGrantCount != 0 || warning.OpenGrantCount != 0 {
		t.Errorf("an unread count must be 0, got = %d / %d",
			warning.FrozenGrantCount, warning.OpenGrantCount)
	}
	if !strings.Contains(warning.Message, "FR-090") {
		t.Errorf("the message must say how to obtain the real figure, got: %q", warning.Message)
	}
	if !strings.Contains(warning.Message, "FR-029c") {
		t.Errorf("the message must keep the replay requirement, got: %q", warning.Message)
	}
}

var errCountUnavailable = errors.New("count unavailable")
