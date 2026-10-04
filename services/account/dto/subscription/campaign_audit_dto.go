package dto

import (
	"time"

	"github.com/google/uuid"
)

// 026 T069 (FR-063, contract `campaign-registry-api.md` §6.2): the audit trail of ONE
// campaign, as `GET /admin/campaigns/{code}/audit` returns it.
//
// WHY THIS RESPONSE TYPE LIVES IN A FILE OF ITS OWN. It was declared in
// `campaign_dto.go` by T069 and moved here at the end of the feature, and the move is
// recorded here rather than left silent because it is a file boundary and nothing else:
// same package, same JSON, same mapper, no behavioural change of any kind. The reason is
// size — `campaign_dto.go` had grown to 578 lines and crossed the production warning
// threshold of `code-hygiene.md` §H1 (500) because this feature added the write requests
// and the audit response to a file that already held three read responses. A pure
// data-shape file is not covered by H2's two-behaviour rule, so it is H1 alone, and the
// split is the cheapest way to satisfy it: one file per question the API answers, in the
// same shape the handler package already uses (`CampaignAdminHandler` reads,
// `CampaignRegistryHandler` writes).
//
// THE TYPE IS ITS OWN AND NOT `dto/admin.AdminAuditLogDTO`, and that is the whole point
// of FR-063's response shape.
type CampaignAuditLogRes struct {
	ID          uuid.UUID `json:"id"`
	ActorUserID uuid.UUID `json:"actorUserId"`
	// Action is `campaign.create`, `campaign.update` or `campaign.close`. They are
	// actions ON the campaign rather than on the subscription because a campaign
	// operation moves nobody's entitlement (FR-033), and an operator answering "who
	// gave away Premium" filters on `subscription.grant`.
	Action string `json:"action"`
	// TargetType is `campaign` for every row on this endpoint.
	TargetType string `json:"targetType"`
	// TargetID is ALWAYS null here, and it is emitted rather than omitted — see below.
	//
	// IT CARRIES NO `omitempty` BECAUSE THE SHARED DTO CANNOT CARRY FR-063.
	// `dto/admin.AdminAuditLogDTO` declares `json:"targetId,omitempty"`, so a nil pointer
	// is OMITTED rather than emitted as `null` — and EVERY campaign row has a null
	// `target_id`, because the audit trail's existing filter takes a UUID while a
	// campaign is keyed by a text code (FR-063, research.md R8). Reusing it would make
	// the invariant invisible on the one endpoint that exists to show it, and dropping
	// `omitempty` from the shared DTO would break the audit endpoint already in
	// production. So `TargetID` HERE CARRIES NO `omitempty`, and that single difference
	// is what makes FR-063 observable — and assertable.
	TargetID *uuid.UUID `json:"targetId"`
	// Payload carries `campaignCode` plus whatever the operation changed: the created
	// values, `before`/`after` for an edit, the closure `reason`. It is a document
	// because the three operations genuinely carry different keys, and a struct of
	// three optional sub-objects would add nesting to every reader for no gain.
	Payload map[string]any `json:"payload"`
	// RequestIP is omitted when the caller sent none — the one field here where
	// absence means "not provided" rather than "null".
	RequestIP *string `json:"requestIp,omitempty"`
	// CreatedAt is AFTER the operator's operation succeeded, because the trail is
	// written asynchronously from the outbox (FR-060).
	CreatedAt time.Time `json:"createdAt"`
}
