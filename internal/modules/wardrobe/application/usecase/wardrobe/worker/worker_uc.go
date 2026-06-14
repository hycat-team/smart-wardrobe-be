package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
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
	cfg             *config.Config
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryRepo    repositories.ICategoryRepository
	mediaService    media.IMediaService
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	eventPublisher  event.IEventPublisher
}

const processingFailureReasonMessage = "Hệ thống chưa thể phân tích trang phục lúc này. Vui lòng thử lại sau."

func NewWardrobeWorkerUseCase(
	cfg *config.Config,
	logger logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryRepo repositories.ICategoryRepository,
	mediaService media.IMediaService,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
) uc_interfaces.IWardrobeWorkerUseCase {
	return &WardrobeWorkerUseCase{
		cfg:             cfg,
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
	if isTransientError(err) && job.RetryCount < uc.cfg.Wardrobe.MaxRetryCount {
		job.RetryCount++

		baseDelay := time.Duration(uc.cfg.Wardrobe.RetryDelay1Seconds) * time.Second
		switch job.RetryCount {
		case 2:
			baseDelay = time.Duration(uc.cfg.Wardrobe.RetryDelay2Seconds) * time.Second
		case 3:
			baseDelay = time.Duration(uc.cfg.Wardrobe.RetryDelay3Seconds) * time.Second
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
	_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, job.ItemID, job.ProcessingVersion, processingFailureReasonMessage, nil)
}

func (uc *WardrobeWorkerUseCase) ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, job.ItemID)
	if err != nil {
		return err
	}
	if item == nil || item.Status != wardrobestatus.Processing || item.ProcessingVersion != job.ProcessingVersion {
		return nil
	}

	categories, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		uc.handleJobFailure(ctx, job, fmt.Errorf("category lookup failed: %w", err))
		return nil
	}

	aiCatRefs := make([]dto.AICategoryRef, len(categories))
	categoryMap := make(map[string]uuid.UUID)
	categoryNameMap := make(map[uuid.UUID]string)
	var otherCategoryID uuid.UUID

	for i, cat := range categories {
		aiCatRefs[i] = dto.AICategoryRef{
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
		dto.FashionMetadataResult
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
	if !aiMeta.IsSingleItem {
		reason := aiMeta.ReviewReason
		if reason == "" {
			reason = "uncertain_category"
		}
		_, _ = uc.wardrobeRepo.MarkProcessingNeedsReview(ctx, job.ItemID, job.ProcessingVersion, reason)
		return nil
	}

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

	updateMap := map[string]any{
		"category_id":  detectedCategoryID,
		"color":        aiMeta.Color,
		"style":        aiMeta.Style,
		"material":     aiMeta.Material,
		"pattern":      aiMeta.Pattern,
		"fit":          aiMeta.Fit,
		"seasonality":  aiMeta.Seasonality,
		"description":  aiMeta.Description,
		"embedding":    entities.Vector(embedding),
	}
	if h, s, l, hex, ok := colorutils.ResolveFashionColor(aiMeta.Color, aiMeta.ColorHex); ok {
		updateMap["color_hex"] = hex
		updateMap["color_hue"] = h
		updateMap["color_saturation"] = s
		updateMap["color_lightness"] = l
	} else {
		uc.logger.Warn("[WardrobeBatchUploadWorker] Failed to resolve HSL for item",
			zap.String("item_id", job.ItemID.String()),
			zap.String("color_hex", aiMeta.ColorHex),
			zap.String("color_name", aiMeta.Color),
		)
		updateMap["color_hex"] = nil
		updateMap["color_hue"] = nil
		updateMap["color_saturation"] = nil
		updateMap["color_lightness"] = nil
	}

	completed, err := uc.wardrobeRepo.CompleteProcessingSuccess(ctx, job.ItemID, job.ProcessingVersion, updateMap)
	if err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}
	if !completed {
		return nil
	}

	payload := dto.WardrobeEventPayload{
		ItemID: job.ItemID,
		UserID: job.UserID,
		Action: eventconstants.ActionCreated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)

	return nil
}

func (uc *WardrobeWorkerUseCase) markJobFailed(ctx context.Context, itemID uuid.UUID) {
	_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, itemID, 0, "Xử lý trang phục thất bại.", nil)
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

func (uc *WardrobeWorkerUseCase) RecoverStaleProcessingItems(ctx context.Context) error {
	staleBefore := time.Now().UTC().Add(-time.Duration(uc.cfg.Wardrobe.StaleMinutes) * time.Minute)
	items, err := uc.wardrobeRepo.GetStaleProcessingItems(ctx, staleBefore, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch stale processing items: %w", err)
	}

	for _, item := range items {
		if item.ProcessingRetryCount >= uc.cfg.Wardrobe.MaxRetryCount {
			failed, err := uc.wardrobeRepo.MarkProcessingFailed(ctx, item.ID, item.ProcessingVersion, "Đã vượt quá số lần thử xử lý tự động.", nil)
			if err != nil {
				return err
			}
			if !failed {
				continue
			}
			continue
		}

		now := time.Now().UTC()
		claimedItem, claimed, err := uc.wardrobeRepo.ClaimStaleProcessingRetry(ctx, item.ID, item.ProcessingVersion, staleBefore, now)
		if err != nil {
			return err
		}
		if !claimed || claimedItem == nil {
			continue
		}

		job := dto.WardrobeBatchUploadJobDTO{
			ItemID:            claimedItem.ID,
			UserID:            claimedItem.UserID,
			CategoryID:        claimedItem.CategoryID,
			ImageUrl:          claimedItem.ImageUrl,
			ImagePublicID:     claimedItem.ImagePublicID,
			RetryCount:        claimedItem.ProcessingRetryCount,
			ProcessingVersion: claimedItem.ProcessingVersion,
		}
		if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeBatchUpload, job); err != nil {
			_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, claimedItem.ID, claimedItem.ProcessingVersion, processingFailureReasonMessage, nil)
			return err
		}
	}

	return nil
}
