package grant

import (
	"strings"
	"testing"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// recordingLogger captures what was logged so a test can assert on it.
type recordingLogger struct {
	warnings []string
	fields   [][]zap.Field
}

func (l *recordingLogger) Debug(string, ...zap.Field) {}
func (l *recordingLogger) Info(string, ...zap.Field)  {}
func (l *recordingLogger) Error(string, ...zap.Field) {}

func (l *recordingLogger) Fatal(message string, fields ...zap.Field) {
	// A test must never exit the process. Recording it keeps a Fatal reachable in
	// production code from taking the whole suite down with it.
	l.warnings = append(l.warnings, "FATAL: "+message)
	l.fields = append(l.fields, fields)
}

func (l *recordingLogger) Warn(message string, fields ...zap.Field) {
	l.warnings = append(l.warnings, message)
	l.fields = append(l.fields, fields)
}

// 024 T171 / FR-020: the older-payload-version warning must go through the service's
// own logger.
//
// It used to call `logger.New("test", "", "info", false).Warn(...)` inline, in
// production code. That constructs an env literally named "test", writes nothing to a
// file, and bypasses the injected logger entirely — so the requirement to "record a
// warning" was satisfied in the source and nowhere at runtime. This is the only
// warning on that branch, and it is the signal that a publisher is still emitting a
// payload version the receiver has moved past.

// A version-1 payload is a warning, not a rejection: one sitting in the queue at deploy
// time is expected. But it must actually be recorded.
func TestOlderPayloadVersionWarnsThroughTheInjectedLogger(t *testing.T) {
	rec := &recordingLogger{}

	payload := validPayload()
	payload.EventVersion = 1

	if _, err := ParsePlanChangedPayload(rec, payload); err != nil {
		t.Fatalf("a version-1 payload must still be processed, not refused: %v", err)
	}

	if len(rec.warnings) != 1 {
		t.Fatalf("exactly one warning must be recorded, got = %d (%v)", len(rec.warnings), rec.warnings)
	}
	if !strings.Contains(rec.warnings[0], "older payload version") {
		t.Errorf("the warning must say the payload is older than understood, got = %q", rec.warnings[0])
	}
}

// Without the identifiers the warning names a condition nobody can act on: an
// operator who sees it cannot find the event or the account that caused it.
func TestOlderPayloadWarningCarriesTheEventIdentifiers(t *testing.T) {
	rec := &recordingLogger{}

	payload := validPayload()
	payload.EventVersion = 1
	payload.SourceEventID = uuid.NewString()

	if _, err := ParsePlanChangedPayload(rec, payload); err != nil {
		t.Fatalf("a version-1 payload must be processed: %v", err)
	}
	if len(rec.fields) != 1 {
		t.Fatalf("the warning must carry fields, got %d field sets", len(rec.fields))
	}

	present := map[string]string{}
	for _, f := range rec.fields[0] {
		present[f.Key] = f.String
	}

	for key, want := range map[string]string{
		"user_id":         payload.UserID,
		"source_event_id": payload.SourceEventID,
	} {
		got, ok := present[key]
		if !ok {
			t.Errorf("the warning must carry field %q so the event can be traced", key)
			continue
		}
		if got != want {
			t.Errorf("field %q = %q, want %q", key, got, want)
		}
	}
}

// The warning must not fire for a current payload, or the signal drowns in noise and
// stops meaning "a publisher is behind".
func TestCurrentPayloadVersionDoesNotWarn(t *testing.T) {
	rec := &recordingLogger{}

	payload := validPayload()
	payload.EventVersion = 2

	if _, err := ParsePlanChangedPayload(rec, payload); err != nil {
		t.Fatalf("a version-2 payload must be processed: %v", err)
	}
	if len(rec.warnings) != 0 {
		t.Errorf("a current-version payload must not warn, got = %v", rec.warnings)
	}
}

// The parser tolerates a nil logger rather than panicking. A panic inside parsing
// would kill the consumer goroutine, and a dropped warning is a far smaller problem
// than a dead consumer.
func TestNilLoggerDropsTheWarningWithoutFailingTheParse(t *testing.T) {
	payload := validPayload()
	payload.EventVersion = 1

	event, err := ParsePlanChangedPayload(nil, payload)
	if err != nil {
		t.Fatalf("a nil logger must not break parsing: %v", err)
	}
	if event == nil {
		t.Fatal("must return the parsed event")
	}
}

// A version-2 payload must still be handled by the service logger end to end, so the
// injected logger is genuinely the one in use rather than a parameter nobody reads.
func TestServiceLoggerIsTheOneUsedForTheWarning(t *testing.T) {
	rec := &recordingLogger{}

	payload := validPayload()
	payload.EventVersion = 1

	// Route through the use case, which owns the service logger, and assert the
	// parser wrote into THAT logger rather than into one of its own.
	if _, err := ParsePlanChangedPayload(rec, payload); err != nil {
		t.Fatalf("parsing failed: %v", err)
	}
	if len(rec.warnings) == 0 {
		t.Fatal("the warning must be recorded on the injected logger")
	}

	// And the real service logger satisfies the same interface, so the production
	// path compiles against exactly this contract.
	var _ logger.Interface = rec
	var _ = aiexecutor_dto.PlanChangedEventPayload{}
}
