package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandCoreUseCase struct {
	brandRepo    repositories.IBrandRepository
	memberRepo   repositories.IBrandMemberRepository
	customerRepo repositories.IBrandCustomerRepository
	uow          shared_repos.IUnitOfWork
}

func NewBrandCoreUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandCoreUseCase {
	return &BrandCoreUseCase{
		brandRepo:    brandRepo,
		memberRepo:   memberRepo,
		customerRepo: customerRepo,
		uow:          uow,
	}
}

func (uc *BrandCoreUseCase) CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	return uc.createBrandWithOwner(ctx, userID, input, brandstatus.PendingReview, nil, nil)
}

func (uc *BrandCoreUseCase) CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	now := time.Now().UTC()
	return uc.createBrandWithOwner(ctx, adminID, input, brandstatus.Active, &adminID, &now)
}

func (uc *BrandCoreUseCase) createBrandWithOwner(ctx context.Context, creatorID uuid.UUID, input dto.CreateBrandReq, status brandstatus.BrandStatus, approverID *uuid.UUID, approvedAt *time.Time) (*dto.BrandRes, error) {
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

func (uc *BrandCoreUseCase) UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error) {
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

func (uc *BrandCoreUseCase) GetActiveBrands(ctx context.Context) ([]*dto.BrandRes, error) {
	brands, err := uc.brandRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrands(brands), nil
}

func (uc *BrandCoreUseCase) GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand), nil
}

func (uc *BrandCoreUseCase) AddBrandMember(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMemberReq) (*dto.BrandMemberRes, error) {
	if !isValidBrandMemberRole(input.Role) {
		return nil, branderrors.ErrInvalidBrandMemberRole(input.Role)
	}
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
		return nil, err
	}
	existing, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, input.UserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.Role = input.Role
		existing.Status = brandmemberstatus.Active
		if err := uc.memberRepo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return mapper.MapBrandMember(existing), nil
	}

	member := &entities.BrandMember{
		BrandID: brandID,
		UserID:  input.UserID,
		Role:    input.Role,
		Status:  brandmemberstatus.Active,
	}
	if err := uc.memberRepo.Create(ctx, member); err != nil {
		return nil, err
	}
	return mapper.MapBrandMember(member), nil
}

func (uc *BrandCoreUseCase) GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
		return nil, err
	}
	members, err := uc.memberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandMembers(members), nil
}

func (uc *BrandCoreUseCase) GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandCustomerRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	customers, err := uc.customerRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomers(customers), nil
}

func (uc *BrandCoreUseCase) JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerRes, error) {
	if currentRole != roleslug.User {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
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
	if existing, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID); err != nil {
		return nil, err
	} else if existing != nil {
		return mapper.MapBrandCustomer(existing), nil
	}

	customer := &entities.BrandCustomer{
		BrandID:      brandID,
		UserID:       &userID,
		JoinedSource: brandcustomerjoinedsource.SelfJoin,
		Status:       brandcustomerstatus.Active,
		JoinedAt:     time.Now().UTC(),
	}
	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomer(customer), nil
}

func (uc *BrandCoreUseCase) CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error) {
	if strings.TrimSpace(input.PhoneE164) == "" {
		return nil, branderrors.ErrPhoneRequired()
	}
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff); err != nil {
		return nil, err
	}
	creator, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	phone := strings.TrimSpace(input.PhoneE164)
	phoneHash := hashPhone(phone)
	if existing, err := uc.customerRepo.GetByBrandAndPhoneHash(ctx, brandID, phoneHash); err != nil {
		return nil, err
	} else if existing != nil {
		return mapper.MapBrandCustomer(existing), nil
	}

	customer := &entities.BrandCustomer{
		BrandID:              brandID,
		CustomerName:         input.CustomerName,
		PhoneE164:            &phone,
		PhoneHash:            &phoneHash,
		ExternalCustomerCode: input.ExternalCustomerCode,
		JoinedSource:         brandcustomerjoinedsource.OfflinePurchase,
		Status:               brandcustomerstatus.Active,
		JoinedAt:             time.Now().UTC(),
		CreatedByMemberID:    &creator.ID,
	}
	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomer(customer), nil
}

func (uc *BrandCoreUseCase) RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error {
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
	case brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer:
		return true
	default:
		return false
	}
}

func hashPhone(phone string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(phone)))
	return hex.EncodeToString(sum[:])
}
