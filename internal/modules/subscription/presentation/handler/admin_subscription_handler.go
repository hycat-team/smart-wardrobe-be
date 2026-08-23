package handler

import (
	admin_dto "smart-wardrobe-be/internal/modules/admin/application/dto"
	admin_contract "smart-wardrobe-be/internal/modules/admin/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/usersubscriptionstatus"
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
	successAdminListPlans         = "Lấy danh sách gói cước thành công"
	successAdminCreatePlan        = "Tạo gói cước mới thành công"
	successAdminUpdatePlan        = "Cập nhật gói cước thành công"
	successAdminSetPlanActive     = "Cập nhật trạng thái hoạt động của gói cước thành công"
	successAdminListSubscriptions = "Lấy danh sách đăng ký hội viên thành công"
	successAdminCancelSub         = "Hủy gói hội viên của người dùng thành công"
	successAdminGrantSub          = "Cấp gói hội viên cho người dùng thành công"
)

type AdminSubscriptionHandler struct {
	logger           logger.Interface
	planUseCase      uc_interfaces.ISubscriptionPlanUseCase
	subscriptionUC   uc_interfaces.ISubscriptionUseCase
	auditLogContract admin_contract.IAuditLogContract
}

func NewAdminSubscriptionHandler(
	logger logger.Interface,
	planUseCase uc_interfaces.ISubscriptionPlanUseCase,
	subscriptionUC uc_interfaces.ISubscriptionUseCase,
	auditLogContract admin_contract.IAuditLogContract,
) *AdminSubscriptionHandler {
	return &AdminSubscriptionHandler{
		logger:           logger,
		planUseCase:      planUseCase,
		subscriptionUC:   subscriptionUC,
		auditLogContract: auditLogContract,
	}
}

// ListPlans
// @Summary Danh sách tất cả gói cước
// @Description Lấy toàn bộ danh sách gói cước trong hệ thống (bao gồm cả gói đang tắt).
// @Tags Admin Subscription
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans [get]
func (h *AdminSubscriptionHandler) ListPlans(c *gin.Context) error {
	plans, err := h.planUseCase.ListAdminPlans(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListPlans, plans)
	return nil
}

// CreatePlan
// @Summary Tạo gói cước mới
// @Description Tạo một gói cước mới trong hệ thống.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param body body dto.CreateSubscriptionPlanReq true "Thông tin gói cước"
// @Success 201 {object} shared_pres.APIResponse{data=dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans [post]
func (h *AdminSubscriptionHandler) CreatePlan(c *gin.Context) error {
	var req dto.CreateSubscriptionPlanReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	created, err := h.planUseCase.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionPlanCreate),
		TargetType:  string(adminaudit.SubscriptionPlan),
		TargetID:    &created.ID,
		Payload: map[string]any{
			"planCode": created.Slug,
			"name":     created.Name,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Created(c, successAdminCreatePlan, created)
	return nil
}

type adminPlanIDParam struct {
	ID string `uri:"id" binding:"required"`
}

// UpdatePlan
// @Summary Cập nhật thông tin gói cước
// @Description Cập nhật thông tin chi tiết một gói cước theo ID.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param id path string true "ID gói cước"
// @Param body body dto.UpdateSubscriptionPlanReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans/{id} [put]
func (h *AdminSubscriptionHandler) UpdatePlan(c *gin.Context) error {
	var params adminPlanIDParam
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	id, err := uuid.Parse(params.ID)
	if err != nil {
		return err
	}
	var req dto.UpdateSubscriptionPlanReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	updated, err := h.planUseCase.UpdatePlan(c.Request.Context(), id, &req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionPlanUpdate),
		TargetType:  string(adminaudit.SubscriptionPlan),
		TargetID:    &id,
		RequestIP:   &clientIP,
	})

	shared_pres.Success(c, successAdminUpdatePlan, updated)
	return nil
}

// SetPlanActive
// @Summary Bật hoặc tắt trạng thái hoạt động gói cước
// @Description Cập nhật trạng thái isActive của gói cước.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param id path string true "ID gói cước"
// @Param body body dto.SetPlanActiveReq true "Trạng thái hoạt động"
// @Success 200 {object} shared_pres.APIResponse{data=dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans/{id}/status [patch]
func (h *AdminSubscriptionHandler) SetPlanActive(c *gin.Context) error {
	var params adminPlanIDParam
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	id, err := uuid.Parse(params.ID)
	if err != nil {
		return err
	}
	var req dto.SetPlanActiveReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	updated, err := h.planUseCase.SetPlanActive(c.Request.Context(), id, req.IsActive)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionPlanDeactivate),
		TargetType:  string(adminaudit.SubscriptionPlan),
		TargetID:    &id,
		Payload: map[string]any{
			"isActive": req.IsActive,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminSetPlanActive, updated)
	return nil
}

type adminListSubscriptionsQuery struct {
	shared_dto.PaginationQuery
	PlanCode *string `form:"planCode"`
	Status   *string `form:"status"`
}

// ListSubscriptions
// @Summary Danh sách đăng ký gói của người dùng
// @Description Lấy danh sách đăng ký gói hội viên của người dùng (lọc theo planCode, status).
// @Tags Admin Subscription
// @Produce json
// @Param planCode query string false "Mã slug gói"
// @Param status query string false "Trạng thái (active, expired)"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.AdminUserSubscriptionItemDTO]}
// @Router /api/v1/admin/subscriptions [get]
func (h *AdminSubscriptionHandler) ListSubscriptions(c *gin.Context) error {
	var req adminListSubscriptionsQuery
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	var statusEnum *usersubscriptionstatus.UserSubscriptionStatus
	if req.Status != nil && *req.Status != "" {
		if st, ok := usersubscriptionstatus.ParseUserSubscriptionStatus(*req.Status); ok {
			statusEnum = &st
		}
	}
	items, total, err := h.subscriptionUC.AdminList(c.Request.Context(), req.PlanCode, statusEnum, req.Page, req.Limit)
	if err != nil {
		return err
	}

	response := &shared_dto.PaginationResult[*dto.AdminUserSubscriptionItemDTO]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(req.PaginationQuery, total),
	}

	shared_pres.Success(c, successAdminListSubscriptions, response)
	return nil
}

type adminUserIDParam struct {
	UserID string `uri:"userId" binding:"required"`
}

// CancelSubscription
// @Summary Hủy gói hội viên của người dùng
// @Description Hủy gói hội viên của người dùng và chuyển về gói miễn phí (Free).
// @Tags Admin Subscription
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/{userId}/cancel [post]
func (h *AdminSubscriptionHandler) CancelSubscription(c *gin.Context) error {
	var params adminUserIDParam
	if err := c.ShouldBindUri(&params); err != nil {
		return err
	}
	userID, err := uuid.Parse(params.UserID)
	if err != nil {
		return err
	}
	if err := h.subscriptionUC.AdminCancel(c.Request.Context(), userID); err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionCancel),
		TargetType:  string(adminaudit.UserSubscription),
		TargetID:    &userID,
		RequestIP:   &clientIP,
	})

	shared_pres.Success(c, successAdminCancelSub, nil)
	return nil
}

type adminGrantPlanReq struct {
	PlanID string `json:"planId" binding:"required" label:"ID gói hội viên"`
}

// GrantSubscription
// @Summary Cấp gói hội viên thủ công cho người dùng
// @Description Gán một gói hội viên cụ thể cho người dùng.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param body body adminGrantPlanReq true "Thông tin gói cấp"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/{userId}/grant [post]
func (h *AdminSubscriptionHandler) GrantSubscription(c *gin.Context) error {
	var params adminUserIDParam
	if err := c.ShouldBindUri(&params); err != nil {
		return err
	}
	userID, err := uuid.Parse(params.UserID)
	if err != nil {
		return err
	}
	var req adminGrantPlanReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		return err
	}
	if err := h.subscriptionUC.AdminGrant(c.Request.Context(), userID, planID); err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionGrant),
		TargetType:  string(adminaudit.UserSubscription),
		TargetID:    &userID,
		Payload: map[string]any{
			"planId": planID,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminGrantSub, nil)
	return nil
}
