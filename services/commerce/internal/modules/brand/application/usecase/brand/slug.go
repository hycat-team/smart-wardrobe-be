package brand

import (
	"context"
	"regexp"
	"strings"

	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandslug"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/shared/pkg/utils/validation"

	"github.com/google/uuid"
)

var repeatedDashPattern = regexp.MustCompile(`-+`)

func NormalizeBrandSlug(raw string) string {
	slug := strings.ToLower(strings.TrimSpace(raw))
	slug = strings.ReplaceAll(slug, " ", "-")
	return repeatedDashPattern.ReplaceAllString(slug, "-")
}

func (uc *BrandUseCase) ResolveBrandRef(ctx context.Context, ref string) (uuid.UUID, error) {
	raw := strings.TrimSpace(ref)
	if raw == "" {
		return uuid.Nil, branderrors.ErrBrandNotFound()
	}
	if id, err := uuid.Parse(raw); err == nil {
		return id, nil
	}

	slug := NormalizeBrandSlug(raw)
	if !validation.IsBrandSlugFormatValid(slug) {
		return uuid.Nil, branderrors.ErrBrandNotFound()
	}

	brand, err := uc.brandRepo.GetBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, err
	}
	if brand == nil {
		return uuid.Nil, branderrors.ErrBrandNotFound()
	}
	return brand.ID, nil
}

func (uc *BrandUseCase) ResolveActiveBrandRef(ctx context.Context, ref string) (uuid.UUID, error) {
	brandID, err := uc.ResolveBrandRef(ctx, ref)
	if err != nil {
		return uuid.Nil, err
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return uuid.Nil, err
	}
	if brand == nil || !strings.EqualFold(string(brand.Status), string(brandstatus.Active)) {
		return uuid.Nil, branderrors.ErrBrandNotFound()
	}
	return brand.ID, nil
}

func normalizedBrandSlugOrError(raw string) (string, error) {
	slug := NormalizeBrandSlug(raw)
	if !validation.IsBrandSlugFormatValid(slug) {
		return "", branderrors.ErrBrandSlugInvalid()
	}
	if brandslug.IsReserved(slug) {
		return "", branderrors.ErrBrandSlugReserved(slug)
	}
	return slug, nil
}
