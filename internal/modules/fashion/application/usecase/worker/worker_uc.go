package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"smart-wardrobe-be/config"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
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
	fashionItemRepo repositories.IFashionItemRepository
	brandRepo       repositories.IBrandItemRepository
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
	fashionItemRepo repositories.IFashionItemRepository,
	brandRepo repositories.IBrandItemRepository,
) uc_interfaces.IFashionWorkerUseCase {
	return &WardrobeWorkerUseCase{
		cfg:             cfg,
		logger:          logger,
		wardrobeRepo:    wardrobeRepo,
		categoryCache:   categoryCache,
		mediaService:    mediaService,
		aiService:       aiService,
		userSubContract: userSubContract,
		eventPublisher:  eventPublisher,
		fashionItemRepo: fashionItemRepo,
		brandRepo:       brandRepo,
	}
}

// ProcessBackgroundBatchUploadJob analyzes a processing wardrobe item and finalizes its AI metadata.
func (uc *WardrobeWorkerUseCase) ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, run *workerlog.Run) error {
	run.AddTotal(1)
	item, err := uc.loadProcessableItem(ctx, job, run)
	if err != nil {
		return err
	}
	if item == nil {
		run.AddSkipped(1)
		return nil
	}

	snapshot, err := uc.categoryCache.Get(ctx)
	if err != nil {
		uc.handleJobFailure(ctx, job, fmt.Errorf("category lookup failed: %w", err), run)
		return nil
	}

	aiMeta, err := uc.analyzeWardrobeImage(ctx, job, snapshot)
	if err != nil {
		uc.handleJobFailure(ctx, job, err, run)
		return nil
	}

	if !aiMeta.IsSingleItem {
		run.AddSuccess(1)
		run.AddSummaryFields(zap.Bool("reviewRequired", true))
		run.ChildInfo(uc.logger, "Wardrobe batch upload requires review",
			zap.String("itemId", job.ItemID.String()),
			zap.String("reviewReason", aiMeta.ReviewReason),
		)
		uc.markNeedsReview(ctx, job, aiMeta.ReviewReason)
		return nil
	}

	updateMap, err := uc.buildCompletedItemUpdate(ctx, aiMeta, snapshot, job.ItemID)
	if err != nil {
		uc.handleJobFailure(ctx, job, err, run)
		return nil
	}

	if err := uc.completeProcessedItem(ctx, job, updateMap); err != nil {
		uc.handleJobFailure(ctx, job, err, run)
		return nil
	}

	run.AddSuccess(1)
	run.AddSummaryFields(zap.Bool("reviewRequired", false))

	return nil
}

// CleanupFailedItems removes failed items older than the retention window and publishes delete events.
func (uc *WardrobeWorkerUseCase) CleanupFailedItems(ctx context.Context, run *workerlog.Run) error {
	limit := 100
	totalDeleted := 0
	publishedDeleteEventCount := 0

	for {
		items, err := uc.wardrobeRepo.GetFailedItemsForCleanup(ctx, limit)
		if err != nil {
			return fmt.Errorf("failed to fetch items for cleanup: %w", err)
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			run.AddTotal(1)
			imagePublicID := item.FashionImagePublicID()
			if imagePublicID != "" {
				if err := uc.mediaService.DeleteImage(ctx, imagePublicID); err != nil {
					run.ChildWarn(uc.logger, "Failed to delete image from Cloudinary during failed item cleanup",
						zap.String("itemId", item.ID.String()),
						zap.String("publicId", imagePublicID),
						zap.Error(err),
					)
				}
			}

			if err := uc.wardrobeRepo.Delete(ctx, item.ID); err != nil {
				run.ChildError(uc.logger, "Failed to delete item from database during failed item cleanup",
					zap.String("itemId", item.ID.String()),
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
			publishedDeleteEventCount++
			totalDeleted++
			run.AddSuccess(1)
		}

		if len(items) < limit {
			break
		}
	}

	run.AddSummaryFields(
		zap.Int("deletedCount", totalDeleted),
		zap.Int("publishedDeleteEventCount", publishedDeleteEventCount),
	)
	return nil
}

// RecoverStaleProcessingItems republishes stale processing items or marks them failed when retries are exhausted.
func (uc *WardrobeWorkerUseCase) RecoverStaleProcessingItems(ctx context.Context, run *workerlog.Run) error {
	staleBefore := time.Now().UTC().Add(-time.Duration(uc.cfg.Wardrobe.StaleMinutes) * time.Minute)
	items, err := uc.wardrobeRepo.GetStaleProcessingItems(ctx, staleBefore, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch stale processing items: %w", err)
	}
	run.AddTotal(len(items))
	republishedCount := 0
	markedFailedCount := 0

	for _, item := range items {
		fashion := item.FashionItem
		if fashion == nil {
			run.AddSkipped(1)
			continue
		}
		if fashion.ProcessingRetryCount >= uc.cfg.Wardrobe.MaxRetryCount {
			failed, err := uc.wardrobeRepo.MarkProcessingFailed(ctx, item.ID, fashion.ProcessingVersion, autoRetryExceededMessage, nil)
			if err != nil {
				run.ChildError(uc.logger, "Failed to mark stale processing item as failed",
					zap.String("itemId", item.ID.String()),
					zap.Int("processingVersion", fashion.ProcessingVersion),
					zap.Int("processingRetryCount", fashion.ProcessingRetryCount),
					zap.Error(err),
				)
				return err
			}
			if !failed {
				run.AddSkipped(1)
				continue
			}
			markedFailedCount++
			run.ChildWarn(uc.logger, "Stale processing item exceeded retry limit and was marked failed",
				zap.String("itemId", item.ID.String()),
				zap.Int("processingVersion", fashion.ProcessingVersion),
				zap.Int("processingRetryCount", fashion.ProcessingRetryCount),
			)
			run.AddSuccess(1)
			continue
		}

		now := time.Now().UTC()
		claimedItem, claimed, err := uc.wardrobeRepo.ClaimStaleProcessingRetry(ctx, item.ID, fashion.ProcessingVersion, staleBefore, now)
		if err != nil {
			run.ChildError(uc.logger, "Failed to claim stale processing retry",
				zap.String("itemId", item.ID.String()),
				zap.Int("processingVersion", fashion.ProcessingVersion),
				zap.Int("processingRetryCount", fashion.ProcessingRetryCount),
				zap.Error(err),
			)
			return err
		}
		if !claimed || claimedItem == nil {
			run.ChildWarn(uc.logger, "Skipped stale processing item because it was not claimable",
				zap.String("itemId", item.ID.String()),
				zap.Int("processingVersion", fashion.ProcessingVersion),
				zap.Int("processingRetryCount", fashion.ProcessingRetryCount),
			)
			run.AddSkipped(1)
			continue
		}

		if err := uc.publishProcessingRetry(ctx, claimedItem); err != nil {
			claimedFashion := claimedItem.FashionItem
			run.ChildError(uc.logger, "Failed to republish stale processing retry",
				zap.String("itemId", claimedItem.ID.String()),
				zap.Int("processingVersion", claimedFashion.ProcessingVersion),
				zap.Int("processingRetryCount", claimedFashion.ProcessingRetryCount),
				zap.Error(err),
			)
			_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, claimedItem.ID, claimedFashion.ProcessingVersion, processingFailureReasonMessage, nil)
			return err
		}
		republishedCount++
		run.AddRetry(1)
		run.AddSuccess(1)
	}

	run.AddSummaryFields(
		zap.Int("staleItemsFoundCount", len(items)),
		zap.Int("republishedCount", republishedCount),
		zap.Int("markedFailedCount", markedFailedCount),
	)
	return nil
}

func (uc *WardrobeWorkerUseCase) handleJobFailure(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, err error, run *workerlog.Run) {
	topic := "fashion.event.analyze_item"
	if job.ItemType == "" {
		topic = eventconstants.TopicWardrobeBatchUpload
	}

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

		run.ChildWarn(uc.logger, "Item analyze failed and scheduled retry",
			zap.String("itemId", job.ItemID.String()),
			zap.Int("retryCount", job.RetryCount),
			zap.Duration("retryDelay", delay),
			zap.Error(err),
		)
		run.AddRetry(1)

		time.AfterFunc(delay, func() {
			publishCtx := context.Background()
			if republishErr := uc.eventPublisher.Publish(publishCtx, topic, job); republishErr != nil {
				run.ChildError(uc.logger, "Failed to republish analyze item retry job",
					zap.String("itemId", job.ItemID.String()),
					zap.Int("retryCount", job.RetryCount),
					zap.Error(republishErr),
				)
			}
		})
		return
	}

	run.ChildError(uc.logger, "Item analyze failed permanently",
		zap.String("itemId", job.ItemID.String()),
		zap.Int("retryCount", job.RetryCount),
		zap.Error(err),
	)
	if job.ItemType == "brand" {
		_, _ = uc.fashionItemRepo.MarkProcessingFailed(ctx, job.FashionItemID, job.ProcessingVersion, processingFailureReasonMessage, nil)
		brandItem, _ := uc.brandRepo.GetByID(ctx, job.ItemID)
		if brandItem != nil {
			brandItem.Status = "DRAFT"
			_ = uc.brandRepo.Update(ctx, brandItem)
		}
		return
	}
	_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, job.ItemID, job.ProcessingVersion, processingFailureReasonMessage, nil)
}

func (uc *WardrobeWorkerUseCase) loadProcessableItem(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, run *workerlog.Run) (any, error) {
	if job.ItemType == "brand" {
		item, err := uc.brandRepo.GetByID(ctx, job.ItemID)
		if err != nil {
			return nil, err
		}
		if item == nil || item.FashionItem == nil || item.FashionItem.ProcessingVersion != job.ProcessingVersion {
			run.ChildWarn(uc.logger, "Skipped brand item analyze job because item is not processable",
				zap.String("itemId", job.ItemID.String()),
				zap.Int("processingVersion", job.ProcessingVersion),
			)
			return nil, nil
		}
		return item, nil
	}

	item, err := uc.wardrobeRepo.GetByID(ctx, job.ItemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.FashionItem == nil || item.Status != wardrobestatus.Processing || item.FashionItem.ProcessingVersion != job.ProcessingVersion {
		run.ChildWarn(uc.logger, "Skipped wardrobe batch upload job because item is not processable",
			zap.String("itemId", job.ItemID.String()),
			zap.Int("processingVersion", job.ProcessingVersion),
		)
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
	if job.ItemType == "brand" {
		_, _ = uc.fashionItemRepo.MarkProcessingNeedsReview(ctx, job.FashionItemID, job.ProcessingVersion, reviewReason)
		brandItem, _ := uc.brandRepo.GetByID(ctx, job.ItemID)
		if brandItem != nil {
			brandItem.Status = "DRAFT"
			_ = uc.brandRepo.Update(ctx, brandItem)
		}
		return
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
	if job.ItemType == "brand" {
		completed, err := uc.fashionItemRepo.CompleteProcessingSuccess(ctx, job.FashionItemID, job.ProcessingVersion, updateMap)
		if err != nil {
			return err
		}
		if !completed {
			return nil
		}
		brandItem, err := uc.brandRepo.GetByID(ctx, job.ItemID)
		if err == nil && brandItem != nil {
			brandItem.Status = "ACTIVE"
			_ = uc.brandRepo.Update(ctx, brandItem)
		}
		return nil
	}

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
		CategoryID:        item.FashionCategoryID(),
		ImageUrl:          item.FashionImageUrl(),
		ImagePublicID:     item.FashionImagePublicID(),
		RetryCount:        item.FashionItem.ProcessingRetryCount,
		ProcessingVersion: item.FashionItem.ProcessingVersion,
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
