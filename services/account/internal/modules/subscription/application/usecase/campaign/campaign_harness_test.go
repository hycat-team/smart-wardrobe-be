package campaign

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	sharedconfig "smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 025: shared scaffolding for the campaign use-case tests.
//
// The instant every test anchors to. Fixed rather than "now" so that a boundary
// test cannot start failing because a day rolled over while the suite ran.
func baseInstant() time.Time {
	return time.Date(2026, 10, 2, 0, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60))
}

func timePtr(t time.Time) *time.Time { return &t }

// testHarness wires a use case against fakes and records what it decided.
type testHarness struct {
	uc        *CampaignUseCase
	claims    *fakeClaimRepo
	subs      *fakeSubRepo
	plans     *fakePlanRepo
	events    *fakeEventRepo
	publisher *fakePublisher
	audit     *fakeAuditContract
	notif     *fakeNotifContract
	logger    *recordingLogger
}

// newTestUseCase builds the use case with a campaign list and a claim ledger.
func newTestUseCase(t *testing.T, campaigns []sharedconfig.CampaignConfig, claims *fakeClaimRepo) *testHarness {
	t.Helper()
	h := &testHarness{
		claims: claims,
		subs:   newFakeSubRepo(),
		plans:  newFakePlanRepo(),
		events: newFakeEventRepo(),
		audit:  &fakeAuditContract{},
		notif:  &fakeNotifContract{},
		logger: &recordingLogger{},
	}
	h.plans.put(testPlan("premium-monthly", 30))
	h.plans.put(testPlan("premium-quarterly", 90))
	h.publisher = &fakePublisher{}
	h.uc = NewCampaignUseCase(
		campaignConfig(campaigns),
		claims,
		h.subs,
		h.plans,
		subscription.NewPlanChangePublisher(h.events, h.publisher),
		h.audit,
		h.notif,
		nil,
		// Releases the fake's advisory lock on the way out, the way COMMIT does.
		releasingUow{claims: claims},
		h.logger,
	).(*CampaignUseCase)
	// Pin the clock inside the campaign's window. Production reads the configured
	// zone; a test that used the real clock would pass or fail depending on the day
	// the suite ran, and a boundary test that stops testing its boundary the day
	// after launch is worse than no test at all.
	h.uc.now = func() time.Time { return baseInstant() }
	return h
}

// releasingUow releases the fake claim ledger's advisory lock when the closure
// returns, modelling `pg_advisory_xact_lock` being released at COMMIT or ROLLBACK
// (R3).
//
// The release is unconditional, including on error: a lock that outlives a
// rollback is exactly the leak the transaction-scoped lock was chosen to avoid,
// and a fake that only released on success would let that pass.
type releasingUow struct {
	claims *fakeClaimRepo
}

func (u releasingUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	defer u.claims.releaseLock()
	return fn(ctx)
}

// granted reports whether processing the payload produced a grant.
// granted reports whether processing the payload produced a NEW grant.
//
// Measured by the plan-change history rows rather than by the subscription's
// current plan: a redelivered message leaves the account on exactly the plan it
// was granted, so "is this account on premium-monthly" answers yes for all one
// hundred deliveries. What separates the first from the hundredth is whether a
// grant HAPPENED, and one grant is one history row (FR-036).
func (h *testHarness) granted(t *testing.T, payload identitydto.UserCreatedEventPayload) bool {
	t.Helper()
	before := len(h.events.published())
	if err := h.uc.ProcessUserCreated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error processing user %s: %v", payload.UserID, err)
	}
	return len(h.events.published()) > before
}

// lastSkip returns the reason the use case LOGGED for the account (FR-061).
//
// Read from the log rather than from a side map the use case maintains for tests:
// the log line is the deliverable FR-061 actually asks for, so asserting on it
// proves the requirement rather than a test-only echo of it.
func (h *testHarness) lastSkip(t *testing.T, userID uuid.UUID) (SkipReason, bool) {
	t.Helper()
	return h.logger.skipReasonFor(userID)
}

// recordingLogger captures what the use case reported.
//
// FR-060 and FR-061 require every grant and every skip to leave a trace naming
// the campaign and the reason, and that trace is a log line. Asserting on the
// captured fields is therefore asserting on the requirement, not on a test hook.
type recordingLogger struct {
	mu       sync.Mutex
	recorded []recordedLine
}

type recordedLine struct {
	level    string
	userID   string
	reason   string
	campaign string
	plan     string
}

func (l *recordingLogger) Debug(message string, fields ...zap.Field) {}
func (l *recordingLogger) Info(message string, fields ...zap.Field)  { l.record("info", fields) }
func (l *recordingLogger) Warn(message string, fields ...zap.Field)  { l.record("warn", fields) }
func (l *recordingLogger) Error(message string, fields ...zap.Field) { l.record("error", fields) }
func (l *recordingLogger) Fatal(message string, fields ...zap.Field) {}

func (l *recordingLogger) record(level string, fields []zap.Field) {
	line := recordedLine{level: level}
	for _, field := range fields {
		switch field.Key {
		case "user_id":
			line.userID = field.String
		case "reason":
			line.reason = field.String
		case "campaign_code":
			line.campaign = field.String
		case "plan_slug":
			line.plan = field.String
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.recorded = append(l.recorded, line)
}

// skipReasonFor returns the most recent skip reason recorded for the account.
func (l *recordingLogger) skipReasonFor(userID uuid.UUID) (SkipReason, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.recorded) - 1; i >= 0; i-- {
		line := l.recorded[i]
		if line.userID == userID.String() && line.reason != "" {
			return SkipReason(line.reason), true
		}
	}
	return "", false
}

// skipsFor returns every skip reason recorded for the account, in order.
func (l *recordingLogger) skipsFor(userID uuid.UUID) []SkipReason {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []SkipReason
	for _, line := range l.recorded {
		if line.userID == userID.String() && line.reason != "" {
			out = append(out, SkipReason(line.reason))
		}
	}
	return out
}

// lines returns every captured line, for assertions that a failure left a trace.
func (l *recordingLogger) lines() []recordedLine {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]recordedLine, len(l.recorded))
	copy(out, l.recorded)
	return out
}

// asMap unwraps a published payload so a test can assert on individual fields.
//
// The payload type is unexported in the mapper package, so it is read as a map
// rather than by importing it. What the test needs is that the RECEIVER gets the
// field, not which Go type carries it.
func asMap(t *testing.T, payload any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal published payload: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal published payload: %v", err)
	}
	return out
}

// campaignConfig wraps a campaign list the way the loader would have produced it.
func campaignConfig(campaigns []sharedconfig.CampaignConfig) *sharedconfig.Config {
	return &sharedconfig.Config{
		Campaigns: campaigns,
		Database:  sharedconfig.Database{TimeZone: "Asia/Ho_Chi_Minh"},
	}
}

// quietLogger discards everything. Used where a test asserts on state only and a
// captured log line would only clutter its failure output.
type quietLogger struct{}

func (quietLogger) Debug(string, ...zap.Field) {}
func (quietLogger) Info(string, ...zap.Field)  {}
func (quietLogger) Warn(string, ...zap.Field)  {}
func (quietLogger) Error(string, ...zap.Field) {}
func (quietLogger) Fatal(string, ...zap.Field) {}

// campaignSourceFile is one production source file of this package.
type campaignSourceFile struct {
	name string
	body string
}

// campaignSourceFiles reads every non-test .go file of this package.
//
// The textual guards (R2b, FR-091) cannot be expressed as Go assertions: the
// compiler cannot see that a WHERE clause subtracts a column, and the bug they
// guard is precisely one that type-checks perfectly.
func campaignSourceFiles(t *testing.T) []campaignSourceFile {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	var out []campaignSourceFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || filepath.Ext(name) == ".test" {
			continue
		}
		if len(name) > 8 && name[len(name)-8:] == "_test.go" {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		out = append(out, campaignSourceFile{name: name, body: string(body)})
	}
	return out
}
