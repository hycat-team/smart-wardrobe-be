package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"

	"github.com/google/uuid"
)

type IUserDashboardReader interface {
	GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error)
	GetActiveUserIDs(ctx context.Context) ([]uuid.UUID, error)
}

type UserDashboardReader struct {
	userUC uc_interfaces.IUserUseCase
}

func NewUserDashboardReader(userUC uc_interfaces.IUserUseCase) IUserDashboardReader {
	return &UserDashboardReader{userUC: userUC}
}

func (r *UserDashboardReader) GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error) {
	return r.userUC.GetAdminUserStats(ctx, snapshotDate)
}

func (r *UserDashboardReader) GetActiveUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	// The whole id set, not a page: it used to flow through GetUsersForAdmin
	// whose pagination defaults cap the result at 20 ids.
	return r.userUC.GetActiveUserIDs(ctx)
}
