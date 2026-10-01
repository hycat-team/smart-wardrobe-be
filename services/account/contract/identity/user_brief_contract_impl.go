package contract

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"

	"github.com/google/uuid"
)

type UserBriefContract struct {
	userBriefUC uc_interfaces.IUserBriefUseCase
	// campaignAccountsUC is the read path the campaign sweep needs. It is a
	// separate dependency rather than a method on IUserBriefUseCase: that
	// interface is the shape of the gRPC contract, and widening it would change
	// the wire format (contracts/identity-contract.md §2).
	campaignAccountsUC uc_interfaces.ICampaignAccountReader
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
func NewUserBriefContract(userBriefUC uc_interfaces.IUserBriefUseCase, campaignAccountsUC uc_interfaces.ICampaignAccountReader) *UserBriefContract {
	return &UserBriefContract{userBriefUC: userBriefUC, campaignAccountsUC: campaignAccountsUC}
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

// CampaignAccountPage is one batch of accounts created inside a window
// (025 T023).
type CampaignAccountPage struct {
	// Items are DTOs, never `entities.User`: constitution §II — "using DTOs,
	// never domain entities". A domain entity pointer must not cross a module
	// boundary, and this type is consumed by the subscription module.
	//
	// An empty result is an EMPTY SLICE, not nil — callers range over it, and a nil
	// slice reads as "not implemented" where it means "nothing in this window".
	Items []*dto.UserBriefContractDTO
	// TotalCount is the number of accounts in the whole window, NOT the size of
	// this batch. The catch-up sweep needs it to know how many batches are left.
	TotalCount int64
}

// CampaignAccountLister is the narrow slice the campaign mechanism needs from the
// identity module (025 T023, FR-079).
//
// It is separate from `IUserBriefContract` on purpose, for the same reason
// `AdminGrantLookup` above is: that contract is served over gRPC, with a client
// adapter (contracts/identity/v1/client/user_brief_client.go) and a server adapter
// (identity/presentation/grpc/server.go). Adding a method there would force a
// change to `identity.proto` — a CROSS-SERVICE contract change — for a need that
// exists only inside this service, which FR-058 forbids.
//
// Declared here, in the contract package, because both sides already depend on it:
// the contract can build it without importing the subscription module, and the
// subscription module can consume it without knowing how identity is wired.
type CampaignAccountLister interface {
	// ListCreatedInWindow returns at most `filter.Limit` accounts whose
	// `created_at` lies in [filter.From, filter.To), oldest first.
	//
	// Pagination is by the COMPOSITE key: the next batch starts at the row whose
	// `created_at` is greater than `filter.CursorTime`, or equal to it with an
	// `id` greater than `filter.CursorID`. The composite key is mandatory:
	// `created_at` has millisecond resolution, so several accounts routinely
	// share one instant, and paging by the timestamp alone skips the rest of that
	// instant or repeats it forever — contracts/identity-contract.md §6, scenario
	// 2.
	//
	// Only accounts that are NOT soft-deleted are issued: a deleted account must
	// not receive new entitlements.
	ListCreatedInWindow(ctx context.Context, filter repositories.CampaignAccountFilter) (*CampaignAccountPage, error)
}

// NewCampaignAccountLister narrows the concrete contract to the campaign's view.
func NewCampaignAccountLister(c *UserBriefContract) CampaignAccountLister { return c }

// ListCreatedInWindow serves the campaign mechanism in-process (025 T023).
func (c *UserBriefContract) ListCreatedInWindow(ctx context.Context, filter repositories.CampaignAccountFilter) (*CampaignAccountPage, error) {
	page, err := c.campaignAccountsUC.ListCreatedInWindow(ctx, filter)
	if err != nil {
		return nil, err
	}
	// The use case already guarantees a non-nil slice; this copy exists so the
	// contract holds no reference to the use case's internal slice.
	items := make([]*dto.UserBriefContractDTO, 0, len(page.Items))
	for _, brief := range page.Items {
		if brief == nil {
			continue
		}
		items = append(items, brief)
	}
	return &CampaignAccountPage{Items: items, TotalCount: page.TotalCount}, nil
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
