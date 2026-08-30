package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/worker/shared"
	"smart-wardrobe-be/internal/modules/fashion/domain/constants/fashionitemtype"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"

	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/colorutils"
	"smart-wardrobe-be/internal/pkg/utils/stringutils"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
		// The raw provider error must not reach the SSE stream; it is already
		// logged above and this message is the only user-facing detail.
		Error: processingFailureReasonMessage,
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
