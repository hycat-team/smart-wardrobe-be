package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type IBrandCoreUseCase interface {
	CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error)
	UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error)
	GetActiveBrands(ctx context.Context) ([]*dto.BrandRes, error)
	GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandRes, error)
	AddBrandMember(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMemberReq) (*dto.BrandMemberRes, error)
	GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error)
	GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandCustomerRes, error)
	JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerRes, error)
	CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error)
	RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error
}
