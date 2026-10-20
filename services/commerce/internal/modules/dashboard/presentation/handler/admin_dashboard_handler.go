package handler

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

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

// parseDashboardPeriod binds fromDate/toDate and turns them into the VN
// half-open range [from, to) every metric query filters on (029 R1/
// data-model §3): from = StartOfDayInVN(fromDate), to = EndOfDayInVN(toDate).
// A missing side yields the zero time so that side stays open; a malformed or
// inverted pair is refused with 400 before any use case runs. One shared
// helper for all seven period endpoints replaced the three duplicated parse
// blocks (T034).
func parseDashboardPeriod(c *gin.Context) (time.Time, time.Time, error) {
	var req dto.GetAIMarginAnalyticsQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return validation.ParseVNDateRangeQuery(req.FromDate, req.ToDate)
}

// GetKPICards lấy các thẻ chỉ số KPI tổng quan cho System Admin
// @Summary Lấy các thẻ chỉ số KPI tổng quan Admin
// @Description Lấy thông tin tổng số lượng người dùng, chỉ số DAU, số nhãn hàng active, số nhãn hàng chờ duyệt, doanh thu subscription và biên lợi nhuận AI net margin. Thẻ KPI là ảnh chụp (snapshot), không nhận tham số kỳ.
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
// @Description Thống kê doanh thu subscription, chi phí thực tế tiêu thụ AI Token (ai_usage_period_ledgers), lợi nhuận ròng AI (net_ai_margin) và phần trăm biên lợi nhuận. Kỳ tính theo ngày lịch Việt Nam, bao gồm trọn ngày hai đầu; không gửi kỳ mặc định 30 ngày gần nhất.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIMarginAnalyticsRes} "Lấy thông tin biên lợi nhuận AI thành công"
// @Router /api/v1/admin/dashboard/ai-margin [get]
func (h *AdminDashboardHandler) GetAIMargin(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
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
// @Description Báo cáo đối soát giao dịch PayOS (cả mua gói trực tiếp lẫn nạp ví) theo ngày lịch Việt Nam: số lượng theo nhóm trạng thái và tổng giá trị giao dịch thành công trong kỳ. Không gửi kỳ mặc định hôm nay.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PayOSReconciliationRes} "Lấy thông tin đối soát PayOS thành công"
// @Router /api/v1/admin/dashboard/payos-reconciliation [get]
func (h *AdminDashboardHandler) GetPayOSReconciliation(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
	}

	res, err := h.adminUC.GetPayOSReconciliation(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetPayOSReconciliationSuccess, res)
	return nil
}

// GetSubscriptionDistribution lấy phân bổ user theo gói subscription
// @Summary Phân bổ người dùng theo gói subscription
// @Description Thống kê số người dùng và tỷ lệ phần trăm theo từng gói, dựa trên các KHOẢNG HIỆU LỰC gói (không phải gói hiện tại), nên người đã đổi/hạ gói sau kỳ vẫn được tính đúng gói đã dùng trong kỳ. Không gửi kỳ: ảnh chụp gói đang có hiệu lực tại thời điểm gọi. Có kỳ: khoảng giao với [fromDate, toDate) theo ngày lịch Việt Nam. totalUsers là TỔNG CỦA CÁC NHÓM (người đổi gói giữa kỳ xuất hiện ở hai nhóm), không phải số người duy nhất. Hai trường tuỳ chọn mô tả độ đầy đủ: dataAvailableFrom (ngày VN YYYY-MM-DD, mốc bắt đầu lịch sử đầy đủ) và dataCompleteness (complete | partial | unavailable). Kỳ kết thúc trước mốc: distribution rỗng và totalUsers = 0, client phải hiển thị trạng thái trống thay vì biểu đồ 0%; kỳ cắt ngang mốc: chỉ tính phần sau mốc và trả partial.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.SubscriptionDistributionRes} "Lấy phân bổ subscription thành công"
// @Router /api/v1/admin/dashboard/subscriptions/distribution [get]
func (h *AdminDashboardHandler) GetSubscriptionDistribution(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
	}

	res, err := h.adminUC.GetSubscriptionDistribution(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetSubscriptionDistributionSuccess, res)
	return nil
}

// GetRevenueBreakdown lấy doanh thu theo plan_code và thời gian
// @Summary Doanh thu subscription theo gói và thời gian
// @Description Thống kê doanh thu subscription theo từng plan_code và theo tháng; kỳ tính theo ngày lịch Việt Nam, bao gồm trọn ngày hai đầu; không gửi kỳ mặc định 6 tháng gần nhất.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.RevenueBreakdownRes} "Lấy doanh thu subscription thành công"
// @Router /api/v1/admin/dashboard/subscriptions/revenue-breakdown [get]
func (h *AdminDashboardHandler) GetRevenueBreakdown(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
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
// @Description Thống kê số lượng request, chi phí và token AI theo từng loại tác vụ (operation) và tuyến xử lý (logical_route); kỳ tính theo ngày lịch Việt Nam, bao gồm trọn ngày hai đầu; không gửi kỳ mặc định 30 ngày gần nhất.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIUsageByOperationRes} "Lấy thống kê sử dụng AI thành công"
// @Router /api/v1/admin/dashboard/ai/usage-by-operation [get]
func (h *AdminDashboardHandler) GetAIUsageByOperation(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
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
// @Description Thống kê số lượng lỗi, tổng số request và tỷ lệ lỗi AI, phân nhóm theo error_code, provider và model. Tử số, mẫu số và phân nhóm cùng một kỳ (ngày lịch Việt Nam); không gửi kỳ mặc định 30 ngày gần nhất.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AIErrorBreakdownRes} "Lấy thống kê lỗi AI thành công"
// @Router /api/v1/admin/dashboard/ai/error-breakdown [get]
func (h *AdminDashboardHandler) GetAIErrorBreakdown(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
	}

	res, err := h.adminUC.GetAIErrorBreakdown(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetAIErrorBreakdownSuccess, res)
	return nil
}

// GetRenewalStats lấy thống kê tỷ lệ gia hạn subscription
// @Summary Thống kê gia hạn subscription
// @Description Thống kê tổng số lần gia hạn, số thành công, số thất bại, tỷ lệ thành công và số lần thử lại trung bình. Có kỳ: chỉ attempt tạo trong khoảng ngày (ngày lịch Việt Nam); không gửi kỳ: toàn thời gian.
// @Tags Dashboard Admin
// @Produce json
// @Param fromDate query string false "Ngày bắt đầu, ngày lịch Việt Nam (YYYY-MM-DD); bắt buộc ≤ toDate, sai định dạng hoặc đảo chiều trả 400"
// @Param toDate query string false "Ngày kết thúc, ngày lịch Việt Nam (YYYY-MM-DD), bao gồm trọn ngày; sai định dạng hoặc đảo chiều trả 400"
// @Success 200 {object} shared_pres.APIResponse{data=dto.RenewalStatsRes} "Lấy thống kê gia hạn subscription thành công"
// @Router /api/v1/admin/dashboard/subscriptions/renewal-stats [get]
func (h *AdminDashboardHandler) GetRenewalStats(c *gin.Context) error {
	fromDate, toDate, err := parseDashboardPeriod(c)
	if err != nil {
		return err
	}

	res, err := h.adminUC.GetRenewalStats(c.Request.Context(), fromDate, toDate)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetRenewalStatsSuccess, res)
	return nil
}

// GetOverview lấy toàn bộ bức tranh tổng quan hệ thống cho Admin
// @Summary Báo cáo tổng quan Admin Dashboard
// @Description Tổng hợp tất cả dữ liệu từ KPI Cards, AI Margin Analytics và PayOS Reconciliation trong 1 kết quả duy nhất. Panel đối soát PayOS mang ngữ nghĩa theo kỳ mặc định hôm nay (giờ Việt Nam).
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
