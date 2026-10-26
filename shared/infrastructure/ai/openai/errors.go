package openai

import (
	"errors"
	"fmt"
	"net/http"
)

// Outcome is the attempt outcome vocabulary from
// specs/032-ollama-lightsail-trial/contracts/ollama-openai-subset.md §4.
type Outcome string

const (
	OutcomeSuccess       Outcome = "success"
	OutcomeTimeout       Outcome = "timeout"
	OutcomeTransport     Outcome = "transport"
	OutcomeHTTP4xx       Outcome = "http_4xx"
	OutcomeHTTP429       Outcome = "http_429"
	OutcomeHTTP5xx       Outcome = "http_5xx"
	OutcomeMalformed     Outcome = "malformed"
	OutcomeSchemaInvalid Outcome = "schema_invalid"
	OutcomeMaxTokens     Outcome = "max_tokens"
	OutcomeCanceled      Outcome = "canceled"
)

// UpstreamError carries the provider HTTP status together with the classified
// outcome. Callers can recover it from a wrapped app error with AsUpstreamError
// (apperror preserves the cause chain), so the HTTP status is not flattened into
// a generic 503 before the fallback decision is made.
type UpstreamError struct {
	HTTPStatus int
	Outcome    Outcome
	Cause      error
}

func (e *UpstreamError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("openai upstream (HTTP %d, %s): %v", e.HTTPStatus, e.Outcome, e.Cause)
	}
	return fmt.Sprintf("openai upstream (HTTP %d, %s)", e.HTTPStatus, e.Outcome)
}

func (e *UpstreamError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// FallbackEligible reports whether the attempt may be retried once on Gemini per
// the fallback matrix. 4xx other than 429 and client cancellation never retry.
func (e *UpstreamError) FallbackEligible() bool {
	if e == nil {
		return false
	}
	switch e.Outcome {
	case OutcomeTimeout, OutcomeTransport, OutcomeHTTP429, OutcomeHTTP5xx,
		OutcomeMalformed, OutcomeSchemaInvalid, OutcomeMaxTokens:
		return true
	default:
		return false
	}
}

// OutcomeForStatus maps an HTTP status to the contract outcome.
func OutcomeForStatus(status int) Outcome {
	switch {
	case status == http.StatusTooManyRequests:
		return OutcomeHTTP429
	case status >= 500:
		return OutcomeHTTP5xx
	case status >= 400:
		return OutcomeHTTP4xx
	case status >= 200 && status < 300:
		return OutcomeSuccess
	default:
		return OutcomeTransport
	}
}

// NewHTTPError builds a typed error for a non-2xx response.
func NewHTTPError(status int, cause error) *UpstreamError {
	return &UpstreamError{
		HTTPStatus: status,
		Outcome:    OutcomeForStatus(status),
		Cause:      cause,
	}
}

// NewMaxTokensError marks finish_reason=length. The HTTP call itself succeeded,
// so the status stays 200 and the outcome decides the fallback.
func NewMaxTokensError(cause error) *UpstreamError {
	return &UpstreamError{
		HTTPStatus: http.StatusOK,
		Outcome:    OutcomeMaxTokens,
		Cause:      cause,
	}
}

// NewMalformedError marks a 2xx response whose body could not be parsed into the
// expected shape.
func NewMalformedError(cause error) *UpstreamError {
	return &UpstreamError{
		HTTPStatus: http.StatusOK,
		Outcome:    OutcomeMalformed,
		Cause:      cause,
	}
}

// AsUpstreamError recovers the typed error from any wrapper in the chain.
func AsUpstreamError(err error) (*UpstreamError, bool) {
	var upstream *UpstreamError
	if errors.As(err, &upstream) {
		return upstream, true
	}
	return nil, false
}
