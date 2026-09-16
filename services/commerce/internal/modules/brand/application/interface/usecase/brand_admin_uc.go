package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"

	"github.com/google/uuid"
)

type IBrandAdminUseCase interface {
	ListMembers(ctx context.Context, brandID uuid.UUID) ([]*dto.BrandMemberRes, error)
	AddStaff(ctx context.Context, brandID uuid.UUID, req dto.AdminAddBrandStaffReq) (*dto.BrandMemberRes, error)
	UpdateMember(ctx context.Context, brandID uuid.UUID, memberID uuid.UUID, req dto.AdminUpdateBrandMemberReq) (*dto.BrandMemberRes, error)
	RemoveMember(ctx context.Context, brandID uuid.UUID, memberID uuid.UUID) error
	TransferOwnership(ctx context.Context, brandID uuid.UUID, req dto.TransferBrandOwnershipReq) error
}
