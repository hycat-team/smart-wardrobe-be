package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestNewInternalErrorCapturesOriginStack(t *testing.T) {
	err := NewInternalError("boom")

	if err.Status != 500 {
		t.Fatalf("expected status 500, got %d", err.Status)
	}
	if err.Title == "" || err.Message != "boom" {
		t.Fatalf("unexpected error payload: %+v", err)
	}
	if err.Stack() == "" {
		t.Fatal("expected stack to be captured")
	}
	if !strings.Contains(err.Stack(), "apperror.TestNewInternalErrorCapturesOriginStack") {
		t.Fatalf("expected stack to contain test frame, got %q", err.Stack())
	}
}

func TestWrapPreservesExistingAppErrorStack(t *testing.T) {
	original := NewBadRequest("bad input")
	original.Errors = []ValidationErrorItem{{Field: "email", Message: "Vui lòng nhập email"}}

	wrapped := Wrap(original)
	appErr, ok := wrapped.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", wrapped)
	}

	if appErr != original {
		t.Fatal("expected existing app error instance to be preserved")
	}
	if appErr.Stack() != original.Stack() {
		t.Fatal("expected original stack to be preserved")
	}
	if len(appErr.Errors) != 1 || appErr.Errors[0].Field != "email" {
		t.Fatalf("expected validation errors to be preserved, got %+v", appErr.Errors)
	}
}

func TestDefaultConstructorCapturesOriginStack(t *testing.T) {
	err := ErrUnauthorized()

	if err.Status != 401 {
		t.Fatalf("expected 401, got %d", err.Status)
	}
	if err.Message != "Vui lòng đăng nhập" {
		t.Fatalf("unexpected message: %q", err.Message)
	}
	if err.Stack() == "" {
		t.Fatal("expected stack to be captured")
	}
	if !strings.Contains(err.Stack(), "apperror.TestDefaultConstructorCapturesOriginStack") {
		t.Fatalf("expected stack to contain test frame, got %q", err.Stack())
	}
}

func TestWrapPlainErrorFallsBackToInternalError(t *testing.T) {
	plain := errors.New("plain failure")
	err := Wrap(plain)

	appErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
	if appErr.Message != GenericInternalMessage {
		t.Fatalf("expected generic message, got %q", appErr.Message)
	}
	if !errors.Is(appErr, plain) && appErr.Cause != plain {
		t.Fatalf("expected cause chain to preserve the original error, got %v", appErr.Cause)
	}
}

func TestFromUnknownErrorHidesRawText(t *testing.T) {
	raw := errors.New(`pq: duplicate key value violates unique constraint "idx_x"`)
	appErr := From(raw)

	if appErr == nil {
		t.Fatal("expected app error")
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
	if appErr.Message != GenericInternalMessage {
		t.Fatalf("expected generic message, got %q", appErr.Message)
	}
	if appErr.Cause != raw {
		t.Fatalf("expected cause to preserve raw error for logging, got %v", appErr.Cause)
	}
	if strings.Contains(appErr.Error(), "duplicate key") {
		t.Fatalf("raw error text must not appear in client-facing message")
	}
}

func TestNewUpstreamUnavailableKeepsProviderDetailOutOfMessage(t *testing.T) {
	cause := fmt.Errorf("OpenAI Text API error (HTTP 429): {\"error\":{\"message\":\"quota exceeded for project\"}}")
	appErr := NewUpstreamUnavailable(cause)

	if appErr.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", appErr.Status)
	}
	if appErr.Message == "" || strings.Contains(appErr.Message, "quota") || strings.Contains(appErr.Message, "OpenAI") {
		t.Fatalf("expected static client-safe message without provider detail, got %q", appErr.Message)
	}
	if appErr.Cause != cause {
		t.Fatalf("expected cause to preserve provider detail for logs, got %v", appErr.Cause)
	}
}

func TestErrorIs(t *testing.T) {
	err1 := New(400, "Bad Request", "Invalid input data")
	err2 := New(400, "Bad Request", "Invalid input data")
	err3 := New(400, "Bad Request", "Different message")
	err4 := New(500, "Internal Server Error", "Invalid input data")

	if !errors.Is(err1, err2) {
		t.Error("expected err1 to match err2 (same status and message)")
	}

	if errors.Is(err1, err3) {
		t.Error("expected err1 to not match err3 (different message)")
	}

	if errors.Is(err1, err4) {
		t.Error("expected err1 to not match err4 (different status)")
	}
}
