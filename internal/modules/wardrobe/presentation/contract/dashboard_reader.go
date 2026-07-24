package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"

	"github.com/google/uuid"
)

type IWardrobeDashboardReader interface {
	GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error)
	GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.UnderutilizedItemInfoDTO, error)
	GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error)
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

func (r *WardrobeDashboardReader) GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.UnderutilizedItemInfoDTO, error) {
	return r.wardrobeUC.GetUnderutilizedItems(ctx, userID, limit)
}

func (r *WardrobeDashboardReader) GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error) {
	return r.wardrobeUC.GetOutfitItemFashionItemIDs(ctx, outfitIDs)
}
