package handler

import (
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	_ "smart-wardrobe-be/internal/shared/presentation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type AdminDashboardHandler struct {
	adminUC uc_interfaces.IAdminDashboardUseCase
}

func NewAdminDashboardHandler(adminUC uc_interfaces.IAdminDashboardUseCase) *AdminDashboardHandler {
	return &AdminDashboardHandler{adminUC: adminUC}
}

// GetKPICards lấy các thẻ chỉ số KPI tổng quan cho System Admin
// @Summary Lấy các thẻ chỉ số KPI tổng quan Admin
// @Description Lấy thông tin tổng số lượng người dùng, chỉ số DAU, số nhãn hàng active, số nhãn hàng chờ duyệt, doanh thu subscription và biên lợi nhuận AI net margin.
// @Tags Admin Dashboard
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminKPICardsRes} "Lấy danh sách thẻ KPI Admin thành công"
// @Router /api/v1/admin/dashboard/kpi-cards [get]
func (h *AdminDashboardHandler) GetKPICards(c *gin.Context) error {
	res, err := h.adminUC.GetAdminKPICards(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách thẻ KPI Admin thành công", res)
	return nil
}

// GetAIMargin phân tích doanh thu và chi phí AI Token thực tế
// @Summary Phân tích biên lợi nhuận AI Token
// @Description Thống kê doanh thu subscription, chi phí thực tế tiêu thụ AI Token (ai_usage_period_ledgers), lợi nhuận ròng AI (net_ai_margin) và phần trăm biên lợi nhuận.
// @Tags Admin Dashboard
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu (định dạng YYYY-MM-DD)"
// @Param toDate query string false "Ngày kết thúc (định dạng YYYY-MM-DD)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIMarginAnalyticsRes} "Lấy thông tin biên lợi nhuận AI thành công"
// @Router /api/v1/admin/dashboard/ai-margin [get]
func (h *AdminDashboardHandler) GetAIMargin(c *gin.Context) error {
	var req dto.GetAIMarginAnalyticsQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}

	var fromDate, toDate time.Time
	if req.FromDate != "" {
		fromDate, _ = time.Parse("2006-01-02", req.FromDate)
	}
	if req.ToDate != "" {
		toDate, _ = time.Parse("2006-01-02", req.ToDate)
	}

	res, err := h.adminUC.GetAIMarginAnalytics(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thông tin biên lợi nhuận AI thành công", res)
	return nil
}

// GetPayOSReconciliation lấy thông tin đối soát giao dịch thanh toán PayOS
// @Summary Thống kê đối soát giao dịch PayOS
// @Description Báo cáo đối soát số lượng giao dịch thanh toán PayOS, tỷ lệ thành công/thất bại và tổng số tiền thanh toán thực tế.
// @Tags Admin Dashboard
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.PayOSReconciliationRes} "Lấy thông tin đối soát PayOS thành công"
// @Router /api/v1/admin/dashboard/payos-reconciliation [get]
func (h *AdminDashboardHandler) GetPayOSReconciliation(c *gin.Context) error {
	res, err := h.adminUC.GetPayOSReconciliation(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thông tin đối soát PayOS thành công", res)
	return nil
}

// GetOverview lấy toàn bộ bức tranh tổng quan hệ thống cho Admin
// @Summary Báo cáo tổng quan Admin Dashboard
// @Description Tổng hợp tất cả dữ liệu từ KPI Cards, AI Margin Analytics và PayOS Reconciliation trong 1 kết quả duy nhất.
// @Tags Admin Dashboard
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminOverviewRes} "Lấy thông tin tổng quan hệ thống thành công"
// @Router /api/v1/admin/dashboard/overview [get]
func (h *AdminDashboardHandler) GetOverview(c *gin.Context) error {
	res, err := h.adminUC.GetSystemOverview(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thông tin tổng quan hệ thống thành công", res)
	return nil
}
