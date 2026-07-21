package item

import (
	"context"
	"time"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/constants/fashionitemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const manualRetryCooldown = 15 * time.Second
const processingFailureMessage = "Hệ thống chưa thể phân tích trang phục lúc này. Vui lòng thử lại sau."

func (uc *WardrobeItemUseCase) CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error) {
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

	err = uc.wardrobeRepo.BulkCreate(ctx, clonedItems)
	if err != nil {
		return nil, err
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
		_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)
	}

	fashionIDs := make([]uuid.UUID, 0, len(clonedItems))
	for _, item := range clonedItems {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)

	fashionItems, err := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
	if err != nil {
		return nil, err
	}
	fashionMap := sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemRes) uuid.UUID { return f.ID })

	resList := make([]*dto.WardrobeItemRes, quantity)
	for i := range quantity {
		var fi *fashion_dto.FashionItemRes
		if clonedItems[i].FashionItemID != nil {
			fi = fashionMap[*clonedItems[i].FashionItemID]
		}
		resList[i] = mapper.MapToWardrobeItemRes(clonedItems[i], fi)
		resList[i].IsLocked = false
	}

	return resList, nil
}

func (uc *WardrobeItemUseCase) ManualClassify(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.ManualClassifyReq) (*dto.WardrobeItemRes, error) {
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

	if item.Status == wardrobestatus.Sold {
		return nil, wardrobeerrors.ErrManualClassifySoldItem()
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	if shared.IsItemLocked(items, itemID, subOverview.MaxWardrobeItems) {
		return nil, wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
	}

	category, err := uc.fashionContract.GetCategory(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound()
	}

	var existingFI *fashion_dto.FashionItemRes
	if item.FashionItemID != nil {
		existingFI, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}

	item.Price = input.Price
	item.Status = wardrobestatus.Processing

	err = uc.wardrobeRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	imageURL := ""
	imagePublicID := ""
	processingVersion := 1
	if existingFI != nil {
		imageURL = existingFI.ImageUrl
		imagePublicID = existingFI.ImagePublicID
	}
	fashionJob := fashion_dto.FashionAnalyzeJobDTO{
		FashionItemID:     item.FashionItemID,
		ItemID:            item.ID,
		ItemType:          fashionitemtype.Wardrobe,
		UserID:            userID,
		ImageUrl:          imageURL,
		ImagePublicID:     imagePublicID,
		CategoryID:        &category.ID,
		ProcessingVersion: processingVersion,
		SkipVision:        true,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicFashionInit, fashionJob)

	var fi *fashion_dto.FashionItemRes
	if item.FashionItemID != nil {
		fi = existingFI
	}
	return mapper.MapToWardrobeItemRes(item, fi), nil
}

func (uc *WardrobeItemUseCase) RetryWardrobeAnalysis(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.WardrobeItemRes, error) {
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
	var existingFI *fashion_dto.FashionItemRes
	if item.FashionItemID != nil {
		existingFI, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}

	now := time.Now().UTC()
	item, claimed, err := uc.wardrobeRepo.ClaimManualAnalysisRetry(ctx, userID, itemID, now)
	if err != nil {
		return nil, err
	}
	if !claimed || item == nil {
		return nil, wardrobeerrors.ErrRetryWardrobeAnalysisInProgress()
	}

	categoryID := (*uuid.UUID)(nil)
	imageURL := ""
	imagePublicID := ""
	processingVersion := 0
	if existingFI != nil {
		if existingFI.Category != nil {
			categoryID = &existingFI.Category.ID
		}
		imageURL = existingFI.ImageUrl
		imagePublicID = existingFI.ImagePublicID
	}
	job := fashion_dto.FashionAnalyzeJobDTO{
		FashionItemID:     item.FashionItemID,
		ItemID:            item.ID,
		ItemType:          fashionitemtype.Wardrobe,
		UserID:            item.UserID,
		CategoryID:        categoryID,
		ImageUrl:          imageURL,
		ImagePublicID:     imagePublicID,
		ProcessingVersion: processingVersion,
		RetryCount:        0,
	}
	if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicFashionInit, job); err != nil {
		uc.logger.Error("[RetryWardrobeAnalysis] Event publishing failed", zap.Error(err))
		_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, item.ID, processingVersion, processingFailureMessage, nil)
		return nil, err
	}

	var fi *fashion_dto.FashionItemRes
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

	for _, id := range ids {
		item := itemMap[id]
		item.IsDeleted = true
		if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
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
		_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeDeleted, payload)
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

	for _, item := range lockedItems {
		item.IsDeleted = true
		if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
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
		_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeDeleted, payload)
	}

	return nil
}

func (uc *WardrobeItemUseCase) BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemRes, error) {
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

	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i, item := range newItems {
		resList[i] = mapper.MapToWardrobeItemRes(item, nil)
		resList[i].IsLocked = false
	}

	return resList, nil
}
