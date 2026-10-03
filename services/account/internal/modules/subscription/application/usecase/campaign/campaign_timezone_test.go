package campaign

import (
	"context"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 026 T028 (FR-015): every `now` comparison in this package goes through
// `cfg.Database.TimeZone`, never through the machine's clock.
//
// THE FAILURE THIS GUARDS, and why a passing suite in Asia/Ho_Chi_Minh cannot catch
// it. `campaigns.starts_at` and `campaigns.ends_at` are TIMESTAMPTZ columns, so the
// instant they hold is unambiguous — the ambiguity is in WHICH INSTANT `now` is. A
// container running UTC against a database configured for Asia/Ho_Chi_Minh compares
// a Vietnam-time window boundary with a UTC clock and is wrong by seven hours. The
// wrong answers are the expensive kind:
//
//   - a campaign that opens at 00:00 ICT is treated as already closed at 00:00 UTC
//     the same day, so nobody is granted anything on launch morning;
//   - a campaign that ends at 23:59 ICT is treated as still running, so it grants
//     seven hours past its end.
//
// 025 attacked this by NORMALISING the campaign boundaries at load time
// (`parseCampaignWindows`), which is why the old fixtures carried both `StartsAt`
// and `StartsAtTime`. 026 removed that function, and the normalisation with it, so
// the guarantee now rests entirely on the clock FUNCTION rather than on a derived
// field kept in sync by hand.
//
// So the test does what a real deployment does: it runs the same evaluation under
// two DIFFERENT machine time zones, with a campaign whose boundary sits exactly on
// the hour boundary between them, and requires the SAME ANSWER. `time.Local` is the
// machine's zone; nothing in the campaign package is allowed to read it.

// TestCampaignDecisionsDoNotFollowTheMachineTimeZone is FR-015's behavioural half.
//
// The campaign is written in ICT (+07:00) and its window starts on the exact instant
// that is a different calendar day in UTC. If any comparison read the machine's
// clock, running the suite in a UTC CI container would produce a different answer
// from a developer machine in ICT — and both would look correct.
func TestCampaignDecisionsDoNotFollowTheMachineTimeZone(t *testing.T) {
	// 2026-10-02T00:00:00+07:00 is 2026-10-01T17:00:00Z — a DIFFERENT DAY in UTC.
	// That is the whole point of the fixture: a zone-blind comparison crosses a date
	// boundary, and a date boundary is where an operator notices.
	startsAt := time.Date(2026, 10, 2, 0, 0, 0, 0, ictLocation())
	endsAt := time.Date(2026, 10, 31, 23, 59, 0, 0, ictLocation())
	campaign := testCampaign("launch-2026-10", "premium-monthly", 50, 5, &startsAt, &endsAt)

	// `now` is what the use case reads: the CONFIGURED zone, read once per call.
	// 2026-10-15T12:00:00+07:00 — comfortably inside the window under any clock.
	inside := time.Date(2026, 10, 15, 12, 0, 0, 0, ictLocation())

	// One minute BEFORE the window, in the configured zone. Any implementation that
	// read the machine's zone would get a different answer from the one below.
	beforeWindow := startsAt.Add(-time.Minute)

	for _, machineZone := range []string{"UTC", "America/New_York", "Asia/Tokyo", "Pacific/Kiritimati"} {
		t.Run(machineZone, func(t *testing.T) {
			restore := swapMachineTimeZone(t, machineZone)

			insideCode := grantDecision(t, campaign, inside, inside)
			if insideCode != campaign.Code {
				t.Errorf("machine zone %s: an account registered inside the window was NOT granted; "+
					"the answer followed the machine clock instead of cfg.Database.TimeZone", machineZone)
			}

			// An account registered BEFORE the window is out of it in every zone.
			beforeCode := grantDecision(t, campaign, inside, beforeWindow)
			if beforeCode != "" {
				t.Errorf("machine zone %s: selected %q for an account registered before the window "+
					"opened", machineZone, beforeCode)
			}

			restore()
		})
	}
}

// TestThePackageNeverReadsTheMachineClock is FR-015's structural half.
//
// The behavioural test above can only check the comparisons it thought of. This one
// checks the RULE: no file in this package may call `time.Now()` at all, because
// every instant has to come from the `now func() time.Time` field the constructors
// fill from `timeutils.GetNow(cfg.Database.TimeZone)`.
//
// `timeutils` is the single allowed door, and it is excluded because the sanctioned
// path to the configured zone goes through it.
func TestThePackageNeverReadsTheMachineClock(t *testing.T) {
	for _, source := range campaignSourceFiles(t) {
		if containsForbiddenCall(source.body, "time.Now(") {
			t.Errorf("FR-015: %s calls time.Now(). Every instant in this package must come from "+
				"the injected `now` field, which is filled from cfg.Database.TimeZone — reading "+
				"the machine clock makes a campaign window mean a different day depending on "+
				"which host the replica runs on", source.name)
		}
		if containsForbiddenCall(source.body, "time.Local") {
			t.Errorf("FR-015: %s reads time.Local. The zone is configuration, not a property of "+
				"the host (constitution §VIII)", source.name)
		}
	}
}

// TestCampaignBoundariesCarryAnExplicitZone is the other half of FR-015, on the
// data rather than on the code.
//
// Before 026 a campaign boundary was an RFC3339 STRING and `parseCampaignWindows`
// refused a zone-less one at startup. Now it is a `time.Time` from a TIMESTAMPTZ
// column, and the equivalent guarantee is that the tests and the seed never
// construct a boundary in a zone-less way — a bare `time.Date(...)` with
// `time.UTC` where the production value is ICT is how a boundary test stops testing
// the boundary.
func TestCampaignBoundariesCarryAnExplicitZone(t *testing.T) {
	startsAt := time.Date(2026, 10, 2, 0, 0, 0, 0, ictLocation())
	endsAt := time.Date(2026, 10, 31, 23, 59, 0, 0, ictLocation())
	campaign := testCampaign("launch-2026-10", "premium-monthly", 50, 5, &startsAt, &endsAt)

	zone := campaign.StartsAt.Location()
	if offset := zoneOffsetSeconds(zone); offset != 7*60*60 {
		t.Errorf("fixture start offset = %d seconds, want %d (ICT). A fixture whose boundary sits "+
			"in the machine's own zone cannot detect a zone-blind comparison", offset, 7*60*60)
	}
	// The boundary really does cross a UTC date line, or the fixture proves nothing.
	if campaign.StartsAt.UTC().Day() == campaign.StartsAt.Day() {
		t.Error("fixture start does not cross a UTC date boundary; a boundary that is the same " +
			"calendar day everywhere cannot catch a seven-hour offset")
	}
}

// ---- helpers ----

// grantDecision runs one message through the grant path with a clock pinned to
// `now`, and returns the campaign code it granted from — or "".
func grantDecision(t *testing.T, campaign *entities.Campaign, now, registeredAt time.Time) string {
	t.Helper()

	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{campaign}, claims)
	h.uc.now = func() time.Time { return now }

	payload := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: registeredAt,
	}
	if err := h.uc.ProcessUserCreated(context.Background(), payload); err != nil {
		t.Fatalf("process account: %v", err)
	}
	claim := claims.claimFor(payload.UserID)
	if claim == nil {
		return ""
	}
	return claim.CampaignCode
}

// swapMachineTimeZone points `time.Local` at another zone for the duration of a
// sub-test and returns the restore function.
//
// `time.Local` is process-wide, so this is mutated rather than injected — which is
// exactly WHY the production code may not read it. A parallel test in this package
// would see the swapped zone; none of them read the machine zone, which the
// structural test above enforces.
func swapMachineTimeZone(t *testing.T, name string) func() {
	t.Helper()
	previous := time.Local
	time.Local = mustLoadLocation(t, name)
	return func() { time.Local = previous }
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %s: %v", name, err)
	}
	return loc
}

// ictLocation is the zone the deployed configuration uses.
func ictLocation() *time.Location {
	// Fixed rather than loaded: `Asia/Ho_Chi_Minh` has had no DST since 1975, and a
	// fixed zone keeps the fixture identical on a machine whose zone database is
	// missing or out of date.
	return time.FixedZone("ICT", 7*60*60)
}

func zoneOffsetSeconds(loc *time.Location) int {
	_, offset := time.Date(2026, 10, 2, 0, 0, 0, 0, loc).Zone()
	return offset
}

// containsForbiddenCall reports whether the source calls the forbidden function.
//
// It matches the CALL and not the bare name, because these files legitimately MENTION
// `time.Now` in the comments that explain why they must not call it — and a guard
// that fired on its own explanation would be deleted within a week.
func containsForbiddenCall(body, call string) bool {
	for index := 0; ; {
		found := indexOf(body, call, index)
		if found < 0 {
			return false
		}
		// Skip the occurrence if it is inside a line comment.
		lineStart := lastIndexOf(body, "\n", found) + 1
		if indexOfFrom(body[lineStart:found], "//") < 0 {
			return true
		}
		index = found + len(call)
	}
}

func indexOf(haystack, needle string, from int) int {
	if from >= len(haystack) {
		return -1
	}
	relative := indexOfFrom(haystack[from:], needle)
	if relative < 0 {
		return -1
	}
	return from + relative
}

func indexOfFrom(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func lastIndexOf(haystack, needle string, before int) int {
	if before > len(haystack) {
		before = len(haystack)
	}
	for i := before; i >= 0; i-- {
		if i+len(needle) <= len(haystack) && haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
