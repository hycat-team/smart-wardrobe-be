package campaign

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// 026 T052 — how an edit becomes a row, and how the row says what moved.
//
// WHY A SEPARATE FILE. `registry_uc.go` holds the two operations and their ordering;
// this file holds the one piece of PURE logic an edit needs before either of them
// runs. Keeping it separate is what lets the ordering above be read as ordering
// rather than as a diff of pointer comparisons, and it is the only part of US3 with
// no `ctx`, no transaction and no clock — a function a reader can check by hand.
//
// THE RULE, IN ONE SENTENCE: a field the request did not send keeps the stored value,
// and a field whose new value EQUALS the stored one is not a change at all.
// Both halves matter. The first is what makes a `PATCH` a `PATCH` (contract §4.1);
// the second is FR-055's no-op exception and US3 acceptance scenario 4.

// campaignChanges records WHICH FIELDS an edit moved, and it is the answer to the
// two questions the update path asks.
//
// It is a struct rather than a list of strings because both questions need the
// values, not only the names: the audit payload has to print the old and the new
// value of each changed field (FR-060), and only a change list built while the merge
// was happening has both of them to hand.
type campaignChanges struct {
	planSlug *string
	quota    *int
	reserve  *int
	startsAt *time.Time
	endsAt   *time.Time
}

// touchesBudget reports whether any of the three columns FR-020 freezes moved.
//
// IT IS THE WHOLE QUESTION FR-021's application layer asks, and the reason it is a
// method rather than three comparisons at the call site: the answer decides whether
// the ledger is read AT ALL. Moving `endsAt` on a campaign that has already granted
// is legal (FR-023), so a check that asked "does this campaign have grants?" first
// would refuse an edit the spec explicitly allows — and would pay for a ledger read
// on every timestamp edit to do it.
func (c campaignChanges) touchesBudget() bool {
	return c.planSlug != nil || c.quota != nil || c.reserve != nil
}

// empty reports whether the edit asked for nothing.
//
// It is FR-055's no-op condition, and it is asked of the CHANGE LIST rather than of
// the request: a request full of fields whose values already match the row is not an
// empty request, but it is a change of nothing, and it is the change list that knows
// the difference.
func (c campaignChanges) empty() bool {
	return !c.touchesBudget() && c.startsAt == nil && c.endsAt == nil
}

// names lists the JSON field names that moved, for the log line.
//
// In the API's spelling rather than Go's, for the same reason the validator's field
// constants are: a log an operator greps must name the field their form shows.
func (c campaignChanges) names() []string {
	names := make([]string, 0, 5)
	for _, field := range []struct {
		name  string
		moved bool
	}{
		{fieldPlanSlug, c.planSlug != nil},
		{fieldQuota, c.quota != nil},
		{fieldReserve, c.reserve != nil},
		{fieldStartsAt, c.startsAt != nil},
		{fieldEndsAt, c.endsAt != nil},
	} {
		if field.moved {
			names = append(names, field.name)
		}
	}
	return names
}

// auditSide renders one side of `payload.before` / `payload.after` (FR-060).
//
// ONLY THE CHANGED FIELDS, and `version` always — see `recordUpdateAudit`'s header
// for why an audit payload must not repeat unchanged values. Times are UTC RFC3339
// for the reason `endsAtForAudit` gives, and the window end renders as `""` rather
// than being omitted, so a reader never has to distinguish "absent" from "empty".
func (c campaignChanges) auditSide(campaign *entities.Campaign) map[string]any {
	side := map[string]any{"version": campaign.Version}
	if c.planSlug != nil {
		side[fieldPlanSlug] = campaign.PlanSlug
	}
	if c.quota != nil {
		side[fieldQuota] = campaign.Quota
	}
	if c.reserve != nil {
		side[fieldReserve] = campaign.Reserve
	}
	if c.startsAt != nil {
		side[fieldStartsAt] = campaign.StartsAt.UTC().Format(time.RFC3339)
	}
	if c.endsAt != nil {
		side[fieldEndsAt] = endsAtForAudit(campaign.EndsAt)
	}
	return side
}

// mergeCampaignUpdate builds the row the edit asks for and the list of fields it
// actually moves.
//
// THE CANDIDATE IS BUILT FROM THE STORED ROW, and that is forced rather than
// chosen: `CampaignRepository.Update` writes all five mutable columns in one
// statement (it must, so that `code` can never be rewritten and `closed_at` cannot
// be cleared — FR-024/FR-034), which means there is no such thing as "update only
// what was sent" at the SQL level. An omitted field therefore has to arrive
// pre-filled with the current value, or the statement would silently blank it.
//
// `closed_at`, `version`, `code` and the timestamps the entity carries are NOT
// copied into the change list, and the candidate deliberately leaves `ClosedAt` and
// `Version` as they are on the stored row: this function is not allowed to change
// them. Closing is `Close`'s statement and the version is `version + 1` inside
// PostgreSQL.
//
// EQUALITY IS COMPARED ON INSTANTS, not on their strings. A client sending
// `startsAt` in a different offset for the same moment has not moved the campaign,
// and treating that as a change would bump the version and write an audit row for an
// edit that did nothing — the same defect FR-055's no-op rule exists to prevent.
func mergeCampaignUpdate(
	stored *entities.Campaign,
	edit campaignUpdate,
) (*entities.Campaign, campaignChanges) {
	candidate := *stored
	var changes campaignChanges

	if edit.planSlug != nil && *edit.planSlug != stored.PlanSlug {
		candidate.PlanSlug = *edit.planSlug
		changes.planSlug = edit.planSlug
	}
	if edit.quota != nil && *edit.quota != stored.Quota {
		candidate.Quota = *edit.quota
		changes.quota = edit.quota
	}
	if edit.reserve != nil && *edit.reserve != stored.Reserve {
		candidate.Reserve = *edit.reserve
		changes.reserve = edit.reserve
	}
	if edit.startsAt != nil && !edit.startsAt.Equal(stored.StartsAt) {
		candidate.StartsAt = *edit.startsAt
		changes.startsAt = edit.startsAt
	}
	// `EndsAt` is compared as instants for the same reason, with the pointer taken
	// apart first: `stored.EndsAt` nil and an edit instant both being non-nil
	// pointers would make `*a == *b` compare instants, and nil-against-nil has to be
	// read as "unchanged" rather than dereferenced.
	if edit.endsAt != nil && (stored.EndsAt == nil || !edit.endsAt.Equal(*stored.EndsAt)) {
		candidate.EndsAt = edit.endsAt
		changes.endsAt = edit.endsAt
	}

	return &candidate, changes
}

// endsAtForAudit renders the optional window end for an audit payload.
//
// IT LIVES HERE BECAUSE THIS IS WHERE AUDIT PAYLOADS ARE RENDERED, and it has three
// callers — the create record, `campaignChanges.auditSide`, and any future audit
// shape — which is more than one, so it is not a `_helper.go`.
//
// UTC and RFC3339, and the reason is that the payload is read WEEKS LATER by somebody
// comparing two rows: a value that keeps the writer's offset (`+07:00`) would render
// differently depending on which administrator's browser produced it, and two audit
// lines for the same change would not compare equal. `""` rather than null for an
// open-ended campaign, so the key is present and a reader does not have to
// distinguish "absent" from "empty".
func endsAtForAudit(endsAt *time.Time) string {
	if endsAt == nil {
		return ""
	}
	return endsAt.UTC().Format(time.RFC3339)
}
