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

const (
	processingFailureReasonMessage = "Hệ thống chưa thể phân tích trang phục lúc này. Vui lòng thử lại sau."
	autoRetryExceededMessage       = "Đã vượt quá số lần thử xử lý tự động."
)

// WardrobeWorkerUseCase orchestrates asynchronous wardrobe item processing and recovery workflows.
type WardrobeWorkerUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryCache   *VisionCategoryCache
	mediaService    media.IMediaService
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	eventPublisher  event.IEventPublisher
}

// NewWardrobeWorkerUseCase creates the worker use case responsible for AI item processing jobs.
func NewWardrobeWorkerUseCase(
	cfg *config.Config,
	logger logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryCache *VisionCategoryCache,
	mediaService media.IMediaService,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
) uc_interfaces.IWardrobeWorkerUseCase {
	return &WardrobeWorkerUseCase{
		cfg:             cfg,
		logger:          logger,
		wardrobeRepo:    wardrobeRepo,
		categoryCache:   categoryCache,
		mediaService:    mediaService,
		aiService:       aiService,
		userSubContract: userSubContract,
		eventPublisher:  eventPublisher,
	}
}

// ProcessBackgroundBatchUploadJob analyzes a processing wardrobe item and finalizes its AI metadata.
func (uc *WardrobeWorkerUseCase) ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) error {
	item, err := uc.loadProcessableItem(ctx, job)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	snapshot, err := uc.categoryCache.Get(ctx)
	if err != nil {
		uc.handleJobFailure(ctx, job, fmt.Errorf("category lookup failed: %w", err))
		return nil
	}

	aiMeta, err := uc.analyzeWardrobeImage(ctx, job, snapshot)
	if err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	if !aiMeta.IsSingleItem {
		uc.markNeedsReview(ctx, job, aiMeta.ReviewReason)
		return nil
	}

	updateMap, err := uc.buildCompletedItemUpdate(ctx, aiMeta, snapshot, job.ItemID)
	if err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	if err := uc.completeProcessedItem(ctx, job, updateMap); err != nil {
		uc.handleJobFailure(ctx, job, err)
	}

	return nil
}

// CleanupFailedItems removes failed items older than the retention window and publishes delete events.
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

// RecoverStaleProcessingItems republishes stale processing items or marks them failed when retries are exhausted.
func (uc *WardrobeWorkerUseCase) RecoverStaleProcessingItems(ctx context.Context) error {
	staleBefore := time.Now().UTC().Add(-time.Duration(uc.cfg.Wardrobe.StaleMinutes) * time.Minute)
	items, err := uc.wardrobeRepo.GetStaleProcessingItems(ctx, staleBefore, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch stale processing items: %w", err)
	}

	for _, item := range items {
		if item.ProcessingRetryCount >= uc.cfg.Wardrobe.MaxRetryCount {
			failed, err := uc.wardrobeRepo.MarkProcessingFailed(ctx, item.ID, item.ProcessingVersion, autoRetryExceededMessage, nil)
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

		if err := uc.publishProcessingRetry(ctx, claimedItem); err != nil {
			_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, claimedItem.ID, claimedItem.ProcessingVersion, processingFailureReasonMessage, nil)
			return err
		}
	}

	return nil
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

func (uc *WardrobeWorkerUseCase) loadProcessableItem(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) (*entities.WardrobeItem, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, job.ItemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.Status != wardrobestatus.Processing || item.ProcessingVersion != job.ProcessingVersion {
		return nil, nil
	}
	return item, nil
}

func (uc *WardrobeWorkerUseCase) analyzeWardrobeImage(
	ctx context.Context,
	job dto.WardrobeBatchUploadJobDTO,
	snapshot *VisionCategorySnapshot,
) (*dto.FashionMetadataResult, error) {
	responseText, err := uc.aiService.AnalyzeImage(ctx, job.ImageUrl, snapshot.Prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		dto.FashionMetadataResult
		Error string `json:"error"`
	}
	cleanedJSON := stringutils.CleanJSONMarkdown(responseText)
	if err := json.Unmarshal([]byte(cleanedJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from AI: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("AI Error: %s", result.Error)
	}

	return &result.FashionMetadataResult, nil
}

func (uc *WardrobeWorkerUseCase) markNeedsReview(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, reviewReason string) {
	if reviewReason == "" {
		reviewReason = "uncertain_category"
	}
	_, _ = uc.wardrobeRepo.MarkProcessingNeedsReview(ctx, job.ItemID, job.ProcessingVersion, reviewReason)
}

func (uc *WardrobeWorkerUseCase) buildCompletedItemUpdate(
	ctx context.Context,
	aiMeta *dto.FashionMetadataResult,
	snapshot *VisionCategorySnapshot,
	itemID uuid.UUID,
) (map[string]any, error) {
	detectedCategoryID, detectedCategoryName := resolveDetectedCategory(aiMeta.CategorySlug, snapshot)
	richTextContext := shared.BuildRichTextContext(detectedCategoryName, aiMeta.Color, aiMeta.Style, aiMeta.Material, aiMeta.Pattern, aiMeta.Fit, aiMeta.Seasonality, aiMeta.Description)

	embedding, err := shared.GenerateItemEmbedding(ctx, uc.aiService, richTextContext)
	if err != nil {
		return nil, err
	}

	updateMap := map[string]any{
		"category_id": detectedCategoryID,
		"color":       aiMeta.Color,
		"style":       aiMeta.Style,
		"material":    aiMeta.Material,
		"pattern":     aiMeta.Pattern,
		"fit":         aiMeta.Fit,
		"seasonality": aiMeta.Seasonality,
		"description": aiMeta.Description,
		"embedding":   entities.Vector(embedding),
	}
	applyResolvedColor(updateMap, aiMeta, uc.logger, itemID)
	return updateMap, nil
}

func (uc *WardrobeWorkerUseCase) completeProcessedItem(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, updateMap map[string]any) error {
	completed, err := uc.wardrobeRepo.CompleteProcessingSuccess(ctx, job.ItemID, job.ProcessingVersion, updateMap)
	if err != nil {
		return err
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

func (uc *WardrobeWorkerUseCase) publishProcessingRetry(ctx context.Context, item *entities.WardrobeItem) error {
	job := dto.WardrobeBatchUploadJobDTO{
		ItemID:            item.ID,
		UserID:            item.UserID,
		CategoryID:        item.CategoryID,
		ImageUrl:          item.ImageUrl,
		ImagePublicID:     item.ImagePublicID,
		RetryCount:        item.ProcessingRetryCount,
		ProcessingVersion: item.ProcessingVersion,
	}

	return uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeBatchUpload, job)
}

func resolveDetectedCategory(categorySlug string, snapshot *VisionCategorySnapshot) (*uuid.UUID, string) {
	detectedCategoryName := "Khác"

	if id, exists := snapshot.CategoryMap[categorySlug]; exists {
		if name, existsName := snapshot.CategoryNameMap[id]; existsName {
			detectedCategoryName = name
		}
		return &id, detectedCategoryName
	}

	if snapshot.OtherCategoryID != uuid.Nil {
		if name, existsName := snapshot.CategoryNameMap[snapshot.OtherCategoryID]; existsName {
			detectedCategoryName = name
		}
		return &snapshot.OtherCategoryID, detectedCategoryName
	}

	return nil, detectedCategoryName
}

func applyResolvedColor(updateMap map[string]any, aiMeta *dto.FashionMetadataResult, logger logger.Interface, itemID uuid.UUID) {
	if h, s, l, hex, ok := colorutils.ResolveFashionColor(aiMeta.Color, aiMeta.ColorHex); ok {
		updateMap["color_hex"] = hex
		updateMap["color_hue"] = h
		updateMap["color_saturation"] = s
		updateMap["color_lightness"] = l
		return
	}

	logger.Warn("[WardrobeBatchUploadWorker] Failed to resolve HSL for item",
		zap.String("item_id", itemID.String()),
		zap.String("color_hex", aiMeta.ColorHex),
		zap.String("color_name", aiMeta.Color),
	)
	updateMap["color_hex"] = nil
	updateMap["color_hue"] = nil
	updateMap["color_saturation"] = nil
	updateMap["color_lightness"] = nil
}
