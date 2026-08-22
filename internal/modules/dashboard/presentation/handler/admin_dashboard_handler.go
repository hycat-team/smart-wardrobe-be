package handler

import (
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgAdminGetKPICardsSuccess                 = "Lấy danh sách thẻ KPI Admin thành công"
	msgAdminGetAIMarginSuccess                 = "Lấy thông tin biên lợi nhuận AI thành công"
	msgAdminGetPayOSReconciliationSuccess      = "Lấy thông tin đối soát PayOS thành công"
	msgAdminGetSubscriptionDistributionSuccess = "Lấy phân bổ subscription thành công"
	msgAdminGetRevenueBreakdownSuccess         = "Lấy doanh thu subscription thành công"
	msgAdminGetAIUsageByOperationSuccess       = "Lấy thống kê sử dụng AI thành công"
	msgAdminGetAIErrorBreakdownSuccess         = "Lấy thống kê lỗi AI thành công"
	msgAdminGetRenewalStatsSuccess             = "Lấy thống kê gia hạn subscription thành công"
	msgAdminGetOverviewSuccess                 = "Lấy thông tin tổng quan hệ thống thành công"
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
// @Tags Dashboard Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminKPICardsRes} "Lấy danh sách thẻ KPI Admin thành công"
// @Router /api/v1/admin/dashboard/kpi-cards [get]
func (h *AdminDashboardHandler) GetKPICards(c *gin.Context) error {
	res, err := h.adminUC.GetAdminKPICards(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetKPICardsSuccess, res)
	return nil
}

// GetAIMargin phân tích doanh thu và chi phí AI Token thực tế
// @Summary Phân tích biên lợi nhuận AI Token
// @Description Thống kê doanh thu subscription, chi phí thực tế tiêu thụ AI Token (ai_usage_period_ledgers), lợi nhuận ròng AI (net_ai_margin) và phần trăm biên lợi nhuận.
// @Tags Dashboard Admin
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
		var err error
		fromDate, err = time.Parse("2006-01-02", req.FromDate)
		if err != nil {
			return apperror.NewBadRequest("Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
	}
	if req.ToDate != "" {
		var err error
		toDate, err = time.Parse("2006-01-02", req.ToDate)
		if err != nil {
			return apperror.NewBadRequest("Định dạng ngày kết thúc (toDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
	}

	res, err := h.adminUC.GetAIMarginAnalytics(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetAIMarginSuccess, res)
	return nil
}

// GetPayOSReconciliation lấy thông tin đối soát giao dịch thanh toán PayOS
// @Summary Thống kê đối soát giao dịch PayOS
// @Description Báo cáo đối soát số lượng giao dịch thanh toán PayOS, tỷ lệ thành công/thất bại và tổng số tiền thanh toán thực tế.
// @Tags Dashboard Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.PayOSReconciliationRes} "Lấy thông tin đối soát PayOS thành công"
// @Router /api/v1/admin/dashboard/payos-reconciliation [get]
func (h *AdminDashboardHandler) GetPayOSReconciliation(c *gin.Context) error {
	res, err := h.adminUC.GetPayOSReconciliation(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetPayOSReconciliationSuccess, res)
	return nil
}

// GetSubscriptionDistribution lấy phân bổ user theo gói subscription
// @Summary Phân bổ người dùng theo gói subscription
// @Description Thống kê số lượng người dùng và tỷ lệ phần trăm theo từng gói subscription (FREE, PREMIUM_MONTHLY, PREMIUM_YEARLY, v.v.)
// @Tags Dashboard Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.SubscriptionDistributionRes} "Lấy phân bổ subscription thành công"
// @Router /api/v1/admin/dashboard/subscriptions/distribution [get]
func (h *AdminDashboardHandler) GetSubscriptionDistribution(c *gin.Context) error {
	res, err := h.adminUC.GetSubscriptionDistribution(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetSubscriptionDistributionSuccess, res)
	return nil
}

// GetRevenueBreakdown lấy doanh thu theo plan_code và thời gian
// @Summary Doanh thu subscription theo gói và thời gian
// @Description Thống kê doanh thu subscription theo từng plan_code và theo tháng, hỗ trợ lọc khoảng thời gian
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu (định dạng YYYY-MM-DD)"
// @Param toDate query string false "Ngày kết thúc (định dạng YYYY-MM-DD)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.RevenueBreakdownRes} "Lấy doanh thu subscription thành công"
// @Router /api/v1/admin/dashboard/subscriptions/revenue-breakdown [get]
func (h *AdminDashboardHandler) GetRevenueBreakdown(c *gin.Context) error {
	var req dto.GetAIMarginAnalyticsQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}

	var fromDate, toDate time.Time
	if req.FromDate != "" {
		var err error
		fromDate, err = time.Parse("2006-01-02", req.FromDate)
		if err != nil {
			return apperror.NewBadRequest("Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
	}
	if req.ToDate != "" {
		var err error
		toDate, err = time.Parse("2006-01-02", req.ToDate)
		if err != nil {
			return apperror.NewBadRequest("Định dạng ngày kết thúc (toDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
	}

	res, err := h.adminUC.GetRevenueBreakdown(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetRevenueBreakdownSuccess, res)
	return nil
}

// GetAIUsageByOperation lấy thống kê sử dụng AI theo operation
// @Summary Thống kê sử dụng AI theo tác vụ
// @Description Thống kê số lượng request, chi phí và token AI theo từng loại tác vụ (operation) và tuyến xử lý (logical_route), hỗ trợ lọc khoảng thời gian
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu (định dạng YYYY-MM-DD)"
// @Param toDate query string false "Ngày kết thúc (định dạng YYYY-MM-DD)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIUsageByOperationRes} "Lấy thống kê sử dụng AI thành công"
// @Router /api/v1/admin/dashboard/ai/usage-by-operation [get]
func (h *AdminDashboardHandler) GetAIUsageByOperation(c *gin.Context) error {
	var req dto.GetAIMarginAnalyticsQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}

	var fromDate, toDate time.Time
	if req.FromDate != "" {
		var err error
		fromDate, err = time.Parse("2006-01-02", req.FromDate)
		if err != nil {
			return apperror.NewBadRequest("Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
	}
	if req.ToDate != "" {
		var err error
		toDate, err = time.Parse("2006-01-02", req.ToDate)
		if err != nil {
			return apperror.NewBadRequest("Định dạng ngày kết thúc (toDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
	}

	res, err := h.adminUC.GetAIUsageByOperation(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetAIUsageByOperationSuccess, res)
	return nil
}

// GetAIErrorBreakdown lấy thống kê lỗi AI theo error_code, provider, model
// @Summary Thống kê lỗi AI theo mã lỗi, nhà cung cấp và mô hình
// @Description Thống kê số lượng lỗi, tổng số request và tỷ lệ lỗi AI, phân nhóm theo error_code, provider và model
// @Tags Dashboard Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIErrorBreakdownRes} "Lấy thống kê lỗi AI thành công"
// @Router /api/v1/admin/dashboard/ai/error-breakdown [get]
func (h *AdminDashboardHandler) GetAIErrorBreakdown(c *gin.Context) error {
	res, err := h.adminUC.GetAIErrorBreakdown(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetAIErrorBreakdownSuccess, res)
	return nil
}

// GetRenewalStats lấy thống kê tỷ lệ gia hạn subscription
// @Summary Thống kê gia hạn subscription
// @Description Thống kê tổng số lần gia hạn, số thành công, số thất bại, tỷ lệ thành công và số lần thử lại trung bình
// @Tags Dashboard Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.RenewalStatsRes} "Lấy thống kê gia hạn subscription thành công"
// @Router /api/v1/admin/dashboard/subscriptions/renewal-stats [get]
func (h *AdminDashboardHandler) GetRenewalStats(c *gin.Context) error {
	res, err := h.adminUC.GetRenewalStats(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetRenewalStatsSuccess, res)
	return nil
}

// GetOverview lấy toàn bộ bức tranh tổng quan hệ thống cho Admin
// @Summary Báo cáo tổng quan Admin Dashboard
// @Description Tổng hợp tất cả dữ liệu từ KPI Cards, AI Margin Analytics và PayOS Reconciliation trong 1 kết quả duy nhất.
// @Tags Dashboard Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminOverviewRes} "Lấy thông tin tổng quan hệ thống thành công"
// @Router /api/v1/admin/dashboard/overview [get]
func (h *AdminDashboardHandler) GetOverview(c *gin.Context) error {
	res, err := h.adminUC.GetSystemOverview(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetOverviewSuccess, res)
	return nil
}
