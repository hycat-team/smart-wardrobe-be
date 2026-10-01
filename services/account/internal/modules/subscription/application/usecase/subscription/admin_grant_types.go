package subscription

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 024 T147: the value types of the batch grant path, in one file.

// code-hygiene H2 allows at most two behavioural types per production file, and
// this path had four. Splitting them out is not cosmetic: the structs are read
// together but change for different reasons, and one of them changing used to
// move the whole file in review.

type grantRequest struct {
	actorAdminID uuid.UUID
	// requestIP is the administrator's network address, recorded on the audit row
	// (FR-082). It travels as an argument rather than being read from the context
	// because the application layer must not depend on a transport concept.
	requestIP *string
	// requestIP is the administrator's address, recorded on the audit row (FR-082).
	// It travels as an argument rather than being read from the context because the
	// application layer must not depend on a transport concept.
	userID        uuid.UUID
	plan          *entities.SubscriptionPlan
	identifierKey string
	itemIndex     int
	// plannedVersion is what the batch read BEFORE any write. The transaction
	// re-reads it and compares (FR-069).
	plannedVersion int64
	// plannedFound distinguishes "no subscription yet" (version 0) from
	// "subscription exists at version 0", which are different states: granting to
	// the first is a creation, granting over the second is an update.
	plannedFound bool
}

// grantedItem is what one committed grant produced.
type grantedItem struct {
	expiresAt *time.Time
	// replayed is true when the same Idempotency-Key and index had already been
	// granted, so nothing was extended this time (FR-068).
	replayed bool
}

// planIssue is a per-item plan resolution failure.
type planIssue struct {
	reason  string
	message string
}

// adminGrantEventKey builds the de-duplication key for one item (024 T097).
//
// It embeds the ITEM INDEX rather than the account, because the same
// Idempotency-Key is meant to cover a whole request: a retried request must
// return the same per-item outcomes, and only the index distinguishes one line
// from another.

// loadPlannedVersions reads the current version of every account the batch will
// touch, in ONE query (024 T098).
//
// Accounts the list does not resolve to are skipped: they fail earlier, on
// identifier or plan grounds, and reading rows for them would be wasted I/O.
