package handler

import (
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
)

type MeHandler struct {
	userUseCase usecase_interfaces.IUserUseCase
}

func NewMeHandler(uc usecase_interfaces.IUserUseCase) *MeHandler {
	return &MeHandler{
		userUseCase: uc,
	}
}

// GetCurrentUser get current user profile
// @Summary Lấy thông tin cá nhân
// @Description Lấy thông tin chi tiết tài khoản của người dùng hiện tại đang đăng nhập
// @Tags Me
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UserRes} "Thông tin người dùng"
// @Router /api/v1/me [get]
func (h *MeHandler) GetCurrentUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	userRes, err := h.userUseCase.GetByID(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy thông tin cá nhân thành công", userRes)
	return nil
}

// UpdateCurrentUser update current user profile
// @Summary Cập nhật thông tin cá nhân
// @Description Cập nhật các trường thông tin của người dùng hiện tại đang đăng nhập
// @Tags Me
// @Accept json
// @Produce json
// @Param body body dto.UpdateProfileReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.UserRes} "Thông tin người dùng sau khi cập nhật"
// @Router /api/v1/me [put]
func (h *MeHandler) UpdateCurrentUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.UpdateProfileReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.userUseCase.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Cập nhật thông tin cá nhân thành công", response)
	return nil
}

// ChangePassword change password
// @Summary Đổi mật khẩu
// @Description Thay đổi mật khẩu cho người dùng hiện tại đang đăng nhập
// @Tags Me
// @Accept json
// @Produce json
// @Param body body dto.ChangePasswordReq true "Mật khẩu cũ và mới"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/me/change-password [put]
func (h *MeHandler) ChangePassword(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.ChangePasswordReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, err = h.userUseCase.ChangePassword(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Đổi mật khẩu thành công", nil)
	return nil
}
