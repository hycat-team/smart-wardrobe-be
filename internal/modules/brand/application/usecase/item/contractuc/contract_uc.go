package contractuc

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

type BrandItemContractUseCase struct {
	brandRepo     repositories.IBrandRepository
	customerRepo  repositories.IBrandCustomerRepository
	brandItemRepo repositories.IBrandItemRepository
	benefitUC     uc_interfaces.IBrandBenefitUseCase
}

func NewBrandItemContractUseCase(
	brandRepo repositories.IBrandRepository,
	customerRepo repositories.IBrandCustomerRepository,
	brandItemRepo repositories.IBrandItemRepository,
	benefitUC uc_interfaces.IBrandBenefitUseCase,
) uc_interfaces.IBrandItemContractUseCase {
	return &BrandItemContractUseCase{
		brandRepo:     brandRepo,
		customerRepo:  customerRepo,
		brandItemRepo: brandItemRepo,
		benefitUC:     benefitUC,
	}
}

func (uc *BrandItemContractUseCase) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemContractDTO, error) {
	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	activeBrandIDs := make([]uuid.UUID, 0, len(customers))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active || customer.Brand == nil || customer.Brand.Status != brandstatus.Active {
			continue
		}
		activeBrandIDs = append(activeBrandIDs, customer.BrandID)
	}

	brandItems, err := uc.brandItemRepo.GetByBrandIDs(ctx, activeBrandIDs)
	if err != nil {
		return nil, err
	}
	brandItemsByBrandID := make(map[uuid.UUID][]*entities.BrandItem, len(activeBrandIDs))
	for _, item := range brandItems {
		brandItemsByBrandID[item.BrandID] = append(brandItemsByBrandID[item.BrandID], item)
	}

	eligibleItems := make([]*entities.BrandItem, 0, len(brandItems))
	sampleAccessByBrandID := make(map[uuid.UUID]bool, len(activeBrandIDs))
	sampleAccessChecked := make(map[uuid.UUID]struct{}, len(activeBrandIDs))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active || customer.Brand == nil || customer.Brand.Status != brandstatus.Active {
			continue
		}

		if _, checked := sampleAccessChecked[customer.BrandID]; !checked {
			hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, customer.BrandID, benefitfeaturecode.SampleMixAccess)
			if err != nil {
				return nil, err
			}
			sampleAccessByBrandID[customer.BrandID] = hasSampleAccess
			sampleAccessChecked[customer.BrandID] = struct{}{}
		}
		for _, item := range brandItemsByBrandID[customer.BrandID] {
			if item.Status != branditemstatus.Active {
				continue
			}

			canUseItem := item.ItemType == branditemtype.Product ||
				(item.ItemType == branditemtype.Sample && sampleAccessByBrandID[item.BrandID])

			if canUseItem {
				eligibleItems = append(eligibleItems, item)
			}
		}
	}

	brandIDs := make([]uuid.UUID, 0, len(eligibleItems))
	for _, item := range eligibleItems {
		brandIDs = append(brandIDs, item.BrandID)
	}
	brandIDs = sliceutils.UniqueUUIDs(brandIDs)
	brandNameMap := make(map[uuid.UUID]string, len(brandIDs))
	if len(brandIDs) > 0 {
		brands, _ := uc.brandRepo.GetByIDs(ctx, brandIDs)
		for _, b := range brands {
			brandNameMap[b.ID] = b.Name
		}
	}

	return mapper.MapToBrandItemContractDTOList(eligibleItems, brandNameMap), nil
}

func (uc *BrandItemContractUseCase) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemContractDTO, error) {
	brandItem, err := uc.brandItemRepo.GetByFashionItemID(ctx, fashionItemID)
	if err != nil {
		return false, nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return false, nil, nil
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandItem.BrandID)
	if err != nil {
		return false, nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandItem.BrandID, userID)
	if err != nil {
		return false, nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil, nil
	}

	if brandItem.ItemType == branditemtype.Sample {
		hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandItem.BrandID, benefitfeaturecode.SampleMixAccess)
		if err != nil {
			return false, nil, err
		}
		if !hasSampleAccess {
			return false, nil, nil
		}
	}

	return true, mapper.MapToBrandItemContractDTO(brandItem, brand.Name), nil
}

func (uc *BrandItemContractUseCase) ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(fashionItemIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}
	items, err := uc.brandItemRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	activeMap := make(map[uuid.UUID]bool, len(fashionItemIDs))
	for _, id := range fashionItemIDs {
		activeMap[id] = false
	}
	activeStatus := branditemstatus.Active
	for _, item := range items {
		if item.FashionItemID != nil && item.Status == activeStatus {
			activeMap[*item.FashionItemID] = true
		}
	}
	return activeMap, nil
}

func (uc *BrandItemContractUseCase) UpdateBrandItemStatusDirect(ctx context.Context, brandItemID uuid.UUID, status branditemstatus.BrandItemStatus) error {
	item, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	item.Status = status
	return uc.brandItemRepo.Update(ctx, item)
}

func (uc *BrandItemContractUseCase) GetBrandItemByIDDirect(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemContractDTO, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	brandName := ""
	if item.Brand != nil {
		brandName = item.Brand.Name
	} else {
		brand, _ := uc.brandRepo.GetByID(ctx, item.BrandID)
		if brand != nil {
			brandName = brand.Name
		}
	}
	return mapper.MapToBrandItemContractDTO(item, brandName), nil
}

func (uc *BrandItemContractUseCase) GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemContractDTO, error) {
	if len(fashionItemIDs) == 0 {
		return []*dto.BrandItemContractDTO{}, nil
	}
	items, err := uc.brandItemRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	activeStatus := branditemstatus.Active
	filtered := make([]*entities.BrandItem, 0, len(items))
	for _, item := range items {
		if item.Status == activeStatus {
			filtered = append(filtered, item)
		}
	}

	brandIDs := make([]uuid.UUID, 0, len(filtered))
	for _, item := range filtered {
		brandIDs = append(brandIDs, item.BrandID)
	}
	brandIDs = sliceutils.UniqueUUIDs(brandIDs)
	brandNameMap := make(map[uuid.UUID]string, len(brandIDs))
	if len(brandIDs) > 0 {
		brands, _ := uc.brandRepo.GetByIDs(ctx, brandIDs)
		for _, b := range brands {
			brandNameMap[b.ID] = b.Name
		}
	}

	return mapper.MapToBrandItemContractDTOList(filtered, brandNameMap), nil
}

func (uc *BrandItemContractUseCase) GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemContractDTO, error) {
	if len(brandItemIDs) == 0 {
		return []*dto.BrandItemContractDTO{}, nil
	}
	items, err := uc.brandItemRepo.GetByIDs(ctx, brandItemIDs)
	if err != nil {
		return nil, err
	}

	brandIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		brandIDs = append(brandIDs, item.BrandID)
	}
	brandIDs = sliceutils.UniqueUUIDs(brandIDs)
	brandNameMap := make(map[uuid.UUID]string, len(brandIDs))
	if len(brandIDs) > 0 {
		brands, _ := uc.brandRepo.GetByIDs(ctx, brandIDs)
		for _, b := range brands {
			brandNameMap[b.ID] = b.Name
		}
	}

	return mapper.MapToBrandItemContractDTOList(items, brandNameMap), nil
}
