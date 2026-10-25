package benefit

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	caching "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/usecase/benefit/strategies"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandBenefitUseCase struct {
	brandRepo      repositories.IBrandRepository
	memberRepo     repositories.IBrandMemberRepository
	customerRepo   repositories.IBrandCustomerRepository
	accountRepo    repositories.ILoyaltyAccountRepository
	txRepo         repositories.ILoyaltyPointTransactionRepository
	lotRepo        repositories.ILoyaltyPointLotRepository
	benefitRepo    repositories.IBrandBenefitRepository
	redemptionRepo repositories.IBenefitRedemptionRepository
	tierRepo       repositories.ILoyaltyTierRepository
	uow            shared_repos.IUnitOfWork
	strategies     map[benefitunlocktype.BenefitUnlockType]UnlockStrategy
	featureCache   caching.IBrandFeatureAccessCache
}

func NewBrandBenefitUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	accountRepo repositories.ILoyaltyAccountRepository,
	txRepo repositories.ILoyaltyPointTransactionRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	benefitRepo repositories.IBrandBenefitRepository,
	redemptionRepo repositories.IBenefitRedemptionRepository,
	tierRepo repositories.ILoyaltyTierRepository,
	uow shared_repos.IUnitOfWork,
	featureCache caching.IBrandFeatureAccessCache,
) uc_interfaces.IBrandBenefitUseCase {
	strategies := map[benefitunlocktype.BenefitUnlockType]UnlockStrategy{
		benefitunlocktype.TierPrivilege:   strategies.NewTierPrivilegeStrategy(accountRepo, tierRepo, redemptionRepo),
		benefitunlocktype.PointRedemption: strategies.NewPointRedemptionStrategy(accountRepo, lotRepo, txRepo, redemptionRepo),
		benefitunlocktype.ManualGrant:     strategies.NewManualGrantStrategy(),
	}

	return &BrandBenefitUseCase{
		brandRepo:      brandRepo,
		memberRepo:     memberRepo,
		customerRepo:   customerRepo,
		accountRepo:    accountRepo,
		txRepo:         txRepo,
		lotRepo:        lotRepo,
		benefitRepo:    benefitRepo,
		redemptionRepo: redemptionRepo,
		tierRepo:       tierRepo,
		uow:            uow,
		strategies:     strategies,
		featureCache:   featureCache,
	}
}

func (uc *BrandBenefitUseCase) CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	bType := benefittype.BenefitType(strings.ToLower(input.BenefitType))
	uType := benefitunlocktype.BenefitUnlockType(strings.ToLower(input.UnlockType))

	benefit := &entities.BrandBenefit{
		BrandID:     brandID,
		Name:        strings.TrimSpace(input.Name),
		Description: input.Description,
		BenefitType: bType,
		UnlockType:  uType,
		Status:      benefitstatus.Active,
	}

	switch uType {
	case benefitunlocktype.PointRedemption:
		if input.RequiredPoints != nil && *input.RequiredPoints > 0 {
			benefit.RequiredPoints = input.RequiredPoints
		}
	case benefitunlocktype.TierPrivilege:
		if input.RequiredTierID == nil {
			return nil, branderrors.ErrInvalidBrandMemberRole("Yeu cau phai co requiredTierId")
		}
		tier, err := uc.tierRepo.GetByID(ctx, *input.RequiredTierID)
		if err != nil {
			return nil, err
		}
		if tier == nil || tier.BrandID != brandID {
			return nil, branderrors.ErrBenefitInvalidStatus()
		}
		benefit.RequiredTierID = input.RequiredTierID
	}

	if bType == benefittype.FeatureAccess {
		if input.FeatureCode == nil {
			return nil, branderrors.ErrBenefitFeatureCodeRequired()
		}
		fCode, ok := benefitfeaturecode.Parse(*input.FeatureCode)
		if !ok {
			return nil, branderrors.ErrInvalidBenefitFeatureCode(*input.FeatureCode)
		}
		benefit.FeatureCode = &fCode
	} else if input.FeatureCode != nil && strings.TrimSpace(*input.FeatureCode) != "" {
		return nil, branderrors.ErrInvalidBenefitFeatureCode(*input.FeatureCode)
	}

	if input.FeatureConfig != nil {
		bytes, err := json.Marshal(input.FeatureConfig)
		if err != nil {
			return nil, branderrors.ErrBenefitInvalidStatus()
		}
		benefit.FeatureConfig = entities.JSONDocument(bytes)
	}

	if err := uc.benefitRepo.Create(ctx, benefit); err != nil {
		return nil, err
	}

	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandBenefitUseCase) ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, query dto.GetBrandBenefitsQueryReq) (*dto.BrandBenefitListRes, error) {
	var unlockType *benefitunlocktype.BenefitUnlockType
	if query.UnlockType != nil {
		val := benefitunlocktype.BenefitUnlockType(strings.ToLower(*query.UnlockType))
		unlockType = &val
	}

	var benefitType *benefittype.BenefitType
	if query.BenefitType != nil {
		val := benefittype.BenefitType(strings.ToLower(*query.BenefitType))
		benefitType = &val
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	filter := repositories.BrandBenefitFilter{
		BrandID:     brandID,
		UnlockType:  unlockType,
		BenefitType: benefitType,
		Page:        page,
		Limit:       limit,
	}

	result, err := uc.benefitRepo.GetActiveByBrandIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &dto.BrandBenefitListRes{
		Items:    mapper.MapBrandBenefits(result.Benefits),
		Metadata: shared_dto.BuildPaginationMetadata(shared_dto.PaginationQuery{Page: page, Limit: limit}, result.TotalCount),
	}, nil
}

func (uc *BrandBenefitUseCase) ListActiveBenefitsForUser(ctx context.Context, brandID uuid.UUID, query dto.GetBrandBenefitsQueryReq) (*dto.BrandBenefitListRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	var unlockType *benefitunlocktype.BenefitUnlockType
	if query.UnlockType != nil {
		val := benefitunlocktype.BenefitUnlockType(strings.ToLower(*query.UnlockType))
		unlockType = &val
	}

	var benefitType *benefittype.BenefitType
	if query.BenefitType != nil {
		val := benefittype.BenefitType(strings.ToLower(*query.BenefitType))
		benefitType = &val
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	filter := repositories.BrandBenefitFilter{
		BrandID:     brandID,
		UnlockType:  unlockType,
		BenefitType: benefitType,
		Page:        page,
		Limit:       limit,
	}

	result, err := uc.benefitRepo.GetActiveByBrandIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &dto.BrandBenefitListRes{
		Items:    mapper.MapBrandBenefits(result.Benefits),
		Metadata: shared_dto.BuildPaginationMetadata(shared_dto.PaginationQuery{Page: page, Limit: limit}, result.TotalCount),
	}, nil
}

func (uc *BrandBenefitUseCase) GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error) {
	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil {
		return nil, branderrors.ErrBenefitNotFound()
	}
	// Verify customer status via user loyalty helper (local check)
	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, benefit.BrandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrCustomerNotFound()
	}
	if benefit.Status != benefitstatus.Active {
		return nil, branderrors.ErrBenefitNotActive()
	}
	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandBenefitUseCase) ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID, brandID *uuid.UUID, page, limit int) (*dto.BenefitRedemptionListRes, error) {
	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	customerIDs := make([]uuid.UUID, 0, len(customers))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active {
			continue
		}
		customerIDs = append(customerIDs, customer.ID)
	}

	filter := repositories.BenefitRedemptionFilter{
		BrandCustomerIDs: customerIDs,
		BrandID:          brandID,
		Page:             page,
		Limit:            limit,
	}

	result, err := uc.redemptionRepo.GetByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.BenefitRedemptionRes, 0, len(result.Items))
	for _, redemption := range result.Items {
		items = append(items, mapper.MapBenefitRedemption(redemption))
	}

	return &dto.BenefitRedemptionListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit}, result.TotalCount),
	}, nil
}

func (uc *BrandBenefitUseCase) ListMyBenefits(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandBenefitsQueryReq) (*dto.MyBrandBenefitListRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrCustomerNotFound()
	}

	account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, branderrors.ErrActiveLoyaltyProgramRequired()
	}

	var unlockType *benefitunlocktype.BenefitUnlockType
	if query.UnlockType != nil {
		val := benefitunlocktype.BenefitUnlockType(strings.ToLower(*query.UnlockType))
		unlockType = &val
	}

	var benefitType *benefittype.BenefitType
	if query.BenefitType != nil {
		val := benefittype.BenefitType(strings.ToLower(*query.BenefitType))
		benefitType = &val
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	result, err := uc.benefitRepo.GetActiveByBrandIDPaginated(ctx, repositories.BrandBenefitFilter{
		BrandID:     brandID,
		UnlockType:  unlockType,
		BenefitType: benefitType,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	redemptions, err := uc.redemptionRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil {
		return nil, err
	}
	latestRedemptionByBenefit := make(map[uuid.UUID]*entities.BenefitRedemption, len(redemptions))
	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		current, ok := latestRedemptionByBenefit[redemption.BenefitID]
		if !ok || redemption.RedeemedAt.After(current.RedeemedAt) {
			latestRedemptionByBenefit[redemption.BenefitID] = redemption
		}
	}

	unlockedByBenefit := make(map[uuid.UUID]bool, len(result.Benefits))
	for _, benefit := range result.Benefits {
		unlockedByBenefit[benefit.ID] = isBenefitUnlockedForAccount(benefit, account)
	}

	return &dto.MyBrandBenefitListRes{
		CurrentTier:   mapper.MapLoyaltyTier(account.CurrentTier),
		CurrentPoints: account.CurrentPoints,
		Items:         mapper.MapMyBrandBenefits(result.Benefits, unlockedByBenefit, latestRedemptionByBenefit),
		Metadata:      shared_dto.BuildPaginationMetadata(shared_dto.PaginationQuery{Page: page, Limit: limit}, result.TotalCount),
	}, nil
}

// isBenefitUnlockedForAccount mirrors the unlock rules enforced by the redeem
// strategies: tier privileges need an equal or higher tier rank, point
// redemptions need enough points, and manual grants are never self-unlocked.
func isBenefitUnlockedForAccount(benefit *entities.BrandBenefit, account *entities.LoyaltyAccount) bool {
	if benefit == nil || account == nil {
		return false
	}

	switch benefit.UnlockType {
	case benefitunlocktype.TierPrivilege:
		if benefit.RequiredTier == nil || account.CurrentTier == nil {
			return false
		}
		return account.CurrentTier.Rank >= benefit.RequiredTier.Rank
	case benefitunlocktype.PointRedemption:
		if benefit.RequiredPoints == nil || *benefit.RequiredPoints <= 0 {
			return true
		}
		return account.CurrentPoints >= *benefit.RequiredPoints
	default:
		return false
	}
}

func (uc *BrandBenefitUseCase) UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error) {
	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil || benefit.BrandID != brandID {
		return nil, branderrors.ErrBenefitNotFound()
	}

	bStatus := benefitstatus.BenefitStatus(strings.ToLower(status))
	if bStatus != benefitstatus.Active && bStatus != benefitstatus.Inactive && bStatus != benefitstatus.Archived {
		return nil, branderrors.ErrBenefitInvalidStatus()
	}

	benefit.Status = bStatus
	if err := uc.benefitRepo.Update(ctx, benefit); err != nil {
		return nil, err
	}

	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandBenefitUseCase) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode benefitfeaturecode.BenefitFeatureCode) (bool, error) {
	if !featureCode.IsValid() {
		return false, branderrors.ErrInvalidBenefitFeatureCode(featureCode)
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return false, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return false, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil
	}

	cachedCodes, err := uc.featureCache.Get(ctx, userID, brandID)
	if err == nil && cachedCodes != nil {
		for _, code := range cachedCodes {
			if code == string(featureCode) {
				return true, nil
			}
		}
		return false, nil
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return false, err
	}

	account, _ := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)

	now := time.Now().UTC()
	var grantedCodes []string
	for _, benefit := range benefits {
		if benefit.BenefitType != benefittype.FeatureAccess || benefit.FeatureCode == nil {
			continue
		}

		strategy, ok := uc.strategies[benefit.UnlockType]
		if !ok {
			continue
		}

		hasAccess, err := strategy.CheckAccess(ctx, benefit, customer, account, now)
		if err != nil {
			return false, err
		}
		if hasAccess {
			grantedCodes = append(grantedCodes, string(*benefit.FeatureCode))
		}
	}

	if len(grantedCodes) > 0 {
		_ = uc.featureCache.Set(ctx, userID, brandID, grantedCodes)
	}

	for _, code := range grantedCodes {
		if code == string(featureCode) {
			return true, nil
		}
	}

	return false, nil
}

func (uc *BrandBenefitUseCase) RedeemBenefit(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error) {
	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil {
		return nil, branderrors.ErrBenefitNotFound()
	}
	brandID := benefit.BrandID

	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	if benefit.Status != benefitstatus.Active {
		return nil, branderrors.ErrBenefitNotActive()
	}

	strategy, ok := uc.strategies[benefit.UnlockType]
	if !ok {
		return nil, branderrors.ErrBenefitUnlockTypeNotSupported()
	}

	now := time.Now().UTC()
	var redemption *entities.BenefitRedemption

	if benefit.UnlockType == benefitunlocktype.PointRedemption {
		err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
			var redeemErr error
			redemption, redeemErr = strategy.Redeem(txCtx, benefit, customer, userID, now)
			return redeemErr
		})
	} else {
		redemption, err = strategy.Redeem(ctx, benefit, customer, userID, now)
	}

	if err != nil {
		return nil, err
	}

	if err := uc.redemptionRepo.Create(ctx, redemption); err != nil {
		return nil, err
	}

	_ = uc.featureCache.Invalidate(ctx, userID, brandID)

	return mapper.MapBenefitRedemption(redemption), nil
}
