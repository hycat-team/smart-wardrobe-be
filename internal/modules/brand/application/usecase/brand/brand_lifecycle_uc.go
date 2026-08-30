package brand

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

func (uc *BrandUseCase) CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	return uc.createBrandWithOwner(ctx, userID, input, brandstatus.PendingReview, nil, nil)
}

func (uc *BrandUseCase) CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	now := time.Now().UTC()
	return uc.createBrandWithOwner(ctx, adminID, input, brandstatus.Active, &adminID, &now)
}

func (uc *BrandUseCase) createBrandWithOwner(ctx context.Context, creatorID uuid.UUID, input dto.CreateBrandReq, status brandstatus.BrandStatus, approverID *uuid.UUID, approvedAt *time.Time) (*dto.BrandRes, error) {
	slug := strings.TrimSpace(strings.ToLower(input.Slug))
	if existing, err := uc.brandRepo.GetBySlug(ctx, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, branderrors.ErrBrandSlugExists()
	}

	brand := &entities.Brand{
		Slug:             slug,
		Name:             strings.TrimSpace(input.Name),
		Description:      input.Description,
		LogoURL:          input.LogoURL,
		LogoPublicID:     input.LogoPublicID,
		Status:           status,
		CreatedByUserID:  creatorID,
		ApprovedByUserID: approverID,
		ApprovedAt:       approvedAt,
	}
	member := &entities.BrandMember{
		UserID: creatorID,
		Role:   brandmemberrole.Owner,
		Status: brandmemberstatus.Active,
	}

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.brandRepo.Create(txCtx, brand); err != nil {
			return err
		}
		member.BrandID = brand.ID
		return uc.memberRepo.Create(txCtx, member)
	}); err != nil {
		return nil, err
	}

	return mapper.MapBrand(brand, -1), nil
}

func (uc *BrandUseCase) UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error) {
	if !isValidBrandStatus(input.Status) {
		return nil, branderrors.ErrInvalidBrandStatus(input.Status)
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	brand.Status = input.Status
	if input.Status == brandstatus.Active {
		now := time.Now().UTC()
		brand.ApprovedByUserID = &adminID
		brand.ApprovedAt = &now
	}
	if err := uc.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}
	// Changing a brand's status changes its members' effective authorization,
	// so invalidate their membership cache immediately.
	if err := uc.InvalidateBrandMembersCache(ctx, brandID); err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand, -1), nil
}

func (uc *BrandUseCase) GetBrandsForAdmin(ctx context.Context, query dto.GetBrandsAdminQueryReq) (*dto.AdminBrandListRes, error) {
	filter := repositories.BrandFilter{
		Status: query.Status,
		Query:  query.Query,
		Page:   query.Page,
		Limit:  query.Limit,
	}

	result, err := uc.brandRepo.GetBrandsForAdmin(ctx, filter)
	if err != nil {
		return nil, err
	}

	brandIDs := make([]uuid.UUID, len(result.Brands))
	for i, b := range result.Brands {
		brandIDs[i] = b.ID
	}
	counts, err := uc.customerRepo.CountByBrandIDs(ctx, brandIDs)
	if err != nil {
		return nil, err
	}
	totalCustomerByBrandID := make(map[uuid.UUID]int, len(counts))
	for id, cnt := range counts {
		totalCustomerByBrandID[id] = int(cnt)
	}

	mappedItems := mapper.MapBrands(result.Brands, totalCustomerByBrandID)
	metadata := shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount)

	return &shared_dto.PaginationResult[*dto.BrandRes]{
		Items:    mappedItems,
		Metadata: metadata,
	}, nil
}

func (uc *BrandUseCase) GetActiveBrands(ctx context.Context, query dto.GetActiveBrandsQueryReq) (*dto.PublicBrandBriefListRes, error) {
	filter := repositories.BrandFilter{
		Query: query.Query,
		Page:  query.Page,
		Limit: query.Limit,
	}

	result, err := uc.brandRepo.GetActiveFiltered(ctx, filter)
	if err != nil {
		return nil, err
	}

	mappedItems := mapper.MapBrandBriefs(result.Brands)
	metadata := shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount)

	return &shared_dto.PaginationResult[*dto.BrandBriefRes]{
		Items:    mappedItems,
		Metadata: metadata,
	}, nil
}

func (uc *BrandUseCase) GetActiveBrand(ctx context.Context, brandID uuid.UUID) (*dto.BrandRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}
	totalCustomer, err := uc.customerRepo.CountByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand, int(totalCustomer)), nil
}

func (uc *BrandUseCase) GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.PortalBrandRes, error) {
	members, err := uc.memberRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	brandIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		brandIDs[i] = m.BrandID
	}
	counts, err := uc.customerRepo.CountByBrandIDs(ctx, brandIDs)
	if err != nil {
		return nil, err
	}
	totalCustomerByBrandID := make(map[uuid.UUID]int, len(counts))
	for id, cnt := range counts {
		totalCustomerByBrandID[id] = int(cnt)
	}
	return mapper.MapPortalBrands(members, totalCustomerByBrandID), nil
}

func (uc *BrandUseCase) GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: fmt.Sprintf("%s/%s", uc.cfg.Cloudinary.BrandsFolder, userID.String()),
	})
}

func (uc *BrandUseCase) UpdateBrandImages(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandImagesReq) (*dto.BrandRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	if input.LogoURL != nil {
		brand.LogoURL = input.LogoURL
	}
	if input.LogoPublicID != nil {
		brand.LogoPublicID = input.LogoPublicID
	}
	if input.BackgroundURL != nil {
		brand.BackgroundURL = input.BackgroundURL
	}
	if input.BackgroundPublicID != nil {
		brand.BackgroundPublicID = input.BackgroundPublicID
	}
	brand.UpdatedAt = time.Now().UTC()
	if err := uc.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand, -1), nil
}

func (uc *BrandUseCase) GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.PortalBrandRes, error) {
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.Brand == nil {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	totalCustomer, err := uc.customerRepo.CountByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapPortalBrand(member, int(totalCustomer)), nil
}
