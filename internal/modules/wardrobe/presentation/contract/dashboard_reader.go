package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"

	"github.com/google/uuid"
)

type IWardrobeDashboardReader interface {
	GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error)
}

type WardrobeDashboardReader struct {
	wardrobeUC uc_interfaces.IWardrobeItemUseCase
}

func NewWardrobeDashboardReader(wardrobeUC uc_interfaces.IWardrobeItemUseCase) IWardrobeDashboardReader {
	return &WardrobeDashboardReader{wardrobeUC: wardrobeUC}
}

func (r *WardrobeDashboardReader) GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error) {
	return r.wardrobeUC.GetUserWardrobeSummaryStats(ctx, userID)
}
