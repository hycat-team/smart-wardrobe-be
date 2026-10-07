package grant

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// 027 US1: resolveGrantWindow is the function the whole feature turns on, so it
// gets its own test, driven by the SAME fakeGrantRepo used by grant_uc_test.go -
// no second fake in this package. The fake already answers
// GetNewestGrantForOrdering, which is the only read resolveGrantWindow makes.
//
// The logger is the package's existing `recordingLogger` (grant_payload_parser_logger_test.go),
// reused rather than redeclared.
// ---------------------------------------------------------------------------

// fallbackWarning is one ordering-floor fallback line lifted out of the captured
// log, tagged with the level that produced it.
type fallbackWarning struct {
	message string
	fields  map[string]string
	source  clockSource
}

// The two fallback warnings are told apart by message CONTENT. Both FR-003 and
// FR-018 carry user_id and grant_id, so the fields alone cannot distinguish "the
// snapshot had no occurred_at" from "created_at was unusable" - only the message
// can, and the fragments below are the words that differ.
const (
	occurredAtFallbackFragment    = "occurred_at"
	effectiveFromFallbackFragment = "effective_from"
)

// floorWarnings returns the captured lines that are ordering-floor fallbacks, each
// tagged with the level that produced it. Unrelated warnings (the rejection
// counter's, for example) carry neither fragment and are dropped.
func floorWarnings(log *recordingLogger) []fallbackWarning {
	var out []fallbackWarning
	for i, message := range log.warnings {
		var source clockSource
		switch {
		case strings.Contains(message, occurredAtFallbackFragment):
			source = clockCreatedAt
		case strings.Contains(message, effectiveFromFallbackFragment):
			source = clockEffectiveFrom
		default:
			continue
		}
		fields := map[string]string{}
		for _, f := range log.fields[i] {
			fields[f.Key] = f.String
		}
		out = append(out, fallbackWarning{message: message, fields: fields, source: source})
	}
	return out
}

// warningsForSource filters floorWarnings down to one fallback branch.
func warningsForSource(log *recordingLogger, source clockSource) []fallbackWarning {
	var out []fallbackWarning
	for _, warning := range floorWarnings(log) {
		if warning.source == source {
			out = append(out, warning)
		}
	}
	return out
}

func newResolutionUC(repo repositories.IAICostRepository, log *recordingLogger) *GrantUseCase {
	return NewGrantUseCase(repo, &recordingUow{}, log, NewGrantMetrics(log)).(*GrantUseCase)
}

// priorGrantForTest builds the account's newest stored grant WITHOUT a readable
// snapshot, so the three-level floor falls through level 1 to created_at or
// effective_from. It is the production shape the repair is about: a row written
// before the snapshot envelope existed.
func priorGrantForTest(createdAt, effectiveFrom time.Time) *aiexecutor_entities.AIPolicyGrant {
	return &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New(), CreatedAt: createdAt},
		},
		UserID:        uuid.MustParse(validUserID),
		EffectiveFrom: effectiveFrom,
	}
}

// priorGrantWithOccurredAt builds the healthy shape: a readable snapshot whose
// envelope carries occurred_at, so the stale floor is taken from the BUSINESS time
// (level 1) and nothing is logged. created_at and effective_from are set too, so a
// test can prove which of the three levels the floor actually came from.
func priorGrantWithOccurredAt(t *testing.T, occurredAt, createdAt, effectiveFrom time.Time) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	prior := priorGrantForTest(createdAt, effectiveFrom)
	prior.PolicySnapshot = snapshotWithOccurredAt(t, occurredAt)
	return prior
}

// activeEvent builds a plain (non-renewal) change whose window starts at
// effectiveFrom. EventGrantWindow returns EffectiveAt verbatim for this type, so
// the expectation is the event's own declared start (K1).
func activeEvent(effectiveFrom, effectiveTo, occurredAt time.Time) *PlanChangedEvent {
	to := effectiveTo
	return &PlanChangedEvent{
		EventVersion: 2,
		EventType:    subscriptioneventtype.ACTIVATED,
		UserID:       uuid.MustParse(validUserID),
		OccurredAt:   occurredAt,
		EffectiveAt:  effectiveFrom,
		EffectiveTo:  &to,
	}
}

func resolveOnce(t *testing.T, uc *GrantUseCase, event *PlanChangedEvent) GrantWindow {
	t.Helper()
	window, err := uc.resolveGrantWindow(context.Background(), event)
	if err != nil {
		t.Fatalf("resolveGrantWindow: %v", err)
	}
	return window
}

func requireStaleRejection(t *testing.T, err error) {
	t.Helper()
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("expected a RejectionError, got %T: %v", err, err)
	}
	if rejection.Reason != ReasonStaleEvent {
		t.Fatalf("expected reason %q, got %q", ReasonStaleEvent, rejection.Reason)
	}
}

func TestResolveGrantWindow(t *testing.T) {
	s := windowBase

	t.Run("no prior grant keeps the event window and logs nothing", func(t *testing.T) {
		event := activeEvent(s, s.Add(30*24*time.Hour), s)
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{}, log)

		window := resolveOnce(t, uc, event)
		if !window.EffectiveFrom.Equal(EventGrantWindow(event).EffectiveFrom) {
			t.Fatalf("expected the event window %v, got %v",
				EventGrantWindow(event).EffectiveFrom, window.EffectiveFrom)
		}
		requireNoFloorWarnings(t, log)
	})

	t.Run("event newer than the new floor is not shifted and logs nothing", func(t *testing.T) {
		// occurred_at s-1h (level 1); effective_from s-3h. order floor = occurred_at
		// s-1h, event starts at s.
		prior := priorGrantWithOccurredAt(t, s.Add(-time.Hour), s.Add(-2*time.Hour), s.Add(-3*time.Hour))
		event := activeEvent(s, s.Add(30*24*time.Hour), s)
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window := resolveOnce(t, uc, event)
		if !window.EffectiveFrom.Equal(s) {
			t.Fatalf("expected no shift (FR-007), got %v", window.EffectiveFrom)
		}
		requireNoFloorWarnings(t, log)
	})

	t.Run("occurred_at in the snapshot is the floor and logs nothing (FR-001)", func(t *testing.T) {
		// occurred_at is s+5s, created_at is s (usable), effective_from s-1h. A
		// created_at floor would be s and would leave the window unshifted; the
		// occurred_at floor is s+5s, so a window declared to start there must move one
		// microsecond past it. Level 1 is not allowed to warn.
		prior := priorGrantWithOccurredAt(t, s.Add(5*time.Second), s, s.Add(-time.Hour))
		event := activeEvent(s.Add(5*time.Second), s.Add(time.Hour), s.Add(6*time.Second))
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window := resolveOnce(t, uc, event)
		if want := s.Add(5*time.Second + time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected the occurred_at floor %v, got %v", want, window.EffectiveFrom)
		}
		requireNoFloorWarnings(t, log)
	})

	t.Run("unreadable snapshot falls back to created_at and warns (FR-003)", func(t *testing.T) {
		// No snapshot -> level 2. created_at s, effective_from s-1h. The event is
		// newer than created_at (so it is not refused), but its declared start is one
		// microsecond before the created_at floor, so it shifts to s+1us.
		prior := priorGrantForTest(s, s.Add(-time.Hour))
		event := activeEvent(s.Add(-time.Microsecond), s.Add(time.Hour), s.Add(time.Second))
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window := resolveOnce(t, uc, event)
		if want := s.Add(time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected the created_at floor %v, got %v", want, window.EffectiveFrom)
		}
		requireFloorWarning(t, log, prior, clockCreatedAt)
	})

	t.Run("event one microsecond below the floor shifts to floor+1us", func(t *testing.T) {
		prior := priorGrantWithOccurredAt(t, s, s.Add(-time.Hour), s.Add(-2*time.Hour))
		event := activeEvent(s.Add(-time.Microsecond), s.Add(time.Hour), s.Add(time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		if want := s.Add(time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected %v, got %v", want, window.EffectiveFrom)
		}
	})

	t.Run("event past the stale floor but before the order floor shifts up to it", func(t *testing.T) {
		// stale floor = occurred_at s; order floor = effective_from s+100us.
		prior := priorGrantWithOccurredAt(t, s, s.Add(-time.Hour), s.Add(100*time.Microsecond))
		event := activeEvent(s.Add(50*time.Microsecond), s.Add(time.Hour), s.Add(time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		if want := s.Add(101 * time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected %v, got %v", want, window.EffectiveFrom)
		}
	})

	t.Run("a skew larger than one microsecond only overshoots the floor by one microsecond", func(t *testing.T) {
		// FR-006 measures slack around the floor, not the total shift: the floor is
		// s+5s, the event starts at s, and the result is s+5s+1us.
		prior := priorGrantWithOccurredAt(t, s.Add(5*time.Second), s.Add(-time.Hour), s.Add(4*time.Second))
		event := activeEvent(s, s.Add(time.Hour), s.Add(6*time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		if want := s.Add(5*time.Second + time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected %v, got %v", want, window.EffectiveFrom)
		}
	})

	t.Run("delayed queue: an event between occurred_at and created_at is accepted (FR-001 regression)", func(t *testing.T) {
		// The refusal that prioritising occurred_at removes. The stored change
		// happened at T1 but the row was written at T1+10s; the new event happened at
		// T1+5s. It is newer than T1, so it must be accepted. A created_at floor
		// (T1+10s) would refuse it, because T1+5s <= T1+10s. This case is red on the
		// old two-level code.
		t1 := s
		prior := priorGrantWithOccurredAt(t, t1, t1.Add(10*time.Second), t1.Add(-time.Hour))
		event := activeEvent(t1.Add(5*time.Second), t1.Add(time.Hour), t1.Add(5*time.Second))
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window, err := uc.resolveGrantWindow(context.Background(), event)
		if err != nil {
			t.Fatalf("an event newer than occurred_at must be accepted, not refused: %v", err)
		}
		if !window.EffectiveFrom.Equal(t1.Add(5 * time.Second)) {
			t.Fatalf("expected the event window start %v, got %v",
				t1.Add(5*time.Second), window.EffectiveFrom)
		}
		requireNoFloorWarnings(t, log)
	})

	t.Run("created_at in the future falls back to effective_from and warns (FR-018)", func(t *testing.T) {
		futureCreatedAt := time.Now().UTC().Add(time.Hour)
		prior := priorGrantForTest(futureCreatedAt, s.Add(-time.Hour))
		event := activeEvent(s, s.Add(time.Hour), s)
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window := resolveOnce(t, uc, event)
		if !window.EffectiveFrom.Equal(s) {
			t.Fatalf("expected the event window %v, got %v", s, window.EffectiveFrom)
		}
		requireFloorWarning(t, log, prior, clockEffectiveFrom)
	})

	t.Run("zero created_at falls back to effective_from and warns (FR-018)", func(t *testing.T) {
		prior := priorGrantForTest(time.Time{}, s.Add(-time.Hour))
		event := activeEvent(s, s.Add(time.Hour), s)
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window := resolveOnce(t, uc, event)
		if !window.EffectiveFrom.Equal(s) {
			t.Fatalf("expected the event window %v, got %v", s, window.EffectiveFrom)
		}
		requireFloorWarning(t, log, prior, clockEffectiveFrom)
	})

	t.Run("a future grant is shifted past by its effective_from, not its created_at (H2)", func(t *testing.T) {
		// occurred_at and created_at are in the past, effective_from is 30 days out.
		// The order floor must be the future effective_from.
		prior := priorGrantWithOccurredAt(t, s, s, s.AddDate(0, 0, 30))
		event := activeEvent(s.Add(time.Second), s.AddDate(0, 0, 60), s.Add(time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		want := s.AddDate(0, 0, 30).Add(time.Microsecond)
		if !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected %v (future.effective_from+1us), got %v", want, window.EffectiveFrom)
		}
	})

	t.Run("event whose occurrence equals the stale floor is refused (FR-002)", func(t *testing.T) {
		prior := priorGrantForTest(s, s.Add(-time.Hour))
		event := activeEvent(s.Add(time.Hour), s.Add(2*time.Hour), s)
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		_, err := uc.resolveGrantWindow(context.Background(), event)
		requireStaleRejection(t, err)
	})

	t.Run("event older than the stale floor is refused (FR-002)", func(t *testing.T) {
		prior := priorGrantForTest(s, s.Add(-time.Hour))
		event := activeEvent(s.Add(time.Hour), s.Add(2*time.Hour), s.Add(-time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		_, err := uc.resolveGrantWindow(context.Background(), event)
		requireStaleRejection(t, err)
	})

	t.Run("a shift that empties the window is refused (FR-017)", func(t *testing.T) {
		prior := priorGrantForTest(s, s.Add(-time.Hour))
		// stale floor = s, order floor = s, so the start shifts to s+1us; the end is
		// exactly s+1us, which is not strictly after the shifted start.
		event := activeEvent(s.Add(-time.Second), s.Add(time.Microsecond), s.Add(time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		_, err := uc.resolveGrantWindow(context.Background(), event)
		requireStaleRejection(t, err)
	})

	t.Run("EXTENDED with no prior keeps the computed renewal start", func(t *testing.T) {
		// K1: the no-shift path must not collapse an EXTENDED window onto OccurredAt.
		event := renewalEvent(s.AddDate(0, 0, 30), 30)
		uc := newResolutionUC(&fakeGrantRepo{}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		if !window.EffectiveFrom.Equal(EventGrantWindow(event).EffectiveFrom) {
			t.Fatalf("expected the computed renewal start %v, got %v",
				EventGrantWindow(event).EffectiveFrom, window.EffectiveFrom)
		}
	})
}

// requireFloorWarning asserts the log carries exactly one ordering-floor fallback
// line, from the expected level, with the identifiers an operator needs.
func requireFloorWarning(t *testing.T, log *recordingLogger, prior *aiexecutor_entities.AIPolicyGrant, source clockSource) {
	t.Helper()
	warnings := floorWarnings(log)
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one floor fallback warning, got %d (%v)", len(warnings), log.warnings)
	}
	if warnings[0].source != source {
		t.Fatalf("expected the fallback from source %d, got %d (message %q)",
			source, warnings[0].source, warnings[0].message)
	}
	if got := warnings[0].fields["user_id"]; got != validUserID {
		t.Fatalf("the warning must carry user_id %q, got %q", validUserID, got)
	}
	if got := warnings[0].fields["grant_id"]; got != prior.ID.String() {
		t.Fatalf("the warning must carry grant_id %q, got %q", prior.ID.String(), got)
	}
}

// requireNoFloorWarnings asserts no ordering-floor fallback was logged at all.
func requireNoFloorWarnings(t *testing.T, log *recordingLogger) {
	t.Helper()
	if got := floorWarnings(log); len(got) != 0 {
		t.Fatalf("no floor fallback must be logged, got %v", got)
	}
}
