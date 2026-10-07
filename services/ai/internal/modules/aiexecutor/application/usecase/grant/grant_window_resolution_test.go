package grant

import (
	"context"
	"errors"
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

// fallbackWarning is one FR-018 line lifted out of the captured log.
type fallbackWarning struct {
	message string
	fields  map[string]string
}

// fallbackWarnings filters the captured lines down to FR-018's warning. The
// rejection counter also logs through this sink, keyed by `reason`; FR-018's line
// is the one carrying `grant_id`, and asserting on that field is what keeps the
// two apart.
func fallbackWarnings(log *recordingLogger) []fallbackWarning {
	var out []fallbackWarning
	for i, message := range log.warnings {
		fields := map[string]string{}
		for _, f := range log.fields[i] {
			fields[f.Key] = f.String
		}
		if _, ok := fields["grant_id"]; ok {
			out = append(out, fallbackWarning{message: message, fields: fields})
		}
	}
	return out
}

func newResolutionUC(repo repositories.IAICostRepository, log *recordingLogger) *GrantUseCase {
	return NewGrantUseCase(repo, &recordingUow{}, log, NewGrantMetrics(log)).(*GrantUseCase)
}

// priorGrantForTest builds the account's newest stored grant. 027 decides
// ordering from CreatedAt (falling back to EffectiveFrom), so both are explicit.
func priorGrantForTest(createdAt, effectiveFrom time.Time) *aiexecutor_entities.AIPolicyGrant {
	return &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New(), CreatedAt: createdAt},
		},
		UserID:        uuid.MustParse(validUserID),
		EffectiveFrom: effectiveFrom,
	}
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
		if got := fallbackWarnings(log); len(got) != 0 {
			t.Fatalf("no floor fallback must be logged, got %v", got)
		}
	})

	t.Run("event newer than the new floor is not shifted", func(t *testing.T) {
		// created_at s-1h; effective_from s-2h. order floor = s-1h, event starts at s.
		prior := priorGrantForTest(s.Add(-time.Hour), s.Add(-2*time.Hour))
		event := activeEvent(s, s.Add(30*24*time.Hour), s)
		log := &recordingLogger{}
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, log)

		window := resolveOnce(t, uc, event)
		if !window.EffectiveFrom.Equal(s) {
			t.Fatalf("expected no shift (FR-007), got %v", window.EffectiveFrom)
		}
		if got := fallbackWarnings(log); len(got) != 0 {
			t.Fatalf("no floor fallback must be logged, got %v", got)
		}
	})

	t.Run("event one microsecond below the floor shifts to floor+1us", func(t *testing.T) {
		prior := priorGrantForTest(s, s.Add(-time.Hour))
		event := activeEvent(s.Add(-time.Microsecond), s.Add(time.Hour), s.Add(time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		if want := s.Add(time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected %v, got %v", want, window.EffectiveFrom)
		}
	})

	t.Run("event past the stale floor but before the order floor shifts up to it", func(t *testing.T) {
		// stale floor = created_at s; order floor = effective_from s+100us.
		prior := priorGrantForTest(s, s.Add(100*time.Microsecond))
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
		prior := priorGrantForTest(s.Add(5*time.Second), s.Add(4*time.Second))
		event := activeEvent(s, s.Add(time.Hour), s.Add(6*time.Second))
		uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

		window := resolveOnce(t, uc, event)
		if want := s.Add(5*time.Second + time.Microsecond); !window.EffectiveFrom.Equal(want) {
			t.Fatalf("expected %v, got %v", want, window.EffectiveFrom)
		}
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
		requireFallbackWarning(t, log, prior)
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
		requireFallbackWarning(t, log, prior)
	})

	t.Run("a future grant is shifted past by its effective_from, not its created_at (H2)", func(t *testing.T) {
		// created_at is in the past, effective_from is 30 days out. A floor taken
		// from created_at alone would let the new window start BEFORE this row and
		// TruncateGrantsBefore would not cut it. The order floor must be the
		// future effective_from.
		prior := priorGrantForTest(s, s.AddDate(0, 0, 30))
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

func requireFallbackWarning(t *testing.T, log *recordingLogger, prior *aiexecutor_entities.AIPolicyGrant) {
	t.Helper()
	warnings := fallbackWarnings(log)
	if len(warnings) != 1 {
		t.Fatalf("FR-018: expected exactly one fallback warning, got %d (%v)", len(warnings), log.warnings)
	}
	if got := warnings[0].fields["user_id"]; got != validUserID {
		t.Fatalf("FR-018: the warning must carry user_id %q, got %q", validUserID, got)
	}
	if got := warnings[0].fields["grant_id"]; got != prior.ID.String() {
		t.Fatalf("FR-018: the warning must carry grant_id %q, got %q", prior.ID.String(), got)
	}
}
