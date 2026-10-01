package contract

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"

	"github.com/google/uuid"
)

type UserBriefContract struct {
	userBriefUC uc_interfaces.IUserBriefUseCase
}

// NewUserBriefContract returns the CONCRETE type, not the interface.
//
// 024 T136: it used to return `IUserBriefContract`, which hid the one method the
// admin grant flow needs that the interface deliberately omits. Returning the
// concrete type lets `NewAdminGrantLookup` below expose that narrow view, while
// every existing consumer keeps depending on the interface - which
// `*UserBriefContract` still satisfies.
//
// It also moves interface satisfaction into a single visible place: adding a
// method to `IUserBriefContract` now fails HERE, in the type that implements it,
// instead of surfacing as a nil interface at some consumer.
func NewUserBriefContract(userBriefUC uc_interfaces.IUserBriefUseCase) *UserBriefContract {
	return &UserBriefContract{userBriefUC: userBriefUC}
}

// AdminGrantLookup is the narrow view the administrator grant flow needs from the
// identity module (024 T136).
//
// It is separate from `IUserBriefContract` on purpose. That contract is served
// over gRPC, and a soft-deleted account has no business travelling between
// services - commerce callers must keep excluding them. The grant flow runs inside
// this service, so it can be told the difference without widening the wire format.
//
// Declared here, in the contract package, because both sides already depend on it:
// the contract can build it without importing the subscription module, and the
// subscription module can consume it without knowing how identity is wired.
type AdminGrantLookup interface {
	// GetByUsernamesOrEmailsIncludingDeleted returns soft-deleted accounts with
	// their flag, so the grant can report `user_soft_deleted` rather than a
	// misleading `user_not_found` (FR-049).
	GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error)
	// GetByIDs resolves UUID identifiers.
	//
	// A UUID cannot be probed for deletion without a second round trip per item,
	// and an administrator typing UUIDs already has the id in hand, so this path
	// keeps the existing excluding behaviour. The reason code therefore applies to
	// the username/email path - the one a person types from memory and gets wrong.
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefContractDTO, error)
}

// NewAdminGrantLookup adapts the concrete contract to the narrow grant view.
func NewAdminGrantLookup(c *UserBriefContract) AdminGrantLookup {
	return c
}

func (c *UserBriefContract) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefContractDTO, error) {
	return c.userBriefUC.GetByID(ctx, userID)
}

func (c *UserBriefContract) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefContractDTO, error) {
	return c.userBriefUC.GetByIDs(ctx, userIDs)
}

func (c *UserBriefContract) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error) {
	return c.userBriefUC.GetByUsernamesOrEmails(ctx, identifiers)
}

// GetByUsernamesOrEmailsIncludingDeleted delegates to the use case (024 T136).
//
// Served in-process for the admin grant flow. The gRPC surface deliberately does
// NOT expose it: cross-service callers must keep excluding deleted accounts, and
// the only consumer is a flow inside this service.
func (c *UserBriefContract) GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error) {
	return c.userBriefUC.GetByUsernamesOrEmailsIncludingDeleted(ctx, identifiers)
}

func (c *UserBriefContract) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*dto.UserBriefContractDTO, error) {
	return c.userBriefUC.SearchUserBriefs(ctx, keyword, limit)
}
