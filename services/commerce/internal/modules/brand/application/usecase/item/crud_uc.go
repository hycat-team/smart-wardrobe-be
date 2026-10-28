package item

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/fashionitemtype"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

func (uc *BrandItemUseCase) GetBrandItemUploadSignature(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: fmt.Sprintf("brands/%s/items", brandID.String()),
	})
}

func (uc *BrandItemUseCase) CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error) {
	if input.ProductCode != nil && *input.ProductCode != "" {
		existing, err := uc.brandItemRepo.GetByProductCode(ctx, brandID, *input.ProductCode)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, branderrors.ErrProductCodeExists()
		}
	}

	brandItemID := uuid.New()
	fashionItemID := uuid.New()
	taskID := uuid.New()

	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID:        brandItemID,
				CreatedAt: time.Now().UTC(),
			},
			UpdatedAt: time.Now().UTC(),
		},
		BrandID:       brandID,
		FashionItemID: &fashionItemID,
		ProductCode:   input.ProductCode,
		Name:          input.Name,
		Description:   input.Description,
		Price:         input.Price,
		ItemType:      branditemtype.BrandItemType(strings.ToLower(strings.TrimSpace(input.ItemType))),
		Status:        branditemstatus.BrandItemStatus(strings.ToLower(strings.TrimSpace(input.Status))),
		TaskID:        &taskID,
	}
	if item.Status == "" {
		item.Status = branditemstatus.Draft
	}

	fashionInput := fashion_dto.CreateFashionItemInput{
		ID:                fashionItemID,
		ImageUrl:          input.ImageUrl,
		ImagePublicID:     input.ImagePublicID,
		CategoryID:        input.CategoryID,
		ProcessingVersion: 1,
	}

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.fashionContract.CreateFashionItem(txCtx, fashionInput); err != nil {
			return err
		}
		if err := uc.brandItemRepo.Create(txCtx, item); err != nil {
			return err
		}
		job := fashion_dto.FashionAnalyzeJobDTO{
			FashionItemID:     item.FashionItemID,
			ItemID:            brandItemID,
			ItemType:          fashionitemtype.Brand,
			UserID:            staffUserID,
			ImageUrl:          input.ImageUrl,
			ImagePublicID:     input.ImagePublicID,
			CategoryID:        input.CategoryID,
			ProcessingVersion: 1,
			TaskID:            taskID.String(),
			Total:             1,
			Index:             0,
		}
		if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionAnalyze, job); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	fi := &fashion_dto.FashionItemContractDTO{
		ID:            fashionItemID,
		ImageUrl:      input.ImageUrl,
		ImagePublicID: input.ImagePublicID,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	return mapper.MapToBrandItemRes(item, fi), nil
}

func (uc *BrandItemUseCase) GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) ([]*dto.BrandItemRes, error) {
	filter := repositories.BrandItemFilter{BrandID: brandID}
	if query.ItemType != nil && *query.ItemType != "" {
		itemType := branditemtype.BrandItemType(strings.ToLower(strings.TrimSpace(*query.ItemType)))
		filter.ItemType = &itemType
	}
	if query.Status != nil && *query.Status != "" {
		status := branditemstatus.BrandItemStatus(strings.ToLower(strings.TrimSpace(*query.Status)))
		filter.Status = &status
	}
	items, err := uc.brandItemRepo.GetByBrandIDFiltered(ctx, filter)
	if err != nil {
		return nil, err
	}
	return uc.mapSliceToRes(ctx, items), nil
}

func (uc *BrandItemUseCase) ConvertSampleToProduct(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	if item.ItemType == branditemtype.Product {
		return nil, branderrors.ErrBrandItemAlreadyProduct()
	}
	if item.ItemType != branditemtype.Sample {
		return nil, branderrors.ErrBrandItemNotSample()
	}
	if item.Status == branditemstatus.Failed {
		return nil, branderrors.ErrBrandItemStatusLocked()
	}
	if item.FashionItemID == nil {
		return nil, branderrors.ErrBrandItemNotReadyForProduct()
	}
	fi, err := uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	if err != nil {
		return nil, err
	}
	if fi == nil || fi.Category == nil {
		return nil, branderrors.ErrBrandItemNotReadyForProduct()
	}

	item.ItemType = branditemtype.Product
	item.UpdatedAt = time.Now().UTC()
	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) AssertTaskBelongsToBrand(ctx context.Context, brandID uuid.UUID, taskID uuid.UUID) error {
	item, err := uc.brandItemRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	if item == nil || item.BrandID != brandID {
		return branderrors.ErrBrandNotFound()
	}
	return nil
}

func (uc *BrandItemUseCase) GetBrandItemForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) GetBrandItemForUser(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.Status != branditemstatus.Active {
		return nil, branderrors.ErrBrandNotFound()
	}
	brand, err := uc.brandRepo.GetByID(ctx, item.BrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || !strings.EqualFold(string(brand.Status), string(brandstatus.Active)) {
		return nil, branderrors.ErrBrandNotFound()
	}
	if item.ItemType == branditemtype.Sample {
		if userID == uuid.Nil {
			return nil, branderrors.ErrBrandPortalForbidden()
		}
		hasAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, item.BrandID, benefitfeaturecode.SampleMixAccess)
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, branderrors.ErrBrandPortalForbidden()
		}
	}
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	statusInput := branditemstatus.BrandItemStatus(strings.ToLower(strings.TrimSpace(input.Status)))
	if item.Status == branditemstatus.Failed && statusInput != "" {
		return nil, branderrors.ErrBrandItemStatusLocked()
	}
	if statusInput != "" && statusInput != branditemstatus.Draft &&
		statusInput != branditemstatus.Active && statusInput != branditemstatus.Archived {
		return nil, branderrors.ErrBrandItemInvalidStatus()
	}
	if input.ProductCode != nil && (item.ProductCode == nil || *input.ProductCode != *item.ProductCode) {
		existing, err := uc.brandItemRepo.GetByProductCode(ctx, brandID, *input.ProductCode)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != itemID {
			return nil, branderrors.ErrProductCodeExists()
		}
	}
	item.ProductCode = input.ProductCode
	item.Name = input.Name
	item.Description = input.Description
	item.Price = input.Price
	if statusInput != "" {
		item.Status = statusInput
	}
	item.UpdatedAt = time.Now().UTC()

	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) UpdateBrandItemStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, status branditemstatus.BrandItemStatus) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	if item.Status == branditemstatus.Failed {
		return nil, branderrors.ErrBrandItemStatusLocked()
	}
	if status != branditemstatus.Draft && status != branditemstatus.Active && status != branditemstatus.Archived {
		return nil, branderrors.ErrBrandItemInvalidStatus()
	}
	item.Status = status
	item.UpdatedAt = time.Now().UTC()
	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	return uc.mapToRes(ctx, item), nil
}
