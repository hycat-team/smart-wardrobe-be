package usecase

import (
	"context"
	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type IWardrobeItemUseCase interface {
	GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error)
	GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemBriefRes], error)
	GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error)
	UpdateWardrobeItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.UpdateWardrobeItemReq) (*dto.WardrobeItemRes, error)
	CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemBriefRes, error)
	GetSystemCatalogWardrobeItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.SearchWardrobeItemRes], error)
	RetryWardrobeAnalysis(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.WardrobeItemRes, error)
	DeleteWardrobeItemsBulk(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
	DeleteLockedWardrobeItems(ctx context.Context, userID uuid.UUID) error
	BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemBriefRes, error)
	GetWardrobeStats(ctx context.Context, userID uuid.UUID) (*dto.WardrobeStatsRes, error)
	GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeSummaryStatsDTO, error)
	GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.UnderutilizedItemInfoDTO, error)
	GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error)
	GetUserWardrobeFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type IWardrobeCatalogUseCase interface {
	InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemBriefRes, error)
	GetSystemCatalogItems(ctx context.Context, query dto.GetSystemCatalogItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error)
	UpdateSystemCatalogItem(ctx context.Context, id uuid.UUID, input dto.UpdateSystemCatalogItemReq) (*dto.WardrobeItemRes, error)
	DeleteSystemCatalogItem(ctx context.Context, id uuid.UUID) error
}

type IWardrobeFashionEventUseCase interface {
	HandleFashionAnalyzed(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error
	HandleFashionFailed(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error
	HandleFashionItemCreated(ctx context.Context, payload fashionDto.FashionItemCreatedPayload) error
	HandleFashionProcessingExhausted(ctx context.Context, payload fashionDto.FashionProcessingExhaustedPayload) error
	HandleWardrobeDeleted(ctx context.Context, payload dto.WardrobeEventPayload) error
}
