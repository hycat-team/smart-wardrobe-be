package handler

import (
	admin_dto "smart-wardrobe-be/internal/modules/admin/application/dto"
	admin_contract "smart-wardrobe-be/internal/modules/admin/presentation/contract"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	"smart-wardrobe-be/internal/shared/domain/constants/adminaudit"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	userUseCase      usecase_interfaces.IUserUseCase
	auditLogContract admin_contract.IAuditLogContract
}

const (
	successUpdateUserStatus   = "Cập nhật trạng thái tài khoản thành công"
	successGetUsers           = "Lấy danh sách tài khoản thành công"
	successGetUserDetail      = "Lấy chi tiết tài khoản người dùng thành công"
	successCreateUser         = "Tạo tài khoản người dùng thành công"
	successUpdateUserRole     = "Cập nhật vai trò tài khoản thành công"
	successDeactivateUser     = "Vô hiệu hóa tài khoản người dùng thành công"
	successListUserSessions   = "Lấy danh sách phiên đăng nhập thành công"
	successRevokeUserSessions = "Thu hồi toàn bộ phiên đăng nhập thành công"
)

func NewAdminHandler(
	uc usecase_interfaces.IUserUseCase,
	auditLogContract admin_contract.IAuditLogContract,
) *AdminHandler {
	return &AdminHandler{
		userUseCase:      uc,
		auditLogContract: auditLogContract,
	}
}

// GetUserDetail lấy chi tiết tài khoản người dùng
// @Summary Chi tiết tài khoản người dùng
// @Description Lấy thông tin chi tiết tài khoản người dùng và trạng thái gói cước.
// @Tags Admin User
// @Produce json
// @Param id path string true "ID người dùng"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/users/{id} [get]
func (h *AdminHandler) GetUserDetail(c *gin.Context) error {
	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	detail, err := h.userUseCase.GetUserDetail(c.Request.Context(), targetUserID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetUserDetail, detail)
	return nil
}

// CreateUser tạo tài khoản người dùng mới
// @Summary Tạo tài khoản người dùng
// @Description Cho phép quản trị viên tạo tài khoản người dùng mới (mặc định vai trò user).
// @Tags Admin User
// @Accept json
// @Produce json
// @Param body body dto.CreateAdminUserReq true "Thông tin tài khoản mới"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/users [post]
func (h *AdminHandler) CreateUser(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req dto.CreateAdminUserReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	res, err := h.userUseCase.CreateUserByAdmin(c.Request.Context(), adminUserID, req)
	if err != nil {
		return err
	}

	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminUserID,
		Action:      string(adminaudit.UserCreate),
		TargetType:  string(adminaudit.User),
		TargetID:    &res.ID,
		Payload: map[string]any{
			"username": res.Username,
			"email":    res.Email,
			"roleSlug": res.RoleSlug,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successCreateUser, res)
	return nil
}

// UpdateUserRole thay đổi vai trò tài khoản
// @Summary Đổi vai trò tài khoản người dùng
// @Description Đổi vai trò (role) của tài khoản người dùng. Chỉ có tài khoản admin seed mới được quyền nâng cấp tài khoản lên admin.
// @Tags Admin User
// @Accept json
// @Produce json
// @Param id path string true "ID người dùng"
// @Param body body dto.UpdateUserRoleReq true "Vai trò mới"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/users/{id}/role [patch]
func (h *AdminHandler) UpdateUserRole(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	var req dto.UpdateUserRoleReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	res, err := h.userUseCase.UpdateUserRole(c.Request.Context(), adminUserID, targetUserID, req)
	if err != nil {
		return err
	}

	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminUserID,
		Action:      string(adminaudit.UserRoleChange),
		TargetType:  string(adminaudit.User),
		TargetID:    &targetUserID,
		Payload: map[string]any{
			"newRole": req.RoleSlug,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successUpdateUserRole, res)
	return nil
}

// DeactivateUser vô hiệu hóa tài khoản người dùng
// @Summary Vô hiệu hóa tài khoản người dùng
// @Description Vô hiệu hóa mềm tài khoản người dùng và thu hồi toàn bộ phiên đăng nhập.
// @Tags Admin User
// @Produce json
// @Param id path string true "ID người dùng"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/users/{id} [delete]
func (h *AdminHandler) DeactivateUser(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	res, err := h.userUseCase.DeactivateUser(c.Request.Context(), adminUserID, targetUserID)
	if err != nil {
		return err
	}

	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminUserID,
		Action:      string(adminaudit.UserDeactivate),
		TargetType:  string(adminaudit.User),
		TargetID:    &targetUserID,
		Payload: map[string]any{
			"status": "inactive",
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successDeactivateUser, res)
	return nil
}

// ListUserSessions danh sách phiên đăng nhập của người dùng
// @Summary Danh sách phiên đăng nhập
// @Description Lấy danh sách các phiên làm việc / thiết bị đăng nhập của người dùng.
// @Tags Admin User
// @Produce json
// @Param id path string true "ID người dùng"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/users/{id}/sessions [get]
func (h *AdminHandler) ListUserSessions(c *gin.Context) error {
	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	sessions, err := h.userUseCase.ListUserSessions(c.Request.Context(), targetUserID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successListUserSessions, sessions)
	return nil
}

// RevokeUserSessions thu hồi toàn bộ phiên đăng nhập
// @Summary Thu hồi toàn bộ phiên đăng nhập
// @Description Thu hồi và vô hiệu hóa tất cả các phiên làm việc hiện tại của người dùng.
// @Tags Admin User
// @Produce json
// @Param id path string true "ID người dùng"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/users/{id}/sessions/revoke [post]
func (h *AdminHandler) RevokeUserSessions(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	if err := h.userUseCase.RevokeUserSessions(c.Request.Context(), targetUserID); err != nil {
		return err
	}

	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminUserID,
		Action:      string(adminaudit.UserSessionsRevoke),
		TargetType:  string(adminaudit.User),
		TargetID:    &targetUserID,
		RequestIP:   &clientIP,
	})

	shared_pres.Success(c, successRevokeUserSessions, nil)
	return nil
}

// UpdateUserStatus cập nhật trạng thái tài khoản người dùng
// @Summary Cập nhật trạng thái tài khoản người dùng
// @Description Cho phép admin khóa hoặc mở lại tài khoản user.
// @Tags Admin User
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

	action := adminaudit.UserActivate
	if input.Status == 0 {
		action = adminaudit.UserDeactivate
	}
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminUserID,
		Action:      string(action),
		TargetType:  string(adminaudit.User),
		TargetID:    &targetUserID,
		Payload: map[string]any{
			"status": input.Status,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successUpdateUserStatus, response)
	return nil
}

// GetUsers lấy danh sách người dùng cho Admin
// @Summary Lấy danh sách người dùng
// @Description Cho phép admin lấy danh sách người dùng phân trang, tìm kiếm và lọc theo trạng thái/phân quyền.
// @Tags Admin User
// @Accept json
// @Produce json
// @Param query query dto.GetUsersQueryReq false "Bộ lọc danh sách người dùng"
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
