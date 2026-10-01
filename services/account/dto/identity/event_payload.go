package dto

import (
	"time"

	"github.com/google/uuid"
)

// UserCreatedEventPayload is the body of topic `identity.user.created`
// (025 T025, contracts/account-created-event.md).
//
// Exactly four fields, and NOTHING ELSE — FR-087 forbids personally identifiable
// data in the message. No email, no first or last name, no date of birth, no
// address, no password hash. A receiver that needs more reads the account table:
// same service, so it costs no network call, and the alternative would put
// customer data into a broker that persists and fans messages out.
//
// The set of seven topics stays untouched (FR-092): this is an identity event,
// not a new plan-change event type.
type UserCreatedEventPayload struct {
	// UserID is generated immediately before the account row is written.
	UserID uuid.UUID `json:"userId"`
	// Provider is the sign-up source: `local` or `google`.
	Provider string `json:"provider"`
	// RegisteredAt is users.created_at, snapshotted so FR-031 can compare an
	// account against the frozen priority watermark without a second read.
	RegisteredAt time.Time `json:"registeredAt"`
	// CreatedByAdmin says whether the executing flow was the administrator one.
	//
	// Obtainable ONLY from the call path: `identity.users` recorded nothing about
	// who created the row before 025 T015, so two flows built structurally
	// identical rows. It is NOT derivable from email verification either — an
	// administrator may create an already-verified account, and every self-serve
	// account is verified (research.md R4).
	CreatedByAdmin bool `json:"createdByAdmin"`
}
