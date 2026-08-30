package item

import (
	"context"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/votetype"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

func (uc *BrandItemUseCase) mapToRes(ctx context.Context, item *entities.BrandItem) *dto.BrandItemRes {
	if item == nil {
		return nil
	}
	var fi *fashion_dto.FashionItemRes
	if item.FashionItemID != nil {
		fi, _ = uc.fashionContract.GetFashionItem(ctx, *item.FashionItemID)
	}
	return mapper.MapToBrandItemRes(item, fi)
}

func (uc *BrandItemUseCase) mapSliceToRes(ctx context.Context, items []*entities.BrandItem) []*dto.BrandItemRes {
	if len(items) == 0 {
		return []*dto.BrandItemRes{}
	}
	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionItemMap := map[uuid.UUID]*fashion_dto.FashionItemRes{}
	if len(fashionIDs) > 0 {
		fashionContractItems, _ := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
		fashionItemMap = sliceutils.IndexBy(fashionContractItems, func(f *fashion_dto.FashionItemRes) uuid.UUID { return f.ID })
	}

	res := make([]*dto.BrandItemRes, len(items))
	for i, item := range items {
		var fi *fashion_dto.FashionItemRes
		if item.FashionItemID != nil {
			fi = fashionItemMap[*item.FashionItemID]
		}
		res[i] = mapper.MapToBrandItemRes(item, fi)
	}
	return res
}

func (uc *BrandItemUseCase) mapSliceToBriefRes(ctx context.Context, items []*entities.BrandItem) []*dto.BrandItemBriefRes {
	if len(items) == 0 {
		return []*dto.BrandItemBriefRes{}
	}

	brandIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		brandIDs = append(brandIDs, item.BrandID)
	}
	brandIDs = sliceutils.UniqueUUIDs(brandIDs)
	brandMap := make(map[uuid.UUID]*entities.Brand, len(brandIDs))
	if len(brandIDs) > 0 {
		brands, _ := uc.brandRepo.GetByIDs(ctx, brandIDs)
		for _, b := range brands {
			brandMap[b.ID] = b
		}
	}

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionBriefMap := make(map[uuid.UUID]*fashion_dto.FashionItemBriefRes)
	if len(fashionIDs) > 0 {
		briefs, _ := uc.fashionContract.ListFashionItemsBrief(ctx, fashionIDs)
		for _, b := range briefs {
			fashionBriefMap[b.ID] = b
		}
	}

	res := make([]*dto.BrandItemBriefRes, len(items))
	for i, item := range items {
		var fi *fashion_dto.FashionItemBriefRes
		if item.FashionItemID != nil {
			fi = fashionBriefMap[*item.FashionItemID]
		}
		res[i] = mapper.MapBrandItemBrief(item, brandMap[item.BrandID], fi)
	}
	return res
}

func isValidVoteType(vote votetype.VoteType) bool {
	switch vote {
	case votetype.Like, votetype.Dislike, votetype.WouldBuy, votetype.NotInterested:
		return true
	default:
		return false
	}
}
