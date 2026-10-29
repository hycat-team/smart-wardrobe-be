package mapper

import (
	"encoding/json"
	"testing"
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

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

	// All 18 keys must be present when every optional field is set. `omitempty`
	// on an optional field is correct — but a field that is set and still
	// missing would be a mapping bug. `user` and `targetPlan` are asserted too:
	// they carry NO `omitempty`, so even the list item (which leaves them nil)
	// publishes both keys, which is what keeps the detail's key set identical.
	for _, key := range []string{
		"id", "userId", "orderCode", "amount", "currency", "status", "transactionType",
		"targetPlanId", "paymentUrl", "reference", "paymentLinkId", "failureReason",
		"cancelledAt", "nextReconciliationAt", "createdAt", "updatedAt",
		"user", "targetPlan",
	} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("FR-007: the DTO must carry %q, got keys of %s", key, raw)
		}
	}

	// The list item carries the labels as null rather than omitting them.
	for _, key := range []string{"user", "targetPlan"} {
		if string(parsed[key]) != "null" {
			t.Errorf("FR-007: list %q must be null (the list does not resolve labels), got %s", key, parsed[key])
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

// fullUserBrief is the identity contract payload the detail enrichment reads.
// It sets every field the contract carries — including the ones the response
// must NOT publish — so the whitelist test below can prove they were dropped
// rather than merely absent by accident.
func fullUserBrief() *identity_dto.UserBriefContractDTO {
	return &identity_dto.UserBriefContractDTO{
		ID:             uuid.New(),
		Username:       "khachhang",
		Email:          "khachhang@example.com",
		FirstName:      "Nguyễn",
		LastName:       "An",
		Gender:         gender.Female,
		RoleSlug:       roleslug.Admin,
		IsDeleted:      true,
		CreatedByAdmin: true,
		CreatedAt:      time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
}

func fullSubscriptionPlan() *entities.SubscriptionPlan {
	durationDays := 30
	plan := &entities.SubscriptionPlan{
		Slug:               "premium-monthly",
		Name:               "Premium tháng",
		Price:              decimal.NewFromInt(250000),
		MaxWardrobeItems:   500,
		MaxOutfits:         100,
		AiOutfitDailyQuota: 30,
		AiChatDailyQuota:   10,
		DurationDays:       &durationDays,
		TierRank:           2,
	}
	plan.ID = uuid.New()
	return plan
}

// TestDepositTransactionDetailToResFillsTheBuyerAndTargetPlan pins the whole
// enrichment contract: the base fields still come from the list mapper, and the
// two labels are filled from the entity's own ids.
func TestDepositTransactionDetailToResFillsTheBuyerAndTargetPlan(t *testing.T) {
	tx := fullDepositTransaction()
	user := fullUserBrief()
	plan := fullSubscriptionPlan()

	res := DepositTransactionDetailToRes(tx, user, plan)
	if res == nil {
		t.Fatal("a non-nil entity must map to a non-nil DTO")
	}
	if res.ID != tx.ID || res.OrderCode != tx.OrderCode {
		t.Errorf("the base fields must still come from DepositTransactionToRes, got %+v", res)
	}

	if res.User == nil {
		t.Fatal("the detail must carry the buyer label")
	}
	if res.User.ID != user.ID || res.User.Username != user.Username {
		t.Errorf("user = %+v, want id %s and username %q", res.User, user.ID, user.Username)
	}
	if res.User.Email == nil || *res.User.Email != user.Email {
		t.Errorf("user.email = %v, want %q", res.User.Email, user.Email)
	}
	if res.User.FirstName == nil || *res.User.FirstName != user.FirstName {
		t.Errorf("user.firstName = %v, want %q", res.User.FirstName, user.FirstName)
	}
	if res.User.LastName == nil || *res.User.LastName != user.LastName {
		t.Errorf("user.lastName = %v, want %q", res.User.LastName, user.LastName)
	}

	if res.TargetPlan == nil {
		t.Fatal("the detail must carry the target plan label")
	}
	if res.TargetPlan.ID != plan.ID || res.TargetPlan.Slug != plan.Slug || res.TargetPlan.Name != plan.Name {
		t.Errorf("targetPlan = %+v, want id %s slug %q name %q", res.TargetPlan, plan.ID, plan.Slug, plan.Name)
	}
	if res.TargetPlan.Price != 250000 {
		t.Errorf("targetPlan.price = %v, want 250000", res.TargetPlan.Price)
	}
	if res.TargetPlan.DurationDays == nil || *res.TargetPlan.DurationDays != 30 {
		t.Errorf("targetPlan.durationDays = %v, want 30", res.TargetPlan.DurationDays)
	}
}

// TestDepositTransactionDetailToResToleratesMissingLabels covers the three
// normal ways a label is absent: a soft-deleted buyer, a plan removed from the
// catalogue, and a wallet top-up that names no plan at all. None of them is an
// error, and the transaction itself must still be answered in full.
func TestDepositTransactionDetailToResToleratesMissingLabels(t *testing.T) {
	res := DepositTransactionDetailToRes(fullDepositTransaction(), nil, nil)
	if res == nil {
		t.Fatal("a transaction without labels must still map")
	}
	if res.User != nil || res.TargetPlan != nil {
		t.Errorf("missing labels must stay nil, got user %+v plan %+v", res.User, res.TargetPlan)
	}
	if res.ID == uuid.Nil {
		t.Error("the transaction fields must still be mapped when the labels are missing")
	}

	if got := DepositTransactionDetailToRes(nil, fullUserBrief(), fullSubscriptionPlan()); got != nil {
		t.Errorf("a nil entity must map to nil even with labels present, got %+v", got)
	}
}

// TestMapTransactionUserBriefPublishesOnlyTheDisplayFields is the whitelist
// assertion, read off the JSON a client receives. The identity contract carries
// `roleSlug`, `isDeleted`, `createdByAdmin`, `gender`, `status` and `createdAt`;
// publishing any of them would leak an internal fact onto a payment screen, and
// would also make this response depend on a contract it does not own.
func TestMapTransactionUserBriefPublishesOnlyTheDisplayFields(t *testing.T) {
	raw, err := json.Marshal(MapTransactionUserBrief(fullUserBrief()))
	if err != nil {
		t.Fatalf("marshal the user brief: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse the user brief %s: %v", raw, err)
	}

	for _, key := range []string{"id", "username", "email", "firstName", "lastName", "avatarUrl"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("the buyer brief must publish %q, got %s", key, raw)
		}
	}
	for _, internal := range []string{"roleSlug", "isDeleted", "createdByAdmin", "gender", "status", "createdAt"} {
		if _, ok := parsed[internal]; ok {
			t.Errorf("the buyer brief must not publish the identity-internal field %q: %s", internal, raw)
		}
	}

	if got := MapTransactionUserBrief(nil); got != nil {
		t.Errorf("a nil contract payload must map to nil, got %+v", got)
	}
}

// TestMapTransactionUserBriefOmitsEmptyStrings keeps "not filled in" distinct
// from the empty string: an account with no last name must not make the screen
// render a blank line where a name would be.
func TestMapTransactionUserBriefOmitsEmptyStrings(t *testing.T) {
	user := &identity_dto.UserBriefContractDTO{ID: uuid.New(), Username: "khachhang"}

	res := MapTransactionUserBrief(user)
	if res == nil {
		t.Fatal("a payload with only the required fields must still map")
	}
	if res.Email != nil || res.FirstName != nil || res.LastName != nil {
		t.Errorf("empty strings must become nil, got %+v", res)
	}
}
