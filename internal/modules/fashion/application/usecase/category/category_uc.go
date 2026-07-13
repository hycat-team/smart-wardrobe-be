package category

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"

	"github.com/google/uuid"
)

type CategoryUseCase struct {
	categoryRepo repositories.ICategoryRepository
}

func NewCategoryUseCase(categoryRepo repositories.ICategoryRepository) uc_interfaces.ICategoryUseCase {
	return &CategoryUseCase{categoryRepo: categoryRepo}
}

func mapToCategoryRes(cat *entities.Category) *dto.CategoryRes {
	if cat == nil {
		return nil
	}
	return &dto.CategoryRes{
		ID:        cat.ID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		SortOrder: cat.SortOrder,
	}
}

func (uc *CategoryUseCase) GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error) {
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) ListCategories(ctx context.Context, ids []uuid.UUID) ([]*dto.CategoryRes, error) {
	items, err := uc.categoryRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.CategoryRes, len(items))
	for i, item := range items {
		res[i] = mapToCategoryRes(item)
	}
	return res, nil
}

func (uc *CategoryUseCase) GetAllCategories(ctx context.Context) ([]*dto.CategoryRes, error) {
	items, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.CategoryRes, len(items))
	for i, item := range items {
		res[i] = mapToCategoryRes(item)
	}
	return res, nil
}

func (uc *CategoryUseCase) GetCategoryBySlug(ctx context.Context, slug string) (*dto.CategoryRes, error) {
	cat, err := uc.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) GetCategoryByName(ctx context.Context, name string) (*dto.CategoryRes, error) {
	cat, err := uc.categoryRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryRes, error) {
	cat := &entities.Category{
		Name:      input.Name,
		Slug:      input.Slug,
		SortOrder: input.SortOrder,
	}
	if err := uc.categoryRepo.Create(ctx, cat); err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryInput) (*dto.CategoryRes, error) {
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, nil
	}
	if input.Name != nil {
		cat.Name = *input.Name
	}
	if input.Slug != nil {
		cat.Slug = *input.Slug
	}
	if input.SortOrder != nil {
		cat.SortOrder = *input.SortOrder
	}
	if err := uc.categoryRepo.Update(ctx, cat); err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return uc.categoryRepo.Delete(ctx, id)
}
