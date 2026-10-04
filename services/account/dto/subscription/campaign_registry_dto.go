package dto

import (
	"bytes"
	"encoding/json"
)

// 026 T029/T042/T051 (FR-052, FR-054): the three WRITE requests of the campaign registry.
//
// WHY THESE ARE NOT IN `campaign_dto.go`. They were declared there by T029, T042 and
// T051, and they moved here at the end of the feature. The move is a FILE boundary and
// nothing else — same package, same JSON tags, same `binding` tags, same binder, same
// validator; the split changed no request shape and no status code. It was made because
// `campaign_dto.go` had reached 578 lines, past the 500-line production warning
// threshold of `code-hygiene.md` §H1, and the honest reading of that file is that it
// answers two unrelated questions:
//
//   - `campaign_dto.go` — what a campaign looks like WHEN READ (`CampaignSummaryRes`,
//     `CampaignClaimRes`, `CampaignAccountEligibilityRes`); the four read endpoints of
//     025, plus the three fields 026 added to them.
//   - `campaign_registry_dto.go` — what an administrator sends TO WRITE one, and only
//     ever to the three write endpoints of 026.
//
// A reader asking "may this field be blank?" must not have to read past the answer to
// "what does this campaign's remaining budget look like?" to find it, and the file
// header of `campaign_dto.go` — written when the feature was read-only — already had to
// be corrected twice to explain why write requests had appeared underneath it. Three
// files of one question each is smaller than one file of three questions, and each is
// comfortably under the threshold.

// CampaignCreateReq opens a campaign (026 T029, FR-052, contract
// `campaign-registry-api.md` §3).
//
// WHY `quota` AND `planSlug` ARE POINTERS AND `reserve` IS NOT. FR-054 separates
// "thiếu trường bắt buộc" (400) from "`quota = 0`" and "`planSlug` rỗng" (422), and
// those two cannot be told apart on a plain `int`/`string`: both arrive as the zero
// value. A pointer is what distinguishes "the field was absent" from "the field was
// sent with an empty value", which is the only way to answer 400 and 422 correctly
// without a second rule that guesses.
//
// `reserve` is a pointer too even though its absent value has a meaning of its own —
// 0 (data-model.md §2.1 `DEFAULT 0`) — because "absent" and "sent 0" producing the same
// row is fine for the row and NOT fine for the check: a negative value has to reach the
// validator as a value rather than being silently dropped as an absent field.
//
// WHY `startsAt`/`endsAt` ARE STRINGS. Two reasons, both from FR-013. A
// `time.Time` field would hand parsing to Gin's decoder, which accepts a format the
// operator's documentation does not show, so the accepted inputs would differ from the
// documented ones with nothing to say so (the same argument
// `campaignClaimsQuery.from`/`.to` makes). And the zone-less value FR-013 rejects
// cannot be represented by a Go `time.Time` at all — it would be read in the SERVER's
// zone, seven hours off in production, and stored as an instant nobody typed. Parsing
// the string here keeps "must carry a zone" a rule the code can state.
type CampaignCreateReq struct {
	// Code identifies the campaign and is immutable afterwards (FR-024).
	//
	// Restricted to `^[a-z0-9][a-z0-9-]{2,63}$`, matching `chk_campaigns_code_format`
	// (FR-011). A malformed code is a 400 — a broken shape, not a wrong meaning
	// (FR-054) — and the alphabet is restricted because the code reaches an audit
	// payload and possibly a queue or file name.
	Code string `json:"code" binding:"required"`
	// PlanSlug is the plan granted to an eligible account. Required (FR-014).
	//
	// Whether the plan EXISTS and is ACTIVE is checked at RUN time, not here and not
	// at startup (FR-003, FR-014): a wrong slug must not be able to crash a service,
	// and it makes that one campaign grant nothing rather than failing a write.
	PlanSlug *string `json:"planSlug" binding:"required"`
	// Quota is the main budget. `>= 1` (FR-012); zero is 422, not 400, because the
	// field was sent.
	Quota *int `json:"quota" binding:"required"`
	// Reserve is the compensation CEILING, not a target. Optional, `>= 0`, default 0.
	Reserve *int `json:"reserve"`
	// StartsAt is the instant the campaign opens, RFC3339 WITH an explicit zone.
	StartsAt string `json:"startsAt" binding:"required"`
	// EndsAt is the instant it closes, RFC3339 with a zone, or absent/null for an
	// open-ended campaign. When present it must be strictly AFTER StartsAt (FR-013).
	//
	// A pointer, and that is what makes `"endsAt": null` differ from `"endsAt": ""`:
	// the first is an open-ended campaign, the second is a malformed timestamp.
	EndsAt *string `json:"endsAt"`
}

// CampaignCloseReq force-closes one campaign (026 T042, FR-030, contract
// `campaign-registry-api.md` §5).
//
// TWO REQUIRED FIELDS, AND THEY ANSWER DIFFERENT QUESTIONS. `version` is the
// client's If-Match: "close the campaign I was looking at, not the one somebody
// edited while the confirmation box was open" (FR-055). `reason` is why a human
// ended the promotion early, and it is the whole reason the operation is audited
// — a closure with no reason leaves an operator six weeks later unable to tell a
// deliberate stop from a mistake (spec.md Câu 4).
//
// WHY `reason` IS A POINTER AND NOT A STRING. FR-054 puts "thiếu `reason`" on the
// 400 row and "`reason` rỗng" on the 422 row, and on a plain `string` those two
// arrive as the same zero value — which is the same reasoning that made
// `planSlug` a pointer on `CampaignCreateReq`, and it is the reason the two codes
// can be told apart at all.
type CampaignCloseReq struct {
	// Version is the version the client read. `0` is not a valid version (the column
	// defaults to 1 and only ever increments), so `binding:"required"` refusing the
	// zero value is exactly the "thiếu trường bắt buộc → 400" rule and nothing more.
	Version int `json:"version" binding:"required"`
	// Reason is the operator's explanation, stored in `payload.reason` of the audit
	// record (contract §6.3). Trimmed by the validator before use, so a box holding
	// three spaces is a 422 rather than an audit row that reads as blank.
	Reason *string `json:"reason" binding:"required"`
}

// CampaignUpdateReq edits a campaign that already exists (026 T051, FR-052,
// contract `campaign-registry-api.md` §4).
//
// EVERY FIELD BUT `version` IS OPTIONAL, AND AN ABSENT FIELD MEANS "UNCHANGED".
// That is what makes a `PATCH` a `PATCH`: an operator raising the budget sends
// only `quota`, and a campaign whose budget is frozen cannot be edited by sending
// the whole row back with one number changed — a form that always submits
// everything would have to re-derive the current values on the client, and two
// clients would disagree about what "unchanged" means.
//
// WHY THE THREE MONEY FIELDS ARE POINTERS. Same reasoning as `CampaignCreateReq`:
// FR-054 separates "thiếu trường bắt buộc" (400) from "sai nghĩa" (422), and on a
// plain `int` an absent `quota` arrives as `0` — which is also a real quota that
// must be refused with 422. Only a pointer can tell "leave it alone" from "set it
// to zero, which is nonsense".
//
// WHY `code` IS NOT A FIELD. FR-024 makes the code immutable, and a body field
// that must be rejected would be a second way to ask the same question the URL
// already answers. Instead the code is simply NOT HERE: a client that sends it
// has it ignored, nothing changes, and the request is a no-op — which is T054's
// assertion, and it holds because the field does not exist rather than because a
// check refuses it.
//
// WHY THE TIMESTAMPS ARE STRINGS. Identical to `CampaignCreateReq`: RFC3339 WITH a
// zone is a rule this layer can state and a `time.Time` cannot (FR-013).
//
// WHY `EndsAt` NEEDS A CUSTOM `UnmarshalJSON` (026 T061, contract §4.1). `endsAt`
// has THREE states, not two: the key ABSENT means "leave the stored end date alone",
// an explicit `null` means "CLEAR it", and a string means "set it". Every ordinary Go
// representation collapses two of them: `encoding/json` maps a JSON `null` to a nil
// pointer, so `*string` cannot tell absent from clear — and `**string` does not help,
// because for a null literal the decoder stops at the first settable pointer and zeroes
// the OUTER one, leaving `{}` and `{"endsAt":null}` both nil. Measured on this
// repository's Go toolchain, not assumed.
//
// So the KEY'S PRESENCE is recorded by this struct rather than by the decoder, and
// `EndsAtUpdate` below is how the use case asks. The field itself stays a plain
// `*string`, so the JSON shape, the Swagger schema and anything that round-trips this
// struct are untouched.
type CampaignUpdateReq struct {
	// Version is the version the client read, and it is REQUIRED (FR-055). `0` is
	// not a version any row can hold — the column defaults to 1 and only ever
	// increments — so `binding:"required"` refusing the zero value is exactly the
	// "thiếu trường bắt buộc → 400" row of FR-054 and nothing more.
	Version int `json:"version" binding:"required"`
	// PlanSlug is the plan this campaign grants. Optional. Changeable only while
	// `budgetLocked` is false (FR-020); a blank value is 422, not 400.
	PlanSlug *string `json:"planSlug"`
	// Quota is the main budget, `>= 1` (FR-012). Optional. Changeable only while
	// `budgetLocked` is false.
	Quota *int `json:"quota"`
	// Reserve is the compensation ceiling, `>= 0`. Optional, and it defaults to
	// NOTHING rather than to 0: an omitted `reserve` leaves the stored ceiling
	// alone, which is the opposite of the create rule and is right here (an
	// omitted field means unchanged).
	Reserve *int `json:"reserve"`
	// StartsAt is when the campaign opens, RFC3339 with a zone. Optional, and
	// editable whether or not the campaign has granted (FR-023).
	StartsAt *string `json:"startsAt"`
	// EndsAt is when it closes, RFC3339 with a zone. Optional and editable
	// regardless of grants (FR-023).
	//
	// `null` CLEARS the end date and makes the campaign open-ended again; the key
	// being absent leaves it alone. The two are different requests, which is why
	// this type decodes itself — see the header and `EndsAtUpdate` below.
	EndsAt *string `json:"endsAt"`

	// endsAtSent and endsAtCleared record what the KEY carried, which the JSON
	// decoder cannot express on a `*string`.
	//
	// UNEXPORTED and read through `EndsAtUpdate`, because the consumer of this DTO is
	// another package: an exported flag would become part of the request contract and
	// a client could not set it anyway, while a method keeps the question and its
	// answer in one place.
	endsAtSent    bool
	endsAtCleared bool
}

// UnmarshalJSON decodes the request and records whether `endsAt` was sent at all.
//
// THE ALIAS IS THE WHOLE POINT. `type plain CampaignUpdateReq` drops this method, so
// the first decode is exactly the one the struct would have received on its own — and
// adding a field to the struct needs no change here. The SECOND decode exists only to
// answer a question the first one cannot: `encoding/json` reports a JSON `null` and an
// absent key identically for a pointer field, so the raw key set is what tells them
// apart.
//
// ERRORS ARE RETURNED UNWRAPPED, on purpose: `TranslateValidationError` recognises
// `*json.UnmarshalTypeError` and answers with the service's standard 400 naming the
// field, while a wrapped error would degrade into a generic "cannot unmarshal" — a
// worse answer for the same wrong request.
func (r *CampaignUpdateReq) UnmarshalJSON(data []byte) error {
	type plain CampaignUpdateReq
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = CampaignUpdateReq(decoded)

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	if value, present := keys["endsAt"]; present {
		r.endsAtSent = true
		r.endsAtCleared = bytes.Equal(bytes.TrimSpace(value), []byte("null"))
	}
	return nil
}

// EndsAtUpdate reports what `endsAt` asked for, in the three states the contract
// defines (026 T061, contract §4.1): key ABSENT keeps the stored `ends_at`, an
// explicit `null` CLEARS it, and a string sets it.
//
// A caller asks two questions and never has to guess a third — and it must test
// `cleared` FIRST, because a cleared request arrives with a nil `value` and must not
// be read as an absent one.
func (r CampaignUpdateReq) EndsAtUpdate() (cleared bool, value *string) {
	return r.endsAtCleared, r.EndsAt
}
