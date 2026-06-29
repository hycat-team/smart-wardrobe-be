package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
)

type IWardrobeItemUseCase interface {
	GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error)
	GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error)
	GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error)
	CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error)
	GetSystemCatalogWardrobeItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) (*shared_dto.PaginationResult[*dto.SearchWardrobeItemRes], error)
	ManualClassify(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.ManualClassifyReq) (*dto.WardrobeItemRes, error)
	RetryWardrobeAnalysis(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.WardrobeItemRes, error)
	DeleteWardrobeItemsBulk(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
	DeleteLockedWardrobeItems(ctx context.Context, userID uuid.UUID) error
	BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemRes, error)
	GetWardrobeStats(ctx context.Context, userID uuid.UUID) (*dto.WardrobeStatsRes, error)
}

type IWardrobeCatalogUseCase interface {
	InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetSystemCatalogItems(ctx context.Context, query dto.GetSystemCatalogItemsQueryReq) (*shared_dto.PaginationResult[*dto.WardrobeItemRes], error)
	UpdateSystemCatalogItem(ctx context.Context, id uuid.UUID, input dto.UpdateSystemCatalogItemReq) (*dto.WardrobeItemRes, error)
	DeleteSystemCatalogItem(ctx context.Context, id uuid.UUID) error
}

type IWardrobeWorkerUseCase interface {
	ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, run *workerlog.Run) error
}
