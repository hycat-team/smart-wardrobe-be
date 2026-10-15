package mapper

import (
	"encoding/json"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 029 T017 / FR-007 — `DepositTransactionToRes` is the single mapping behind
// BOTH the admin list and the admin detail endpoint (R5). FR-007 exists because
// the detail used to return the raw entity: Go-cased keys (`ID`, `UserID`,
// `OrderCode`) and a numeric status, while the list answered camelCase. The
// requirement is that the two cannot differ, so the mapper test pins the whole
// field set here and the handler test (T019) pins that both endpoints go
// through it.

func fullDepositTransaction() *entities.DepositTransaction {
	id := uuid.New()
	userID := uuid.New()
	planID := uuid.New()
	paymentURL := "https://pay.payos.vn/web/abc"
	reference := "REF-20261015-0001"
	paymentLinkID := "pay_link_123"
	failureReason := "gateway_timeout"
	cancelledAt := time.Date(2026, 10, 15, 10, 30, 0, 0, time.UTC)
	nextReconciliationAt := time.Date(2026, 10, 15, 11, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)

	tx := &entities.DepositTransaction{
		UserID:                      userID,
		Amount:                      decimal.NewFromInt(250000),
		Currency:                    currency.VND,
		Status:                      depositstatus.ReconciliationRequired,
		TransactionType:             deposittransactiontype.DirectPurchase,
		SubscriptionPlanID:          &planID,
		OrderCode:                   1234567890,
		PaymentLinkID:               &paymentLinkID,
		SuccessfulProviderReference: &reference,
		PaymentUrl:                  &paymentURL,
		NextReconciliationAt:        &nextReconciliationAt,
		FailureReason:               &failureReason,
		CancelledAt:                 &cancelledAt,
	}
	tx.ID = id
	tx.CreatedAt = createdAt
	tx.UpdatedAt = updatedAt
	return tx
}

// TestDepositTransactionToResMapsTheFullAdminContract asserts EVERY field of
// `DepositTransactionRes` (data-model §2), including the renames that the old
// raw-entity response got wrong: `SubscriptionPlanID` -> `targetPlanId`,
// `SuccessfulProviderReference` -> `reference`, `PaymentLinkID` ->
// `paymentLinkId`. A field-by-field assertion rather than a subset: a mapping
// that drops one of the optional fields still answers 200 and still looks
// plausible on screen.
func TestDepositTransactionToResMapsTheFullAdminContract(t *testing.T) {
	tx := fullDepositTransaction()

	res := DepositTransactionToRes(tx)
	if res == nil {
		t.Fatal("a non-nil entity must map to a non-nil DTO")
	}

	if res.ID != tx.ID {
		t.Errorf("id = %s, want %s", res.ID, tx.ID)
	}
	if res.UserID != tx.UserID {
		t.Errorf("userId = %s, want %s", res.UserID, tx.UserID)
	}
	if res.OrderCode != tx.OrderCode {
		t.Errorf("orderCode = %d, want %d", res.OrderCode, tx.OrderCode)
	}
	if res.Amount != 250000 {
		t.Errorf("amount = %v, want 250000", res.Amount)
	}
	if res.Currency != string(currency.VND) {
		t.Errorf("currency = %q, want %q", res.Currency, currency.VND)
	}
	if res.Status != "reconciliation_required" {
		t.Errorf("status = %q, want the contract name %q", res.Status, "reconciliation_required")
	}
	if res.TransactionType != string(deposittransactiontype.DirectPurchase) {
		t.Errorf("transactionType = %q, want %q", res.TransactionType, deposittransactiontype.DirectPurchase)
	}
	if res.TargetPlanID == nil || *res.TargetPlanID != *tx.SubscriptionPlanID {
		t.Errorf("targetPlanId = %v, want %v", res.TargetPlanID, tx.SubscriptionPlanID)
	}
	if res.PaymentUrl == nil || *res.PaymentUrl != *tx.PaymentUrl {
		t.Errorf("paymentUrl = %v, want %v", res.PaymentUrl, tx.PaymentUrl)
	}
	if res.Reference == nil || *res.Reference != *tx.SuccessfulProviderReference {
		t.Errorf("reference = %v, want %v", res.Reference, tx.SuccessfulProviderReference)
	}
	if res.PaymentLinkId == nil || *res.PaymentLinkId != *tx.PaymentLinkID {
		t.Errorf("paymentLinkId = %v, want %v", res.PaymentLinkId, tx.PaymentLinkID)
	}
	if res.FailureReason == nil || *res.FailureReason != *tx.FailureReason {
		t.Errorf("failureReason = %v, want %v", res.FailureReason, tx.FailureReason)
	}
	if res.CancelledAt == nil || !res.CancelledAt.Equal(*tx.CancelledAt) {
		t.Errorf("cancelledAt = %v, want %v", res.CancelledAt, tx.CancelledAt)
	}
	if res.NextReconciliationAt == nil || !res.NextReconciliationAt.Equal(*tx.NextReconciliationAt) {
		t.Errorf("nextReconciliationAt = %v, want %v", res.NextReconciliationAt, tx.NextReconciliationAt)
	}
	if !res.CreatedAt.Equal(tx.CreatedAt) {
		t.Errorf("createdAt = %v, want %v", res.CreatedAt, tx.CreatedAt)
	}
	if !res.UpdatedAt.Equal(tx.UpdatedAt) {
		t.Errorf("updatedAt = %v, want %v", res.UpdatedAt, tx.UpdatedAt)
	}
}

// TestDepositTransactionToResSerialisesCamelCaseKeysWithStringStatus reads the
// contract off the JSON, which is what a client actually consumes: the exact
// camelCase key set, and `status` as a quoted STRING rather than the number the
// raw entity used to emit.
func TestDepositTransactionToResSerialisesCamelCaseKeysWithStringStatus(t *testing.T) {
	raw, err := json.Marshal(DepositTransactionToRes(fullDepositTransaction()))
	if err != nil {
		t.Fatalf("marshal the DTO: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse the DTO JSON %s: %v", raw, err)
	}

	// All 16 keys must be present when every optional field is set. `omitempty`
	// on an optional field is correct — but a field that is set and still
	// missing would be a mapping bug.
	for _, key := range []string{
		"id", "userId", "orderCode", "amount", "currency", "status", "transactionType",
		"targetPlanId", "paymentUrl", "reference", "paymentLinkId", "failureReason",
		"cancelledAt", "nextReconciliationAt", "createdAt", "updatedAt",
	} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("FR-007: the DTO must carry %q, got keys of %s", key, raw)
		}
	}

	var status string
	if err := json.Unmarshal(parsed["status"], &status); err != nil {
		t.Fatalf("FR-008: status must serialise as a JSON string, got %s", parsed["status"])
	}
	if status != "reconciliation_required" {
		t.Errorf("status = %q, want %q", status, "reconciliation_required")
	}

	// The Go names the raw entity used must not reappear. Case matters here:
	// `"ID"` and `"ID"`-style keys are exactly what FR-007 removes.
	for _, goName := range []string{`"ID"`, `"UserID"`, `"OrderCode"`, `"Status"`, `"CreatedAt"`} {
		if _, ok := parsed[goName[1:len(goName)-1]]; ok {
			t.Errorf("FR-007: the DTO must not expose the Go field name %s: %s", goName, raw)
		}
	}
}

func TestDepositTransactionToResMapsANilEntityToNil(t *testing.T) {
	if got := DepositTransactionToRes(nil); got != nil {
		t.Errorf("a nil entity must map to nil, got %+v", got)
	}
}
