package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var _ shared_dto.PaginationQuery

const (
	msgOutfitGetUploadSignatureSuccess = "Lấy chữ ký tải ảnh bìa bộ phối đồ thành công"
	msgOutfitSaveSuccess               = "Tạo bộ phối đồ thành công"
	msgOutfitUpdateSuccess             = "Cập nhật bộ phối đồ thành công"
	msgOutfitGetOutfitsSuccess         = "Lấy danh sách bộ phối đồ thành công"
	msgOutfitGetByIDSuccess            = "Lấy chi tiết bộ phối đồ thành công"
	msgOutfitDeleteSuccess             = "Xóa bộ phối đồ thành công"
)

type OutfitHandler struct {
	outfitUseCase usecase_interfaces.IOutfitUseCase
}

func NewOutfitHandler(uc usecase_interfaces.IOutfitUseCase) *OutfitHandler {
	return &OutfitHandler{
		outfitUseCase: uc,
	}
}

// GetUploadSignature get secure signature for uploading cover image
// @Summary Lấy chữ ký tải ảnh bìa bộ phối đồ
// @Description Lấy thông tin chữ ký bảo mật từ Cloudinary để upload ảnh cover của outfit từ Client
// @Tags Outfits
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.UploadSignatureResult} "Lấy chữ ký thành công"
// @Router /api/v1/outfits/upload-signature [get]
func (h *OutfitHandler) GetUploadSignature(c *gin.Context) error {
	signatureRes, err := h.outfitUseCase.GetUploadSignature(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgOutfitGetUploadSignatureSuccess, signatureRes)
	return nil
}

// SaveOutfit creates a new outfit coordinate canvas
// @Summary Tạo bộ phối đồ mới
// @Description Lưu bộ phối đồ tự thiết kế cùng danh sách trang phục kèm tọa độ kéo thả 2D và layer order.
// @Tags Outfits
// @Accept json
// @Produce json
// @Param body body dto.SaveOutfitReq true "Tọa độ 2D và thông tin phối đồ"
// @Success 201 {object} shared_pres.APIResponse{data=dto.OutfitRes} "Tạo bộ phối đồ thành công"
// @Router /api/v1/outfits [post]
func (h *OutfitHandler) SaveOutfit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.SaveOutfitReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.outfitUseCase.SaveOutfit(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgOutfitSaveSuccess, response)
	return nil
}

// UpdateOutfit updates coordinates of outfits on canvas board
// @Summary Cập nhật bộ phối đồ
// @Description Cập nhật lại thông tin hoặc điều chỉnh tọa độ kéo thả của bộ phối đồ.
// @Tags Outfits
// @Accept json
// @Produce json
// @Param id path string true "ID bộ phối đồ"
// @Param body body dto.SaveOutfitReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.OutfitRes} "Cập nhật thành công"
// @Router /api/v1/outfits/{id} [put]
func (h *OutfitHandler) UpdateOutfit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return wardrobeerrors.ErrInvalidOutfitIDFormat
	}

	var input dto.SaveOutfitReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.outfitUseCase.UpdateOutfit(c.Request.Context(), userID, id, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgOutfitUpdateSuccess, response)
	return nil
}

// GetOutfits gets all outfits of logged in user
// @Summary Lấy danh sách bộ phối đồ của tôi
// @Description Trả về danh sách tất cả các bộ phối đồ do tôi thiết kế.
// @Tags Outfits
// @Produce json
// @Param page query int false "Số trang (mặc định: 1)"
// @Param limit query int false "Số lượng phần tử trên trang (mặc định: 20)"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.OutfitRes]} "Danh sách bộ phối đồ"
// @Router /api/v1/me/outfits [get]
func (h *OutfitHandler) GetOutfits(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var query dto.GetOutfitsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.outfitUseCase.GetOutfits(c.Request.Context(), userID, query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgOutfitGetOutfitsSuccess, response)
	return nil
}

// GetOutfitByID gets full details of a specific outfit
// @Summary Chi tiết bộ phối đồ và tọa độ canvas
// @Description Trả về chi tiết bộ phối đồ kèm danh sách trang phục đầy đủ và tọa độ 2D để render lên canvas.
// @Tags Outfits
// @Produce json
// @Param id path string true "ID bộ phối đồ"
// @Success 200 {object} shared_pres.APIResponse{data=dto.OutfitRes} "Chi tiết bộ phối đồ"
// @Router /api/v1/outfits/{id} [get]
func (h *OutfitHandler) GetOutfitByID(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return wardrobeerrors.ErrInvalidOutfitIDFormat
	}

	response, err := h.outfitUseCase.GetOutfitByID(c.Request.Context(), userID, id)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgOutfitGetByIDSuccess, response)
	return nil
}

// DeleteOutfit deletes custom outfit coordinates record
// @Summary Xóa bộ phối đồ tự thiết kế
// @Description Xóa bộ phối đồ khỏi bộ sưu tập cá nhân.
// @Tags Outfits
// @Produce json
// @Param id path string true "ID bộ phối đồ"
// @Success 200 {object} shared_pres.APIResponse "Xóa bộ phối đồ thành công"
// @Router /api/v1/outfits/{id} [delete]
func (h *OutfitHandler) DeleteOutfit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return wardrobeerrors.ErrInvalidOutfitIDFormat
	}

	err = h.outfitUseCase.DeleteOutfit(c.Request.Context(), userID, id)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgOutfitDeleteSuccess, nil)
	return nil
}

