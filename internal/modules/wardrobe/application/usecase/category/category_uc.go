package category

import (
	"context"
	"regexp"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
)

type CategoryUseCase struct {
	logger       logger.Interface
	categoryRepo repositories.ICategoryRepository
	uow          shared_repos.IUnitOfWork
}

func NewCategoryUseCase(
	l logger.Interface,
	categoryRepo repositories.ICategoryRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.ICategoryUseCase {
	return &CategoryUseCase{
		logger:       l,
		categoryRepo: categoryRepo,
		uow:          uow,
	}
}

func (uc *CategoryUseCase) GetCategories(ctx context.Context) ([]*dto.CategoryRes, error) {
	categories, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return mapper.MapToCategoryResList(categories), nil
}

func (uc *CategoryUseCase) GetCategoryByID(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error) {
	category, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound
	}

	return mapper.MapToCategoryRes(category), nil
}

func (uc *CategoryUseCase) CreateCategory(ctx context.Context, input dto.CreateCategoryReq) (*dto.CategoryRes, error) {
	category, err := uc.prepareCategoryEntity(ctx, nil, input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	if err := uc.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	return mapper.MapToCategoryRes(category), nil
}

func (uc *CategoryUseCase) UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryReq) (*dto.CategoryRes, error) {
	current, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound
	}

	category, err := uc.prepareCategoryEntity(ctx, current, input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	if err := uc.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	return mapper.MapToCategoryRes(category), nil
}

func (uc *CategoryUseCase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		category, err := uc.categoryRepo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if category == nil {
			return wardrobeerrors.ErrCategoryNotFound
		}
		if category.Slug == "other" {
			return wardrobeerrors.ErrCategoryOtherImmutable
		}

		userItemCount, err := uc.categoryRepo.CountWardrobeItemsByCategoryAndItemType(txCtx, category.ID, itemtype.UserItem)
		if err != nil {
			return err
		}
		if userItemCount > 0 {
			return wardrobeerrors.ErrCategoryHasUserItems
		}

		systemItemCount, err := uc.categoryRepo.CountWardrobeItemsByCategoryAndItemType(txCtx, category.ID, itemtype.SystemCatalogItem)
		if err != nil {
			return err
		}
		if systemItemCount > 0 {
			fallbackCategory, err := uc.categoryRepo.GetBySlug(txCtx, "other")
			if err != nil {
				return err
			}
			if fallbackCategory == nil {
				return wardrobeerrors.ErrFallbackCategoryNotFound
			}
			if err := uc.categoryRepo.ReassignSystemCatalogItemsToCategory(txCtx, category.ID, fallbackCategory.ID); err != nil {
				return err
			}
		}

		return uc.categoryRepo.Delete(txCtx, category.ID)
	})
}

func (uc *CategoryUseCase) prepareCategoryEntity(ctx context.Context, current *entities.Category, name string, slug string) (*entities.Category, error) {
	normalizedName := strings.TrimSpace(name)
	normalizedSlug := normalizeCategorySlug(slug)

	existingByName, err := uc.categoryRepo.GetByName(ctx, normalizedName)
	if err != nil {
		return nil, err
	}
	if existingByName != nil && (current == nil || existingByName.ID != current.ID) {
		return nil, wardrobeerrors.ErrCategoryNameAlreadyExists
	}

	existingBySlug, err := uc.categoryRepo.GetBySlug(ctx, normalizedSlug)
	if err != nil {
		return nil, err
	}
	if existingBySlug != nil && (current == nil || existingBySlug.ID != current.ID) {
		return nil, wardrobeerrors.ErrCategorySlugAlreadyExists
	}

	if current == nil {
		return &entities.Category{
			Name: normalizedName,
			Slug: normalizedSlug,
		}, nil
	}

	current.Name = normalizedName
	current.Slug = normalizedSlug
	return current, nil
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
