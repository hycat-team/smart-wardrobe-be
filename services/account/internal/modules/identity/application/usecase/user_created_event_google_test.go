package usecase_test

import (
	"context"
	"testing"

	// The dto/identity directory declares `package dto`, so alias it rather than
	// shadowing the import path with a name that is not a real identifier here.
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoogleFirstLoginPublishesUserCreated covers the Google door of 025 T031.
//
// The local door was proven end to end; this one was not, and "the code reads
// correctly" is not evidence - the stale_event defect in the grant path passed
// every unit test and only appeared against a real AI service. A signup path
// that quietly stopped emitting the event would be invisible: the account would
// be created, the free plan provisioned, and the customer would simply never
// receive the promotion, with nothing in any log saying why.
//
// A Google signup is a real signup for the campaign's purposes, and half the
// registrations may be Google ones. It therefore has to emit the same event with
// the same shape as the local door.
func TestGoogleFirstLoginPublishesUserCreated(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	publisher := newRecordingPublisher()

	uc := newEagerGoogleUCWithPublisher(repo, spy, uow, identityClaim(), publisher)

	_, err := uc.LoginWithIDToken(context.Background(), identitydto.GoogleLoginReq{IDToken: "token"})
	require.NoError(t, err)

	record, err := singleRecord(t, publisher)
	require.NoError(t, err)

	assert.Equal(t, string(authprovider.Google), record.payload.Provider,
		"the receiver reads the payload, not the column; a wrong provider here sends the sweep down the wrong branch")
	assert.False(t, record.payload.CreatedByAdmin,
		"a Google signup is an ordinary customer; the flag is only true on the administrator door")
	assert.NotEqual(t, uuid.Nil, record.payload.UserID,
		"the message must name the account that was just created")
	assert.False(t, record.payload.RegisteredAt.IsZero(),
		"FR-031 compares this instant against the frozen watermark, so it cannot be empty")

	// The event has to commit with the account. A published message for an account
	// that then rolls back is a grant the catch-up sweep would make for somebody
	// who does not exist (FR-042).
	assert.Equal(t, 1, uow.commits, "the account and its event must commit once, in one transaction")
	assert.True(t, record.inTx, "the publish happened outside the creation transaction")
}

// TestGoogleAutoLinkDoesNotPublishUserCreated is the other half of the same door.
//
// When the email already belongs to an account, Google login LINKS an identity to
// it. No account is created, so there is nothing to announce. Publishing anyway
// would re-enter the campaign on every subsequent Google login, and the only
// thing standing between that and a duplicate grant is the UNIQUE constraint on
// campaign_claims - which is a last-resort guard, not a design.
func TestGoogleAutoLinkDoesNotPublishUserCreated(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	publisher := newRecordingPublisher()

	// An existing, email-verified LOCAL account, which is the case that takes the
	// auto-link branch: the email is taken, so the Google identity is bound to it.
	existing := &entities.User{
		Username:      "duck",
		Email:         "duck@gmail.com",
		Provider:      authprovider.Local,
		EmailVerified: true,
		RoleSlug:      roleslug.User,
		Status:        userstatus.Active,
	}
	existing.ID = uuid.New()
	repo.byEmail[existing.Email] = existing

	claim := identityClaim()
	claim.Email = existing.Email
	uc := newEagerGoogleUCWithPublisher(repo, spy, uow, claim, publisher)

	_, err := uc.LoginWithIDToken(context.Background(), identitydto.GoogleLoginReq{IDToken: "token"})
	require.NoError(t, err)

	records := publisher.all()
	assert.Empty(t, records,
		"linking a Google identity creates no account, so it must announce none; got %d publish(es)", len(records))
}
