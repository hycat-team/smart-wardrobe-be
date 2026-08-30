package item

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

func (uc *BrandItemUseCase) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, []*fashion_dto.FashionItemRes, error) {
	user, err := uc.userBriefContract.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return []*dto.BrandItemRes{}, []*fashion_dto.FashionItemRes{}, nil
	}

	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	brandItemsByBrandID := make(map[uuid.UUID][]*entities.BrandItem, len(activeBrandIDs))
	for _, item := range brandItems {
		brandItemsByBrandID[item.BrandID] = append(brandItemsByBrandID[item.BrandID], item)
	}

	eligibleDTOS := make([]*dto.BrandItemRes, 0, len(brandItems))
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
				return nil, nil, err
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

	fashionIDs := make([]uuid.UUID, 0, len(eligibleItems))
	for _, item := range eligibleItems {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionItemMap := map[uuid.UUID]*fashion_dto.FashionItemRes{}
	if len(fashionIDs) > 0 {
		fashionContractItems, err := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
		if err != nil {
			return nil, nil, err
		}
		fashionItemMap = sliceutils.IndexBy(fashionContractItems, func(f *fashion_dto.FashionItemRes) uuid.UUID { return f.ID })
	}

	for _, item := range eligibleItems {
		var fi *fashion_dto.FashionItemRes
		if item.FashionItemID != nil {
			fi = fashionItemMap[*item.FashionItemID]
		}
		eligibleDTOS = append(eligibleDTOS, mapper.MapToBrandItemRes(item, fi))
	}

	fashionItems := make([]*fashion_dto.FashionItemRes, len(eligibleItems))
	for i := range eligibleItems {
		if eligibleItems[i].FashionItemID != nil {
			if fi := fashionItemMap[*eligibleItems[i].FashionItemID]; fi != nil {
				fashionItems[i] = fi
			}
		}
	}

	return eligibleDTOS, fashionItems, nil
}

func (uc *BrandItemUseCase) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *fashion_dto.FashionItemRes, error) {
	user, err := uc.userBriefContract.GetByID(ctx, userID)
	if err != nil {
		return false, nil, nil, err
	}
	if user == nil {
		return false, nil, nil, nil
	}

	brandItem, err := uc.brandItemRepo.GetByFashionItemID(ctx, fashionItemID)
	if err != nil {
		return false, nil, nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return false, nil, nil, nil
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandItem.BrandID)
	if err != nil {
		return false, nil, nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil, nil, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandItem.BrandID, userID)
	if err != nil {
		return false, nil, nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil, nil, nil
	}

	if brandItem.ItemType == branditemtype.Sample {
		hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandItem.BrandID, benefitfeaturecode.SampleMixAccess)
		if err != nil {
			return false, nil, nil, err
		}
		if !hasSampleAccess {
			return false, nil, nil, nil
		}
	}

	fi, _ := uc.fashionContract.GetFashionItem(ctx, *brandItem.FashionItemID)
	return true, mapper.MapToBrandItemRes(brandItem, fi), fi, nil
}

func (uc *BrandItemUseCase) ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(fashionItemIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}

	brandItems, err := uc.brandItemRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]bool, len(fashionItemIDs))
	for _, bi := range brandItems {
		if bi.FashionItemID != nil && bi.Status == branditemstatus.Active {
			result[*bi.FashionItemID] = true
		}
	}
	return result, nil
}
