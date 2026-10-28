package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgAnalyticsExecuteDynamicQuerySuccess            = "Truy vấn dữ liệu tổng hợp thành công"
	msgAnalyticsGetAIEventsDrillDownSuccess           = "Lấy chi tiết sự kiện AI thành công"
	msgAnalyticsGetSampleLabFeedbacksDrillDownSuccess = "Lấy chi tiết đánh giá mẫu thử thành công"
)

type AnalyticsHandler struct {
	dynamicQueryUC uc_interfaces.IDynamicQueryUseCase
	drillDownUC    uc_interfaces.IDrillDownUseCase
}

func NewAnalyticsHandler(
	dynamicQueryUC uc_interfaces.IDynamicQueryUseCase,
	drillDownUC uc_interfaces.IDrillDownUseCase,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		dynamicQueryUC: dynamicQueryUC,
		drillDownUC:    drillDownUC,
	}
}

// ExecuteDynamicQuery truy vấn tổng hợp linh hoạt đa chiều cho Dashboard Widgets
// @Summary Truy vấn tổng hợp linh hoạt đa chiều (Dynamic Query Engine)
// @Description Gửi các chỉ số (metrics), khoảng thời gian (timeframe), độ mịn (granularity) và bộ lọc để nhận dữ liệu Time-Series linh hoạt. Kiểm tra quyền truy cập theo TargetDashboard.
// @Tags Dashboard Analytics
// @Accept json
// @Produce json
// @Param request body dto.DynamicQueryReq true "Thông số truy vấn động"
// @Success 200 {object} shared_pres.APIResponse{data=dto.DynamicQueryRes} "Truy vấn dữ liệu tổng hợp thành công"
// @Router /api/v1/dashboard/analytics/query [post]
func (h *AnalyticsHandler) ExecuteDynamicQuery(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	userRole, err := contextutils.GetRoleSlug(c)
	if err != nil {
		return err
	}

	var req dto.DynamicQueryReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	res, err := h.dynamicQueryUC.ExecuteDynamicQuery(c.Request.Context(), userID, userRole, &req)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAnalyticsExecuteDynamicQuerySuccess, res)
	return nil
}

// GetAIEventsDrillDown xem danh sách chi tiết các sự kiện/lỗi AI (Admin)
// @Summary Truy vết chi tiết các sự kiện lỗi AI (Drill-down Admin)
// @Description Lấy danh sách phân trang các sự kiện lỗi AI token với thông tin provider, model, status code.
// @Tags Dashboard Admin
// @Produce json
// @Param status query string false "Trạng thái lỗi (ví dụ: FAILED)"
// @Param page query int false "Số trang" default(1)
// @Param limit query int false "Số lượng phần tử mỗi trang" default(20)
// @Success 200 {object} shared_pres.APIResponse{data=dto.DrillDownAIEventListRes} "Lấy chi tiết sự kiện AI thành công"
// @Router /api/v1/admin/dashboard/drilldown/ai-events [get]
func (h *AnalyticsHandler) GetAIEventsDrillDown(c *gin.Context) error {
	var req dto.DrillDownAIEventQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}

	res, err := h.drillDownUC.GetAIEventsDrillDown(c.Request.Context(), &req)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAnalyticsGetAIEventsDrillDownSuccess, res)
	return nil
}

// GetSampleLabFeedbacksDrillDown xem chi tiết đánh giá mẫu thử Digital Sample Lab (Brand)
// @Summary Truy vết chi tiết đánh giá mẫu thử 3D (Drill-down Brand)
// @Description Lấy danh sách phân trang chi tiết các nhận xét và lượt bình chọn của mẫu thử sản phẩm.
// @Tags Dashboard Brand
// @Produce json
// @Param itemId query string true "ID Sản phẩm mẫu thử"
// @Param page query int false "Số trang" default(1)
// @Param limit query int false "Số lượng phần tử mỗi trang" default(20)
// @Success 200 {object} shared_pres.APIResponse{data=dto.DrillDownSampleFeedbackListRes} "Lấy chi tiết đánh giá mẫu thử thành công"
// @Router /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks [get]
func (h *AnalyticsHandler) GetSampleLabFeedbacksDrillDown(c *gin.Context) error {
	brandID, err := contextutils.GetBrandId(c)
	if err != nil {
		return err
	}
	var req dto.DrillDownSampleFeedbackQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}

	res, err := h.drillDownUC.GetSampleLabFeedbacksDrillDown(c.Request.Context(), brandID, &req)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAnalyticsGetSampleLabFeedbacksDrillDownSuccess, res)
	return nil
}
