package contract

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type IFashionContract interface {
	GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemContractDTO, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemContractDTO, error)
	ListFashionItemsBrief(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemBriefRes, error)
	CreateFashionItem(ctx context.Context, input dto.CreateFashionItemInput) error
	BulkCreateFashionItems(ctx context.Context, inputs []dto.CreateFashionItemInput) error
	UpdateFashionItemCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error
	GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error)
	SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error)
	ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error
	// SetProcessingErrorReason records a processing-error code (and clears the
	// review reason) for an item, e.g. when a retry fails to publish.
	SetProcessingErrorReason(ctx context.Context, fashionItemID uuid.UUID, reason string) error
	// IsWearableCategory reports whether a category can participate in outfit
	// pairing (i.e. maps to a real wearing role, not "other"). Feature 023 US1.
	IsWearableCategory(ctx context.Context, categoryID uuid.UUID) (bool, error)
}
