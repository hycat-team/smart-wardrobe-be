package category

import (
	"context"
	"regexp"
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashionerrors "smart-wardrobe-be/internal/modules/fashion/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

	"github.com/google/uuid"
)

type CategoryUseCase struct {
	categoryRepo     repositories.ICategoryRepository
	fashionItemRepo  repositories.IFashionItemRepository
	wardrobeContract wardrobe_contract.IWardrobeContract
}

func NewCategoryUseCase(
	categoryRepo repositories.ICategoryRepository,
	fashionItemRepo repositories.IFashionItemRepository,
	wardrobeContract wardrobe_contract.IWardrobeContract,
) uc_interfaces.ICategoryUseCase {
	return &CategoryUseCase{
		categoryRepo:     categoryRepo,
		fashionItemRepo:  fashionItemRepo,
		wardrobeContract: wardrobeContract,
	}
}

func mapToCategoryRes(cat *entities.Category) *dto.CategoryContractDTO {
	if cat == nil {
		return nil
	}
	return &dto.CategoryContractDTO{
		ID:        cat.ID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		SortOrder: cat.SortOrder,
	}
}

func (uc *CategoryUseCase) GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryContractDTO, error) {
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) ListCategories(ctx context.Context, ids []uuid.UUID) ([]*dto.CategoryContractDTO, error) {
	items, err := uc.categoryRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.CategoryContractDTO, len(items))
	for i, item := range items {
		res[i] = mapToCategoryRes(item)
	}
	return res, nil
}

func (uc *CategoryUseCase) GetAllCategories(ctx context.Context) ([]*dto.CategoryContractDTO, error) {
	items, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.CategoryContractDTO, len(items))
	for i, item := range items {
		res[i] = mapToCategoryRes(item)
	}
	return res, nil
}

func (uc *CategoryUseCase) GetCategoryBySlug(ctx context.Context, slug string) (*dto.CategoryContractDTO, error) {
	cat, err := uc.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) GetCategoryByName(ctx context.Context, name string) (*dto.CategoryContractDTO, error) {
	cat, err := uc.categoryRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryContractDTO, error) {
	normalizedName := strings.TrimSpace(input.Name)
	normalizedSlug := normalizeCategorySlug(input.Slug)

	if normalizedSlug == "vay" {
		return nil, fashionerrors.ErrCategoryLegacySlugForbidden()
	}

	existingByName, err := uc.categoryRepo.GetByName(ctx, normalizedName)
	if err != nil {
		return nil, err
	}
	if existingByName != nil {
		return nil, fashionerrors.ErrCategoryNameAlreadyExists()
	}

	existingBySlug, err := uc.categoryRepo.GetBySlug(ctx, normalizedSlug)
	if err != nil {
		return nil, err
	}
	if existingBySlug != nil {
		return nil, fashionerrors.ErrCategorySlugAlreadyExists()
	}

	cat := &entities.Category{
		Name:      normalizedName,
		Slug:      normalizedSlug,
		SortOrder: input.SortOrder,
	}
	if err := uc.categoryRepo.Create(ctx, cat); err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (uc *CategoryUseCase) UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryInput) (*dto.CategoryContractDTO, error) {
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, fashionerrors.ErrCategoryNotFound()
	}

	if input.Name != nil {
		normalizedName := strings.TrimSpace(*input.Name)
		existingByName, err := uc.categoryRepo.GetByName(ctx, normalizedName)
		if err != nil {
			return nil, err
		}
		if existingByName != nil && existingByName.ID != id {
			return nil, fashionerrors.ErrCategoryNameAlreadyExists()
		}
		cat.Name = normalizedName
	}

	if input.Slug != nil {
		normalizedSlug := normalizeCategorySlug(*input.Slug)
		if normalizedSlug == "vay" {
			return nil, fashionerrors.ErrCategoryLegacySlugForbidden()
		}
		existingBySlug, err := uc.categoryRepo.GetBySlug(ctx, normalizedSlug)
		if err != nil {
			return nil, err
		}
		if existingBySlug != nil && existingBySlug.ID != id {
			return nil, fashionerrors.ErrCategorySlugAlreadyExists()
		}
		cat.Slug = normalizedSlug
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
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cat == nil {
		return fashionerrors.ErrCategoryNotFound()
	}
	if cat.Slug == "other" {
		return fashionerrors.ErrCategoryOtherImmutable()
	}

	fashionIDs, err := uc.fashionItemRepo.GetFashionItemIDsByCategory(ctx, id)
	if err != nil {
		return err
	}

	if len(fashionIDs) > 0 {
		userItemCount, err := uc.wardrobeContract.CountItemsByFashionIDs(ctx, fashionIDs)
		if err != nil {
			return err
		}
		if userItemCount > 0 {
			return fashionerrors.ErrCategoryHasUserItems()
		}
	}

	return uc.categoryRepo.Delete(ctx, id)
}

func normalizeCategorySlug(input string) string {
	slug := strings.TrimSpace(strings.ToLower(input))
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.Join(strings.Fields(slug), "-")

	invalidChars := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = invalidChars.ReplaceAllString(slug, "-")

	multiDashes := regexp.MustCompile(`-+`)
	slug = multiDashes.ReplaceAllString(slug, "-")

	return strings.Trim(slug, "-")
}
