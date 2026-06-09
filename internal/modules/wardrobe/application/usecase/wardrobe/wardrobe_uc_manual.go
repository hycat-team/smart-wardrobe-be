package wardrobe

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/colorutils"

	"github.com/google/uuid"
)

func (uc *WardrobeItemUseCase) ManualClassify(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, input dto.ManualClassifyReq) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, wardrobeerrors.ErrItemNotFound
	}

	if item.UserID != userID {
		return nil, wardrobeerrors.ErrUpdateItemForbidden
	}

	if item.Status == wardrobestatus.Sold {
		return nil, wardrobeerrors.ErrManualClassifySoldItem
	}

	category, err := uc.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, wardrobeerrors.ErrCategoryNotFound
	}

	tokens := fmt.Sprintf("[CAT:%s][COL:%s][STY:%s][MAT:%s][PAT:%s][FIT:%s][SEA:%s]",
		category.Slug, input.Color, input.Style, input.Material, input.Pattern, input.Fit, input.Seasonality)

	freeForm := fmt.Sprintf("Món đồ thời trang %s màu %s phong cách %s được làm từ %s với họa tiết %s, dáng %s thích hợp mặc vào %s.",
		category.Name, input.Color, input.Style, input.Material, input.Pattern, input.Fit, input.Seasonality)

	description := tokens + " " + freeForm

	richTextContext := fmt.Sprintf(
		"Danh mục trang phục: %s, Thuộc tính màu sắc: %s, Định hình phong cách thiết kế: %s, Chất liệu: %s, Họa tiết: %s, Kiểu dáng: %s, Mùa phù hợp: %s. Mô tả chi tiết: %s",
		category.Name,
		input.Color,
		input.Style,
		input.Material,
		input.Pattern,
		input.Fit,
		input.Seasonality,
		freeForm,
	)

	embeddings, err := uc.aiService.GenerateEmbeddings(ctx, []string{richTextContext})
	if err != nil || len(embeddings) == 0 {
		return nil, wardrobeerrors.ErrProcessFashionTextFailed
	}
	embedding := embeddings[0]

	item.CategoryID = &category.ID
	item.Color = &input.Color
	item.Style = &input.Style
	item.Material = &input.Material
	item.Pattern = &input.Pattern
	item.Fit = &input.Fit
	item.Seasonality = &input.Seasonality
	item.Description = &description
	item.Price = input.Price
	item.Embedding = entities.Vector(embedding)
	item.Status = wardrobestatus.InWardrobe

	if h, s, l, hex, ok := colorutils.ResolveHSLFromColorName(input.Color); ok {
		item.ColorHex = &hex
		item.ColorHue = &h
		item.ColorSaturation = &s
		item.ColorLightness = &l
	}

	err = uc.wardrobeRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	payload := dto.WardrobeEventPayload{
		ItemID: item.ID,
		UserID: item.UserID,
		Action: eventconstants.ActionCreated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)

	return mapper.MapToWardrobeItemRes(item), nil
}
