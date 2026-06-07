package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type IWardrobeUseCase interface {
	IWardrobeContractUseCase
	GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error)
	GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) ([]*dto.WardrobeItemRes, error)
	GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error)
	CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error)
	InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error)
	BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemRes, error)
	ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) error
	SearchWardrobeItems(ctx context.Context, query dto.SearchWardrobeItemsQueryReq) ([]*dto.SearchWardrobeItemRes, error)
	ManualClassify(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.ManualClassifyReq) (*dto.WardrobeItemRes, error)
	CleanupFailedItems(ctx context.Context) error
	RecommendOutfit(ctx context.Context, userID uuid.UUID, input dto.RecommendOutfitReq) (*dto.RecommendedOutfitRes, error)
	CreateChatSession(ctx context.Context, userID uuid.UUID, title *string) (*dto.ChatSessionRes, error)
	GetChatSessions(ctx context.Context, userID uuid.UUID) ([]*dto.ChatSessionRes, error)
	GetChatMessages(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) ([]*dto.ChatMessageRes, error)
	ArchiveChatSession(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) error
	ProcessChatMessage(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, content string) (*dto.ChatMessageRes, *dto.ChatMessageRes, error)
}
