package handler

import (
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	userUseCase usecase_interfaces.IUserUseCase
}

const (
	successUpdateUserStatus = "Cập nhật trạng thái tài khoản thành công"
)

func NewAdminHandler(uc usecase_interfaces.IUserUseCase) *AdminHandler {
	return &AdminHandler{
		userUseCase: uc,
	}
}

// UpdateUserStatus cập nhật trạng thái tài khoản người dùng
// @Summary Cập nhật trạng thái tài khoản người dùng
// @Description Cho phép admin khóa hoặc mở lại tài khoản member
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "ID người dùng"
// @Param body body dto.UpdateUserStatusReq true "Trạng thái tài khoản mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.UserRes} "Cập nhật trạng thái tài khoản thành công"
// @Router /api/v1/admin/users/{id}/status [patch]
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	var input dto.UpdateUserStatusReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.userUseCase.UpdateUserStatus(c.Request.Context(), adminUserID, targetUserID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successUpdateUserStatus, response)
	return nil
}
