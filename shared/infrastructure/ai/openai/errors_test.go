package openai

import (
	"errors"
	"net/http"
	"testing"

	"smart-wardrobe-be/shared/application/constants/apperror"
)

func TestOutcomeForStatus(t *testing.T) {
	cases := map[int]Outcome{
		http.StatusOK:                  OutcomeSuccess,
		http.StatusCreated:             OutcomeSuccess,
		http.StatusBadRequest:          OutcomeHTTP4xx,
		http.StatusUnauthorized:        OutcomeHTTP4xx,
		http.StatusForbidden:           OutcomeHTTP4xx,
		http.StatusNotFound:            OutcomeHTTP4xx,
		http.StatusUnprocessableEntity: OutcomeHTTP4xx,
		http.StatusTooManyRequests:     OutcomeHTTP429,
		http.StatusInternalServerError: OutcomeHTTP5xx,
		http.StatusBadGateway:          OutcomeHTTP5xx,
		http.StatusServiceUnavailable:  OutcomeHTTP5xx,
		http.StatusGatewayTimeout:      OutcomeHTTP5xx,
	}

	for status, want := range cases {
		if got := OutcomeForStatus(status); got != want {
			t.Errorf("OutcomeForStatus(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestNewHTTPErrorCarriesStatusAndOutcome(t *testing.T) {
	cause := errors.New("upstream said no")
	err := NewHTTPError(http.StatusTooManyRequests, cause)

	if err.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("HTTPStatus = %d, want 429", err.HTTPStatus)
	}
	if err.Outcome != OutcomeHTTP429 {
		t.Errorf("Outcome = %q, want http_429", err.Outcome)
	}
	if !errors.Is(err, cause) {
		t.Error("the cause must remain reachable through Unwrap")
	}
}

func TestNewMaxTokensErrorUsesMaxTokensOutcome(t *testing.T) {
	err := NewMaxTokensError(errors.New("finish_reason=length"))

	if err.Outcome != OutcomeMaxTokens {
		t.Errorf("Outcome = %q, want max_tokens", err.Outcome)
	}
	if err.HTTPStatus != http.StatusOK {
		t.Errorf("HTTPStatus = %d, want 200 (the call itself succeeded)", err.HTTPStatus)
	}
	if !err.FallbackEligible() {
		t.Error("max_tokens must be eligible for the one fallback attempt")
	}
}

func TestUpstreamErrorFallbackEligibility(t *testing.T) {
	cases := []struct {
		outcome Outcome
		want    bool
	}{
		{OutcomeSuccess, false},
		{OutcomeTimeout, true},
		{OutcomeTransport, true},
		{OutcomeHTTP4xx, false},
		{OutcomeHTTP429, true},
		{OutcomeHTTP5xx, true},
		{OutcomeMalformed, true},
		{OutcomeSchemaInvalid, true},
		{OutcomeMaxTokens, true},
		{OutcomeCanceled, false},
	}

	for _, tc := range cases {
		err := &UpstreamError{HTTPStatus: http.StatusOK, Outcome: tc.outcome}
		if got := err.FallbackEligible(); got != tc.want {
			t.Errorf("FallbackEligible(%q) = %v, want %v", tc.outcome, got, tc.want)
		}
	}
}

// The adapter wraps the typed error in apperror to keep a client-safe 503. The
// typed value must still be recoverable for the fallback decision, which is the
// whole reason it exists.
func TestAsUpstreamErrorRecoversTypedErrorFromAppError(t *testing.T) {
	typed := NewHTTPError(http.StatusServiceUnavailable, errors.New("upstream down"))
	wrapped := apperror.NewUpstreamUnavailable(typed)

	got, ok := AsUpstreamError(wrapped)
	if !ok {
		t.Fatal("AsUpstreamError must find the typed error through apperror")
	}
	if got.HTTPStatus != http.StatusServiceUnavailable || got.Outcome != OutcomeHTTP5xx {
		t.Fatalf("recovered = %+v, want status 503 and outcome http_5xx", got)
	}
}
