package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
)

type IUserDashboardReader interface {
	GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error)
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
