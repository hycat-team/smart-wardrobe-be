package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type IWardrobeUseCase interface {
	GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error)
	GetWardrobeItems(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error)
	CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error)
	InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error)
	BatchCropWardrobeItems(ctx context.Context, userID uuid.UUID, input dto.BatchCropWardrobeItemsReq) ([]*dto.WardrobeItemRes, error)
	ProcessBackgroundCropJob(ctx context.Context, job dto.BatchCropJobDTO) error
}
