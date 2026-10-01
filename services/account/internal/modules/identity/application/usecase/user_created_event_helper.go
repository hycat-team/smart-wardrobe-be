package usecase

import (
	"context"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_event "smart-wardrobe-be/shared/application/event"
)

// publishUserCreated writes the `identity.user.created` outbox record for a newly
// inserted account (025 T030-T032).
//
// It is ONE helper for all three creation doors rather than three inline calls, so
// the payload shape cannot drift between them — and the payload is the contract
// (contracts/account-created-event.md §2).
//
// `createdByAdmin` is passed in by the CALLER rather than read from the entity,
// and that is the whole point of the parameter: the flag is only knowable from the
// flow that is executing. The entity field exists for the catch-up sweep, which
// reads the database and has no event to read (research.md R4).
//
// `user.CreatedAt` is read AFTER the insert, because GORM populates it from the
// column default (`now()`) during Create. That is the value FR-031 compares
// against the frozen priority watermark, so it has to be the stored instant and
// not a second `time.Now()` taken a moment later.
//
// The caller MUST be inside the transaction that inserted the account. The outbox
// publisher resolves the transaction from the context, so the record commits with
// the account or not at all (FR-042, FR-088).
func publishUserCreated(
	ctx context.Context,
	publisher shared_event.IEventPublisher,
	user *entities.User,
	createdByAdmin bool,
) error {
	return publisher.Publish(ctx, eventconstants.TopicUserCreated, identitydto.UserCreatedEventPayload{
		UserID:         user.ID,
		Provider:       string(user.Provider),
		RegisteredAt:   user.CreatedAt,
		CreatedByAdmin: createdByAdmin,
	})
}
