package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

type IBrandUseCase interface {
	ResolveBrandRef(ctx context.Context, ref string) (uuid.UUID, error)
	ResolveActiveBrandRef(ctx context.Context, ref string) (uuid.UUID, error)
	CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error)
	GetBrandsForAdmin(ctx context.Context, query dto.GetBrandsAdminQueryReq) (*dto.AdminBrandListRes, error)
	GetActiveBrands(ctx context.Context, query dto.GetActiveBrandsQueryReq) (*dto.PublicBrandBriefListRes, error)
	GetActiveBrand(ctx context.Context, brandID uuid.UUID) (*dto.BrandRes, error)
	GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.PortalBrandRes, error)
	GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.PortalBrandRes, error)
	GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	UpdateBrandImages(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandImagesReq) (*dto.BrandRes, error)
	UpdateBrandProfile(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandProfileReq) (*dto.BrandRes, error)
	AddBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMembersReq) (*dto.AddBrandMembersRes, error)
	GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error)
	RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error
	GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error)
	GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error)
	GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error)
	InvalidateUserMembershipCache(ctx context.Context, userID uuid.UUID) error
	InvalidateBrandMembersCache(ctx context.Context, brandID uuid.UUID) error
	GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error)
	GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error)
	GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error)
}
