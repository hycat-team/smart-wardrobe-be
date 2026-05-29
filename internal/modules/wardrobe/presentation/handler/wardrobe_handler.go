package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	_ "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
)

type WardrobeHandler struct {
	wardrobeUseCase usecase_interfaces.IWardrobeUseCase
}

func NewWardrobeHandler(uc usecase_interfaces.IWardrobeUseCase) *WardrobeHandler {
	return &WardrobeHandler{
		wardrobeUseCase: uc,
	}
}

// GetUploadSignature get secure cloudinary signature for item upload
// @Summary Lấy chữ ký tải ảnh trang phục
// @Description Lấy chữ ký bảo mật từ Cloudinary để client tải trực tiếp ảnh trang phục lên
// @Tags Wardrobe
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult} "Chữ ký và thông tin upload"
// @Router /api/v1/wardrobe-items/upload-signature [get]
func (h *WardrobeHandler) GetUploadSignature(c *gin.Context) error {
	signatureRes, err := h.wardrobeUseCase.GetUploadSignature(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy chữ ký tải ảnh trang phục thành công", signatureRes)
	return nil
}

// CreateWardrobeItem create new wardrobe item with AI analysis
// @Summary Thêm trang phục mới
// @Description Tải lên trang phục mới, tự động phân tích và số hóa gu thời trang bằng trí tuệ nhân tạo
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param body body dto.CreateWardrobeItemReq true "Thông tin trang phục"
// @Success 201 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Thông tin trang phục sau khi lưu trữ"
// @Router /api/v1/wardrobe-items [post]
func (h *WardrobeHandler) CreateWardrobeItem(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.CreateWardrobeItemReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.CreateWardrobeItem(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Created(c, "Tải lên và phân tích trang phục thành công", response)
	return nil
}
