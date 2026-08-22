package handler

import (
	admin_dto "smart-wardrobe-be/internal/modules/admin/application/dto"
	admin_contract "smart-wardrobe-be/internal/modules/admin/presentation/contract"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	"smart-wardrobe-be/internal/shared/domain/constants/adminaudit"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	successAdminListMembers       = "Lấy danh sách thành viên nhãn hàng thành công"
	successAdminAddStaff          = "Thêm nhân viên cho nhãn hàng thành công"
	successAdminUpdateMember      = "Cập nhật thành viên nhãn hàng thành công"
	successAdminRemoveMember      = "Gỡ bỏ thành viên khỏi nhãn hàng thành công"
	successAdminTransferOwnership = "Chuyển giao quyền sở hữu nhãn hàng thành công"
)

type AdminBrandHandler struct {
	logger           logger.Interface
	brandAdminUC     uc_interfaces.IBrandAdminUseCase
	auditLogContract admin_contract.IAuditLogContract
}

func NewAdminBrandHandler(
	logger logger.Interface,
	brandAdminUC uc_interfaces.IBrandAdminUseCase,
	auditLogContract admin_contract.IAuditLogContract,
) *AdminBrandHandler {
	return &AdminBrandHandler{
		logger:           logger,
		brandAdminUC:     brandAdminUC,
		auditLogContract: auditLogContract,
	}
}

type brandIDUriParam struct {
	BrandID string `uri:"brandId" binding:"required"`
}

type brandMemberUriParam struct {
	BrandID  string `uri:"brandId" binding:"required"`
	MemberID string `uri:"memberId" binding:"required"`
}

// ListMembers
// @Summary Danh sách thành viên nhãn hàng
// @Description Quản trị viên xem danh sách tất cả thành viên của nhãn hàng.
// @Tags Admin Brand
// @Produce json
// @Param brandId path string true "Brand ID"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandMemberRes}
// @Router /api/v1/admin/brands/{brandId}/members [get]
func (h *AdminBrandHandler) ListMembers(c *gin.Context) error {
	var uri brandIDUriParam
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	brandID, err := uuid.Parse(uri.BrandID)
	if err != nil {
		return err
	}

	members, err := h.brandAdminUC.ListMembers(c.Request.Context(), brandID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successAdminListMembers, members)
	return nil
}

// AddStaff
// @Summary Thêm nhân viên cho nhãn hàng
// @Description Quản trị viên gán người dùng làm nhân viên (staff) của nhãn hàng.
// @Tags Admin Brand
// @Accept json
// @Produce json
// @Param brandId path string true "Brand ID"
// @Param body body dto.AdminAddBrandStaffReq true "Thông tin người dùng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandMemberRes}
// @Router /api/v1/admin/brands/{brandId}/members [post]
func (h *AdminBrandHandler) AddStaff(c *gin.Context) error {
	var uri brandIDUriParam
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	brandID, err := uuid.Parse(uri.BrandID)
	if err != nil {
		return err
	}

	var req dto.AdminAddBrandStaffReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	member, err := h.brandAdminUC.AddStaff(c.Request.Context(), brandID, req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.BrandMemberAdd),
		TargetType:  string(adminaudit.BrandMember),
		TargetID:    &member.ID,
		Payload: map[string]any{
			"brandId": brandID,
			"userId":  req.UserID,
			"role":    member.Role,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminAddStaff, member)
	return nil
}

// UpdateMember
// @Summary Cập nhật vai trò/trạng thái thành viên
// @Description Quản trị viên cập nhật vai trò hoặc trạng thái hoạt động của thành viên nhãn hàng.
// @Tags Admin Brand
// @Accept json
// @Produce json
// @Param brandId path string true "Brand ID"
// @Param memberId path string true "Member ID"
// @Param body body dto.AdminUpdateBrandMemberReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandMemberRes}
// @Router /api/v1/admin/brands/{brandId}/members/{memberId} [patch]
func (h *AdminBrandHandler) UpdateMember(c *gin.Context) error {
	var uri brandMemberUriParam
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	brandID, err := uuid.Parse(uri.BrandID)
	if err != nil {
		return err
	}
	memberID, err := uuid.Parse(uri.MemberID)
	if err != nil {
		return err
	}

	var req dto.AdminUpdateBrandMemberReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	member, err := h.brandAdminUC.UpdateMember(c.Request.Context(), brandID, memberID, req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.BrandMemberRoleChange),
		TargetType:  string(adminaudit.BrandMember),
		TargetID:    &memberID,
		Payload: map[string]any{
			"brandId": brandID,
			"role":    member.Role,
			"status":  member.Status,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminUpdateMember, member)
	return nil
}

// RemoveMember
// @Summary Gỡ bỏ thành viên khỏi nhãn hàng
// @Description Quản trị viên xóa thành viên khỏi nhãn hàng (bảo vệ chủ sở hữu duy nhất).
// @Tags Admin Brand
// @Produce json
// @Param brandId path string true "Brand ID"
// @Param memberId path string true "Member ID"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/brands/{brandId}/members/{memberId} [delete]
func (h *AdminBrandHandler) RemoveMember(c *gin.Context) error {
	var uri brandMemberUriParam
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	brandID, err := uuid.Parse(uri.BrandID)
	if err != nil {
		return err
	}
	memberID, err := uuid.Parse(uri.MemberID)
	if err != nil {
		return err
	}

	if err := h.brandAdminUC.RemoveMember(c.Request.Context(), brandID, memberID); err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.BrandMemberRemove),
		TargetType:  string(adminaudit.BrandMember),
		TargetID:    &memberID,
		Payload: map[string]any{
			"brandId": brandID,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminRemoveMember, nil)
	return nil
}

// TransferOwnership
// @Summary Chuyển giao quyền sở hữu nhãn hàng
// @Description Quản trị viên chuyển quyền Owner nhãn hàng cho người dùng khác trong nhãn hàng, đồng thời chọn hạ chủ cũ thành staff hoặc gỡ hoàn toàn.
// @Tags Admin Brand
// @Accept json
// @Produce json
// @Param brandId path string true "Brand ID"
// @Param body body dto.TransferBrandOwnershipReq true "Thông tin chuyển giao"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/brands/{brandId}/transfer-ownership [post]
func (h *AdminBrandHandler) TransferOwnership(c *gin.Context) error {
	var uri brandIDUriParam
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	brandID, err := uuid.Parse(uri.BrandID)
	if err != nil {
		return err
	}

	var req dto.TransferBrandOwnershipReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	if err := h.brandAdminUC.TransferOwnership(c.Request.Context(), brandID, req); err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.BrandOwnershipTransfer),
		TargetType:  string(adminaudit.Brand),
		TargetID:    &brandID,
		Payload: map[string]any{
			"newOwnerUserId":   req.NewOwnerUserID,
			"oldOwnerHandling": req.OldOwnerHandling,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminTransferOwnership, nil)
	return nil
}
