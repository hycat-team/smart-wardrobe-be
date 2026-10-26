package orchestrator

import (
	"context"
	"net/http"
	"testing"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type trialTelemetryRecorder struct {
	message string
	fields  []zap.Field
}

func (l *trialTelemetryRecorder) Debug(string, ...zap.Field) {}
func (l *trialTelemetryRecorder) Info(message string, fields ...zap.Field) {
	l.message = message
	l.fields = append([]zap.Field(nil), fields...)
}
func (l *trialTelemetryRecorder) Warn(message string, fields ...zap.Field) {
	l.message = message
	l.fields = append([]zap.Field(nil), fields...)
}
func (l *trialTelemetryRecorder) Error(string, ...zap.Field) {}
func (l *trialTelemetryRecorder) Fatal(string, ...zap.Field) {}

// The trial attempt log must be labelled aws-course by the contract, while the
// deployment still reports SERVER_ENV=vps through uc.environment(). The label is
// trial-only and must not require changing SERVER_ENV or the business routing.
func TestTrialAttemptTelemetryUsesAWSourseEnvironment(t *testing.T) {
	recorder := &trialTelemetryRecorder{}

	cfg := &config.Config{}
	cfg.Server.Env = "vps"

	uc := &OrchestratorUseCase{cfg: cfg, logger: recorder}
	if got := uc.environment(); got != "vps" {
		t.Fatalf("uc.environment() = %q, want the unchanged SERVER_ENV vps", got)
	}

	state := &trialAttemptState{plan: &uc_interfaces.ExecutionPlan{
		Route:    airoute.Paid,
		Provider: providerOpenAI,
		Model:    "qwen3:4b",
	}}
	meta := app_ai.AttemptMeta{
		RequestID:    uuid.New(),
		AttemptIndex: 1,
		Provider:     providerOpenAI,
		Model:        "qwen3:4b",
		Operation:    string(aioperation.Chat),
		Capability:   config.CapabilityChatText,
	}
	uc.afterTrialAttempt(context.Background(), state, meta, app_ai.AttemptResult{
		Outcome:    "success",
		HTTPStatus: http.StatusOK,
	})

	if recorder.message != "ai provider attempt" {
		t.Fatalf("message = %q, want the attempt log message", recorder.message)
	}

	var environment string
	var found bool
	for _, field := range recorder.fields {
		if field.Key == "environment" {
			environment = field.String
			found = true
			break
		}
	}
	if !found {
		t.Fatal("attempt telemetry is missing the environment field")
	}
	if environment != "aws-course" {
		t.Errorf("environment = %q, want aws-course for the trial attempt log", environment)
	}
}
