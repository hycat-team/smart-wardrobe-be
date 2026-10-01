package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type UserFilter struct {
	RoleSlug *string
	IsActive *bool
	Query    *string
	Page     int
	Limit    int
}

type UserListResult struct {
	Users      []*entities.User
	TotalCount int64
}

// IUserRepository handles repository operations for basic user identities
type IUserRepository interface {
	repositories.IGenericRepository[entities.User, uuid.UUID]
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByProviderID(ctx context.Context, provider string, providerID string) (*entities.User, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	IsEmailExists(ctx context.Context, email string) (bool, error)
	IsUsernameExists(ctx context.Context, username string) (bool, error)
	GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*entities.User, error)
	// GetByUsernamesOrEmailsIncludingDeleted is the admin-grant-only variant that
	// returns soft-deleted rows with their flag intact, so the caller can report
	// `user_soft_deleted` instead of a misleading `user_not_found` (024 T136).
	//
	// Separate from GetByUsernamesOrEmails on purpose: that one is shared with
	// commerce, which must keep excluding deleted accounts.
	GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*entities.User, error)
	SearchByKeyword(ctx context.Context, keyword string, limit int) ([]*entities.User, error)
	// ListCreatedInWindow returns at most `filter.Limit` non-deleted accounts whose
	// `created_at` falls in [filter.From, filter.To), oldest first (025 T021).
	//
	// It exists for the campaign catch-up sweep (FR-079), which has to find
	// eligible accounts whose grant was never written. The sweep runs inside this
	// service, which is why this stays on the repository rather than becoming a
	// contract method — see contracts/identity-contract.md §2.
	//
	// READ-ONLY, deliberately. The campaign mechanism has no business reason to
	// change an account; every account state change goes through the existing
	// flows.
	ListCreatedInWindow(ctx context.Context, filter CampaignAccountFilter) (*CampaignAccountPage, error)
	GetUsersForAdmin(ctx context.Context, filter UserFilter) (*UserListResult, error)
	GetStyleProfile(ctx context.Context, userID uuid.UUID) (*UserStyleProfileRes, error)
	GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*UserAdminStatsDTO, error)
}
