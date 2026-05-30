package usecase

import (
	"context"
	"fmt"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

type WardrobeUseCase struct {
	cfg          *config.Config
	wardrobeRepo repositories.IWardrobeItemRepository
	categoryRepo repositories.ICategoryRepository
	mediaService media.IMediaService
	aiService    ai.IAIService
}

func NewWardrobeUseCase(
	cfg *config.Config,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryRepo repositories.ICategoryRepository,
	mediaService media.IMediaService,
	aiService ai.IAIService,
) uc_interfaces.IWardrobeUseCase {
	return &WardrobeUseCase{
		cfg:          cfg,
		wardrobeRepo: wardrobeRepo,
		categoryRepo: categoryRepo,
		mediaService: mediaService,
		aiService:    aiService,
	}
}

func (uc *WardrobeUseCase) GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error) {
	folder := uc.cfg.Cloudinary.ItemFolder
	if folder == "" {
		folder = "smart_wardrobe/items"
	}
	params := shared_dto.UploadSignatureParams{
		Folder: folder,
	}
	return uc.mediaService.GenerateUploadSignature(ctx, params)
}

func (uc *WardrobeUseCase) CreateWardrobeItem(ctx context.Context, userID uuid.UUID, input dto.CreateWardrobeItemReq) (*dto.WardrobeItemRes, error) {
	category, err := uc.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy danh mục trang phục tương ứng.")
	}

	aiMeta, err := uc.aiService.AnalyzeFashionImage(ctx, input.ImageUrl)
	if err != nil {
		return nil, err
	}

	richTextContext := fmt.Sprintf(
		"Danh mục trang phục: %s, Thuộc tính màu sắc: %s, Định hình phong cách thiết kế: %s, Chất liệu: %s, Họa tiết: %s, Kiểu dáng: %s, Mùa phù hợp: %s. Mô tả chi tiết: %s",
		category.Name,
		aiMeta.Color,
		aiMeta.Style,
		aiMeta.Material,
		aiMeta.Pattern,
		aiMeta.Fit,
		aiMeta.Seasonality,
		aiMeta.Description,
	)

	embeddings, err := uc.aiService.GenerateEmbeddings(ctx, []string{richTextContext})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, errorcode.NewInternalError("Không thể sinh mã hóa vector cho trang phục.")
	}
	embedding := embeddings[0]

	item := &entities.WardrobeItem{
		UserID:        userID,
		CategoryID:    input.CategoryID,
		ImageUrl:      input.ImageUrl,
		ImagePublicID: input.ImagePublicID,
		Color:         &aiMeta.Color,
		Style:         &aiMeta.Style,
		Material:      &aiMeta.Material,
		Pattern:       &aiMeta.Pattern,
		Fit:           &aiMeta.Fit,
		Seasonality:   &aiMeta.Seasonality,
		Description:   &aiMeta.Description,
		Embedding:     entities.Vector(embedding),
		Status:        wardrobestatus.InWardrobe,
	}

	err = uc.wardrobeRepo.Create(ctx, item)
	if err != nil {
		return nil, err
	}

	item.Category = category
	return mapper.MapToWardrobeItemRes(item), nil
}
