package admin

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandAdminUseCase struct {
	brandRepo         repositories.IBrandRepository
	brandMemberRepo   repositories.IBrandMemberRepository
	membershipCache   caching.IBrandMembershipCache
	userBriefContract identity_contract.IUserBriefContract
	uow               shared_repos.IUnitOfWork
}

func NewBrandAdminUseCase(
	brandRepo repositories.IBrandRepository,
	brandMemberRepo repositories.IBrandMemberRepository,
	membershipCache caching.IBrandMembershipCache,
	userBriefContract identity_contract.IUserBriefContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandAdminUseCase {
	return &BrandAdminUseCase{
		brandRepo:         brandRepo,
		brandMemberRepo:   brandMemberRepo,
		membershipCache:   membershipCache,
		userBriefContract: userBriefContract,
		uow:               uow,
	}
}

func (uc *BrandAdminUseCase) ListMembers(ctx context.Context, brandID uuid.UUID) ([]*dto.BrandMemberRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	members, err := uc.brandMemberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}

	users, err := uc.userBriefContract.GetByIDs(ctx, userIDs)
	if err != nil {
		users = nil
	}

	userMap := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	return mapper.MapBrandMembersWithUsers(members, userMap), nil
}

func (uc *BrandAdminUseCase) AddStaff(ctx context.Context, brandID uuid.UUID, req dto.AdminAddBrandStaffReq) (*dto.BrandMemberRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	user, err := uc.userBriefContract.GetByID(ctx, req.UserID)
	if err != nil || user == nil {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	existing, err := uc.brandMemberRepo.GetByBrandAndUser(ctx, brandID, req.UserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrMemberAlreadyExists()
	}

	member := &entities.BrandMember{
		BrandID: brandID,
		UserID:  req.UserID,
		Role:    brandmemberrole.Staff,
		Status:  brandmemberstatus.Active,
	}

	if err := uc.brandMemberRepo.Create(ctx, member); err != nil {
		return nil, err
	}

	_ = uc.membershipCache.InvalidateByUserID(ctx, req.UserID)

	return mapper.MapBrandMemberWithUser(member, user), nil
}

func (uc *BrandAdminUseCase) UpdateMember(ctx context.Context, brandID uuid.UUID, memberID uuid.UUID, req dto.AdminUpdateBrandMemberReq) (*dto.BrandMemberRes, error) {
	member, err := uc.brandMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.BrandID != brandID {
		return nil, branderrors.ErrBrandMemberNotFound()
	}

	// If attempting to demote or change status of owner, guard against last owner
	if member.Role == brandmemberrole.Owner {
		if (req.Role != nil && *req.Role != brandmemberrole.Owner) ||
			(req.Status != nil && *req.Status != brandmemberstatus.Active) {
			members, err := uc.brandMemberRepo.GetByBrandID(ctx, brandID)
			if err != nil {
				return nil, err
			}
			activeOwnerCount := 0
			for _, m := range members {
				if m.Role == brandmemberrole.Owner && m.Status == brandmemberstatus.Active {
					activeOwnerCount++
				}
			}
			if activeOwnerCount <= 1 {
				return nil, branderrors.ErrCannotDemoteLastOwner()
			}
		}
	}

	if req.Role != nil {
		member.Role = *req.Role
	}
	if req.Status != nil {
		member.Status = *req.Status
	}

	if err := uc.brandMemberRepo.Update(ctx, member); err != nil {
		return nil, err
	}

	_ = uc.membershipCache.InvalidateByUserID(ctx, member.UserID)

	user, _ := uc.userBriefContract.GetByID(ctx, member.UserID)
	return mapper.MapBrandMemberWithUser(member, user), nil
}

func (uc *BrandAdminUseCase) RemoveMember(ctx context.Context, brandID uuid.UUID, memberID uuid.UUID) error {
	member, err := uc.brandMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil || member.BrandID != brandID {
		return branderrors.ErrBrandMemberNotFound()
	}

	if member.Role == brandmemberrole.Owner {
		members, err := uc.brandMemberRepo.GetByBrandID(ctx, brandID)
		if err != nil {
			return err
		}
		activeOwnerCount := 0
		for _, m := range members {
			if m.Role == brandmemberrole.Owner && m.Status == brandmemberstatus.Active {
				activeOwnerCount++
			}
		}
		if activeOwnerCount <= 1 {
			return branderrors.ErrCannotRemoveLastOwner()
		}
	}

	if err := uc.brandMemberRepo.Delete(ctx, memberID); err != nil {
		return err
	}

	_ = uc.membershipCache.InvalidateByUserID(ctx, member.UserID)
	return nil
}

func (uc *BrandAdminUseCase) TransferOwnership(ctx context.Context, brandID uuid.UUID, req dto.TransferBrandOwnershipReq) error {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return err
	}
	if brand == nil {
		return branderrors.ErrBrandNotFound()
	}

	members, err := uc.brandMemberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return err
	}

	var currentOwner *entities.BrandMember
	var newOwner *entities.BrandMember

	for _, m := range members {
		if m.Role == brandmemberrole.Owner && m.Status == brandmemberstatus.Active {
			currentOwner = m
		}
		if m.UserID == req.NewOwnerUserID {
			newOwner = m
		}
	}

	if newOwner == nil {
		return branderrors.ErrNewOwnerNotMember()
	}

	txFn := func(txCtx context.Context) error {
		// 1. Promote new owner
		newOwner.Role = brandmemberrole.Owner
		newOwner.Status = brandmemberstatus.Active
		if err := uc.brandMemberRepo.Update(txCtx, newOwner); err != nil {
			return err
		}

		// 2. Handle old owner
		if currentOwner != nil && currentOwner.ID != newOwner.ID {
			if req.OldOwnerHandling == "remove" {
				if err := uc.brandMemberRepo.Delete(txCtx, currentOwner.ID); err != nil {
					return err
				}
			} else { // default or "staff"
				currentOwner.Role = brandmemberrole.Staff
				if err := uc.brandMemberRepo.Update(txCtx, currentOwner); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := uc.uow.Execute(ctx, txFn); err != nil {
		return err
	}

	// Invalidate cache for both users
	_ = uc.membershipCache.InvalidateByUserID(ctx, req.NewOwnerUserID)
	if currentOwner != nil {
		_ = uc.membershipCache.InvalidateByUserID(ctx, currentOwner.UserID)
	}

	return nil
}
