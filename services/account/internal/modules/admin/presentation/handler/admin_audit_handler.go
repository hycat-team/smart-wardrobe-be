package handler

import (
	"time"

	"smart-wardrobe-be/services/account/dto/admin"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/admin/application/interface/usecase"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	successGetAuditLogs = "Lấy danh sách nhật ký quản trị thành công"
)

type AdminAuditHandler struct {
	logger     logger.Interface
	auditLogUC uc_interfaces.IAuditLogUseCase
}

func NewAdminAuditHandler(
	logger logger.Interface,
	auditLogUC uc_interfaces.IAuditLogUseCase,
) *AdminAuditHandler {
	return &AdminAuditHandler{
		logger:     logger,
		auditLogUC: auditLogUC,
	}
}

// GetAuditLogs
// @Summary Danh sách nhật ký quản trị (Audit Logs)
// @Description Quản trị viên tra cứu lịch sử thao tác của các quản trị viên trong hệ thống.
// @Tags Admin Audit
// @Produce json
// @Param actorUserId query string false "ID admin thực hiện"
// @Param action query string false "Hành động"
// @Param targetType query string false "Loại đối tượng"
// @Param targetId query string false "ID đối tượng"
// @Param fromDate query string false "Từ ngày (YYYY-MM-DD)"
// @Param toDate query string false "Đến ngày (YYYY-MM-DD)"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminAuditLogListRes}
// @Router /api/v1/admin/audit-logs [get]
func (h *AdminAuditHandler) GetAuditLogs(c *gin.Context) error {
	var query dto.GetAuditLogsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	filter := dto.AuditLogFilter{
		Action:     query.Action,
		TargetType: query.TargetType,
	}

	if query.ActorUserID != nil && *query.ActorUserID != "" {
		if uid, err := uuid.Parse(*query.ActorUserID); err == nil {
			filter.ActorUserID = &uid
		}
	}
	if query.TargetID != nil && *query.TargetID != "" {
		if tid, err := uuid.Parse(*query.TargetID); err == nil {
			filter.TargetID = &tid
		}
	}
	if query.FromDate != nil && *query.FromDate != "" {
		if t, err := time.Parse("2006-01-02", *query.FromDate); err == nil {
			filter.FromDate = &t
		}
	}
	if query.ToDate != nil && *query.ToDate != "" {
		if t, err := time.Parse("2006-01-02", *query.ToDate); err == nil {
			filter.ToDate = &t
		}
	}

	logs, total, err := h.auditLogUC.List(c.Request.Context(), filter, query.Page, query.Limit)
	if err != nil {
		return err
	}

	response := &shared_dto.PaginationResult[*dto.AdminAuditLogDTO]{
		Items:    logs,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, total),
	}

	shared_pres.Success(c, successGetAuditLogs, response)
	return nil
}
