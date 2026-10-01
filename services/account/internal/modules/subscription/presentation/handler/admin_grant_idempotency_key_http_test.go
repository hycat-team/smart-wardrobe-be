package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 024 T179 / FR-085 + FR-068: the two branches added at
// admin_subscription_handler.go - `strings.TrimSpace` and the
// `utf8.RuneCountInString` limit - had no test reaching them.
//
// Both go through HTTP rather than calling the handler directly, because the
// properties under test are about the bytes a client sends and the JSON a client
// receives. The value the use case is handed is observed by a fake, because the
// trimmed key is not visible anywhere in the response: asserting on the response
// alone would let a handler that trims for display but forwards the padded key
// pass.

const testAdminID = "6c2f6a52-9a4d-4a6b-8d3f-2b7f5c1e0a91"

// recordingGrantUseCase records the identifierKey each request delivered, so the
// test can compare the keys rather than the responses.
//
// The interface is embedded rather than implemented method by method: the batch
// grant is the only method these requests reach, and spelling out the other
// twenty would pin the interface to this file. Same shape as `lateWinningEventRepo`
// in the idempotency race test.
type recordingGrantUseCase struct {
	uc_interfaces.ISubscriptionUseCase

	mu   sync.Mutex
	keys []string
}

func (f *recordingGrantUseCase) AdminGrantBatch(
	_ context.Context,
	_ uuid.UUID,
	items []sub_dto.AdminGrantBatchItem,
	identifierKey string,
	_ *string,
) (*sub_dto.AdminGrantBatchRes, error) {
	f.mu.Lock()
	f.keys = append(f.keys, identifierKey)
	f.mu.Unlock()

	results := make([]sub_dto.AdminGrantBatchResult, 0, len(items))
	for i, item := range items {
		results = append(results, sub_dto.AdminGrantBatchResult{
			Index:          i,
			UserIdentifier: item.UserIdentifier,
			PlanSlug:       item.PlanSlug,
			Status:         sub_dto.AdminGrantStatusSuccess,
		})
	}
	return &sub_dto.AdminGrantBatchRes{
		Total:     len(items),
		Succeeded: len(items),
		Results:   results,
	}, nil
}

func (f *recordingGrantUseCase) recordedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.keys))
	copy(out, f.keys)
	return out
}

// Compile-time proof the fake satisfies the interface the handler holds.
var _ uc_interfaces.ISubscriptionUseCase = (*recordingGrantUseCase)(nil)

// postGrantBatchWithKey sends ONE valid item with an arbitrary Idempotency-Key and
// returns the recorder plus the response.
//
// The existing postGrantBatch hardcodes a key and wires a nil use case, which is
// what the two list-limit tests want; this variant exists because these tests need
// the opposite of both - a chosen key and a use case that observes what arrived.
//
// The administrator is placed in the context because the handler refuses an
// unidentified actor (FR-081) BEFORE calling the use case. Without it, the "this
// key was accepted" cases would stop at a 401 and assert nothing about the key.
func postGrantBatchWithKey(
	t *testing.T,
	uc uc_interfaces.ISubscriptionUseCase,
	idempotencyKey string,
) (*recordingGrantUseCase, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := &recordingGrantUseCase{}
	log := logger.New("test", "", "info", false)
	h := NewAdminSubscriptionHandler(log, nil, recorder, nil)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))
	engine.Use(func(c *gin.Context) {
		if parsed, err := uuid.Parse(testAdminID); err == nil {
			c.Set(contextutils.CtxUserId, parsed)
		}
		c.Next()
	})
	engine.POST(grantBatchPath, shared_pres.WrapHandler(h.GrantSubscriptionsBatch))

	body, err := json.Marshal(map[string]any{
		"items": []map[string]any{
			{"userIdentifier": "user@example.com", "planSlug": "premium"},
		},
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, grantBatchPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Set, not Add, and deliberately no trimming here: the padding under test must
	// survive the transport exactly as the caller wrote it.
	req.Header.Set("Idempotency-Key", idempotencyKey)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return recorder, rec
}

// FR-068: padding must not fork the key. The event key is
// `ADMIN_GRANT:{key}:{index}`, so `" abc "` and `"abc"` are two different event
// keys and therefore two different grants. A retry that picked the padding back up
// would extend the subscription a second time with nothing to report it - the
// silent double grant this feature exists to prevent.
//
// The handler gets the raw header, so the two requests below are what the caller
// would really send; comparing what the USE CASE received is what makes this a
// statement about the stored key rather than about cosmetics.
func TestPaddedIdempotencyKeyYieldsTheSameKeyAsTheUnpaddedOne(t *testing.T) {
	padded, paddedRec := postGrantBatchWithKey(t, nil, "  padded-key  ")
	bare, bareRec := postGrantBatchWithKey(t, nil, "padded-key")

	if paddedRec.Code != http.StatusOK {
		t.Fatalf("a padded but non-empty key must be accepted, got = %d (%s)",
			paddedRec.Code, paddedRec.Body.String())
	}
	if bareRec.Code != http.StatusOK {
		t.Fatalf("an unpadded key must be accepted, got = %d (%s)",
			bareRec.Code, bareRec.Body.String())
	}

	paddedKeys := padded.recordedKeys()
	bareKeys := bare.recordedKeys()
	if len(paddedKeys) != 1 || len(bareKeys) != 1 {
		t.Fatalf("each request must reach the use case exactly once, got = %d and %d",
			len(paddedKeys), len(bareKeys))
	}
	if paddedKeys[0] != bareKeys[0] {
		t.Errorf("padding must not fork the key: a retry carrying the padded form would be a SECOND grant\n"+
			"  padded   = %q\n  unpadded = %q", paddedKeys[0], bareKeys[0])
	}
	if paddedKeys[0] != "padded-key" {
		t.Errorf("the key must arrive trimmed, got = %q", paddedKeys[0])
	}
}

// A whitespace-only header is NOT a key. It has to be refused as missing, which
// only holds if the trim runs BEFORE the emptiness test - the reverse order would
// let a key made of spaces through to a 255-column event key.
func TestWhitespaceOnlyIdempotencyKeyIsRejectedAsMissing(t *testing.T) {
	recorder, rec := postGrantBatchWithKey(t, nil, "   ")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a whitespace-only key must return 400, got = %d (%s)",
			rec.Code, rec.Body.String())
	}
	want := subscriptionerrors.ErrAdminGrantIdempotencyKeyRequired().Message
	if got := responseMessage(t, rec); got != want {
		t.Errorf("the message must be the missing-key one\n  want = %q\n  got  = %q", want, got)
	}
	if keys := recorder.recordedKeys(); len(keys) != 0 {
		t.Errorf("a blank key must never reach the use case, got = %v", keys)
	}
}

// Boundary of FR-068 / FR-085: 239 characters is the widest key that still fits the
// `character varying(255)` event key column, so it must be ACCEPTED. 240 is the
// first rejected length.
//
// Both cases are compared against the error factory rather than a copied string:
// a hand-written copy would keep passing after someone reworded the message, which
// is precisely the drift this test exists to catch.
func TestIdempotencyKeyAtTheCharacterLimitIsAccepted(t *testing.T) {
	keyAtLimit := strings.Repeat("k", adminGrantMaxIdempotencyKeyLength)

	recorder, rec := postGrantBatchWithKey(t, nil, keyAtLimit)

	if rec.Code != http.StatusOK {
		t.Fatalf("a key of exactly %d characters is within the inclusive limit and must be accepted, got = %d (%s)",
			adminGrantMaxIdempotencyKeyLength, rec.Code, rec.Body.String())
	}
	keys := recorder.recordedKeys()
	if len(keys) != 1 {
		t.Fatalf("the key must reach the use case, got %d recorded keys", len(keys))
	}
	if keys[0] != keyAtLimit {
		t.Errorf("the key must arrive unchanged, got = %q", keys[0])
	}
}

func TestIdempotencyKeyOverTheCharacterLimitIsRejectedNamingTheActualLength(t *testing.T) {
	overLimit := adminGrantMaxIdempotencyKeyLength + 1

	recorder, rec := postGrantBatchWithKey(t, nil, strings.Repeat("k", overLimit))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a key of %d characters must return 400, got = %d (%s)",
			overLimit, rec.Code, rec.Body.String())
	}
	want := subscriptionerrors.ErrAdminGrantIdempotencyKeyTooLong(
		adminGrantMaxIdempotencyKeyLength, overLimit,
	).Message
	if got := responseMessage(t, rec); got != want {
		t.Errorf("the message must name the real length, the limit and the cause\n  want = %q\n  got  = %q", want, got)
	}
	if keys := recorder.recordedKeys(); len(keys) != 0 {
		t.Errorf("an over-long key must never reach the use case, got = %v", keys)
	}
}

// The most important assertion in this file, because the limit is counted in the
// wrong unit by accident more easily than by design.
//
// The key becomes part of `user_subscription_events.event_key`, a
// `character varying(255)` column, and PostgreSQL counts CHARACTERS in a varchar -
// not bytes. So a 239-character Vietnamese key is legal and must be accepted even
// though it occupies 717 bytes. A handler that measured `len(key)` would refuse it,
// and it would refuse it for a reason that does not exist: the insert would have
// succeeded. The caller would be told their key is too long when the real problem
// would be that the measurement is wrong.
func TestMultibyteIdempotencyKeyAtTheCharacterLimitIsAccepted(t *testing.T) {
	// "ế" is 3 bytes in UTF-8, so this key is 239 characters and 717 bytes. It is
	// therefore also a byte-count canary: if the handler ever switches to len(),
	// 717 > 239 and this test fails.
	keyAtLimit := strings.Repeat("ế", adminGrantMaxIdempotencyKeyLength)
	if got := len(keyAtLimit); got <= adminGrantMaxIdempotencyKeyLength {
		t.Fatalf("the fixture must be LONGER in bytes than the limit, otherwise it proves nothing: bytes = %d", got)
	}

	recorder, rec := postGrantBatchWithKey(t, nil, keyAtLimit)

	if rec.Code != http.StatusOK {
		t.Fatalf("a %d-character multibyte key (%d bytes) must be accepted because the column counts characters, got = %d (%s)",
			adminGrantMaxIdempotencyKeyLength, len(keyAtLimit), rec.Code, rec.Body.String())
	}
	keys := recorder.recordedKeys()
	if len(keys) != 1 {
		t.Fatalf("the key must reach the use case, got %d recorded keys", len(keys))
	}
	if keys[0] != keyAtLimit {
		t.Errorf("the multibyte key must arrive unchanged, got = %q", keys[0])
	}
}
