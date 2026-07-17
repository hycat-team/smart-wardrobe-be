package category

import (
	"context"
	"regexp"
	"strings"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type CategoryUseCase struct {
	logger          logger.Interface
	fashionContract fashion_contract.IFashionContract
	wardrobeRepo    repositories.IWardrobeItemRepository
	uow             shared_repos.IUnitOfWork
}

func NewCategoryUseCase(
	l logger.Interface,
	fashionContract fashion_contract.IFashionContract,
	wardrobeRepo repositories.IWardrobeItemRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.ICategoryUseCase {
	return &CategoryUseCase{
		logger:          l,
		fashionContract: fashionContract,
		wardrobeRepo:    wardrobeRepo,
		uow:             uow,
	}
}

func (uc *CategoryUseCase) GetCategories(ctx context.Context) ([]*dto.CategoryRes, error) {
	categories, err := uc.fashionContract.GetAllCategories(ctx)
	if err != nil {
		return nil, err
	}

	return mapper.MapToCategoryResList(categories), nil
}

func (uc *CategoryUseCase) GetCategoryByID(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error) {
	category, err := uc.fashionContract.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound()
	}

	return mapper.MapToCategoryRes(category), nil
}

func (uc *CategoryUseCase) CreateCategory(ctx context.Context, input dto.CreateCategoryReq) (*dto.CategoryRes, error) {
	normalizedName := strings.TrimSpace(input.Name)
	normalizedSlug := normalizeCategorySlug(input.Slug)
	if normalizedSlug == "vay" {
		return nil, wardrobeerrors.ErrCategoryLegacySlugForbidden()
	}

	existingByName, err := uc.fashionContract.GetCategoryByName(ctx, normalizedName)
	if err != nil {
		return nil, err
	}
	if existingByName != nil {
		return nil, wardrobeerrors.ErrCategoryNameAlreadyExists()
	}

	existingBySlug, err := uc.fashionContract.GetCategoryBySlug(ctx, normalizedSlug)
	if err != nil {
		return nil, err
	}
	if existingBySlug != nil {
		return nil, wardrobeerrors.ErrCategorySlugAlreadyExists()
	}

	normalizedSortOrder := 0
	if input.SortOrder != nil {
		normalizedSortOrder = *input.SortOrder
	}

	category, err := uc.fashionContract.CreateCategory(ctx, fashion_dto.CreateCategoryInput{
		Name:      normalizedName,
		Slug:      normalizedSlug,
		SortOrder: normalizedSortOrder,
	})
	if err != nil {
		return nil, err
	}

	return mapper.MapToCategoryRes(category), nil
}

func (uc *CategoryUseCase) UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryReq) (*dto.CategoryRes, error) {
	current, err := uc.fashionContract.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound()
	}

	normalizedName := strings.TrimSpace(input.Name)
	normalizedSlug := normalizeCategorySlug(input.Slug)
	if normalizedSlug == "vay" {
		return nil, wardrobeerrors.ErrCategoryLegacySlugForbidden()
	}

	existingByName, err := uc.fashionContract.GetCategoryByName(ctx, normalizedName)
	if err != nil {
		return nil, err
	}
	if existingByName != nil && existingByName.ID != id {
		return nil, wardrobeerrors.ErrCategoryNameAlreadyExists()
	}

	existingBySlug, err := uc.fashionContract.GetCategoryBySlug(ctx, normalizedSlug)
	if err != nil {
		return nil, err
	}
	if existingBySlug != nil && existingBySlug.ID != id {
		return nil, wardrobeerrors.ErrCategorySlugAlreadyExists()
	}

	updateInput := fashion_dto.UpdateCategoryInput{
		Name: &normalizedName,
		Slug: &normalizedSlug,
	}
	if input.SortOrder != nil {
		updateInput.SortOrder = input.SortOrder
	}

	category, err := uc.fashionContract.UpdateCategory(ctx, id, updateInput)
	if err != nil {
		return nil, err
	}

	return mapper.MapToCategoryRes(category), nil
}

func (uc *CategoryUseCase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		category, err := uc.fashionContract.GetCategory(txCtx, id)
		if err != nil {
			return err
		}
		if category == nil {
			return wardrobeerrors.ErrCategoryNotFound()
		}
		if category.Slug == "other" {
			return wardrobeerrors.ErrCategoryOtherImmutable()
		}

		userItemCount, err := uc.wardrobeRepo.CountByCategoryIDAndItemType(txCtx, category.ID, itemtype.UserItem)
		if err != nil {
			return err
		}
		if userItemCount > 0 {
			return wardrobeerrors.ErrCategoryHasUserItems()
		}

		systemItemCount, err := uc.wardrobeRepo.CountByCategoryIDAndItemType(txCtx, category.ID, itemtype.SystemCatalogItem)
		if err != nil {
			return err
		}
		if systemItemCount > 0 {
			fallbackCategory, err := uc.fashionContract.GetCategoryBySlug(txCtx, "other")
			if err != nil {
				return err
			}
			if fallbackCategory == nil {
				return wardrobeerrors.ErrFallbackCategoryNotFound()
			}
			if err := uc.wardrobeRepo.ReassignCategoryForCatalogItems(txCtx, category.ID, fallbackCategory.ID); err != nil {
				return err
			}
		}

		return uc.fashionContract.DeleteCategory(txCtx, category.ID)
	})
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
