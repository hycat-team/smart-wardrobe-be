package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandUseCase struct {
	brandRepo    repositories.IBrandRepository
	memberRepo   repositories.IBrandMemberRepository
	userRepo     identity_repos.IUserRepository
	mediaService media.IMediaService
	uow          shared_repos.IUnitOfWork
	cfg          *config.Config
}

func NewBrandUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	userRepo identity_repos.IUserRepository,
	mediaService media.IMediaService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.IBrandUseCase {
	return &BrandUseCase{
		brandRepo:    brandRepo,
		memberRepo:   memberRepo,
		userRepo:     userRepo,
		mediaService: mediaService,
		uow:          uow,
		cfg:          cfg,
	}
}

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

	return mapper.MapBrand(brand), nil
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
	return mapper.MapBrand(brand), nil
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

	mappedItems := mapper.MapBrands(result.Brands)
	metadata := shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount)

	return &shared_dto.PaginationResult[*dto.BrandRes]{
		Items:    mappedItems,
		Metadata: metadata,
	}, nil
}

func (uc *BrandUseCase) GetActiveBrands(ctx context.Context) ([]*dto.BrandRes, error) {
	brands, err := uc.brandRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrands(brands), nil
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
	return mapper.MapBrand(brand), nil
}

func (uc *BrandUseCase) GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.PortalBrandRes, error) {
	members, err := uc.memberRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapper.MapPortalBrands(members), nil
}

func (uc *BrandUseCase) GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: fmt.Sprintf("brands/logos/%s", userID.String()),
	})
}

func (uc *BrandUseCase) UpdateBrandLogo(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandLogoReq) (*dto.BrandRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	brand.LogoURL = &input.LogoURL
	brand.LogoPublicID = &input.LogoPublicID
	brand.UpdatedAt = time.Now().UTC()
	if err := uc.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand), nil
}

func (uc *BrandUseCase) GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.PortalBrandRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.Brand == nil {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	return mapper.MapPortalBrand(member), nil
}

func (uc *BrandUseCase) AddBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMembersReq) (*dto.AddBrandMembersRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	result := &dto.AddBrandMembersRes{
		Created: []dto.AddBrandMemberItemResult{},
		Updated: []dto.AddBrandMemberItemResult{},
		Failed:  []dto.AddBrandMemberItemResult{},
	}
	resolvedUsersByID := make(map[uuid.UUID]struct {
		emailOrUsername string
		role            brandmemberrole.BrandMemberRole
	})
	resolvedUserOrder := make([]uuid.UUID, 0, len(input.Members))
	seenInputs := make(map[string]struct{})

	for _, memberInput := range input.Members {
		identifier := strings.TrimSpace(memberInput.EmailOrUsername)
		normalizedIdentifier := strings.ToLower(identifier)
		if _, exists := seenInputs[normalizedIdentifier]; exists {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "duplicate_input",
				Message:         "Email hoặc tên đăng nhập bị trùng trong danh sách gửi lên.",
			})
			continue
		}
		seenInputs[normalizedIdentifier] = struct{}{}

		if memberInput.Role != brandmemberrole.Staff {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "invalid_role",
				Message:         "API thêm thành viên chỉ cho phép vai trò staff.",
			})
			continue
		}

		user, err := uc.userRepo.GetByUsernameOrEmail(ctx, identifier)
		if err != nil {
			return nil, err
		}
		if user == nil || user.Status != userstatus.Active {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "user_not_found_or_inactive",
				Message:         "Không tìm thấy user đang hoạt động theo email hoặc tên đăng nhập.",
			})
			continue
		}
		if _, exists := resolvedUsersByID[user.ID]; exists {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "duplicate_user",
				Message:         "Email hoặc tên đăng nhập trỏ đến user đã có trong danh sách gửi lên.",
			})
			continue
		}
		resolvedUsersByID[user.ID] = struct {
			emailOrUsername string
			role            brandmemberrole.BrandMemberRole
		}{emailOrUsername: identifier, role: memberInput.Role}
		resolvedUserOrder = append(resolvedUserOrder, user.ID)
	}

	existingMembers, err := uc.memberRepo.GetByBrandAndUserIDs(ctx, brandID, resolvedUserOrder)
	if err != nil {
		return nil, err
	}
	existingByUserID := make(map[uuid.UUID]*entities.BrandMember, len(existingMembers))
	for _, existing := range existingMembers {
		existingByUserID[existing.UserID] = existing
	}

	for _, resolvedUserID := range resolvedUserOrder {
		resolved := resolvedUsersByID[resolvedUserID]
		if existing := existingByUserID[resolvedUserID]; existing != nil {
			existing.Role = resolved.role
			existing.Status = brandmemberstatus.Active
			if err := uc.memberRepo.Update(ctx, existing); err != nil {
				return nil, err
			}
			result.Updated = append(result.Updated, dto.AddBrandMemberItemResult{
				EmailOrUsername: resolved.emailOrUsername,
				Member:          mapper.MapBrandMember(existing),
			})
			continue
		}

		member := &entities.BrandMember{
			BrandID: brandID,
			UserID:  resolvedUserID,
			Role:    resolved.role,
			Status:  brandmemberstatus.Active,
		}
		if err := uc.memberRepo.Create(ctx, member); err != nil {
			return nil, err
		}
		result.Created = append(result.Created, dto.AddBrandMemberItemResult{
			EmailOrUsername: resolved.emailOrUsername,
			Member:          mapper.MapBrandMember(member),
		})
	}

	return result, nil
}

func (uc *BrandUseCase) GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	members, err := uc.memberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandMembers(members), nil
}

func (uc *BrandUseCase) RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return err
	}
	if brand == nil {
		return branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return branderrors.ErrBrandNotActive()
	}
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return err
	}
	if member == nil || member.Status != brandmemberstatus.Active {
		return branderrors.ErrBrandPortalForbidden()
	}
	for _, allowedRole := range allowedRoles {
		if member.Role == allowedRole {
			return nil
		}
	}
	return branderrors.ErrBrandPortalForbidden()
}

func isValidBrandStatus(status brandstatus.BrandStatus) bool {
	switch status {
	case brandstatus.PendingReview, brandstatus.Active, brandstatus.Suspended, brandstatus.Archived:
		return true
	default:
		return false
	}
}

func isValidBrandMemberRole(role brandmemberrole.BrandMemberRole) bool {
	switch role {
	case brandmemberrole.Owner, brandmemberrole.Staff:
		return true
	default:
		return false
	}
}
