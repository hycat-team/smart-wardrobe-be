package fashion_item

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"

	"github.com/google/uuid"
)

type FashionItemUseCase struct {
	fashionItemRepo repositories.IFashionItemRepository
}

func NewFashionItemUseCase(fashionItemRepo repositories.IFashionItemRepository) uc_interfaces.IFashionItemUseCase {
	return &FashionItemUseCase{fashionItemRepo: fashionItemRepo}
}

func mapToFashionItemRes(fi *entities.FashionItem) *dto.FashionItemContractDTO {
	if fi == nil {
		return nil
	}
	var category *dto.CategoryBriefRes
	if fi.Category != nil {
		category = &dto.CategoryBriefRes{
			ID:   fi.Category.ID,
			Name: fi.Category.Name,
			Slug: fi.Category.Slug,
		}
	}
	return &dto.FashionItemContractDTO{
		ID:              fi.ID,
		Category:        category,
		ImageUrl:        fi.ImageUrl,
		ImagePublicID:   fi.ImagePublicID,
		Color:           fi.Color,
		ColorHex:        fi.ColorHex,
		ColorHue:        fi.ColorHue,
		ColorSaturation: fi.ColorSaturation,
		ColorLightness:  fi.ColorLightness,
		Style:           fi.Style,
		Material:        fi.Material,
		Pattern:         fi.Pattern,
		Fit:             fi.Fit,
		Seasonality:     fi.Seasonality,
		Description:     fi.Description,
		CreatedAt:       fi.CreatedAt,
		UpdatedAt:       fi.UpdatedAt,
	}
}

func (uc *FashionItemUseCase) GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemContractDTO, error) {
	item, err := uc.fashionItemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToFashionItemRes(item), nil
}

func (uc *FashionItemUseCase) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemContractDTO, error) {
	items, err := uc.fashionItemRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.FashionItemContractDTO, len(items))
	for i, item := range items {
		res[i] = mapToFashionItemRes(item)
	}
	return res, nil
}

func (uc *FashionItemUseCase) CreateFashionItem(ctx context.Context, input dto.CreateFashionItemInput) error {
	fi := &entities.FashionItem{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: input.ID},
		},
		ImageUrl:          input.ImageUrl,
		ImagePublicID:     input.ImagePublicID,
		CategoryID:        input.CategoryID,
		ProcessingVersion: input.ProcessingVersion,
	}
	return uc.fashionItemRepo.Create(ctx, fi)
}

func (uc *FashionItemUseCase) BulkCreateFashionItems(ctx context.Context, inputs []dto.CreateFashionItemInput) error {
	if len(inputs) == 0 {
		return nil
	}
	for _, input := range inputs {
		if err := uc.CreateFashionItem(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

func (uc *FashionItemUseCase) UpdateFashionItemCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error {
	return uc.fashionItemRepo.UpdateCategory(ctx, fashionItemID, categoryID)
}

func (uc *FashionItemUseCase) GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error) {
	if len(fashionItemIDs) == 0 {
		return nil, nil
	}
	items, err := uc.fashionItemRepo.GetByIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	result := make([]dto.FashionProcessingInfoDTO, 0, len(items))
	for _, item := range items {
		result = append(result, dto.FashionProcessingInfoDTO{
			ID:                      item.ID,
			ProcessingVersion:       item.ProcessingVersion,
			ProcessingStartedAt:     item.ProcessingStartedAt,
			LastProcessingAttemptAt: item.LastProcessingAttemptAt,
		})
	}
	return result, nil
}

func (uc *FashionItemUseCase) SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error) {
	return uc.fashionItemRepo.SearchFashionItemIDs(ctx, query, categorySlug)
}

func (uc *FashionItemUseCase) GetFashionItemIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	return uc.fashionItemRepo.GetFashionItemIDsByCategory(ctx, categoryID)
}

func (uc *FashionItemUseCase) GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error) {
	return uc.fashionItemRepo.GetCategoryIDsByFashionItemIDs(ctx, fashionItemIDs)
}

func (uc *FashionItemUseCase) ReassignCategoryForCatalogItems(ctx context.Context, fashionItemIDs []uuid.UUID, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error {
	return uc.fashionItemRepo.ReassignCategory(ctx, fashionItemIDs, fromCategoryID, toCategoryID)
}

func (uc *FashionItemUseCase) ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error {
	return uc.fashionItemRepo.ResetProcessingState(ctx, fashionItemID)
}
