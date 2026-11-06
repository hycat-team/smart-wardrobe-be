package ai

import (
	"errors"
	"fmt"
	"testing"

	"smart-wardrobe-be/shared/application/constants/apperror"
)

// errSelfReferencing wraps itself, which a hand-built error could do. rootCause
// must stop instead of looping forever.
type errSelfReferencing struct{}

func (e *errSelfReferencing) Error() string { return "self" }
func (e *errSelfReferencing) Unwrap() error { return e }

func TestRootCauseKeepsProviderErrorVisible(t *testing.T) {
	providerErr := fmt.Errorf("Google Text API error (HTTP 404): this model is no longer available")
	wrapped := apperror.NewUpstreamUnavailable(providerErr)

	// The client-facing message hides the provider reason; the log needs it.
	if wrapped.Error() == providerErr.Error() {
		t.Fatal("test setup broken: wrapper must not expose the cause through Error()")
	}
	if got := rootCause(wrapped); !errors.Is(got, providerErr) {
		t.Fatalf("rootCause = %v, want the provider error", got)
	}
}

func TestRootCauseOfAWrappedChain(t *testing.T) {
	inner := errors.New("innermost")
	err := fmt.Errorf("middle: %w", fmt.Errorf("outer: %w", inner))

	if got := rootCause(err); got.Error() != "innermost" {
		t.Fatalf("rootCause = %q, want %q", got, "innermost")
	}
}

func TestRootCauseOfASingleError(t *testing.T) {
	err := errors.New("only")
	if got := rootCause(err); got != err {
		t.Fatalf("rootCause = %v, want the error itself", got)
	}
}

func TestRootCauseStopsOnASelfReferencingChain(t *testing.T) {
	self := &errSelfReferencing{}
	if got := rootCause(self); got != self {
		t.Fatalf("rootCause = %v, want the original error", got)
	}
}
