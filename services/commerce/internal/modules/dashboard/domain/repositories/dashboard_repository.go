package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"

	"github.com/google/uuid"
)

type IDashboardRepository interface {
	UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error
	GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error)
	GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error)

	UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error
	GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error)
	GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error)

	UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error
	GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error)
}
