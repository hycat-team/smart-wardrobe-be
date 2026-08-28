package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/worker/shared"
	"smart-wardrobe-be/internal/modules/fashion/domain/constants/fashionitemtype"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/colorutils"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"
	"smart-wardrobe-be/internal/pkg/utils/stringutils"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	processingFailureReasonMessage = "Hệ thống chưa thể phân tích trang phục lúc này. Vui lòng thử lại sau."
	autoRetryExceededMessage       = "Đã vượt quá số lần thử xử lý tự động."
)

// FashionWorkerUseCase orchestrates asynchronous wardrobe item processing and recovery workflows.
type FashionWorkerUseCase struct {
	cfg              *config.Config
	logger           logger.Interface
	wardrobeContract wardrobe_contract.IWardrobeContract
	categoryCache    *VisionCategoryCache
	mediaService     media.IMediaService
	aiService        aiexecutor_contract.IAIOrchestrator
	userSubContract  contract.IUserSubscriptionContract
	eventPublisher   event.IEventPublisher
	fashionItemRepo  repositories.IFashionItemRepository
	uow              shared_repos.IUnitOfWork
}

// NewFashionWorkerUseCase creates the worker use case responsible for AI item processing jobs.
func NewFashionWorkerUseCase(
	cfg *config.Config,
	logger logger.Interface,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	categoryCache *VisionCategoryCache,
	mediaService media.IMediaService,
	aiService aiexecutor_contract.IAIOrchestrator,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
	fashionItemRepo repositories.IFashionItemRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IFashionWorkerUseCase {
	return &FashionWorkerUseCase{
		cfg:              cfg,
		logger:           logger,
		wardrobeContract: wardrobeContract,
		categoryCache:    categoryCache,
		mediaService:     mediaService,
		aiService:        aiService,
		userSubContract:  userSubContract,
		eventPublisher:   eventPublisher,
		fashionItemRepo:  fashionItemRepo,
		uow:              uow,
	}
}

// ProcessBackgroundBatchUploadJob analyzes a processing wardrobe item and finalizes its AI metadata.
func (uc *FashionWorkerUseCase) ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.FashionAnalyzeJobDTO, run *workerlog.Run) error {
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
func (uc *FashionWorkerUseCase) CleanupFailedItems(ctx context.Context, run *workerlog.Run) error {
	limit := 100
	totalDeleted := 0
	publishedDeleteEventCount := 0

	for {
		items, err := uc.wardrobeContract.GetFailedItemsForCleanup(ctx, limit)
		if err != nil {
			return fmt.Errorf("failed to fetch items for cleanup: %w", err)
		}

		if len(items) == 0 {
			break
		}

		fashionIDs := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			if item.FashionItemID != nil {
				fashionIDs = append(fashionIDs, *item.FashionItemID)
			}
		}
		fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
		fashionItemMap := map[uuid.UUID]*entities.FashionItem{}
		if len(fashionIDs) > 0 {
			if fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, fashionIDs); err == nil {
				fashionItemMap = sliceutils.IndexBy(fashionItems, func(fi *entities.FashionItem) uuid.UUID { return fi.ID })
			}
		}
		for _, item := range items {
			run.AddTotal(1)
			imagePublicID := ""
			if item.FashionItemID != nil {
				if fi, ok := fashionItemMap[*item.FashionItemID]; ok && fi != nil {
					imagePublicID = fi.ImagePublicID
				}
			}
			if imagePublicID != "" {
				if err := uc.mediaService.DeleteImage(ctx, imagePublicID); err != nil {
					run.ChildWarn(uc.logger, "Failed to delete image from Cloudinary during failed item cleanup",
						zap.String("itemId", item.ID.String()),
						zap.String("publicId", imagePublicID),
						zap.Error(err),
					)
				}
			}

			err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
				payload := wardrobeDto.WardrobeEventPayload{
					ItemID: item.ID,
					UserID: item.UserID,
					Action: eventconstants.ActionDeleted,
				}
				return uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeDeleted, payload)
			})
			if err != nil {
				run.ChildError(uc.logger, "Failed to delete item from database during failed item cleanup",
					zap.String("itemId", item.ID.String()),
					zap.Error(err),
				)
				return fmt.Errorf("failed to delete item %s: %w", item.ID.String(), err)
			}
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
func (uc *FashionWorkerUseCase) RecoverStaleProcessingItems(ctx context.Context, run *workerlog.Run) error {
	staleBefore := time.Now().UTC().Add(-time.Duration(uc.cfg.Wardrobe.StaleMinutes) * time.Minute)
	items, err := uc.wardrobeContract.GetStaleProcessingItems(ctx, staleBefore, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch stale processing items: %w", err)
	}
	run.AddTotal(len(items))
	republishedCount := 0
	markedFailedCount := 0

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionItemMap := map[uuid.UUID]*entities.FashionItem{}
	if len(fashionIDs) > 0 {
		if fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, fashionIDs); err == nil {
			fashionItemMap = sliceutils.IndexBy(fashionItems, func(fi *entities.FashionItem) uuid.UUID { return fi.ID })
		}
	}
	for _, item := range items {
		if item.FashionItemID == nil {
			run.AddSkipped(1)
			continue
		}
		fi, ok := fashionItemMap[*item.FashionItemID]
		if !ok || fi == nil {
			run.AddSkipped(1)
			continue
		}
		if fi.ProcessingRetryCount >= uc.cfg.Wardrobe.MaxRetryCount {
			payload := dto.FashionProcessingExhaustedPayload{
				WardrobeItemID: item.ID,
				FashionItemID:  fi.ID,
				Reason:         autoRetryExceededMessage,
				UserID:         item.UserID,
			}
			err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
				if _, err := uc.fashionItemRepo.MarkProcessingFailed(txCtx, fi.ID, fi.ProcessingVersion, autoRetryExceededMessage, nil); err != nil {
					return err
				}
				return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionProcessingExhausted, payload)
			})
			if err == nil {
				markedFailedCount++
			}
			run.ChildInfo(uc.logger, "Stale processing item marked failed after retries exhausted",
				zap.String("itemId", item.ID.String()),
				zap.Int("processingVersion", fi.ProcessingVersion),
			)
			continue
		}

		if err := uc.fashionItemRepo.IncrementProcessingRetryCount(ctx, fi.ID); err != nil {
			run.ChildError(uc.logger, "Failed to increment retry count for stale processing item",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			continue
		}
		if err := uc.publishProcessingRetry(ctx, item.ID, &fi.ID, item.UserID); err != nil {
			run.ChildError(uc.logger, "Failed to republish stale processing item",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			continue
		}
		republishedCount++
		run.AddSuccess(1)
	}

	run.AddSummaryFields(
		zap.Int("staleItemsFoundCount", len(items)),
		zap.Int("republishedCount", republishedCount),
		zap.Int("markedFailedCount", markedFailedCount),
	)
	return nil
}

func (uc *FashionWorkerUseCase) handleJobFailure(ctx context.Context, job dto.FashionAnalyzeJobDTO, err error, run *workerlog.Run) {
	topic := eventconstants.TopicFashionAnalyze

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

		if republishErr := uc.uow.Execute(context.Background(), func(txCtx context.Context) error {
			return uc.eventPublisher.Publish(txCtx, topic, job)
		}); republishErr != nil {
			run.ChildError(uc.logger, "Failed to republish analyze item retry job",
				zap.String("itemId", job.ItemID.String()),
				zap.Int("retryCount", job.RetryCount),
				zap.Error(republishErr),
			)
		}
		return
	}

	run.ChildError(uc.logger, "Item analyze failed permanently",
		zap.String("itemId", job.ItemID.String()),
		zap.Int("retryCount", job.RetryCount),
		zap.Error(err),
	)
	payload := dto.FashionAnalyzedEventPayload{
		TaskID:        job.TaskID,
		ItemID:        job.ItemID,
		ItemType:      job.ItemType,
		FashionItemID: job.FashionItemID,
		Status:        analyzetaskstatus.Failed,
		Total:         job.Total,
		Index:         job.Index,
		Error:         err.Error(),
	}
	_ = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if job.FashionItemID != nil {
			if _, err := uc.fashionItemRepo.MarkProcessingFailed(txCtx, *job.FashionItemID, job.ProcessingVersion, processingFailureReasonMessage, nil); err != nil {
				return err
			}
		}
		failedTopic := eventconstants.TopicFashionFailedWardrobe
		if job.ItemType == "brand" {
			failedTopic = eventconstants.TopicFashionFailedBrand
		}
		return uc.eventPublisher.Publish(txCtx, failedTopic, payload)
	})
}

func (uc *FashionWorkerUseCase) loadProcessableItem(ctx context.Context, job dto.FashionAnalyzeJobDTO, run *workerlog.Run) (any, error) {
	if job.FashionItemID == nil {
		run.ChildWarn(uc.logger, "Skipped analyze job because fashionItemId is nil",
			zap.String("itemId", job.ItemID.String()),
			zap.Int("processingVersion", job.ProcessingVersion),
		)
		return nil, nil
	}

	fi, err := uc.fashionItemRepo.GetByID(ctx, *job.FashionItemID)
	if err != nil {
		return nil, err
	}
	if fi == nil || fi.ProcessingVersion != job.ProcessingVersion {
		run.ChildWarn(uc.logger, "Skipped analyze job because item not found or processing version mismatch",
			zap.String("itemId", job.ItemID.String()),
			zap.Int("processingVersion", job.ProcessingVersion),
		)
		return nil, nil
	}
	return fi, nil
}

func (uc *FashionWorkerUseCase) analyzeWardrobeImage(
	ctx context.Context,
	job dto.FashionAnalyzeJobDTO,
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

func (uc *FashionWorkerUseCase) markNeedsReview(ctx context.Context, job dto.FashionAnalyzeJobDTO, reviewReason string) {
	if reviewReason == "" {
		reviewReason = "uncertain_category"
	}
	var fashionBrief *dto.FashionItemBriefRes
	if job.FashionItemID != nil {
		if fi, _ := uc.fashionItemRepo.GetByID(ctx, *job.FashionItemID); fi != nil {
			fashionBrief = &dto.FashionItemBriefRes{
				ID:       fi.ID,
				ImageUrl: fi.ImageUrl,
			}
		}
	}

	payload := dto.FashionAnalyzedEventPayload{
		TaskID:        job.TaskID,
		ItemID:        job.ItemID,
		ItemType:      job.ItemType,
		FashionItemID: job.FashionItemID,
		Status:        analyzetaskstatus.NeedsReview,
		Total:         job.Total,
		Index:         job.Index,
		FashionBrief:  fashionBrief,
	}
	_ = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if job.FashionItemID != nil {
			if _, err := uc.fashionItemRepo.MarkProcessingNeedsReview(txCtx, *job.FashionItemID, job.ProcessingVersion, reviewReason); err != nil {
				return err
			}
		}
		analyzedTopic := eventconstants.TopicFashionAnalyzedWardrobe
		if job.ItemType == "brand" {
			analyzedTopic = eventconstants.TopicFashionAnalyzedBrand
		}
		return uc.eventPublisher.Publish(txCtx, analyzedTopic, payload)
	})
}

func (uc *FashionWorkerUseCase) buildCompletedItemUpdate(
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

func (uc *FashionWorkerUseCase) completeProcessedItem(ctx context.Context, job dto.FashionAnalyzeJobDTO, updateMap map[string]any) error {
	if job.FashionItemID == nil {
		return nil
	}

	var fashionBrief *dto.FashionItemBriefRes
	var payload dto.FashionAnalyzedEventPayload
	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		completed, err := uc.fashionItemRepo.CompleteProcessingSuccess(txCtx, *job.FashionItemID, job.ProcessingVersion, updateMap)
		if err != nil {
			return err
		}
		if !completed {
			return nil
		}

		fi, err := uc.fashionItemRepo.GetByID(txCtx, *job.FashionItemID)
		if err != nil || fi == nil {
			return err
		}

		var categoryBrief *dto.CategoryBriefRes
		if fi.CategoryID != nil {
			if snapshot, err := uc.categoryCache.Get(txCtx); err == nil {
				if name, ok := snapshot.CategoryNameMap[*fi.CategoryID]; ok {
					slug := ""
					for s, id := range snapshot.CategoryMap {
						if id == *fi.CategoryID {
							slug = s
							break
						}
					}
					categoryBrief = &dto.CategoryBriefRes{
						ID:   *fi.CategoryID,
						Name: name,
						Slug: slug,
					}
				}
			}
		}

		fashionBrief = &dto.FashionItemBriefRes{
			ID:       fi.ID,
			Category: categoryBrief,
			ImageUrl: fi.ImageUrl,
			Color:    fi.Color,
			ColorHex: fi.ColorHex,
			Style:    fi.Style,
		}

		payload = dto.FashionAnalyzedEventPayload{
			TaskID:        job.TaskID,
			ItemID:        job.ItemID,
			ItemType:      job.ItemType,
			FashionItemID: job.FashionItemID,
			Status:        analyzetaskstatus.Completed,
			Total:         job.Total,
			Index:         job.Index,
			FashionBrief:  fashionBrief,
		}

		analyzedTopic := eventconstants.TopicFashionAnalyzedWardrobe
		if job.ItemType == "brand" {
			analyzedTopic = eventconstants.TopicFashionAnalyzedBrand
		}
		return uc.eventPublisher.Publish(txCtx, analyzedTopic, payload)
	})
	// If CompleteProcessingSuccess returned false (version mismatch), no publish was done; return nil
	if err != nil {
		return err
	}
	return nil
}

func (uc *FashionWorkerUseCase) publishProcessingRetry(ctx context.Context, itemID uuid.UUID, fashionItemID *uuid.UUID, userID uuid.UUID) error {
	categoryID := (*uuid.UUID)(nil)
	imageURL := ""
	imagePublicID := ""
	processingVersion := 0
	if fashionItemID != nil {
		if fi, err := uc.fashionItemRepo.GetByID(ctx, *fashionItemID); err == nil && fi != nil {
			categoryID = fi.CategoryID
			imageURL = fi.ImageUrl
			imagePublicID = fi.ImagePublicID
			processingVersion = fi.ProcessingVersion
		}
	}
	job := dto.FashionAnalyzeJobDTO{
		FashionItemID:     fashionItemID,
		ItemID:            itemID,
		ItemType:          fashionitemtype.Wardrobe,
		UserID:            userID,
		CategoryID:        categoryID,
		ImageUrl:          imageURL,
		ImagePublicID:     imagePublicID,
		RetryCount:        0,
		ProcessingVersion: processingVersion,
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionAnalyze, job)
	})
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
