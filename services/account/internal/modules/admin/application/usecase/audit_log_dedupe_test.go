package usecase_test

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/dto/admin"
	"smart-wardrobe-be/services/account/internal/modules/admin/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
)

// fakeAuditLogRepository records what the use case tried to write and can be
// told to report a previous identical change, which is how a redelivered
// RabbitMQ message looks from here.
type fakeAuditLogRepository struct {
	repositories.IAdminAuditLogRepository

	written  []*entities.AdminAuditLog
	existing bool
	query    repositories.RecentChangeQuery
	queryErr error
}

func (f *fakeAuditLogRepository) Create(_ context.Context, log *entities.AdminAuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	f.written = append(f.written, log)
	return nil
}

// ExistsRecentChange answers from the rows actually written, honouring the query's
// time window.
//
// 024 T140: it used to return a fixed `existing` flag, so no test could tell a
// redelivery from a genuine second edit - the very distinction FR-004e is about.
// The flag is kept as an override for the "repository is broken" cases, but the
// default answer is now derived from `written` and the `Since` bound.
func (f *fakeAuditLogRepository) ExistsRecentChange(_ context.Context, query repositories.RecentChangeQuery) (bool, error) {
	f.query = query
	if f.queryErr != nil {
		return false, f.queryErr
	}
	if f.existing {
		return true, nil
	}
	for _, row := range f.written {
		if !row.CreatedAt.After(query.Since) {
			continue
		}
		if !matchesIdentity(row, query) {
			continue
		}
		return true, nil
	}
	return false, nil
}

// matchesIdentity applies the same fields the real query matches on, so the test
// exercises the identity rather than just the presence of any row.
func matchesIdentity(row *entities.AdminAuditLog, q repositories.RecentChangeQuery) bool {
	if row.ActorUserID != q.ActorUserID || row.Action != q.Action || row.TargetType != q.TargetType {
		return false
	}
	policyCode, _ := row.Payload["policyCode"].(string)
	operation, _ := row.Payload["operation"].(string)
	fieldName, _ := row.Payload["fieldName"].(string)
	if policyCode != q.PolicyCode || operation != q.Operation || fieldName != q.FieldName {
		return false
	}
	var rowValue *string
	if raw, ok := row.Payload["newValue"].(string); ok {
		rowValue = &raw
	}
	if q.NewValue == nil || rowValue == nil {
		return q.NewValue == nil && rowValue == nil
	}
	return *q.NewValue == *rowValue
}

func gate2Input(actorID uuid.UUID, newValue any) dto.RecordAuditLogInput {
	ip := "203.0.113.42"
	return dto.RecordAuditLogInput{
		ActorUserID: actorID,
		Action:      string(adminaudit.AICostPolicyUpdate),
		TargetType:  string(adminaudit.AICostPolicy),
		RequestIP:   &ip,
		Payload: map[string]any{
			"policyCode": "free-default",
			"operation":  "outfit",
			"fieldName":  "maxPaidAttemptsPerDay",
			"oldValue":   "3",
			"newValue":   newValue,
		},
	}
}

// TestGate2EventDeliveredTwiceLogsOneRow is the core guarantee of FR-004e: when
// the outbox retries the same change, the central audit table must not end up
// with two rows for it.
func TestGate2EventDeliveredTwiceLogsOneRow(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	// First delivery: nothing recorded yet, so the row is written.
	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatalf("expected 1 row after the first delivery, got %d", len(repo.written))
	}

	// Redelivery of the same event: the repository now reports the earlier row.
	repo.existing = true
	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatalf("the redelivery must not add a second row, got %d", len(repo.written))
	}
}

// TestGate2DedupeQueryCarriesTheFullIdentity pins down the identity tuple, since
// a field left out of it would make two genuinely different edits look like one.
func TestGate2DedupeQueryCarriesTheFullIdentity(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := repo.query
	if q.ActorUserID != actorID {
		t.Fatalf("actor missing from the identity: %+v", q)
	}
	if q.Action != string(adminaudit.AICostPolicyUpdate) || q.TargetType != string(adminaudit.AICostPolicy) {
		t.Fatalf("action/targetType missing from the identity: %+v", q)
	}
	if q.PolicyCode != "free-default" || q.Operation != "outfit" || q.FieldName != "maxPaidAttemptsPerDay" {
		t.Fatalf("target identity incomplete: %+v", q)
	}
	if q.NewValue == nil || *q.NewValue != "4" {
		t.Fatalf("new value missing from the identity: %+v", q.NewValue)
	}
	if q.Since.IsZero() || time.Since(q.Since) > 15*time.Minute {
		t.Fatalf("the window must look back far enough to cover an outbox retry, got %v", q.Since)
	}
}

// TestClearedValueIsDistinguishedFromAnUnsetOne guards the nullable-column
// subtlety: a change that sets the field to NULL must not match an earlier
// change that set it to a number, and must not be dropped for lacking a value.
func TestClearedValueIsDistinguishedFromAnUnsetOne(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	if err := uc.Record(context.Background(), gate2Input(actorID, nil)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatalf("a cleared value must still be logged, got %d rows", len(repo.written))
	}
	if repo.query.NewValue != nil {
		t.Fatalf("a cleared value must reach the query as nil, not as a string: %+v", *repo.query.NewValue)
	}
}

// TestEventWithoutIdentityIsStillLogged documents the deliberate bias: when the
// event does not carry a full identity, the row is written. Losing an audit
// entry is worse than keeping a redundant one.
func TestEventWithoutIdentityIsStillLogged(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{existing: true}
	uc := usecase.NewAuditLogUseCase(repo)

	input := gate2Input(actorID, "4")
	input.Payload = map[string]any{"note": "identity fields were stripped"}

	if err := uc.Record(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatal("an unidentifiable event must still be logged")
	}
}

// TestOtherActionsAreNotDeduplicated stops the gate-2 rule from swallowing
// audit rows produced by unrelated flows that happen to share a payload.
func TestOtherActionsAreNotDeduplicated(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{existing: true}
	uc := usecase.NewAuditLogUseCase(repo)

	ip := "203.0.113.42"
	input := dto.RecordAuditLogInput{
		ActorUserID: actorID,
		Action:      string(adminaudit.AIPricingUpdate),
		TargetType:  string(adminaudit.AIModelConfig),
		RequestIP:   &ip,
		Payload:     map[string]any{"policyCode": "free-default", "operation": "outfit", "fieldName": "maxPaidAttemptsPerDay", "newValue": "4"},
	}

	if err := uc.Record(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatal("only the gate-2 action is de-duplicated")
	}
}

// ---- 024 T140: FR-004e requires the limit of the mechanism to be stated in tests ----

// Two GENUINELY DIFFERENT changes to the same field must produce two rows. The
// identity tuple includes the new value precisely so that an administrator
// lowering a limit, then raising it, is fully recorded - which is what makes the
// trail worth anything after an investigation.
func TestGate2TwoGenuinelyDifferentChangesBothLogged(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	for _, value := range []string{"4", "5"} {
		if err := uc.Record(context.Background(), gate2Input(actorID, value)); err != nil {
			t.Fatalf("ghi thay doi %s: %v", value, err)
		}
	}

	if len(repo.written) != 2 {
		t.Fatalf("hai gia tri khac nhau phai tao hai dong, co = %d", len(repo.written))
	}
}

// The limit cuts both ways and it is worth being precise about which.
//
// Returning to a value already seen INSIDE the window is indistinguishable from a
// redelivery: the identity has no event id, so 3 -> 4 -> 3 records the second "3"
// as a duplicate of the first and drops it. That is the documented limit of
// FR-004e's mechanism, asserted here so it is a known property rather than a
// surprise discovered during an investigation.
func TestGate2ReturningToAPreviousValueInsideTheWindowIsTreatedAsADuplicate(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	for _, value := range []string{"4", "5", "4"} {
		if err := uc.Record(context.Background(), gate2Input(actorID, value)); err != nil {
			t.Fatalf("ghi thay doi %s: %v", value, err)
		}
	}

	if len(repo.written) != 2 {
		t.Errorf("quay lai mot gia tri da thay trong cua so bi coi trung - gioi han FR-004e chap nhan, co = %d dong", len(repo.written))
	}
}

// FR-004e states the limit of the mechanism, and requires it to be written down:
// the identity has no event id, so a GENUINE second edit of the same field to the
// same value inside the dedupe window is dropped rather than logged twice.
//
// This test is that statement. It documents which direction the mechanism errs in
// - silently losing a row rather than keeping a redundant one - so nobody
// discovers it by wondering why a second identical edit left no trace.
func TestGate2GenuineRepeatInsideTheWindowIsDroppedAndThatIsTheStatedLimit(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Same administrator, same field, same value, inside the window.
	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("second: %v", err)
	}

	if len(repo.written) != 1 {
		t.Errorf("trong cua so khu trung, mot lan doi THAT su lap lai bi bo qua - gioi han duoc FR-004e chap nhan, co = %d dong", len(repo.written))
	}
}

// Outside the window the same edit IS logged, so the window is the only thing
// limiting the loss and it is sized, not unbounded.
func TestGate2SameChangeOutsideTheWindowIsLoggedAgain(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Age the first row past the window.
	repo.written[0].CreatedAt = time.Now().Add(-2 * time.Hour)

	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("second: %v", err)
	}

	if len(repo.written) != 2 {
		t.Errorf("ngoai cua so thi phai ghi lai, co = %d dong", len(repo.written))
	}
}

// The window the code actually queries must match the window it documents. This
// is the assertion that stops the two drifting apart when someone retunes one of
// them.
func TestGate2DedupeWindowMatchesWhatTheQueryAsksFor(t *testing.T) {
	actorID := uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad11")
	repo := &fakeAuditLogRepository{}
	uc := usecase.NewAuditLogUseCase(repo)

	before := time.Now().Add(-time.Minute)
	if err := uc.Record(context.Background(), gate2Input(actorID, "4")); err != nil {
		t.Fatalf("record: %v", err)
	}

	got := time.Since(repo.query.Since)
	if got < 9*time.Minute || got > 11*time.Minute {
		t.Errorf("cua so khu trung phai la 10 phut, co = %v", got)
	}
	if repo.query.Since.After(before) {
		t.Error("Since phai nam o qua khu, khong phai o tuong lai")
	}
}
