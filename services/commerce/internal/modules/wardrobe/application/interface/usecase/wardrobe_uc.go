package usecase

import (
	"context"
	fashionDto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type IWardrobeItemUseCase interface {
	GetUploadSignature(ctx context.Context, userID uuid.UUID, role roleslug.RoleSlug) (*dto.WardrobeUploadSignatureRes, error)
	GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemBriefRes], error)
	GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error)
	UpdateWardrobeItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.UpdateWardrobeItemReq) (*dto.WardrobeItemRes, error)
	CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemBriefRes, error)
	GetSystemCatalogWardrobeItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.SearchWardrobeItemRes], error)
	RetryWardrobeAnalysis(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, categoryID *uuid.UUID) (*dto.WardrobeItemRes, error)
	DeleteWardrobeItemsBulk(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
	DeleteLockedWardrobeItems(ctx context.Context, userID uuid.UUID) error
	BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemBriefRes, error)
	GetWardrobeStats(ctx context.Context, userID uuid.UUID) (*dto.WardrobeStatsRes, error)
	GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error)
	GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.UnderutilizedItemInfoDTO, error)
	GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error)
	GetUserWardrobeFashionItemCounts(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error)
}

type IWardrobeCatalogUseCase interface {
	InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemBriefRes, error)
	GetSystemCatalogItems(ctx context.Context, query dto.GetSystemCatalogItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error)
	UpdateSystemCatalogItem(ctx context.Context, id uuid.UUID, input dto.UpdateSystemCatalogItemReq) (*dto.WardrobeItemRes, error)
	DeleteSystemCatalogItem(ctx context.Context, id uuid.UUID) error
}

type IWardrobeFashionEventUseCase interface {
	HandleFashionEvent(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error
	HandleFashionAnalyzed(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error
	HandleFashionFailed(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error
	HandleFashionProcessing(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error
	HandleFashionItemCreated(ctx context.Context, payload fashionDto.FashionItemCreatedEventPayload) error
	HandleFashionProcessingExhausted(ctx context.Context, payload fashionDto.FashionProcessingExhaustedEventPayload) error
	HandleWardrobeDeleted(ctx context.Context, payload dto.WardrobeEventPayload) error
}
