package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"

	"github.com/google/uuid"
)

type IBrandContract interface {
	CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error)
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemStylingDTO, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, error)
}
