package handler

import (
	"time"

	admin_dto "smart-wardrobe-be/internal/modules/admin/application/dto"
	admin_contract "smart-wardrobe-be/internal/modules/admin/presentation/contract"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/adminaudit"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	successAdminListPricing  = "Lấy danh sách cấu hình giá AI thành công"
	successAdminUpdatePrice  = "Cập nhật cấu hình giá AI thành công"
	successAdminGetUserUsage = "Lấy lịch sử sử dụng AI của người dùng thành công"
	successAdminListGrants   = "Lấy danh sách hạn mức AI của người dùng thành công"
	successAdminCreateGrant  = "Cấp hạn mức AI cho người dùng thành công"
)

type AdminHandler struct {
	logger           logger.Interface
	aiAdminUC        uc_interfaces.IAIAdminUseCase
	auditLogContract admin_contract.IAuditLogContract
}

func NewAdminHandler(
	logger logger.Interface,
	aiAdminUC uc_interfaces.IAIAdminUseCase,
	auditLogContract admin_contract.IAuditLogContract,
) *AdminHandler {
	return &AdminHandler{
		logger:           logger,
		aiAdminUC:        aiAdminUC,
		auditLogContract: auditLogContract,
	}
}

type adminPricingIDUriParam struct {
	ID string `uri:"id" binding:"required"`
}

type adminUserUsageQuery struct {
	shared_dto.PaginationQuery
	UserID   string `form:"userId" binding:"required"`
	FromDate string `form:"fromDate"`
	ToDate   string `form:"toDate"`
}

type adminUserGrantsQuery struct {
	UserID string `form:"userId" binding:"required"`
}

// ListPricing
// @Summary Danh sách cấu hình giá mô hình AI
// @Description Quản trị viên xem danh sách bảng giá các model AI của các nhà cung cấp.
// @Tags Admin AI
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.AIModelConfigDTO}
// @Router /api/v1/admin/ai/pricing [get]
func (h *AdminHandler) ListPricing(c *gin.Context) error {
	list, err := h.aiAdminUC.ListPricing(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListPricing, list)
	return nil
}

// UpdatePricing
// @Summary Cập nhật cấu hình giá mô hình AI
// @Description Quản trị viên cập nhật giá token hoặc tỷ giá quy đổi cho mô hình AI.
// @Tags Admin AI
// @Accept json
// @Produce json
// @Param id path string true "Model Config ID"
// @Param body body dto.UpdatePricingReq true "Thông tin cập nhật giá"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIModelConfigDTO}
// @Router /api/v1/admin/ai/pricing/{id} [put]
func (h *AdminHandler) UpdatePricing(c *gin.Context) error {
	var uri adminPricingIDUriParam
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	id, err := uuid.Parse(uri.ID)
	if err != nil {
		return err
	}

	var req dto.UpdatePricingReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	updated, err := h.aiAdminUC.UpdatePricing(c.Request.Context(), id, req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.AIPricingUpdate),
		TargetType:  string(adminaudit.AIModelConfig),
		TargetID:    &id,
		RequestIP:   &clientIP,
	})

	shared_pres.Success(c, successAdminUpdatePrice, updated)
	return nil
}

// GetUserUsage
// @Summary Lịch sử sử dụng AI của người dùng
// @Description Quản trị viên xem nhật ký sử dụng và chi phí AI của một người dùng.
// @Tags Admin AI
// @Produce json
// @Param userId query string true "User ID"
// @Param fromDate query string false "Từ ngày (YYYY-MM-DD)"
// @Param toDate query string false "Đến ngày (YYYY-MM-DD)"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.AIUsageItemDTO]}
// @Router /api/v1/admin/ai/usage [get]
func (h *AdminHandler) GetUserUsage(c *gin.Context) error {
	var query adminUserUsageQuery
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	userID, err := uuid.Parse(query.UserID)
	if err != nil {
		return err
	}

	var fromDate, toDate *time.Time
	if query.FromDate != "" {
		if t, err := time.Parse("2006-01-02", query.FromDate); err == nil {
			fromDate = &t
		}
	}
	if query.ToDate != "" {
		if t, err := time.Parse("2006-01-02", query.ToDate); err == nil {
			toDate = &t
		}
	}

	items, total, err := h.aiAdminUC.GetUserUsage(c.Request.Context(), userID, fromDate, toDate, query.Page, query.Limit)
	if err != nil {
		return err
	}

	response := &shared_dto.PaginationResult[*dto.AIUsageItemDTO]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, total),
	}

	shared_pres.Success(c, successAdminGetUserUsage, response)
	return nil
}

// ListGrants
// @Summary Danh sách hạn mức chính sách AI của người dùng
// @Description Quản trị viên xem các lượt grant chính sách AI đã cấp cho người dùng.
// @Tags Admin AI
// @Produce json
// @Param userId query string true "User ID"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.AIPolicyGrantDTO}
// @Router /api/v1/admin/ai/grants [get]
func (h *AdminHandler) ListGrants(c *gin.Context) error {
	var query adminUserGrantsQuery
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	userID, err := uuid.Parse(query.UserID)
	if err != nil {
		return err
	}

	grants, err := h.aiAdminUC.ListGrants(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successAdminListGrants, grants)
	return nil
}

// CreateGrant
// @Summary Cấp chính sách hạn mức AI cho người dùng
// @Description Quản trị viên cấp trực tiếp chính sách hạn mức AI mới cho người dùng.
// @Tags Admin AI
// @Accept json
// @Produce json
// @Param body body dto.CreateAIGrantReq true "Thông tin cấp chính sách"
// @Success 201 {object} shared_pres.APIResponse{data=dto.AIPolicyGrantDTO}
// @Router /api/v1/admin/ai/grants [post]
func (h *AdminHandler) CreateGrant(c *gin.Context) error {
	var req dto.CreateAIGrantReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	grant, err := h.aiAdminUC.CreateGrant(c.Request.Context(), req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.AIGrantCreate),
		TargetType:  string(adminaudit.AIPolicyGrant),
		TargetID:    &grant.ID,
		Payload: map[string]any{
			"userId":         req.UserID,
			"aiCostPolicyId": req.AICostPolicyID,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Created(c, successAdminCreateGrant, grant)
	return nil
}
