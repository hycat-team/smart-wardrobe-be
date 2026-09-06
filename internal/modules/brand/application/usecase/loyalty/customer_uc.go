package loyalty

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/stringutils"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

func (uc *BrandLoyaltyUseCase) GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandCustomersQueryReq) (*dto.BrandCustomerListRes, error) {
	var statusFilter *brandcustomerstatus.BrandCustomerStatus
	if query.Status != nil {
		val := brandcustomerstatus.BrandCustomerStatus(*query.Status)
		statusFilter = &val
	}

	filter := repositories.BrandCustomerFilter{
		BrandID: brandID,
		Status:  statusFilter,
		Query:   query.Query,
		Page:    query.Page,
		Limit:   query.Limit,
	}

	result, err := uc.customerRepo.GetByBrandIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	userMap := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO)
	if len(result.Customers) > 0 {
		userIDs := make([]uuid.UUID, 0, len(result.Customers))
		for _, c := range result.Customers {
			if c.UserID != nil {
				userIDs = append(userIDs, *c.UserID)
			}
		}
		if len(userIDs) > 0 {
			users, err := uc.userBriefContract.GetByIDs(ctx, userIDs)
			if err == nil {
				for _, u := range users {
					userMap[u.ID] = u
				}
			}
		}
	}

	return &dto.BrandCustomerListRes{
		Items:    mapper.MapBrandCustomersWithUsers(result.Customers, userMap),
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetBrandCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.BrandCustomerRes, error) {
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	var user *identity_dto.UserBriefContractDTO
	if customer.UserID != nil {
		u, err := uc.userBriefContract.GetByID(ctx, *customer.UserID)
		if err == nil {
			user = u
		}
	}
	return mapper.MapBrandCustomer(customer, user), nil
}

func (uc *BrandLoyaltyUseCase) JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerBriefRes, error) {
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
		if err := uc.ensureLoyaltyAccount(ctx, existing); err != nil {
			return nil, err
		}
		return mapper.MapBrandCustomerBrief(existing, nil), nil
	}

	customer := &entities.BrandCustomer{
		BrandID:      brandID,
		UserID:       &userID,
		JoinedSource: brandcustomerjoinedsource.SelfJoin,
		Status:       brandcustomerstatus.Active,
		JoinedAt:     time.Now().UTC(),
	}
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.customerRepo.Create(txCtx, customer); err != nil {
			return err
		}
		return uc.ensureLoyaltyAccount(txCtx, customer)
	}); err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomerBrief(customer, nil), nil
}

func (uc *BrandLoyaltyUseCase) CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error) {
	if strings.TrimSpace(input.PhoneE164) == "" {
		return nil, branderrors.ErrPhoneRequired()
	}
	creator, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	phone := strings.TrimSpace(input.PhoneE164)
	phoneHash := stringutils.HashSHA256(phone)
	if existing, err := uc.customerRepo.GetByBrandAndPhoneHash(ctx, brandID, phoneHash); err != nil {
		return nil, err
	} else if existing != nil {
		if err := uc.ensureLoyaltyAccount(ctx, existing); err != nil {
			return nil, err
		}
		return mapper.MapBrandCustomer(existing, nil), nil
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
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.customerRepo.Create(txCtx, customer); err != nil {
			return err
		}
		return uc.ensureLoyaltyAccount(txCtx, customer)
	}); err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomer(customer, nil), nil
}
