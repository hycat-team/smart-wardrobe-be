package apperror

import (
	"errors"
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
	err := Wrap(errors.New("plain failure"))

	appErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
	if appErr.Message != "plain failure" {
		t.Fatalf("unexpected message: %q", appErr.Message)
	}
}
