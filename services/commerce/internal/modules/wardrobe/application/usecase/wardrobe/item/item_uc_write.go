package item

import (
	"context"
	"time"

	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/analysisreason"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/fashionitemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const manualRetryCooldown = 15 * time.Second

func (uc *WardrobeItemUseCase) CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemBriefRes, error) {
	if quantity < 1 || quantity > 5 {
		return nil, wardrobeerrors.ErrInvalidCloneQuantity()
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if int(currentCount)+quantity > subOverview.MaxWardrobeItems {
		return nil, wardrobeerrors.ErrWardrobeLimitExceededForClone(int(currentCount), subOverview.MaxWardrobeItems, quantity)
	}

	original, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, wardrobeerrors.ErrOriginalItemToCloneNotFound()
	}

	if original.UserID != userID {
		return nil, wardrobeerrors.ErrCloneOtherUserItemForbidden()
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	if shared.IsItemLocked(items, id, subOverview.MaxWardrobeItems) {
		return nil, wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
	}

	if original.Status == wardrobestatus.Sold {
		return nil, wardrobeerrors.ErrCloneSoldItem()
	}

	clonedItems := make([]*entities.WardrobeItem, quantity)
	fashionItemID := original.FashionItemID
	if fashionItemID == nil {
		fashionItemID = &original.ID
	}
	for i := range quantity {
		clonedItems[i] = &entities.WardrobeItem{
			UserID:        userID,
			FashionItemID: fashionItemID,
			Status:        wardrobestatus.InWardrobe,
			ItemType:      itemtype.UserItem,
		}
	}

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.wardrobeRepo.BulkCreate(txCtx, clonedItems); err != nil {
			return err
		}
		for _, cloned := range clonedItems {
			var fashionItemID uuid.UUID
			if cloned.FashionItemID != nil {
				fashionItemID = *cloned.FashionItemID
			}
			payload := dto.WardrobeEventPayload{
				ItemID:        cloned.ID,
				UserID:        cloned.UserID,
				FashionItemID: fashionItemID,
				ItemType:      int(cloned.ItemType),
				Action:        eventconstants.ActionCreated,
			}
			if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeCreated, payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	fashionIDs := make([]uuid.UUID, 0, len(clonedItems))
	for _, item := range clonedItems {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)

	fashionBriefs, err := uc.fashionContract.ListFashionItemsBrief(ctx, fashionIDs)
	if err != nil {
		return nil, err
	}
	fashionBriefMap := make(map[uuid.UUID]*fashion_dto.FashionItemBriefRes, len(fashionBriefs))
	for _, b := range fashionBriefs {
		fashionBriefMap[b.ID] = b
	}

	brandItemMap := map[uuid.UUID]*shared_dto.BrandItemBriefRes{}
	if len(fashionIDs) > 0 && uc.brandContract != nil {
		brandItems, err := uc.brandContract.GetBrandItemsByFashionItemIDs(ctx, fashionIDs)
		if err == nil {
			for _, bi := range brandItems {
				brandItemMap[bi.FashionItemID] = &shared_dto.BrandItemBriefRes{
					ID:          bi.ID,
					BrandID:     bi.BrandID,
					BrandName:   bi.BrandName,
					ProductCode: bi.ProductCode,
					Name:        bi.Name,
					Price:       bi.Price,
					ItemType:    string(bi.ItemType),
					Status:      string(bi.Status),
				}
			}
		}
	}

	resList := make([]*dto.WardrobeItemBriefRes, quantity)
	for i := range quantity {
		var fiBrief *fashion_dto.FashionItemBriefRes
		if clonedItems[i].FashionItemID != nil {
			fiBrief = fashionBriefMap[*clonedItems[i].FashionItemID]
		}
		var brandItem *shared_dto.BrandItemBriefRes
		if clonedItems[i].FashionItemID != nil {
			brandItem = brandItemMap[*clonedItems[i].FashionItemID]
		}
		resList[i] = mapper.MapToWardrobeItemBrief(clonedItems[i], fiBrief, brandItem)
		resList[i].IsLocked = false
	}

	return resList, nil
}

func (uc *WardrobeItemUseCase) UpdateWardrobeItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.UpdateWardrobeItemReq) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, wardrobeerrors.ErrItemNotFound()
	}
	if item.UserID != userID {
		return nil, wardrobeerrors.ErrUpdateItemForbidden()
	}

	if input.Price != nil {
		item.Price = input.Price
	}

	if input.CategoryID != nil {
		category, err := uc.categoryContract.GetCategory(ctx, *input.CategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, wardrobeerrors.ErrCategoryNotFound()
		}
		if item.FashionItemID != nil {
			if err := uc.fashionContract.UpdateFashionItemCategory(ctx, *item.FashionItemID, *input.CategoryID); err != nil {
				return nil, err
			}
		}
	}

	if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	var fi *fashion_dto.FashionItemContractDTO
	if item.FashionItemID != nil {
		fi, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	return mapper.MapToWardrobeItemRes(item, fi), nil
}

func (uc *WardrobeItemUseCase) RetryWardrobeAnalysis(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, categoryID *uuid.UUID) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, wardrobeerrors.ErrItemNotFound()
	}
	if item.UserID != userID {
		return nil, wardrobeerrors.ErrUpdateItemForbidden()
	}
	if item.Status != wardrobestatus.Failed && item.Status != wardrobestatus.NeedsReview {
		return nil, wardrobeerrors.ErrRetryWardrobeAnalysisForbidden()
	}
	var existingFI *fashion_dto.FashionItemContractDTO
	if item.FashionItemID != nil {
		existingFI, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	// Retry is futile for an unusable image (FR-033).
	if existingFI != nil && existingFI.ProcessingErrorReason != nil && analysisreason.IsInvalidImageReason(*existingFI.ProcessingErrorReason) {
		return nil, wardrobeerrors.ErrRetryInvalidImage()
	}

	// Resolve the fixed category used to re-analyze a needs_review item (FR-037).
	// A category from the request (validated as wearable) wins over the existing one.
	resolvedCategoryID := (*uuid.UUID)(nil)
	if existingFI != nil && existingFI.Category != nil {
		resolvedCategoryID = &existingFI.Category.ID
	}
	categoryFromRequest := false
	if categoryID != nil {
		wearable, err := uc.fashionContract.IsWearableCategory(ctx, *categoryID)
		if err != nil {
			return nil, err
		}
		if !wearable {
			return nil, wardrobeerrors.ErrRetryCategoryRequired()
		}
		resolvedCategoryID = categoryID
		categoryFromRequest = true
	}
	fixedCategory := item.Status == wardrobestatus.NeedsReview
	if fixedCategory && resolvedCategoryID == nil {
		return nil, wardrobeerrors.ErrRetryCategoryRequired()
	}

	now := time.Now().UTC()
	item, claimed, err := uc.wardrobeRepo.ClaimManualAnalysisRetry(ctx, userID, itemID, now)
	if err != nil {
		return nil, err
	}
	if !claimed || item == nil {
		return nil, wardrobeerrors.ErrRetryWardrobeAnalysisInProgress()
	}

	taskID := uuid.New()
	preserveCategory := fixedCategory || (existingFI != nil && existingFI.Category != nil)

	if categoryFromRequest && item.FashionItemID != nil && categoryID != nil {
		if err := uc.fashionContract.UpdateFashionItemCategory(ctx, *item.FashionItemID, *categoryID); err != nil {
			return nil, err
		}
	}

	if item.FashionItemID != nil {
		if err := uc.fashionContract.ResetProcessingState(ctx, *item.FashionItemID); err != nil {
			uc.logger.Error("[RetryWardrobeAnalysis] Failed to reset fashion processing state", zap.Error(err))
		}
	}

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		imageURL := ""
		imagePublicID := ""
		processingVersion := 0
		if existingFI != nil {
			imageURL = existingFI.ImageUrl
			imagePublicID = existingFI.ImagePublicID
		}
		job := fashion_dto.FashionAnalyzeJobDTO{
			FashionItemID:     item.FashionItemID,
			ItemID:            item.ID,
			ItemType:          fashionitemtype.Wardrobe,
			UserID:            item.UserID,
			CategoryID:        resolvedCategoryID,
			ImageUrl:          imageURL,
			ImagePublicID:     imagePublicID,
			ProcessingVersion: processingVersion,
			RetryCount:        0,
			TaskID:            taskID.String(),
			PreserveCategory:  preserveCategory,
			FixedCategory:     fixedCategory,
		}
		item.TaskID = &taskID
		if err := uc.wardrobeRepo.Update(txCtx, item); err != nil {
			return err
		}
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionInit, job)
	})
	if err != nil {
		uc.logger.Error("[RetryWardrobeAnalysis] Event publishing failed", zap.Error(err))
		if item.FashionItemID != nil {
			_ = uc.fashionContract.SetProcessingErrorReason(ctx, *item.FashionItemID, analysisreason.ErrorAnalysisTemporary)
		}
		_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, item.ID, 0, analysisreason.ErrorAnalysisTemporary, nil)
		return nil, err
	}

	var fi *fashion_dto.FashionItemContractDTO
	if item.FashionItemID != nil {
		fi = existingFI
	}
	res := mapper.MapToWardrobeItemRes(item, fi)
	res.IsLocked = false
	return res, nil
}

func (uc *WardrobeItemUseCase) DeleteWardrobeItemsBulk(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	items, err := uc.wardrobeRepo.GetByIDs(ctx, ids)
	if err != nil {
		return err
	}

	itemMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range items {
		itemMap[item.ID] = item
	}

	for _, id := range ids {
		item, exists := itemMap[id]
		if !exists || item.IsDeleted {
			return wardrobeerrors.ErrItemNotFound()
		}
		if item.UserID != userID {
			return wardrobeerrors.ErrUpdateItemForbidden()
		}
	}

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		for _, id := range ids {
			item := itemMap[id]
			item.IsDeleted = true
			if err := uc.wardrobeRepo.Update(txCtx, item); err != nil {
				return err
			}
			var fashionItemID uuid.UUID
			if item.FashionItemID != nil {
				fashionItemID = *item.FashionItemID
			}
			payload := dto.WardrobeEventPayload{
				ItemID:        item.ID,
				UserID:        item.UserID,
				FashionItemID: fashionItemID,
				ItemType:      int(item.ItemType),
				Action:        eventconstants.ActionDeleted,
			}
			if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeDeleted, payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (uc *WardrobeItemUseCase) DeleteLockedWardrobeItems(ctx context.Context, userID uuid.UUID) error {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return err
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return err
	}

	var lockedItems []*entities.WardrobeItem
	for idx, item := range items {
		if idx >= subOverview.MaxWardrobeItems {
			lockedItems = append(lockedItems, item)
		}
	}

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		for _, item := range lockedItems {
			item.IsDeleted = true
			if err := uc.wardrobeRepo.Update(txCtx, item); err != nil {
				return err
			}
			var fashionItemID uuid.UUID
			if item.FashionItemID != nil {
				fashionItemID = *item.FashionItemID
			}
			payload := dto.WardrobeEventPayload{
				ItemID:        item.ID,
				UserID:        item.UserID,
				FashionItemID: fashionItemID,
				ItemType:      int(item.ItemType),
				Action:        eventconstants.ActionDeleted,
			}
			if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeDeleted, payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (uc *WardrobeItemUseCase) BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemBriefRes, error) {
	if len(input.Items) == 0 {
		return nil, wardrobeerrors.ErrUploadImagesEmpty()
	}

	itemType := itemtype.SystemCatalogItem
	if currentRole == roleslug.User {
		itemType = itemtype.UserItem
		subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
		if err != nil {
			return nil, err
		}

		currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if int(currentCount)+len(input.Items) > subOverview.MaxWardrobeItems {
			return nil, wardrobeerrors.ErrWardrobeLimitExceededForUpload(int(currentCount), subOverview.MaxWardrobeItems, len(input.Items))
		}
	}

	taskID := uuid.New()
	newItems := make([]*entities.WardrobeItem, len(input.Items))
	fashionInputs := make([]fashion_dto.CreateFashionItemInput, len(input.Items))

	for i, reqItem := range input.Items {
		itemID := uuid.New()
		fashionItemID := uuid.New()
		newItems[i] = &entities.WardrobeItem{
			AuditableEntity: entities.AuditableEntity{
				BaseEntity: entities.BaseEntity{
					ID: itemID,
				},
			},
			UserID:        userID,
			Status:        wardrobestatus.Processing,
			ItemType:      itemType,
			FashionItemID: &fashionItemID,
			TaskID:        &taskID,
		}
		fashionInputs[i] = fashion_dto.CreateFashionItemInput{
			ID:                fashionItemID,
			ImageUrl:          reqItem.ImageUrl,
			ImagePublicID:     reqItem.ImagePublicID,
			CategoryID:        reqItem.CategoryID,
			ProcessingVersion: 1,
		}
	}

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.fashionContract.BulkCreateFashionItems(txCtx, fashionInputs); err != nil {
			return err
		}
		if err := uc.wardrobeRepo.BulkCreate(txCtx, newItems); err != nil {
			return err
		}
		for i, item := range newItems {
			job := fashion_dto.FashionAnalyzeJobDTO{
				FashionItemID:     item.FashionItemID,
				ItemID:            item.ID,
				ItemType:          fashionitemtype.Wardrobe,
				UserID:            userID,
				CategoryID:        input.Items[i].CategoryID,
				ImageUrl:          input.Items[i].ImageUrl,
				ImagePublicID:     input.Items[i].ImagePublicID,
				ProcessingVersion: 1,
				TaskID:            taskID.String(),
				Total:             len(newItems),
				Index:             i,
			}
			if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionAnalyze, job); err != nil {
				uc.logger.Error("[WardrobeBatchUploadUseCase] Event publishing failed", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemBriefRes, len(newItems))
	for i, item := range newItems {
		resList[i] = mapper.MapToWardrobeItemBrief(item, nil, nil)
		resList[i].IsLocked = false
	}

	return resList, nil
}
