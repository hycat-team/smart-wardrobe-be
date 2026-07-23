package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type IBrandUseCase interface {
	CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error)
	GetBrandsForAdmin(ctx context.Context, query dto.GetBrandsAdminQueryReq) (*dto.AdminBrandListRes, error)
	GetActiveBrands(ctx context.Context, query dto.GetActiveBrandsQueryReq) (*dto.PublicBrandListRes, error)
	GetActiveBrand(ctx context.Context, brandID uuid.UUID) (*dto.BrandRes, error)
	GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.PortalBrandRes, error)
	GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.PortalBrandRes, error)
	GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	UpdateBrandImages(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandImagesReq) (*dto.BrandRes, error)
	AddBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMembersReq) (*dto.AddBrandMembersRes, error)
	GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error)
	RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error
	GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error)
	GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error)
	GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error)
}
