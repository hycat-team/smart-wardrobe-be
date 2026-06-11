package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/colorutils"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WardrobeWorkerUseCase struct {
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryRepo    repositories.ICategoryRepository
	mediaService    media.IMediaService
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	eventPublisher  event.IEventPublisher
}

func NewWardrobeWorkerUseCase(
	logger logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryRepo repositories.ICategoryRepository,
	mediaService media.IMediaService,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
) uc_interfaces.IWardrobeWorkerUseCase {
	return &WardrobeWorkerUseCase{
		logger:          logger,
		wardrobeRepo:    wardrobeRepo,
		categoryRepo:    categoryRepo,
		mediaService:    mediaService,
		aiService:       aiService,
		userSubContract: userSubContract,
		eventPublisher:  eventPublisher,
	}
}

func (uc *WardrobeWorkerUseCase) handleJobFailure(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, err error) {
	if isTransientError(err) && job.RetryCount < 3 {
		job.RetryCount++

		baseDelay := 60 * time.Second
		switch job.RetryCount {
		case 2:
			baseDelay = 300 * time.Second
		case 3:
			baseDelay = 900 * time.Second
		}

		jitter := time.Duration(1+rand.Intn(10)) * time.Second
		delay := baseDelay + jitter

		uc.logger.Warn(fmt.Sprintf("[WardrobeBatchUploadWorker] Transient error encountered. Scheduling retry %d/3 in %v", job.RetryCount, delay),
			zap.String("item_id", job.ItemID.String()),
			zap.Error(err),
		)

		time.AfterFunc(delay, func() {
			publishCtx := context.Background()
			if republishErr := uc.eventPublisher.Publish(publishCtx, eventconstants.TopicWardrobeBatchUpload, job); republishErr != nil {
				uc.logger.Error("[WardrobeBatchUploadWorker] Failed to republish retry job", zap.Error(republishErr))
			}
		})
		return
	}

	uc.logger.Error(fmt.Sprintf("[WardrobeBatchUploadWorker] Fatal error or max retries exceeded. Marking item as Failed. RetryCount: %d", job.RetryCount),
		zap.String("item_id", job.ItemID.String()),
		zap.Error(err),
	)
	uc.markJobFailed(ctx, job.ItemID)
}

func (uc *WardrobeWorkerUseCase) ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) error {
	categories, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		uc.handleJobFailure(ctx, job, fmt.Errorf("category lookup failed: %w", err))
		return nil
	}

	aiCatRefs := make([]shared_dto.AICategoryRef, len(categories))
	categoryMap := make(map[string]uuid.UUID)
	categoryNameMap := make(map[uuid.UUID]string)
	var otherCategoryID uuid.UUID

	for i, cat := range categories {
		aiCatRefs[i] = shared_dto.AICategoryRef{
			Name: cat.Name,
			Slug: cat.Slug,
		}
		categoryMap[cat.Slug] = cat.ID
		categoryNameMap[cat.ID] = cat.Name
		if cat.Slug == "other" {
			otherCategoryID = cat.ID
		}
	}

	prompt := getVisionSystemPrompt(aiCatRefs)
	responseText, err := uc.aiService.AnalyzeImage(ctx, job.ImageUrl, prompt)
	if err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	var result struct {
		shared_dto.FashionMetadataResult
		Error string `json:"error"`
	}
	cleanedJSON := stringutils.CleanJSONMarkdown(responseText)
	if err := json.Unmarshal([]byte(cleanedJSON), &result); err != nil {
		uc.handleJobFailure(ctx, job, fmt.Errorf("failed to parse JSON from AI: %w", err))
		return nil
	}

	if result.Error != "" {
		uc.handleJobFailure(ctx, job, fmt.Errorf("AI Error: %s", result.Error))
		return nil
	}

	aiMeta := result.FashionMetadataResult

	var detectedCategoryID *uuid.UUID
	detectedCategoryName := "Khác"

	if id, exists := categoryMap[aiMeta.CategorySlug]; exists {
		detectedCategoryID = &id
		if name, existsName := categoryNameMap[id]; existsName {
			detectedCategoryName = name
		}
	} else if otherCategoryID != uuid.Nil {
		detectedCategoryID = &otherCategoryID
		if name, existsName := categoryNameMap[otherCategoryID]; existsName {
			detectedCategoryName = name
		}
	}

	richTextContext := shared.BuildRichTextContext(detectedCategoryName, aiMeta.Color, aiMeta.Style, aiMeta.Material, aiMeta.Pattern, aiMeta.Fit, aiMeta.Seasonality, aiMeta.Description)

	embedding, err := shared.GenerateItemEmbedding(ctx, uc.aiService, richTextContext)
	if err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	item, err := uc.wardrobeRepo.GetByID(ctx, job.ItemID)
	if err != nil || item == nil {
		if err == nil {
			err = fmt.Errorf("item not found in database")
		}
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	item.CategoryID = detectedCategoryID
	item.Color = &aiMeta.Color
	item.Style = &aiMeta.Style
	item.Material = &aiMeta.Material
	item.Pattern = &aiMeta.Pattern
	item.Fit = &aiMeta.Fit
	item.Seasonality = &aiMeta.Seasonality
	item.Description = &aiMeta.Description
	item.Embedding = entities.Vector(embedding)
	item.Status = wardrobestatus.InWardrobe

	if h, s, l, hex, ok := colorutils.ResolveFashionColor(aiMeta.Color, aiMeta.ColorHex); ok {
		item.ColorHex = &hex
		item.ColorHue = &h
		item.ColorSaturation = &s
		item.ColorLightness = &l
	} else {
		uc.logger.Warn("[WardrobeBatchUploadWorker] Failed to resolve HSL for item",
			zap.String("item_id", item.ID.String()),
			zap.String("color_hex", aiMeta.ColorHex),
			zap.String("color_name", aiMeta.Color),
		)
	}

	if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	payload := dto.WardrobeEventPayload{
		ItemID: item.ID,
		UserID: item.UserID,
		Action: eventconstants.ActionCreated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)

	return nil
}

func (uc *WardrobeWorkerUseCase) markJobFailed(ctx context.Context, itemID uuid.UUID) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err == nil && item != nil {
		item.Status = wardrobestatus.Failed
		_ = uc.wardrobeRepo.Update(ctx, item)
	}
}

func (uc *WardrobeWorkerUseCase) CleanupFailedItems(ctx context.Context) error {
	limit := 100
	totalDeleted := 0

	for {
		items, err := uc.wardrobeRepo.GetFailedItemsForCleanup(ctx, limit)
		if err != nil {
			return fmt.Errorf("failed to fetch items for cleanup: %w", err)
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if item.ImagePublicID != "" {
				if err := uc.mediaService.DeleteImage(ctx, item.ImagePublicID); err != nil {
					uc.logger.Warn("[CleanupFailedItems] Failed to delete image from Cloudinary",
						zap.String("item_id", item.ID.String()),
						zap.String("public_id", item.ImagePublicID),
						zap.Error(err),
					)
				}
			}

			if err := uc.wardrobeRepo.Delete(ctx, item.ID); err != nil {
				uc.logger.Error("[CleanupFailedItems] Failed to delete item from database",
					zap.String("item_id", item.ID.String()),
					zap.Error(err),
				)
				return fmt.Errorf("failed to delete item %s: %w", item.ID.String(), err)
			}

			payload := dto.WardrobeEventPayload{
				ItemID: item.ID,
				UserID: item.UserID,
				Action: eventconstants.ActionDeleted,
			}
			_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeDeleted, payload)

			totalDeleted++
		}

		if len(items) < limit {
			break
		}
	}

	uc.logger.Info("[CleanupFailedItems] Successfully cleaned up failed items", zap.Int("count", totalDeleted))
	return nil
}
