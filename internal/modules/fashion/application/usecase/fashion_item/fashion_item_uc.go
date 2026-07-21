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

func mapToFashionItemRes(fi *entities.FashionItem) *dto.FashionItemRes {
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
	return &dto.FashionItemRes{
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

func (uc *FashionItemUseCase) GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemRes, error) {
	item, err := uc.fashionItemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToFashionItemRes(item), nil
}

func (uc *FashionItemUseCase) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemRes, error) {
	items, err := uc.fashionItemRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.FashionItemRes, len(items))
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
