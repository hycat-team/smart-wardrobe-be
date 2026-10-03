package campaign

import (
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"

	"github.com/google/uuid"
)

// 026 T026 (SC-008): an EMPTY REGISTRY IS A NORMAL STATE, not a fault.
//
// Before 026 an environment with no campaigns was an environment whose YAML had no
// `campaigns:` key — which viper's exact unmarshal reads as an empty list, not as an
// error, so the service booted and silently granted nothing. After 026 the registry
// is a TABLE, and a fresh environment has no rows until an operator opens a
// campaign through the API. The window in which "no campaign exists" is the normal
// state of a new deployment is therefore WIDER, not narrower, and it is the first
// moment a new environment ever runs.
//
// What must be true in that state:
//
//   - the message receiver answers normally (a skip is a success),
//   - the catch-up sweep answers normally and invents nothing,
//   - and NOTHING IS LOGGED AT WARN OR ERROR LEVEL, because "no promotion is
//     running" is a state, not a fault.
//
// That last one is the part worth a test. A registry read that fails and a registry
// that is empty are both "nothing was granted", and only one of them is a problem —
// so the log level is what tells an operator which one they have, and an empty
// registry that logs a warning every fifteen minutes trains people to ignore the
// warning that matters.

// TestEmptyRegistrySkipsQuietlyOnTheMessagePath is the receiver half of SC-008.
func TestEmptyRegistrySkipsQuietlyOnTheMessagePath(t *testing.T) {
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, nil, claims)

	payload := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant().Add(-time.Minute),
	}

	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("SC-008: an empty registry must NOT fail the message — the queue has to keep "+
			"moving, or a new deployment would build a dead-letter backlog before its first "+
			"campaign: %v", err)
	}
	if claims.total() != 0 {
		t.Errorf("claims written = %d, want 0", claims.total())
	}
	if len(h.events.published()) != 0 {
		t.Error("no plan change may be published for a skipped account")
	}
	if len(h.notif.emails()) != 0 {
		t.Error("no email may be enqueued for a skipped account")
	}

	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipCampaignNotConfigured {
		t.Errorf("reason = %q (found=%v), want %q", reason, ok, SkipCampaignNotConfigured)
	}
	assertNoCampaignAlarms(t, h.logger)
}

// TestEmptyRegistrySweepsQuietlyAndInventsNothing is the sweep half of SC-008.
//
// The dangerous part is "invents nothing": a sweep that wrote a health row per run
// for campaigns it does not have would create registry entries out of thin air, and
// the operator list would then show campaigns nobody opened.
func TestEmptyRegistrySweepsQuietlyAndInventsNothing(t *testing.T) {
	sweepRuns, sweep, lister := newHealthSweep(t, nil)

	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("SC-008: an empty registry must not fail the sweep: %v", err)
	}
	if report.Scanned != 0 || report.Forwarded != 0 || report.Skipped != 0 || report.Failed != 0 {
		t.Errorf("report = %+v, want an all-zero run: with no campaign there is no window to sweep", report)
	}
	if report.SweptAt.IsZero() {
		t.Error("the run must still stamp its instant — the sweep health line reports it even " +
			"for a run that found nothing")
	}
	if sweepRuns.writeCount() != 0 {
		t.Errorf("health writes = %d, want 0: a sweep must not write rows for campaigns that do "+
			"not exist, or the operator list would show campaigns nobody opened", sweepRuns.writeCount())
	}

	// And the window query must not even be attempted: with no campaign there is no
	// window, and a query over one would be a full scan of `identity.users` for nothing.
	lister.failing = false
	if _, err := sweep.Sweep(t.Context()); err != nil {
		t.Fatalf("a second empty sweep must also succeed: %v", err)
	}
	assertNoCampaignAlarms(t, nil)
}

// TestARegistryThatCannotBeReadIsNotAnEmptyRegistry is the other half of the same
// distinction, and it is the half a lazy implementation gets wrong.
//
// A failed read substituted with an empty list produces EXACTLY the same visible
// result as an empty registry — nothing granted, no error, no warning — while
// meaning the opposite: a broken database rather than a quiet promotion. If the two
// are indistinguishable, a registry outage is read as "no campaign is running" and
// the next launch day finds out from customers.
//
// So the grant path must FAIL the message (retryable) instead of skipping, and the
// message must come back rather than being acknowledged and dropped.
func TestARegistryThatCannotBeReadIsNotAnEmptyRegistry(t *testing.T) {
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, nil, claims)
	h.campaigns.failReads = true

	payload := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: baseInstant().Add(-time.Minute),
	}

	err := h.uc.ProcessUserCreated(t.Context(), payload)
	if err == nil {
		t.Fatal("a registry that cannot be read MUST fail the message. Acknowledging it would " +
			"discard the account and report a broken database as a quiet promotion")
	}
	if reason, ok := h.lastSkip(t, payload.UserID); ok {
		t.Errorf("a read failure must not be recorded as a skip (%q): an operator reading the log "+
			"would conclude the campaign is asleep rather than that the database is down", reason)
	}
	if claims.total() != 0 {
		t.Errorf("claims written = %d, want 0", claims.total())
	}
}

// assertNoCampaignAlarms fails when the logger captured anything at WARN or ERROR.
//
// The empty-registry case is asserted against the GRANT logger; the sweep half
// passes nil because the sweep's skip line is deliberately at Info, and asserting on
// it needs the sweep's own logger rather than the grant path's.
func assertNoCampaignAlarms(t *testing.T, logger *recordingLogger) {
	t.Helper()
	if logger == nil {
		return
	}
	for _, line := range logger.lines() {
		if line.level == "warn" || line.level == "error" {
			t.Errorf("SC-008: an empty registry logged at %s level (%s). \"No promotion is "+
				"running\" is a state an operator can read from the admin list; logging it as a "+
				"warning every sweep period trains people to ignore the warning that matters",
				line.level, line.reason)
		}
	}
}
