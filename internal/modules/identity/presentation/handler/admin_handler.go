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
	successGetUsers         = "Lấy danh sách tài khoản thành công"
)

func NewAdminHandler(uc usecase_interfaces.IUserUseCase) *AdminHandler {
	return &AdminHandler{
		userUseCase: uc,
	}
}

// UpdateUserStatus cập nhật trạng thái tài khoản người dùng
// @Summary Cập nhật trạng thái tài khoản người dùng
// @Description Cho phép admin khóa hoặc mở lại tài khoản member. Khi khóa sang inactive, hệ thống chỉ revoke refresh token; access token hiện tại vẫn còn hiệu lực đến hết TTL theo cơ chế JWT stateless.
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

// GetUsers lấy danh sách người dùng cho Admin
// @Summary Lấy danh sách người dùng
// @Description Cho phép admin lấy danh sách người dùng phân trang, tìm kiếm và lọc theo trạng thái/phân quyền.
// @Tags Admin
// @Accept json
// @Produce json
// @Param roleSlug query string false "Phân quyền (e.g. member, admin)"
// @Param isActive query boolean false "Trạng thái hoạt động"
// @Param q query string false "Từ khóa tìm kiếm (username, email, họ tên)"
// @Param page query int false "Số trang"
// @Param limit query int false "Số lượng phần tử mỗi trang"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminUserListRes} "Lấy danh sách tài khoản thành công"
// @Router /api/v1/admin/users [get]
func (h *AdminHandler) GetUsers(c *gin.Context) error {
	var query dto.GetUsersQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.userUseCase.GetUsersForAdmin(c.Request.Context(), query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetUsers, response)
	return nil
}

