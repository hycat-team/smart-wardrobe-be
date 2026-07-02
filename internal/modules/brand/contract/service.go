package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type IBrandContract interface {
	CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode benefitfeaturecode.BenefitFeatureCode) (bool, error)
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemStylingDTO, []*entities.FashionItem, error)
	CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *entities.FashionItem, error)
	SetFashionItemID(ctx context.Context, brandItemID uuid.UUID, fashionItemID uuid.UUID) error
	UpdateBrandItemStatus(ctx context.Context, brandItemID uuid.UUID, status string) error
}
