package campaign

import (
	"regexp"
	"strings"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// 026 T030 — the request-shape and request-meaning rules of the campaign registry's
// three write endpoints.
//
// WHY THIS IS A SEPARATE FILE AND NOT A BLOCK INSIDE `registry_uc.go`. FR-054 draws
// one line that cuts through all three operations: `400` is a request that cannot be
// READ, `422` is a request that reads fine and means nothing useful. Every rule here is
// on one side of that line or the other, so putting them in one place makes the line
// visible — and each operation (create, update, close) adds its rules HERE rather than
// re-deciding where the line is.
//
// WHY THE VALIDATION RUNS BEFORE ANY I/O. All of it is pure: a rejected campaign never
// opens a transaction, never takes a lock and never writes an outbox row. The operator
// gets their 400 or 422 in the same millisecond whether or not the database is
// reachable, and a database that is down cannot change which error a typo produces.
//
// `validation.Bind*` HAS ALREADY DONE PART OF THIS, and this file does not repeat it.
// A missing `code`, `planSlug`, `quota` or `startsAt` is `binding:"required"` on the
// DTO, so the shared binder answers 400 before the use case is reached at all. What is
// left here is exactly the part a struct tag cannot express: a pattern, a comparison
// between two fields, and a timestamp whose zone is mandatory.

// campaignCodePattern is `chk_campaigns_code_format` restated in Go (FR-011).
//
// The SAME pattern as the column's CHECK, and kept as a literal rather than read from
// the schema, for the reason the schema is there in the first place: the constraint is
// the backstop for a hand-written INSERT, and this is the layer that answers the
// operator in Vietnamese with the field named. Two independent expressions of one rule
// is what makes the backstop worth having — if they are generated from each other, the
// backstop proves nothing.
var campaignCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,63}$`)

// Field names as the API spells them.
//
// The names in a 422 message are camelCase JSON names, not Go names: the reader is
// looking at a form whose field is labelled `planSlug`, and `PlanSlug` would send them
// to look for a field that does not exist. Constitution §"Code Style" requires
// client-facing text in Vietnamese, and the field name is part of that text.
const (
	fieldCode     = "code"
	fieldPlanSlug = "planSlug"
	fieldQuota    = "quota"
	fieldReserve  = "reserve"
	fieldStartsAt = "startsAt"
	fieldEndsAt   = "endsAt"
)

// campaignWrite is the validated, ready-to-persist form of a write request.
//
// It exists so that nothing downstream of validation re-reads the raw request. Every
// value here is already an instant or an integer, and the optional ones have already
// had their "absent" case folded into a single representation — which is what lets the
// use case build an entity without asking a second question about any field.
type campaignWrite struct {
	code     string
	planSlug string
	quota    int
	reserve  int
	startsAt time.Time
	endsAt   *time.Time
}

// validateCampaignCreate applies FR-054's table to the create request.
//
// THE ORDER IS THE ORDER OF THE REQUEST, not the order of the table. An operator
// fixing a form reads fields top to bottom and stops at the first thing that is
// wrong; reporting the last violation instead means the second round of edits
// uncovers one more error, and the third another.
//
// REQUIRED FIELDS ARE NOT RE-TESTED HERE. `binding:"required"` already refused a
// missing `code`, `planSlug`, `quota` or `startsAt`, and re-testing them would produce
// a second message for a problem the client was already told about.
//
// THE IDENTITY IS CHECKED HERE AND THE TWO GROUPS BELOW ARE DELEGATED, rather than one
// function walking the whole request. The split is by KIND, which is the axis FR-054
// draws: identity (`code`, `planSlug`) is about whether this names a campaign at all,
// budget is two integers against lower bounds, and the window is two timestamps against
// each other. Each group fails on its own terms and needs no knowledge of the others.
func validateCampaignCreate(req *sub_dto.CampaignCreateReq) (campaignWrite, error) {
	var out campaignWrite

	// `code`: 400 for a malformed code, 400 for a blank one. Both are broken shapes —
	// FR-054 lists "thiếu trường bắt buộc" and "`code` sai định dạng" on the SAME row,
	// and there is no semantic reading of an absent or misspelt code to preserve.
	if !campaignCodePattern.MatchString(req.Code) {
		return out, subscriptionerrors.ErrCampaignCodeInvalid()
	}
	out.code = req.Code

	// `planSlug`: 400 when the field is ABSENT (the binder's `required`), 422 when it
	// is present and blank. FR-054 puts "`planSlug` rỗng" on the 422 row, and the
	// pointer on the DTO is the only way to tell "absent" from "sent empty".
	//
	// Trimmed before use, because `chk_campaigns_plan_slug_not_blank` is
	// `btrim(plan_slug) <> ''`: a slug of three spaces passes a naive emptiness test
	// and is then refused by the constraint as a raw `check_violation` the operator
	// cannot read. Trimming here also makes the stored value the one the operator
	// meant rather than the one their editor padded.
	//
	// A slug that names no plan is NOT refused. FR-014 keeps 025's decision: existence
	// and activity are checked at RUN time, and each grant against a missing plan is
	// skipped with `plan_unavailable`. Refusing the write here would reintroduce the
	// startup-time database reach that FR-003 forbids.
	out.planSlug = strings.TrimSpace(*req.PlanSlug)
	if out.planSlug == "" {
		return out, subscriptionerrors.ErrCampaignInvalidField(fieldPlanSlug, "không được để trống")
	}

	if err := validateCampaignBudget(req, &out); err != nil {
		return out, err
	}
	if err := validateCampaignWindow(req, &out); err != nil {
		return out, err
	}
	return out, nil
}

// fieldReason names the closure reason as the API spells it, for the same reason
// as the other field names above.
const fieldReason = "reason"

// validateCampaignClose applies FR-054's table to the close request.
//
// THE WHOLE OF IT IS THE REASON, because the version is not validated here and
// that is deliberate. `version` is a NUMBER THE CLIENT READ BACK, not an input with
// a shape to check: any integer is well-formed, and whether it is the CURRENT one
// is a concurrency question answered by the conditional UPDATE, not by a validator.
// A validator that compared it against anything here would either duplicate the
// UPDATE's decision or need a read the transaction is going to make anyway.
//
// `reason` IS checked, and it is 422 rather than 400 because the binder has already
// proved the field was SENT (`binding:"required"`): FR-054 puts "thiếu `reason`" on
// the 400 row and "`reason` rỗng" on the 422 row, and the pointer on the DTO is the
// only thing that can tell the two apart.
//
// TRIMMED BEFORE IT IS STORED, for the same reason `planSlug` is trimmed: three
// spaces is a blank answer that would reach the audit trail looking like a reason
// somebody gave, and the row an operator reads weeks later would say nothing while
// appearing to say something.
func validateCampaignClose(req *sub_dto.CampaignCloseReq) (string, error) {
	reason := strings.TrimSpace(*req.Reason)
	if reason == "" {
		return "", subscriptionerrors.ErrCampaignInvalidField(fieldReason, "không được để trống")
	}
	return reason, nil
}

// validateCampaignBudget checks `quota` and `reserve` (FR-012).
//
// BOTH ARE 422, and neither is 400, and that is the whole argument: the binder has
// already proved both fields were SENT (`quota` is required; `reserve` is optional and a
// sent value arrives as a pointer). FR-054's own example for 422 is `quota = 0` — a zero
// budget is a campaign that grants nobody, which is meaningless rather than malformed,
// and an operator who sent `quota: 0` has a typo to fix in the NUMBER, not in the
// request's shape.
//
// It fills the two fields in place rather than returning them, because the caller
// accumulates one struct and a second return value would mean copying the pair out by
// hand at the call site — which is where a swapped `quota`/`reserve` would come from.
func validateCampaignBudget(req *sub_dto.CampaignCreateReq, out *campaignWrite) error {
	out.quota = *req.Quota
	if out.quota < 1 {
		return subscriptionerrors.ErrCampaignInvalidField(fieldQuota, "phải lớn hơn hoặc bằng 1")
	}

	// `reserve`: optional, so an absent value becomes 0 (data-model.md §2.1) and only a
	// negative one is wrong. 422 for the same reason as `quota`.
	out.reserve = 0
	if req.Reserve != nil {
		out.reserve = *req.Reserve
	}
	if out.reserve < 0 {
		return subscriptionerrors.ErrCampaignInvalidField(fieldReserve, "không được âm")
	}
	return nil
}

// validateCampaignWindow checks `startsAt` and `endsAt` (FR-013).
//
// IT IS THE ONLY PLACE THE TWO TIMESTAMPS ARE COMPARED, and only because `endsAt`'s rule
// is relative: "`endsAt` không sau `startsAt`" has no meaning until both are instants.
// A start date in the past is NOT checked at all, and that is deliberate — it is the
// normal way to open a campaign retroactively, and quickstart V1 opens one that way on
// purpose (spec.md US1).
//
// `"endsAt": null` is the open-ended campaign. A pointer present but EMPTY is a
// MALFORMED timestamp rather than an absent one, and that distinction is why the DTO
// field is a pointer: the two arrive as different Go values and must produce different
// errors.
func validateCampaignWindow(req *sub_dto.CampaignCreateReq, out *campaignWrite) error {
	startsAt, err := parseCampaignInstant(req.StartsAt, fieldStartsAt)
	if err != nil {
		return err
	}
	out.startsAt = startsAt

	if req.EndsAt == nil {
		return nil
	}
	endsAt, err := parseCampaignInstant(*req.EndsAt, fieldEndsAt)
	if err != nil {
		return err
	}
	// FR-013 / `chk_campaigns_window`: strictly after. EQUAL is refused as well as
	// before, because a campaign whose window closes the instant it opens grants
	// nobody — the same meaningless state a zero quota describes, and the reason the
	// CHECK says `>` and not `>=`.
	if err := requireWindowAfterStart(out.startsAt, &endsAt); err != nil {
		return err
	}
	out.endsAt = &endsAt
	return nil
}

// requireWindowAfterStart is FR-013's comparison, in ONE place.
//
// IT IS SHARED BY BOTH ENTRY POINTS rather than written twice, and that is the point:
// create checks the window the REQUEST describes, update checks the window the
// request MERGES onto the stored row, and the two must produce the same 422 with the
// same field name and the same Vietnamese text. A second copy of a five-line rule is
// how "the field name in the message" starts disagreeing between two endpoints.
func requireWindowAfterStart(startsAt time.Time, endsAt *time.Time) error {
	if endsAt == nil || endsAt.After(startsAt) {
		return nil
	}
	return subscriptionerrors.ErrCampaignInvalidField(fieldEndsAt,
		"phải sau thời điểm bắt đầu")
}

// validateMergedCampaignWindow re-decides FR-013 on the row an edit would leave
// behind (026 T061, contract §4.1).
//
// IT MUST RUN ON THE MERGED ROW, and that is the whole reason it exists separately
// from `validateCampaignUpdate`. A field the request did not send contributes its
// STORED value, so the request alone cannot answer the question: moving only
// `startsAt` past a stored `endsAt` describes a window that never existed and must be
// refused with the same 422 create would give. Deciding it in the validator would mean
// reading the row before the transaction, and reading it twice is how the two reads
// disagree.
//
// `nil` `EndsAt` is an open-ended campaign, which the CHECK also accepts — so clearing
// the end date is never refused for its own sake, only for landing before the start.
func validateMergedCampaignWindow(merged *entities.Campaign) error {
	return requireWindowAfterStart(merged.StartsAt, merged.EndsAt)
}

// parseCampaignInstant reads one RFC3339 timestamp that MUST carry a zone (FR-013).
//
// WHY THE ZONE IS REQUIRED RATHER THAN ASSUMED. `time.Parse(time.RFC3339, ...)` needs
// it, so a zone-less string fails here — which is the point. Assuming the configured
// zone would accept `2026-10-02 00:00:00` and store an instant seven hours away from
// the one the operator typed in production, and nothing downstream could tell: the
// column is `TIMESTAMPTZ` and it would hold a perfectly valid wrong value. A campaign
// opening at midnight, shifted seven hours, is the whole launch-day feature failing
// silently (FR-015 puts every comparison on the configured zone; this is where that
// zone is EARNED rather than assumed).
//
// The field name travels in the message because FR-054's rule is per-field: an
// operator with two timestamps in the form needs to know which one is wrong.
func parseCampaignInstant(raw, field string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, subscriptionerrors.ErrCampaignTimestampMalformed(field)
	}
	return parsed, nil
}

// campaignUpdate is the validated form of an edit: only the fields the client SENT,
// already turned into the types the row uses.
//
// EVERY FIELD IS A POINTER, and that is the requirement rather than a convenience.
// "Absent" and "present with that value" are different requests, and a plain field
// would collapse them — an absent `quota` arriving as `0` would be written over a
// campaign's real budget. The use case merges this onto the stored row, and the
// pointers are what let it tell which columns the operator actually named.
//
// The consequence is that this function CANNOT produce the row to write, and that is
// correct: `ends_at > starts_at` is a comparison between two instants where either
// may come from the existing row, so it needs the row. What is checked here is
// everything decidable from the request alone, which is exactly the half that must
// happen BEFORE a transaction is opened. The other half — the comparison on the MERGED
// window — is `validateMergedCampaignWindow` (026 T061).
type campaignUpdate struct {
	planSlug *string
	quota    *int
	reserve  *int
	startsAt *time.Time
	// endsAt is the ONE field with THREE states (026 T061, contract §4.1), and the
	// pointer-to-pointer is what carries all of them:
	//
	//	nil            the key was ABSENT — keep the stored `ends_at`
	//	&nil           the key was an explicit `null` — CLEAR the end date
	//	&instant       the key carried a timestamp — set the window end to it
	//
	// The middle state is what makes the type worth the extra indirection: it is the
	// request that makes a finished promotion open-ended again, and on a `*time.Time`
	// it would be indistinguishable from the first state.
	endsAt **time.Time
}

// validateCampaignUpdate applies FR-054's table to an edit, for the fields the client
// sent. Absent fields are not inspected at all: there is nothing to inspect.
//
// SAME ORDER AS THE CREATE VALIDATOR, and for the same reason — an operator fixing a
// form reads top to bottom and stops at the first thing that is wrong, so reporting
// the last violation costs them a round of edits per error.
//
// WHAT IS *NOT* DECIDED HERE, and why it cannot be:
//
//   - whether the campaign exists (404). That is a read, and a read inside a
//     transaction is FR-055's ordering requirement, not a validation rule;
//   - whether the budget has already been granted (422, FR-020). Also a read, and it
//     must be the read of the SAME transaction that writes, or a grant committed in
//     between would slip past;
//   - whether anything changes at all (FR-055's no-op). Needs the stored values;
//   - whether the MERGED window is legal (422, FR-013). Both instants of that
//     comparison may come from the stored row, so the request cannot answer it — see
//     `validateMergedCampaignWindow`.
//
// Those four live in `Update`, and each of them is a decision about the WORLD rather
// than about the shape of a request — which is the line this file has followed since
// T030.
func validateCampaignUpdate(req *sub_dto.CampaignUpdateReq) (campaignUpdate, error) {
	var out campaignUpdate

	if req.PlanSlug != nil {
		// Trimmed before comparison, for the same reason create trims: the column's
		// CHECK is `btrim(plan_slug) <> ''`, so three spaces would pass an emptiness
		// test and then be refused by the schema as a raw `check_violation` no
		// operator can read.
		planSlug := strings.TrimSpace(*req.PlanSlug)
		if planSlug == "" {
			return out, subscriptionerrors.ErrCampaignInvalidField(fieldPlanSlug, "không được để trống")
		}
		out.planSlug = &planSlug
	}

	// 422, NOT 400, AND THE REASON IS THE SAME ONE CREATE GIVES: the binder has proved
	// the field was SENT (it is a pointer), and a quota of zero is a campaign that
	// grants nobody — a wrong number, not an unreadable request (FR-054).
	if req.Quota != nil {
		if *req.Quota < 1 {
			return out, subscriptionerrors.ErrCampaignInvalidField(fieldQuota, "phải lớn hơn hoặc bằng 1")
		}
		out.quota = req.Quota
	}
	if req.Reserve != nil {
		if *req.Reserve < 0 {
			return out, subscriptionerrors.ErrCampaignInvalidField(fieldReserve, "không được âm")
		}
		out.reserve = req.Reserve
	}

	// FR-013 again: RFC3339 WITH a zone, per field, and the field name in the message
	// so an operator with two timestamps knows which one is wrong. An empty string is
	// a MALFORMED timestamp, not an absent one — exactly as on create.
	if req.StartsAt != nil {
		startsAt, err := parseCampaignInstant(*req.StartsAt, fieldStartsAt)
		if err != nil {
			return out, err
		}
		out.startsAt = &startsAt
	}
	// `endsAt` HAS THREE STATES AND IS ASKED ABOUT IN THAT ORDER (026 T061). The DTO
	// recorded whether the key was sent, because `encoding/json` cannot; `cleared` is
	// tested FIRST because a cleared request carries no string and would otherwise be
	// read as an absent one — which is precisely the request that would silently keep
	// a campaign open-ended forever.
	if cleared, raw := req.EndsAtUpdate(); cleared || raw != nil {
		var requested *time.Time
		if raw != nil {
			endsAt, err := parseCampaignInstant(*raw, fieldEndsAt)
			if err != nil {
				return out, err
			}
			requested = &endsAt
		}
		out.endsAt = &requested
	}

	// `version` IS NOT CHECKED HERE, for the same reason `validateCampaignClose` does
	// not check it: any integer is well-formed, and whether it is the CURRENT one is
	// the conditional UPDATE's question. Checking it against the row read above would
	// duplicate the UPDATE's decision and make the 412 unreachable.
	return out, nil
}
