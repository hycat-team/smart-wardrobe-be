package outfit

import (
	"context"
	"fmt"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
)

type OutfitUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	outfitRepo      repositories.IOutfitRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract contract.IUserSubscriptionContract
}

func NewOutfitUseCase(
	cfg *config.Config,
	l logger.Interface,
	outfitRepo repositories.IOutfitRepository,
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract contract.IUserSubscriptionContract,
) uc_interfaces.IOutfitUseCase {
	return &OutfitUseCase{
		cfg:             cfg,
		logger:          l,
		outfitRepo:      outfitRepo,
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
	}
}

func (uc *OutfitUseCase) SaveOutfit(ctx context.Context, userID uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error) {
	// 1. Kiểm tra giới hạn số lượng Outfit của gói cước
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	existingOutfits, err := uc.outfitRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(existingOutfits) >= subOverview.MaxOutfits {
		return nil, errorcode.NewForbidden(fmt.Sprintf("Bạn đã đạt giới hạn tối đa phối đồ (%d bộ đồ) của gói dịch vụ hiện tại. Vui lòng nâng cấp lên gói Premium để mở rộng giới hạn phối đồ!", subOverview.MaxOutfits))
	}

	// 2. Kiểm tra các Wardrobe Items truyền lên có tồn tại và thuộc về User không
	itemIDs := make([]uuid.UUID, len(input.Items))
	for idx, itemReq := range input.Items {
		itemIDs[idx] = itemReq.WardrobeItemID
	}

	verifiedItems, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}

	verifiedMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range verifiedItems {
		if item.UserID == userID {
			verifiedMap[item.ID] = item
		}
	}

	// Xác thực 100% các items hợp lệ
	outfitItems := make([]*entities.OutfitItem, len(input.Items))
	for idx, itemReq := range input.Items {
		verifiedItem, ok := verifiedMap[itemReq.WardrobeItemID]
		if !ok {
			return nil, errorcode.NewBadRequest(fmt.Sprintf("Trang phục ID %s không tồn tại hoặc không thuộc tủ đồ của bạn.", itemReq.WardrobeItemID))
		}

		outfitItems[idx] = &entities.OutfitItem{
			ItemID:     itemReq.WardrobeItemID,
			Wardrobe:   verifiedItem,
			PositionX:  itemReq.PositionX,
			PositionY:  itemReq.PositionY,
			Scale:      itemReq.Scale,
			LayerOrder: itemReq.LayerOrder,
		}
	}

	// 3. Tiến hành Transaction tạo Outfit
	outfit := &entities.Outfit{
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		CoverImageUrl: input.CoverImageUrl,
		Status:        1,
	}

	err = uc.outfitRepo.CreateWithItems(ctx, outfit, outfitItems)
	if err != nil {
		return nil, err
	}

	return mapper.MapToOutfitRes(outfit, outfitItems), nil
}

func (uc *OutfitUseCase) UpdateOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error) {
	// 1. Kiểm tra Outfit có tồn tại và thuộc về User không
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, errorcode.NewNotFound("Không tìm thấy bộ phối đồ tương ứng.")
	}

	// 2. Kiểm tra các Wardrobe Items truyền lên
	itemIDs := make([]uuid.UUID, len(input.Items))
	for idx, itemReq := range input.Items {
		itemIDs[idx] = itemReq.WardrobeItemID
	}

	verifiedItems, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}

	verifiedMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range verifiedItems {
		if item.UserID == userID {
			verifiedMap[item.ID] = item
		}
	}

	outfitItems := make([]*entities.OutfitItem, len(input.Items))
	for idx, itemReq := range input.Items {
		verifiedItem, ok := verifiedMap[itemReq.WardrobeItemID]
		if !ok {
			return nil, errorcode.NewBadRequest(fmt.Sprintf("Trang phục ID %s không tồn tại hoặc không thuộc tủ đồ của bạn.", itemReq.WardrobeItemID))
		}

		outfitItems[idx] = &entities.OutfitItem{
			OutfitID:   id,
			ItemID:     itemReq.WardrobeItemID,
			Wardrobe:   verifiedItem,
			PositionX:  itemReq.PositionX,
			PositionY:  itemReq.PositionY,
			Scale:      itemReq.Scale,
			LayerOrder: itemReq.LayerOrder,
		}
	}

	// 3. Tiến hành Cập nhật
	outfit.Name = input.Name
	outfit.Description = input.Description
	outfit.CoverImageUrl = input.CoverImageUrl

	err = uc.outfitRepo.UpdateWithItems(ctx, outfit, outfitItems)
	if err != nil {
		return nil, err
	}

	return mapper.MapToOutfitRes(outfit, outfitItems), nil
}

func (uc *OutfitUseCase) GetOutfits(ctx context.Context, userID uuid.UUID) ([]*dto.OutfitRes, error) {
	outfits, err := uc.outfitRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.OutfitRes, len(outfits))
	for idx, outfit := range outfits {
		resList[idx] = mapper.MapToOutfitRes(outfit, nil)
	}

	return resList, nil
}

func (uc *OutfitUseCase) GetOutfitByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.OutfitRes, error) {
	outfit, items, err := uc.outfitRepo.GetDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, errorcode.NewNotFound("Không tìm thấy bộ phối đồ tương ứng.")
	}

	return mapper.MapToOutfitRes(outfit, items), nil
}

func (uc *OutfitUseCase) DeleteOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if outfit == nil || outfit.UserID != userID {
		return errorcode.NewNotFound("Không tìm thấy bộ phối đồ tương ứng.")
	}

	return uc.outfitRepo.DeleteOutfit(ctx, id)
}
