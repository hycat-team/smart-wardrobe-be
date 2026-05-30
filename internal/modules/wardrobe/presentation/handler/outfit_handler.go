package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OutfitHandler struct {
	outfitUseCase usecase_interfaces.IOutfitUseCase
}

func NewOutfitHandler(uc usecase_interfaces.IOutfitUseCase) *OutfitHandler {
	return &OutfitHandler{
		outfitUseCase: uc,
	}
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

	shared_pres.Created(c, "Tạo bộ phối đồ thành công", response)
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
		return errorcode.NewBadRequest("Định dạng ID bộ phối đồ không hợp lệ.")
	}

	var input dto.SaveOutfitReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.outfitUseCase.UpdateOutfit(c.Request.Context(), userID, id, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Cập nhật bộ phối đồ thành công", response)
	return nil
}

// GetOutfits gets all outfits of logged in user
// @Summary Lấy danh sách bộ phối đồ của tôi
// @Description Trả về danh sách tất cả các bộ phối đồ do tôi thiết kế.
// @Tags Outfits
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.OutfitRes} "Danh sách bộ phối đồ"
// @Router /api/v1/outfits [get]
func (h *OutfitHandler) GetOutfits(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.outfitUseCase.GetOutfits(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy danh sách bộ phối đồ thành công", response)
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
		return errorcode.NewBadRequest("Định dạng ID bộ phối đồ không hợp lệ.")
	}

	response, err := h.outfitUseCase.GetOutfitByID(c.Request.Context(), userID, id)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy chi tiết bộ phối đồ thành công", response)
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
		return errorcode.NewBadRequest("Định dạng ID bộ phối đồ không hợp lệ.")
	}

	err = h.outfitUseCase.DeleteOutfit(c.Request.Context(), userID, id)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Xóa bộ phối đồ thành công", nil)
	return nil
}
