package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/infrastructure/ai/openai"
)

// This file owns the lifecycle of ONE text provider attempt: the shared
// concurrency slot, the per-attempt deadline, the attempt callbacks and the
// error classification that decides whether the single fallback may run.
//
// Source of truth for the matrix:
// specs/032-ollama-lightsail-trial/contracts/ollama-openai-subset.md section 4
// and contracts/attempt-lifecycle.md section 2.

// errEmptyContent marks a 2xx response that carried no usable text. It is an
// output failure (eligible for the one fallback), not a transport failure.
func errEmptyContent() error {
	return apperror.NewUpstreamUnavailable(openai.NewMalformedError(errors.New("provider returned empty content")))
}

// attemptUsageCapture remembers what the provider reported for one attempt so
// the attempt telemetry can distinguish "zero tokens" from "usage not reported".
type attemptUsageCapture struct {
	usage    app_ai.TokenUsage
	reported bool
}

// captureAttemptUsage wraps the already-stamped usage callback so the attempt
// can read the provider/model winner after the call. The caller's callback still
// runs first-class; capture is additive.
func captureAttemptUsage(options app_ai.TextGenerationOptions) (app_ai.TextGenerationOptions, *attemptUsageCapture) {
	capture := &attemptUsageCapture{}
	forward := options.OnUsage
	options.OnUsage = func(usage app_ai.TokenUsage) {
		capture.usage = usage
		capture.reported = usage.PromptTokens > 0 || usage.OutputTokens > 0 || usage.ThinkingTokens > 0
		if forward != nil {
			forward(usage)
		}
	}
	return options, capture
}

// runTextAttempt executes one non-streaming text attempt and closes it with the
// attempt callbacks. A BeforeAttempt error aborts the attempt before the provider
// is called, which is what keeps admission/reservation ahead of the provider.
func (s *AIService) runTextAttempt(
	ctx context.Context,
	call textCall,
	systemPrompt string,
	userPrompt string,
	operation string,
	attemptIndex int,
	options app_ai.TextGenerationOptions,
) (string, error) {
	attemptCtx, cancel, meta, callOptions, capture, err := s.beginTextAttempt(ctx, call, operation, attemptIndex, options)
	if err != nil {
		return "", err
	}
	defer cancel()
	defer s.releaseTextSlot()

	started := time.Now()
	text, err := s.tryTextProvider(attemptCtx, call.client, call.provider, systemPrompt, userPrompt, callOptions)
	if err == nil && strings.TrimSpace(text) == "" {
		err = errEmptyContent()
	}

	result := buildAttemptResult(err, capture, time.Since(started), 0, false)
	s.afterAttempt(attemptCtx, meta, result, options.Hooks)
	return text, err
}

// runStreamAttempt executes one streaming text attempt. It reports whether any
// chunk reached the caller: once content has been emitted the attempt can no
// longer fall back, because switching providers mid-stream would splice two
// different answers together.
func (s *AIService) runStreamAttempt(
	ctx context.Context,
	call textCall,
	systemPrompt string,
	userPrompt string,
	operation string,
	attemptIndex int,
	options app_ai.TextGenerationOptions,
	outText chan<- string,
) (bool, error) {
	attemptCtx, cancel, meta, callOptions, capture, err := s.beginTextAttempt(ctx, call, operation, attemptIndex, options)
	if err != nil {
		return false, err
	}
	defer cancel()
	defer s.releaseTextSlot()

	started := time.Now()
	textChan, errChan := s.tryTextProviderStream(attemptCtx, call.client, call.provider, systemPrompt, userPrompt, callOptions)
	produced, ttft, err := s.forwardProviderStream(attemptCtx, started, textChan, errChan, outText)
	if err == nil && !produced && attemptCtx.Err() != nil {
		// The provider channels can close without an error when the attempt
		// context died first; report the cancellation instead of a false success.
		err = attemptCtx.Err()
	}

	result := buildAttemptResult(err, capture, time.Since(started), ttft, produced)
	s.afterAttempt(attemptCtx, meta, result, options.Hooks)
	return produced, err
}

// beginTextAttempt takes the shared text slot, waits for the bucket limiter,
// applies the per-attempt deadline and runs BeforeAttempt. The caller owns the
// returned release function through defer.
func (s *AIService) beginTextAttempt(
	ctx context.Context,
	call textCall,
	operation string,
	attemptIndex int,
	options app_ai.TextGenerationOptions,
) (context.Context, context.CancelFunc, app_ai.AttemptMeta, app_ai.TextGenerationOptions, *attemptUsageCapture, error) {
	if err := s.acquireTextSlot(ctx); err != nil {
		return ctx, func() {}, app_ai.AttemptMeta{}, options, nil, err
	}

	// The shared semaphore is taken first and released only when the attempt
	// finishes, so the bound counts every in-flight text attempt including the
	// ones waiting on the per-bucket rate limiter. That is what keeps the Go side
	// at OLLAMA_NUM_PARALLEL even when a burst queues.
	if call.limiter != nil {
		if err := call.limiter.Wait(ctx); err != nil {
			s.releaseTextSlot()
			return ctx, func() {}, app_ai.AttemptMeta{}, options, nil, err
		}
	}

	timeout := s.textAttemptTimeout(call.provider, call.capability, attemptIndex > 1)
	attemptCtx, cancel := s.withTextAttemptDeadline(ctx, timeout)

	meta := s.attemptMeta(options, call, operation, attemptIndex, timeout)
	if err := s.beforeAttempt(attemptCtx, meta, options.Hooks); err != nil {
		cancel()
		s.releaseTextSlot()
		return ctx, func() {}, meta, options, nil, err
	}

	callOptions, capture := captureAttemptUsage(s.withUsageStamp(options, call))
	return attemptCtx, cancel, meta, callOptions, capture, nil
}

// attemptMeta fills the fields the text service owns. Route and BillingClass are
// left to the orchestrator's callbacks, which know the business route.
func (s *AIService) attemptMeta(
	options app_ai.TextGenerationOptions,
	call textCall,
	operation string,
	attemptIndex int,
	timeout time.Duration,
) app_ai.AttemptMeta {
	resolvedOperation := options.Operation
	if resolvedOperation == "" {
		resolvedOperation = operation
	}
	return app_ai.AttemptMeta{
		RequestID:    options.RequestID,
		AttemptIndex: attemptIndex,
		Provider:     call.provider.Provider,
		Model:        call.provider.Model,
		Operation:    resolvedOperation,
		Capability:   call.capability,
		Timeout:      timeout,
	}
}

// attemptBlockedError marks a BeforeAttempt rejection. The provider was never
// called, so the fallback must not run either: the orchestrator already decided
// this request may not execute (limit, admission, reservation failure).
type attemptBlockedError struct {
	err error
}

func (e *attemptBlockedError) Error() string {
	return "attempt blocked before the provider call: " + e.err.Error()
}

func (e *attemptBlockedError) Unwrap() error { return e.err }

// beforeAttempt runs the admission callback. A non-nil error means the provider
// must not be called at all.
func (s *AIService) beforeAttempt(ctx context.Context, meta app_ai.AttemptMeta, hooks *app_ai.AttemptHooks) error {
	if hooks == nil || hooks.BeforeAttempt == nil {
		return nil
	}
	if err := hooks.BeforeAttempt(ctx, meta); err != nil {
		return &attemptBlockedError{err: err}
	}
	return nil
}

// afterAttempt closes the attempt. It runs on a context without cancellation so
// usage/telemetry are still recorded when the caller gave up.
func (s *AIService) afterAttempt(ctx context.Context, meta app_ai.AttemptMeta, result app_ai.AttemptResult, hooks *app_ai.AttemptHooks) {
	if hooks == nil || hooks.AfterAttempt == nil {
		return
	}
	hooks.AfterAttempt(context.WithoutCancel(ctx), meta, result)
}

// buildAttemptResult classifies the attempt outcome for telemetry and the
// fallback decision. Emitting a chunk before failing always disables the
// fallback, regardless of the error class.
func buildAttemptResult(err error, capture *attemptUsageCapture, latency, ttft time.Duration, produced bool) app_ai.AttemptResult {
	outcome, status, eligible := classifyTextAttempt(err)
	if produced {
		eligible = false
	}

	result := app_ai.AttemptResult{
		Outcome:          outcome,
		HTTPStatus:       status,
		FallbackEligible: eligible,
		LatencyMs:        latency.Milliseconds(),
		TTFTMs:           ttft.Milliseconds(),
		Error:            err,
	}
	if capture != nil {
		result.Usage = capture.usage
		result.UsageReported = capture.reported
	}
	return result
}

// classifyTextAttempt maps a provider failure to the contract outcome, the HTTP
// status (when there was a response), and whether the single fallback may run.
//
// Contract rule: fallback only for timeout, transport, 429, 5xx, malformed or
// empty output, schema_invalid and max_tokens. Never for client cancellation or
// a 4xx other than 429.
func classifyTextAttempt(err error) (string, int, bool) {
	if err == nil {
		return string(openai.OutcomeSuccess), http.StatusOK, false
	}
	if errors.Is(err, context.Canceled) {
		return string(openai.OutcomeCanceled), 0, false
	}
	var blocked *attemptBlockedError
	if errors.As(err, &blocked) {
		// Admission/reservation already refused this attempt; retrying the fallback
		// would bypass that decision and could double-charge.
		return string(openai.OutcomeHTTP4xx), 0, false
	}
	if upstream, ok := openai.AsUpstreamError(err); ok {
		return string(upstream.Outcome), upstream.HTTPStatus, upstream.FallbackEligible()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return string(openai.OutcomeTimeout), 0, true
	}
	if status := upstreamHTTPStatus(err); status != 0 {
		return string(openai.OutcomeForStatus(status)), status, fallbackEligibleForStatus(status)
	}
	if isMalformedOutputError(err) {
		return string(openai.OutcomeMalformed), http.StatusOK, true
	}
	if isTimeoutError(err) {
		return string(openai.OutcomeTimeout), 0, true
	}
	if isMaxTokensError(err) {
		// The Gemini adapter reports a truncated answer as a plain error rather
		// than a typed one; it is still an output failure, so it stays eligible.
		return string(openai.OutcomeMaxTokens), http.StatusOK, true
	}
	if isConfigOrRequestError(err) {
		// Authentication/configuration problems are not fixed by the fallback
		// provider; retrying would only hide the misconfiguration.
		return string(openai.OutcomeHTTP4xx), 0, false
	}
	return string(openai.OutcomeTransport), 0, true
}

// fallbackEligibleForStatus applies the 4xx/429/5xx split of the fallback matrix.
func fallbackEligibleForStatus(status int) bool {
	switch {
	case status == http.StatusTooManyRequests:
		return true
	case status >= 500:
		return true
	case status >= 400:
		return false
	default:
		return true
	}
}

var httpStatusInMessage = regexp.MustCompile(`HTTP (\d{3})`)

// upstreamHTTPStatus recovers the upstream status from an error chain. The
// OpenAI adapter carries it as a typed value; the Gemini adapter still embeds it
// in the message text ("Google Text API error (HTTP 400): ..."), which is the
// only place the status survives across an apperror wrapper.
func upstreamHTTPStatus(err error) int {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if upstream, ok := current.(*openai.UpstreamError); ok && upstream.HTTPStatus > 0 {
			return upstream.HTTPStatus
		}
		match := httpStatusInMessage.FindStringSubmatch(current.Error())
		if len(match) == 2 {
			if status, convErr := strconv.Atoi(match[1]); convErr == nil {
				return status
			}
		}
	}
	return 0
}

// isMalformedOutputError recognises a body that could not be decoded into the
// provider's response shape. It is an output failure and may fall back once.
func isMalformedOutputError(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &typeErr)
}

// isTimeoutError recognises a network timeout that was not raised as a
// context deadline.
func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// isMaxTokensError recognises the plain "reached MAX_TOKENS" failure the Gemini
// text adapter returns. A truncated answer is an output failure, never a success.
func isMaxTokensError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "MAX_TOKENS")
}

// isConfigOrRequestError recognises failures the fallback cannot repair: an
// unsupported provider, a missing configuration, or a client-side rejection.
func isConfigOrRequestError(err error) bool {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		switch appErr.Status {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity:
			return true
		}
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "chưa hỗ trợ") ||
		strings.Contains(text, "not configured") ||
		strings.Contains(text, "không được cấu hình")
}
