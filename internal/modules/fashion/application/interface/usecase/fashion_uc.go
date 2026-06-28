package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
)

type IOutfitRecommendationUseCase interface {
	RecommendOutfit(ctx context.Context, userID uuid.UUID, input dto.RecommendOutfitReq) (*dto.RecommendedOutfitRes, error)
}

type IWardrobeChatUseCase interface {
	CreateChatSession(ctx context.Context, userID uuid.UUID, title *string) (*dto.ChatSessionRes, error)
	GetChatSessions(ctx context.Context, userID uuid.UUID) ([]*dto.ChatSessionRes, error)
	GetChatMessages(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, query dto.GetChatMessagesQueryReq) (*shared_dto.PaginationResult[*dto.ChatMessageRes], error)
	ArchiveChatSession(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) error
	DeleteChatSession(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) error
	UpdateChatSession(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, input dto.UpdateChatSessionReq) (*dto.ChatSessionRes, error)
	ProcessChatMessageStream(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, content string) (<-chan string, func(success bool) error, error)
}

type IFashionWorkerUseCase interface {
	ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, run *workerlog.Run) error
	CleanupFailedItems(ctx context.Context, run *workerlog.Run) error
	RecoverStaleProcessingItems(ctx context.Context, run *workerlog.Run) error
}
