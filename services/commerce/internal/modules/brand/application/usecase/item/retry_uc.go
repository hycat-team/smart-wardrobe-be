package item

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/analysisreason"
	"smart-wardrobe-be/shared/application/constants/eventconstants"

	"github.com/google/uuid"
)

// RetryBrandItemAnalysis re-queues a failed brand item (or a draft awaiting
// review) for analysis and returns the item with a fresh taskId (feature 023,
// US4). The status transition is claimed atomically so concurrent retries cannot
// create duplicate sessions; the item's category is preserved.
func (uc *BrandItemUseCase) RetryBrandItemAnalysis(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, categoryID *uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}

	var fi *fashion_dto.FashionItemContractDTO
	if item.FashionItemID != nil {
		fi, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	// Retry is futile for an unusable image (FR-033).
	if fi != nil && fi.ProcessingErrorReason != nil && analysisreason.IsInvalidImageReason(*fi.ProcessingErrorReason) {
		return nil, branderrors.ErrBrandRetryInvalidImage()
	}

	awaitingReview := item.Status == branditemstatus.Draft && fi != nil && fi.ReviewReason != nil
	retryable := item.Status == branditemstatus.Failed || awaitingReview
	if !retryable {
		return nil, branderrors.ErrBrandRetryForbidden()
	}

	// Resolve the fixed category (FR-041): request category wins and must be
	// wearable; a draft awaiting review needs one to leave review.
	resolvedCategoryID := (*uuid.UUID)(nil)
	if fi != nil && fi.Category != nil {
		resolvedCategoryID = &fi.Category.ID
	}
	categoryFromRequest := false
	if categoryID != nil {
		wearable, err := uc.fashionContract.IsWearableCategory(ctx, *categoryID)
		if err != nil {
			return nil, err
		}
		if !wearable {
			return nil, branderrors.ErrBrandRetryCategoryRequired()
		}
		resolvedCategoryID = categoryID
		categoryFromRequest = true
	}
	fixedCategory := awaitingReview || categoryFromRequest
	if awaitingReview && resolvedCategoryID == nil {
		return nil, branderrors.ErrBrandRetryCategoryRequired()
	}

	taskID := uuid.New()
	now := time.Now().UTC()
	if categoryFromRequest && item.FashionItemID != nil && categoryID != nil {
		if err := uc.fashionContract.UpdateFashionItemCategory(ctx, *item.FashionItemID, *categoryID); err != nil {
			return nil, err
		}
	}
	imageURL, imagePublicID := "", ""
	if fi != nil {
		imageURL = fi.ImageUrl
		imagePublicID = fi.ImagePublicID
	}

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		claimed, ok, claimErr := uc.brandItemRepo.ClaimManualAnalysisRetry(txCtx, item.ID, item.TaskID, taskID, now)
		if claimErr != nil {
			return claimErr
		}
		if !ok || claimed == nil {
			return branderrors.ErrBrandRetryInProgress()
		}
		item = claimed

		if item.FashionItemID != nil {
			if resetErr := uc.fashionContract.ResetProcessingState(txCtx, *item.FashionItemID); resetErr != nil {
				return resetErr
			}
		}
		job := fashion_dto.FashionAnalyzeJobDTO{
			FashionItemID:     item.FashionItemID,
			ItemID:            item.ID,
			ItemType:          "brand",
			UserID:            staffUserID,
			CategoryID:        resolvedCategoryID,
			ImageUrl:          imageURL,
			ImagePublicID:     imagePublicID,
			ProcessingVersion: 1,
			TaskID:            taskID.String(),
			PreserveCategory:  true,
			FixedCategory:     fixedCategory,
		}
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionAnalyze, job)
	})
	if err != nil {
		return nil, err
	}

	return uc.mapToRes(ctx, item), nil
}
