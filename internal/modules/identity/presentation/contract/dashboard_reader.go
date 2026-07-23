package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"

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
	res, err := r.userUC.GetUsersForAdmin(ctx, dto.GetUsersQueryReq{})
	if err != nil || res == nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(res.Items))
	for _, u := range res.Items {
		ids = append(ids, u.ID)
	}
	return ids, nil
}
