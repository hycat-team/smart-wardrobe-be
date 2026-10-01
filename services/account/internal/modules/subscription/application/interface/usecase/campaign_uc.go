package usecase

import (
	"context"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
)

// ICampaignUseCase is the application entry point of the signup-campaign
// mechanism (025 T041).
//
// One method, because there is exactly one trigger: the `identity.user.created`
// message (FR-002). The catch-up sweep of Phase 10 will re-enter through the same
// method rather than a second one, so the "is this account eligible" rules have
// exactly one implementation and the two paths cannot drift apart (FR-080).
//
// The use case returns a classified error rather than a boolean, because FR-062
// requires the caller to be able to tell a transient failure from an
// unrecoverable one, and that decision is made where the cause is known: a
// malformed payload can never succeed on redelivery, while a database timeout
// must be retried and the account must not be lost.
type ICampaignUseCase interface {
	// ProcessUserCreated evaluates one account-creation message and, when it is
	// eligible, grants the campaign's plan.
	//
	// It returns nil for every decision that is not a grant — including "skipped,
	// and here is why" — because a skip is a successful evaluation, not a failure.
	// A non-nil error means the decision could NOT be made and the message must be
	// retried or dead-lettered.
	ProcessUserCreated(ctx context.Context, payload identitydto.UserCreatedEventPayload) error
}
