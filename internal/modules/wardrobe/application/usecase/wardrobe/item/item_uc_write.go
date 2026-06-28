package item

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/colorutils"

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
	if fashionItemID == uuid.Nil {
		fashionItemID = original.ID
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
		payload := dto.WardrobeEventPayload{
			ItemID: cloned.ID,
			UserID: cloned.UserID,
			Action: "created",
		}
		_ = uc.eventPublisher.Publish(ctx, "wardrobe.event.created", payload)
	}

	resList := make([]*dto.WardrobeItemRes, quantity)
	for i := range quantity {
		clonedItems[i].FashionItem = original.FashionItem
		resList[i] = mapper.MapToWardrobeItemRes(clonedItems[i])
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

	category, err := uc.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound()
	}

	freeForm := fmt.Sprintf("Món đồ thời trang %s màu %s phong cách %s được làm từ %s với họa tiết %s, dáng %s thích hợp mặc vào %s.",
		category.Name, input.Color, input.Style, input.Material, input.Pattern, input.Fit, input.Seasonality)

	richTextContext := shared.BuildRichTextContext(category.Name, input.Color, input.Style, input.Material, input.Pattern, input.Fit, input.Seasonality, freeForm)

	embedding, err := shared.GenerateItemEmbedding(ctx, uc.aiService, richTextContext)
	if err != nil {
		return nil, wardrobeerrors.ErrProcessFashionTextFailed()
	}

	if item.FashionItem == nil {
		item.FashionItem = &entities.FashionItem{}
	}
	item.FashionItem.CategoryID = &category.ID
	item.FashionItem.Category = category
	item.FashionItem.Color = &input.Color
	item.FashionItem.Style = &input.Style
	item.FashionItem.Material = &input.Material
	item.FashionItem.Pattern = &input.Pattern
	item.FashionItem.Fit = &input.Fit
	item.FashionItem.Seasonality = &input.Seasonality
	item.FashionItem.Description = &freeForm
	item.Price = input.Price
	item.FashionItem.Embedding = entities.Vector(embedding)
	item.Status = wardrobestatus.InWardrobe
	item.FashionItem.ReviewReason = nil
	item.FashionItem.ProcessingErrorReason = nil
	item.FashionItem.ProcessingRetryCount = 0
	item.FashionItem.ProcessingStartedAt = nil
	item.FashionItem.LastProcessingAttemptAt = nil

	if h, s, l, hex, ok := colorutils.ResolveFashionColor(input.Color, ""); ok {
		item.FashionItem.ColorHex = &hex
		item.FashionItem.ColorHue = &h
		item.FashionItem.ColorSaturation = &s
		item.FashionItem.ColorLightness = &l
	}

	err = uc.wardrobeRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	payload := dto.WardrobeEventPayload{
		ItemID: item.ID,
		UserID: item.UserID,
		Action: eventconstants.ActionCreated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)

	return mapper.MapToWardrobeItemRes(item), nil
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
	if item.FashionItem != nil && item.FashionItem.LastProcessingAttemptAt != nil && time.Since(item.FashionItem.LastProcessingAttemptAt.UTC()) < manualRetryCooldown {
		return nil, wardrobeerrors.ErrRetryWardrobeAnalysisCooldown()
	}

	now := time.Now().UTC()
	item, claimed, err := uc.wardrobeRepo.ClaimManualAnalysisRetry(ctx, userID, itemID, now)
	if err != nil {
		return nil, err
	}
	if !claimed || item == nil {
		return nil, wardrobeerrors.ErrRetryWardrobeAnalysisInProgress()
	}

	job := dto.WardrobeBatchUploadJobDTO{
		ItemID:            item.ID,
		UserID:            item.UserID,
		CategoryID:        item.FashionCategoryID(),
		ImageUrl:          item.FashionImageUrl(),
		ImagePublicID:     item.FashionImagePublicID(),
		ProcessingVersion: item.FashionItem.ProcessingVersion,
	}
	if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeBatchUpload, job); err != nil {
		uc.logger.Error("[RetryWardrobeAnalysis] Event publishing failed", zap.Error(err))
		_, _ = uc.wardrobeRepo.MarkProcessingFailed(ctx, item.ID, item.FashionItem.ProcessingVersion, processingFailureMessage, nil)
		return nil, err
	}

	res := mapper.MapToWardrobeItemRes(item)
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

		payload := dto.WardrobeEventPayload{
			ItemID: item.ID,
			UserID: item.UserID,
			Action: eventconstants.ActionDeleted,
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

		payload := dto.WardrobeEventPayload{
			ItemID: item.ID,
			UserID: item.UserID,
			Action: eventconstants.ActionDeleted,
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

	newItems := make([]*entities.WardrobeItem, len(input.Items))
	for i, itemReq := range input.Items {
		itemID := uuid.New()
		fashionItemID, err := uc.fashionContract.CreateFashionItem(ctx, userID, itemID, "wardrobe", itemReq.CategoryID, itemReq.ImageUrl, itemReq.ImagePublicID)
		if err != nil {
			return nil, err
		}
		newItems[i] = &entities.WardrobeItem{
			AuditableEntity: entities.AuditableEntity{
				BaseEntity: entities.BaseEntity{
					ID: itemID,
				},
			},
			UserID:        userID,
			Status:        wardrobestatus.Processing,
			ItemType:      itemType,
			FashionItemID: fashionItemID,
		}
	}

	if err := uc.wardrobeRepo.BulkCreate(ctx, newItems); err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i, item := range newItems {
		job := dto.WardrobeBatchUploadJobDTO{
			ItemID:            item.ID,
			UserID:            userID,
			CategoryID:        item.FashionCategoryID(),
			ImageUrl:          item.FashionImageUrl(),
			ImagePublicID:     item.FashionImagePublicID(),
			ProcessingVersion: item.FashionItem.ProcessingVersion,
		}

		err := uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeBatchUpload, job)
		if err != nil {
			uc.logger.Error("[WardrobeBatchUploadUseCase] Event publishing failed", zap.Error(err))
			item.Status = wardrobestatus.Failed
			reason := processingFailureMessage
			item.FashionItem.ProcessingErrorReason = &reason
			_ = uc.wardrobeRepo.Update(ctx, item)
		}

		resList[i] = mapper.MapToWardrobeItemRes(item)
		resList[i].IsLocked = false
	}

	return resList, nil
}
