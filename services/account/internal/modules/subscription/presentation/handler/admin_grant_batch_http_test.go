package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// 024 T167 / FR-062 + FR-063 + FR-085: an empty list and an oversized list must reject
// the WHOLE request with the two exact messages the API contract specifies.
//
// These went through HTTP rather than being asserted on the handler's return value,
// because the defect was never in the handler — it was in what ran BEFORE it. The
// `binding:"required,min=1,max=500"` tag on the DTO fired during binding, so the
// handler's own two checks were unreachable and the response body came from the
// generic validator. Calling the handler directly would have passed while the
// endpoint was still wrong, which is exactly what happened.
//
// No use case is wired up. Both cases are refused before any dependency is touched,
// so a nil interface here is a stronger assertion than a fake would be: if a future
// change moved either check after the dependency lookup, this test would panic rather
// than quietly start passing.

const grantBatchPath = "/api/v1/admin/subscription-grants"

// postGrantBatch sends a batch request through the real error middleware and the real
// handler wrapper, so the JSON under test is the one a client would receive.
func postGrantBatch(t *testing.T, items []map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	log := logger.New("test", "", "info", false)
	h := NewAdminSubscriptionHandler(log, nil, nil, nil)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))
	engine.POST(grantBatchPath, shared_pres.WrapHandler(h.GrantSubscriptionsBatch))

	body, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, grantBatchPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// The key is required, and supplying it keeps the test focused on the two list
	// limits rather than on the idempotency check that comes after them.
	req.Header.Set("Idempotency-Key", "t167-test-key")

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// responseMessage pulls the human-facing message out of the error response.
func responseMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var parsed struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse error response %q: %v", rec.Body.String(), err)
	}
	if parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return parsed.Message
}

func TestGrantBatchRejectsEmptyListWithTheContractMessage(t *testing.T) {
	const want = "Danh sách cấp gói không được để trống."

	rec := postGrantBatch(t, []map[string]any{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("an empty list must return 400, got = %d (%s)", rec.Code, rec.Body.String())
	}
	if got := responseMessage(t, rec); got != want {
		t.Errorf("the message must match the contract\n  want = %q\n  got  = %q", want, got)
	}
}

func TestGrantBatchRejectsOversizedListWithTheContractMessage(t *testing.T) {
	const want = "Danh sách cấp gói tối đa 500 tài khoản mỗi lần."

	items := make([]map[string]any, 0, adminGrantMaxItems+1)
	for i := 0; i <= adminGrantMaxItems; i++ {
		items = append(items, map[string]any{
			"userIdentifier": fmt.Sprintf("user-%d@example.com", i),
			"planSlug":       "premium",
		})
	}

	rec := postGrantBatch(t, items)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("an oversized list must return 400, got = %d (%s)", rec.Code, rec.Body.String())
	}
	if got := responseMessage(t, rec); got != want {
		t.Errorf("the message must match the contract\n  want = %q\n  got  = %q", want, got)
	}
}

// The wrong-unit bug specifically: a slice limit must never be reported in characters.
// `max` on a slice used to fall through to the string message, so 501 items produced
// "không được vượt quá 500 kí tự". This asserts the word is absent, so the regression
// cannot come back through a different path either.
func TestOversizedListIsNeverReportedInCharacters(t *testing.T) {
	items := make([]map[string]any, 0, adminGrantMaxItems+1)
	for i := 0; i <= adminGrantMaxItems; i++ {
		items = append(items, map[string]any{
			"userIdentifier": fmt.Sprintf("user-%d@example.com", i),
			"planSlug":       "premium",
		})
	}

	rec := postGrantBatch(t, items)
	body := rec.Body.String()

	if strings.Contains(body, "ký tự") || strings.Contains(body, "kí tự") {
		t.Errorf("the limit is a COUNT OF ITEMS, not a string length: the message talks about characters: %s", body)
	}
}

// 500 exactly is allowed by FR-062, so the boundary must not be off by one. The
// request then fails later — on the nil use case — which is the proof that the size
// check let it through.
func TestExactlyFiveHundredItemsPassesTheSizeCheck(t *testing.T) {
	items := make([]map[string]any, 0, adminGrantMaxItems)
	for i := 0; i < adminGrantMaxItems; i++ {
		items = append(items, map[string]any{
			"userIdentifier": fmt.Sprintf("user-%d@example.com", i),
			"planSlug":       "premium",
		})
	}

	rec := postGrantBatch(t, items)

	if strings.Contains(rec.Body.String(), "tối đa 500") {
		t.Errorf("500 items is within the inclusive limit and must not be refused: %s", rec.Body.String())
	}
}
