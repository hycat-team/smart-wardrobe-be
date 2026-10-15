package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	admin_contract "smart-wardrobe-be/services/account/contract/admin"
	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 029 T019 / FR-007, FR-008, FR-010 — the admin billing endpoints over HTTP.
//
// WHY THESE GO THROUGH THE ROUTER. Two of the assertions are about what the
// response IS after JSON serialisation: the detail must carry the same
// camelCase field set as a list item, and `status` must be a quoted string. A
// handler called directly would return the same Go value either way; only the
// serialised response distinguishes the DTO from the raw entity it replaced.
// The route is also where a wrong period rule becomes a 400 rather than a
// silently unfiltered page, which is the before→after this contract is about.
//
// The use case is stubbed: the rules of the query are not under test here (the
// repository test pins the SQL, T018), only what the handler builds from the
// query string and what it hands back.

const adminBillingPath = "/api/v1/admin/subscriptions/transactions"

// stubAdminBillingUC records the filter the handler built and answers from
// fixed rows, so a test can assert on the INSTANTS the use case receives rather
// than on a rendered query.
type stubAdminBillingUC struct {
	calls      int
	lastFilter uc_interfaces.AdminBillingFilter
	lastPage   int
	lastLimit  int
	listRows   []*entities.DepositTransaction
	listTotal  int64
	detail     *entities.DepositTransaction
	detailErr  error
	detailID   uuid.UUID
}

func (s *stubAdminBillingUC) ListTransactions(
	_ context.Context, filter uc_interfaces.AdminBillingFilter, page, limit int,
) ([]*entities.DepositTransaction, int64, error) {
	s.calls++
	s.lastFilter = filter
	s.lastPage, s.lastLimit = page, limit
	return s.listRows, s.listTotal, nil
}

func (s *stubAdminBillingUC) GetTransaction(_ context.Context, id uuid.UUID) (*entities.DepositTransaction, error) {
	s.calls++
	s.detailID = id
	return s.detail, s.detailErr
}

func (s *stubAdminBillingUC) GetTransactionByOrderCode(context.Context, int64) (*entities.DepositTransaction, error) {
	return nil, nil
}

var _ uc_interfaces.IAdminBillingUseCase = (*stubAdminBillingUC)(nil)

// stubPaymentResolutionUC satisfies the handler's second dependency. The read
// endpoints never call it; the stub exists so the constructor is the production
// one, not a test-only shape.
type stubPaymentResolutionUC struct{}

func (stubPaymentResolutionUC) RetryInvestigation(context.Context, uc_interfaces.ManualPaymentResolutionInput) error {
	return nil
}

func (stubPaymentResolutionUC) ResolveAsPaid(context.Context, uc_interfaces.ManualPaymentResolutionInput) error {
	return nil
}

func (stubPaymentResolutionUC) ResolveAsCancelled(context.Context, uc_interfaces.ManualPaymentResolutionInput) error {
	return nil
}

func (stubPaymentResolutionUC) ResolveAsCreationFailed(context.Context, uc_interfaces.ManualPaymentResolutionInput) error {
	return nil
}

var _ uc_interfaces.IPaymentResolutionUseCase = stubPaymentResolutionUC{}

// stubAuditLogContract satisfies the third dependency; the read endpoints never
// record anything (FR-005's read-only shape lives on the activity endpoint, but
// the same reasoning holds here: nothing on this path writes).
type stubAuditLogContract struct{}

func (stubAuditLogContract) Record(context.Context, admin_dto.RecordAuditLogInput) error { return nil }

var _ admin_contract.IAuditLogContract = stubAuditLogContract{}

// newAdminBillingEngine builds the real route group around the stubbed use case,
// with the production middleware chain (error handler + admin role check).
func newAdminBillingEngine(t *testing.T, stub *stubAdminBillingUC) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := logger.New("test", "", "info", false)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))

	h := NewAdminBillingHandler(log, stub, stubPaymentResolutionUC{}, stubAuditLogContract{})
	group := engine.Group(adminBillingPath)
	group.Use(authenticatedAs(roleslug.Admin), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		group.GET("", shared_pres.WrapHandler(h.ListTransactions))
		group.GET("/:id", shared_pres.WrapHandler(h.GetTransaction))
	}
	return engine
}

// billingDepositTransaction builds a fully-populated entity at a chosen status,
// so both list and detail have every optional field to serialise.
func billingDepositTransaction(id uuid.UUID, status depositstatus.DepositStatus, createdAt time.Time) *entities.DepositTransaction {
	userID := uuid.New()
	planID := uuid.New()
	paymentURL := "https://pay.payos.vn/web/abc"
	reference := "REF-20261015-0001"
	paymentLinkID := "pay_link_123"
	cancelledAt := createdAt.Add(-2 * time.Hour)

	tx := &entities.DepositTransaction{
		UserID:                      userID,
		Amount:                      decimal.NewFromInt(250000),
		Currency:                    currency.VND,
		Status:                      status,
		TransactionType:             deposittransactiontype.DirectPurchase,
		SubscriptionPlanID:          &planID,
		OrderCode:                   1234567890,
		PaymentLinkID:               &paymentLinkID,
		SuccessfulProviderReference: &reference,
		PaymentUrl:                  &paymentURL,
		CancelledAt:                 &cancelledAt,
	}
	tx.ID = id
	tx.CreatedAt = createdAt
	tx.UpdatedAt = createdAt.Add(time.Minute)
	return tx
}

// billingResponseKeys pulls the `data` object's keys, sorted, so two responses
// can be compared by SHAPE.
func billingResponseKeys(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var parsed struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse response %q: %v", rec.Body.String(), err)
	}
	keys := make([]string, 0, len(parsed.Data))
	for key := range parsed.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ---- FR-007: the detail is the list item, not the raw entity ----

// TestAdminBillingDetailReturnsTheSameCamelCaseShapeAsTheList is FR-007's
// requirement stated as an equality: the two endpoints are two ways to see one
// DTO. It fails on the old behaviour in both directions — the detail came back
// with Go field names (`ID`, `UserID`, `OrderCode`) and a numeric `status`,
// which the key-set comparison and the explicit string check below catch.
func TestAdminBillingDetailReturnsTheSameCamelCaseShapeAsTheList(t *testing.T) {
	id := uuid.New()
	tx := billingDepositTransaction(id, depositstatus.ReconciliationRequired,
		time.Date(2026, 10, 15, 23, 30, 0, 0, timeutils.VnLoc()))
	stub := &stubAdminBillingUC{
		listRows:  []*entities.DepositTransaction{tx},
		listTotal: 1,
		detail:    tx,
	}
	engine := newAdminBillingEngine(t, stub)

	listRec := request(t, engine, http.MethodGet, adminBillingPath)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d (%s)", listRec.Code, listRec.Body.String())
	}
	detailRec := request(t, engine, http.MethodGet, adminBillingPath+"/"+id.String())
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail: expected 200, got %d (%s)", detailRec.Code, detailRec.Body.String())
	}
	if stub.detailID != id {
		t.Errorf("the detail lookup must receive the path id %s, got %s", id, stub.detailID)
	}

	var listEnvelope struct {
		Data struct {
			Items []map[string]json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("parse list response %q: %v", listRec.Body.String(), err)
	}
	if len(listEnvelope.Data.Items) != 1 {
		t.Fatalf("expected one list item, got %d", len(listEnvelope.Data.Items))
	}

	listKeys := make([]string, 0, len(listEnvelope.Data.Items[0]))
	for key := range listEnvelope.Data.Items[0] {
		listKeys = append(listKeys, key)
	}
	sort.Strings(listKeys)

	detailKeys := billingResponseKeys(t, detailRec)

	if strings.Join(listKeys, ",") != strings.Join(detailKeys, ",") {
		t.Errorf("FR-007: list item keys %v and detail keys %v must be identical — one DTO, two endpoints",
			listKeys, detailKeys)
	}
	for _, key := range []string{"id", "userId", "orderCode", "amount", "currency", "status",
		"transactionType", "createdAt", "updatedAt"} {
		if !containsString(detailKeys, key) {
			t.Errorf("FR-007: the detail must carry the camelCase key %q, got %v", key, detailKeys)
		}
	}
	for _, goName := range []string{"ID", "UserID", "OrderCode", "Status", "CreatedAt"} {
		if containsString(detailKeys, goName) {
			t.Errorf("FR-007: the detail must not expose the Go field name %q: %v", goName, detailKeys)
		}
	}

	// FR-008: on BOTH responses, `status` must be a JSON string, not the number
	// the raw entity used to emit.
	var detailParsed struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detailParsed); err != nil {
		t.Fatalf("parse detail response %q: %v", detailRec.Body.String(), err)
	}
	var detailStatus string
	if err := json.Unmarshal(detailParsed.Data["status"], &detailStatus); err != nil {
		t.Fatalf("FR-008: detail status must serialise as a string, got %s", detailParsed.Data["status"])
	}
	if detailStatus != "reconciliation_required" {
		t.Errorf("detail status = %q, want %q", detailStatus, "reconciliation_required")
	}
	var listStatus string
	if err := json.Unmarshal(listEnvelope.Data.Items[0]["status"], &listStatus); err != nil {
		t.Fatalf("FR-008: list status must serialise as a string, got %s", listEnvelope.Data.Items[0]["status"])
	}
	if listStatus != "reconciliation_required" {
		t.Errorf("list status = %q, want %q", listStatus, "reconciliation_required")
	}
}

// ---- FR-010: the VN calendar period, half-open ----

// TestAdminBillingListParsesTheVNPeriodHalfOpen sends the last two days of the
// VN calendar and asserts the INSTANTS the use case received: from = 00:00
// (+07:00) of the 15th, to = 00:00 (+07:00) of the 16th. A transaction at 23:30
// on the 15th is inside that range — the quickstart's scenario-2 boundary,
// which the old UTC-midnight parse dropped.
func TestAdminBillingListParsesTheVNPeriodHalfOpen(t *testing.T) {
	stub := &stubAdminBillingUC{}
	engine := newAdminBillingEngine(t, stub)

	rec := request(t, engine, http.MethodGet, adminBillingPath+"?fromDate=2026-10-15&toDate=2026-10-15")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	wantFrom := time.Date(2026, 10, 15, 0, 0, 0, 0, timeutils.VnLoc())
	wantTo := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())
	if stub.lastFilter.FromDate == nil || stub.lastFilter.ToDate == nil {
		t.Fatalf("the handler must parse fromDate/toDate into instants, got %#v", stub.lastFilter)
	}
	if !stub.lastFilter.FromDate.Equal(wantFrom) {
		t.Errorf("fromDate = %s, want %s (StartOfDayInVN)", stub.lastFilter.FromDate, wantFrom)
	}
	if !stub.lastFilter.ToDate.Equal(wantTo) {
		t.Errorf("toDate = %s, want %s (EndOfDayInVN — the exclusive end of the VN day)",
			stub.lastFilter.ToDate, wantTo)
	}

	boundary := time.Date(2026, 10, 15, 23, 30, 0, 0, timeutils.VnLoc())
	if boundary.Before(*stub.lastFilter.FromDate) || !boundary.Before(*stub.lastFilter.ToDate) {
		t.Errorf("a transaction at %s (23:30 VN on the selected day) must fall inside [%s, %s)",
			boundary, *stub.lastFilter.FromDate, *stub.lastFilter.ToDate)
	}

	// A transaction at 00:00 of the next VN day belongs to the NEXT period.
	nextDay := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())
	if nextDay.Before(*stub.lastFilter.ToDate) {
		t.Errorf("the upper bound must exclude 00:00 of the next VN day; %s < %s",
			nextDay, *stub.lastFilter.ToDate)
	}
}

// TestAdminBillingListRejectsAMalformedOrInvertedPeriod is the before→after
// the contract insists on: a bad date used to be DROPPED silently, and the
// caller received an unfiltered page that looked like a complete answer. The
// message is Vietnamese and names the offending parameter.
func TestAdminBillingListRejectsAMalformedOrInvertedPeriod(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		needle string
	}{
		{"malformed fromDate", "?fromDate=15-10-2026", "fromDate"},
		{"malformed toDate", "?toDate=2026-13-01", "toDate"},
		{"inverted period", "?fromDate=2026-10-16&toDate=2026-10-15", "không được sau"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAdminBillingUC{}
			engine := newAdminBillingEngine(t, stub)

			rec := request(t, engine, http.MethodGet, adminBillingPath+tc.query)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("FR-010: a period that cannot be honoured must be refused with 400, got %d (%s)",
					rec.Code, rec.Body.String())
			}
			if message := responseMessage(t, rec); !strings.Contains(message, tc.needle) {
				t.Errorf("the 400 message must mention %q, got %q", tc.needle, message)
			}
			// The refusal happens before the use case: an unfiltered page must
			// not be produced for a caller who asked for a different one.
			if stub.calls != 0 {
				t.Errorf("the list use case ran %d times for a refused period; it must not run at all", stub.calls)
			}
		})
	}
}

// TestAdminBillingListAcceptsFailedLegacyAndOpenBounds pins the three input
// shapes the contract adds: `failed_legacy` as a filter (FR-008), one-sided
// bounds (the missing side stays open), and genuinely absent bounds. Present
// but empty query values are ABSENT — gin binds them as empty strings, and an
// empty string must not become a zero instant that filters everything out.
func TestAdminBillingListAcceptsFailedLegacyAndOpenBounds(t *testing.T) {
	t.Run("failed_legacy is accepted", func(t *testing.T) {
		stub := &stubAdminBillingUC{}
		engine := newAdminBillingEngine(t, stub)

		rec := request(t, engine, http.MethodGet, adminBillingPath+"?status=failed_legacy")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		if stub.lastFilter.Status == nil || *stub.lastFilter.Status != depositstatus.FailedLegacy {
			t.Errorf("status filter = %#v, want failed_legacy (%d)", stub.lastFilter.Status, depositstatus.FailedLegacy)
		}
	})

	t.Run("one-sided bounds stay open on the other side", func(t *testing.T) {
		stub := &stubAdminBillingUC{}
		engine := newAdminBillingEngine(t, stub)

		rec := request(t, engine, http.MethodGet, adminBillingPath+"?fromDate=2026-10-15")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		if stub.lastFilter.FromDate == nil {
			t.Error("fromDate must reach the use case as an instant")
		}
		if stub.lastFilter.ToDate != nil {
			t.Errorf("an omitted toDate must stay nil, got %s", stub.lastFilter.ToDate)
		}
	})

	t.Run("no bounds means no bounds", func(t *testing.T) {
		stub := &stubAdminBillingUC{}
		engine := newAdminBillingEngine(t, stub)

		rec := request(t, engine, http.MethodGet, adminBillingPath)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		if stub.lastFilter.FromDate != nil || stub.lastFilter.ToDate != nil {
			t.Errorf("no bounds must stay nil, got %#v", stub.lastFilter)
		}
	})

	t.Run("present but empty bounds are absent", func(t *testing.T) {
		stub := &stubAdminBillingUC{}
		engine := newAdminBillingEngine(t, stub)

		rec := request(t, engine, http.MethodGet, adminBillingPath+"?fromDate=&toDate=")
		if rec.Code != http.StatusOK {
			t.Fatalf("an empty value must not be a validation error, got %d (%s)", rec.Code, rec.Body.String())
		}
		if stub.lastFilter.FromDate != nil || stub.lastFilter.ToDate != nil {
			t.Errorf("empty strings must not become instants, got %#v", stub.lastFilter)
		}
	})
}

func containsString(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
