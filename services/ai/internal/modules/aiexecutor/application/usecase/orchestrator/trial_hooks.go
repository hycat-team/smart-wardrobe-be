package orchestrator

import (
	"context"
	"os"
	"strings"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	app_ai "smart-wardrobe-be/shared/application/ai"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const providerOpenAI = "openai"

const (
	billingClassUnbilled = "unbilled"
	billingClassBilled   = "billed"
)

// trialAttemptEnvironment is the environment label the telemetry contract
// requires for trial attempt logs. The deployment keeps SERVER_ENV as-is (vps on
// Lightsail); this label only describes the log, it is not a business branch.
const trialAttemptEnvironment = "aws-course"

// trialAttemptState carries the decision the hooks need across the two attempts
// of one trial text request. It is written by BeforeAttempt and read after the
// call returns; the text service closes its channels before the orchestrator
// reads it, which is the happens-before edge between the two goroutines.
type trialAttemptState struct {
	plan     *uc_interfaces.ExecutionPlan
	degraded bool
}

// attachTrialHooks installs the attempt-aware admission for provider=openai.
// Non-trial text keeps nil hooks and therefore the pre-existing behaviour.
func (uc *OrchestratorUseCase) attachTrialHooks(plan *uc_interfaces.ExecutionPlan, opt *app_ai.TextGenerationOptions) *trialAttemptState {
	if plan == nil || plan.Provider != providerOpenAI {
		return nil
	}

	uc.warnUnexpectedOpenAIHost(plan, opt)

	state := &trialAttemptState{plan: plan}
	opt.Route = string(plan.Route)
	opt.BillingClass = billingClass(plan.Provider, plan.Route != airoute.Paid)
	opt.Hooks = &app_ai.AttemptHooks{
		BeforeAttempt: func(ctx context.Context, meta app_ai.AttemptMeta) error {
			return uc.beforeTrialAttempt(ctx, state, meta)
		},
		AfterAttempt: func(ctx context.Context, meta app_ai.AttemptMeta, result app_ai.AttemptResult) {
			uc.afterTrialAttempt(ctx, state, meta, result)
		},
	}
	return state
}

// beforeTrialAttempt admits the Ollama attempt and upgrades the same event to a
// billable Gemini attempt only when the request did not degrade to the free
// route. A degraded fallback stays free: it never touches pricing, reservation
// or the paid attempt cap.
func (uc *OrchestratorUseCase) beforeTrialAttempt(ctx context.Context, state *trialAttemptState, meta app_ai.AttemptMeta) error {
	switch meta.AttemptIndex {
	case 1:
		decision, err := uc.costUC.AdmitUnbilledAttempt(ctx, uc_interfaces.UnbilledAttemptAdmission{
			Plan:         state.plan,
			AttemptIndex: meta.AttemptIndex,
			Provider:     meta.Provider,
			Model:        meta.Model,
		})
		if err != nil {
			return err
		}
		if decision != nil {
			state.degraded = decision.Route != airoute.Paid
		}
		return nil
	case 2:
		if state.degraded {
			return nil
		}
		_, err := uc.costUC.PrepareBillableAttempt(ctx, uc_interfaces.BillableAttemptAdmission{
			Plan:         state.plan,
			AttemptIndex: meta.AttemptIndex,
			Provider:     meta.Provider,
			Model:        meta.Model,
		})
		return err
	default:
		return nil
	}
}

// afterTrialAttempt records the attempt outcome through the same telemetry
// helper the text service uses. Route and billing class are filled from the
// business plan and the real provider; no prompt or secret is passed.
func (uc *OrchestratorUseCase) afterTrialAttempt(_ context.Context, state *trialAttemptState, meta app_ai.AttemptMeta, result app_ai.AttemptResult) {
	route := state.plan.Route
	if state.degraded {
		route = airoute.Free
	}

	entry := app_ai.AttemptTelemetry{
		Service:        "ai",
		Version:        appVersion(),
		Environment:    trialAttemptEnvironment,
		RequestID:      requestIDString(meta.RequestID),
		Operation:      meta.Operation,
		Capability:     meta.Capability,
		AttemptIndex:   meta.AttemptIndex,
		Provider:       meta.Provider,
		Model:          meta.Model,
		BillingClass:   billingClass(meta.Provider, state.degraded || route != airoute.Paid),
		Outcome:        result.Outcome,
		HTTPStatus:     result.HTTPStatus,
		TTFTMs:         result.TTFTMs,
		LatencyMs:      result.LatencyMs,
		PromptTokens:   result.Usage.PromptTokens,
		OutputTokens:   result.Usage.OutputTokens,
		ThinkingTokens: result.Usage.ThinkingTokens,
		UsageReported:  result.UsageReported,
		// The attempt result carries token usage only. The actual billed cost is
		// written by the cost use case to ai_usage_events; reading it back here
		// would add a billing lookup, which feature 032 explicitly does not do.
		// OpenAI/Ollama attempts are unbilled and therefore 0 by contract.
		CostMicroVND: 0,
	}
	// A fallback reason is only meaningful on a failed first attempt that is
	// still eligible for the single Gemini fallback.
	if meta.AttemptIndex == 1 && result.Error != nil && result.FallbackEligible {
		entry.FallbackReason = result.Outcome
	}

	app_ai.NewAttemptTelemetryLogger(uc.logger).LogAttempt(entry)
}

func (uc *OrchestratorUseCase) environment() string {
	if uc.cfg == nil {
		return ""
	}
	return uc.cfg.Server.Env
}

// appVersion is the image/app version stamped into attempt telemetry. The
// repository has no build-time version symbol yet, so the deployment-provided
// APP_VERSION is used. An empty value is allowed and documented: the runbook
// records the deployed image tag separately.
func appVersion() string {
	return strings.TrimSpace(os.Getenv("APP_VERSION"))
}

// warnUnexpectedOpenAIHost emits one structured warning per trial request when
// provider=openai points at a host that is not in AI_OPENAI_TRIAL_EXPECTED_HOSTS.
// An empty list disables the check, and only the host is logged: never the full
// endpoint, prompt, credential or request body.
func (uc *OrchestratorUseCase) warnUnexpectedOpenAIHost(plan *uc_interfaces.ExecutionPlan, opt *app_ai.TextGenerationOptions) {
	if uc == nil || uc.logger == nil || uc.cfg == nil || plan == nil || opt == nil || opt.Primary == nil {
		return
	}

	expected := uc.cfg.AI.OpenAITrialExpectedHosts
	host, mismatch := app_ai.OpenAIEndpointHostMismatch(opt.Primary.Endpoint, expected)
	if !mismatch {
		return
	}

	uc.logger.Warn("openai provider endpoint is outside AI_OPENAI_TRIAL_EXPECTED_HOSTS",
		zap.String("service", "ai"),
		zap.String("environment", uc.environment()),
		zap.String("request_id", requestIDString(plan.RequestID)),
		zap.String("provider", plan.Provider),
		zap.String("model", plan.Model),
		zap.String("endpoint_host", host),
		zap.Strings("expected_hosts", expected),
	)
}

// stampedUnbilled trusts the provider/model stamp the AI service attached to a
// usage result. When the result carries no stamp (a direct call or a stub), it
// falls back to the capability list so vision stays billed by default.
func (uc *OrchestratorUseCase) stampedUnbilled(usage app_ai.TokenUsage, capability string) bool {
	if usage.Unbilled || usage.Provider == providerOpenAI {
		return true
	}
	if usage.Provider == "" && usage.Model == "" {
		return uc.unbilledByProvider("", capability)
	}
	return false
}

// unbilledByProvider applies the provider rule BEFORE the capability list, the
// same order the text service uses. The capability list therefore cannot make a
// Gemini fallback free, and it remains the only source for vision/embedding.
func (uc *OrchestratorUseCase) unbilledByProvider(provider, capability string) bool {
	if provider == providerOpenAI {
		return true
	}
	if uc.cfg == nil {
		return false
	}
	return uc.cfg.AI.IsUnbilledCapability(capability)
}

func billingClass(provider string, degraded bool) string {
	if degraded || provider == providerOpenAI {
		return billingClassUnbilled
	}
	return billingClassBilled
}

func requestIDString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
