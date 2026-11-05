package orchestrator

import (
	"testing"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type recordingLogger struct {
	warnings []string
}

func (l *recordingLogger) Debug(string, ...zap.Field) {}
func (l *recordingLogger) Info(string, ...zap.Field)  {}
func (l *recordingLogger) Warn(message string, _ ...zap.Field) {
	l.warnings = append(l.warnings, message)
}
func (l *recordingLogger) Error(string, ...zap.Field) {}
func (l *recordingLogger) Fatal(string, ...zap.Field) {}

var _ logger.Interface = (*recordingLogger)(nil)

func TestCeilingObservationDetectsInputBreach(t *testing.T) {
	tests := []struct {
		name      string
		estimated int64
		maxInput  int
		exceeded  bool
	}{
		{name: "well under", estimated: 2800, maxInput: 6000, exceeded: false},
		{name: "at the ceiling", estimated: 4000, maxInput: 4000, exceeded: false},
		{name: "one token over", estimated: 4001, maxInput: 4000, exceeded: true},
		{name: "no ceiling resolved", estimated: 9999, maxInput: 0, exceeded: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &uc_interfaces.ExecutionPlan{
				Operation:      aioperation.Outfit,
				MaxInputTokens: tt.maxInput,
			}
			observation := newCeilingObservation(plan, tt.estimated, 0)
			if got := observation.inputExceeded(); got != tt.exceeded {
				t.Fatalf("inputExceeded() = %v, want %v", got, tt.exceeded)
			}
		})
	}
}

func TestCeilingObservationDetectsOutputBreach(t *testing.T) {
	plan := &uc_interfaces.ExecutionPlan{
		Operation:       aioperation.Chat,
		MaxInputTokens:  4000,
		MaxOutputTokens: 300,
	}

	if newCeilingObservation(plan, 100, 300).exceeded() {
		t.Fatal("a request exactly at the output ceiling must not be reported")
	}
	if !newCeilingObservation(plan, 100, 301).outputExceeded() {
		t.Fatal("a request above the output ceiling must be reported")
	}
}

func TestCeilingObservationRatio(t *testing.T) {
	plan := &uc_interfaces.ExecutionPlan{MaxInputTokens: 4000}
	observation := newCeilingObservation(plan, 5000, 0)

	if got := observation.inputRatio(); got != 1.25 {
		t.Fatalf("inputRatio() = %v, want 1.25", got)
	}
	if got := newCeilingObservation(&uc_interfaces.ExecutionPlan{}, 5000, 0).inputRatio(); got != 0 {
		t.Fatalf("inputRatio() without a ceiling = %v, want 0", got)
	}
}

// The warning is throttled on powers of two so an incident cannot flood the
// log, but the counter still advances on every breach.
func TestCeilingMetricsThrottleWarnings(t *testing.T) {
	metrics := newCeilingMetrics()
	log := &recordingLogger{}
	observation := ceilingObservation{Operation: string(aioperation.Outfit), EstimatedInput: 5000, MaxInputTokens: 4000}

	const calls = 16
	for range calls {
		metrics.onExceeded(log, observation, uuid.Nil)
	}

	// 1, 2, 4, 8, 16
	if len(log.warnings) != 5 {
		t.Fatalf("logged %d warnings over %d breaches, want 5", len(log.warnings), calls)
	}
	if got := metrics.counters["ceiling:"+observation.Operation]; got != calls {
		t.Fatalf("counter = %d, want %d", got, calls)
	}
}

func TestVisionUsageGapOnlyReportedForGemini(t *testing.T) {
	metrics := newCeilingMetrics()
	log := &recordingLogger{}

	metrics.onVisionUsageMissing(log, string(aioperation.Vision), uuid.Nil)
	if len(log.warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(log.warnings))
	}

	// A zero ceiling means the operation never resolved one (always-free), so it
	// must not be treated as a breach.
	plan := &uc_interfaces.ExecutionPlan{Operation: aioperation.Vision}
	if newCeilingObservation(plan, 0, 0).exceeded() {
		t.Fatal("an unresolved ceiling must never be reported as exceeded")
	}
}

// A response that outgrows the policy ceiling is a different signal from a
// prompt that outgrows its input ceiling, so it must not share the log message.
func TestOutputOverrunUsesItsOwnMessage(t *testing.T) {
	metrics := newCeilingMetrics()
	log := &recordingLogger{}

	metrics.onOutputOverrun(log, ceilingObservation{
		Operation:       string(aioperation.Outfit),
		RequestedOutput: 326,
		MaxOutputTokens: 300,
	}, uuid.Nil)

	if len(log.warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(log.warnings))
	}
	if log.warnings[0] != "AI response exceeded the policy output ceiling" {
		t.Fatalf("unexpected message: %q", log.warnings[0])
	}
}
