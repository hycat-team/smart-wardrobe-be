package handler

import (
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	_ "smart-wardrobe-be/internal/shared/presentation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BrandDashboardHandler struct {
	brandUC uc_interfaces.IBrandDashboardUseCase
}

func NewBrandDashboardHandler(brandUC uc_interfaces.IBrandDashboardUseCase) *BrandDashboardHandler {
	return &BrandDashboardHandler{brandUC: brandUC}
}

// GetKPICards lấy thẻ chỉ số KPI cho Brand Portal
// @Summary Lấy thẻ KPI tổng quan B2B Brand Portal
// @Description Lấy thống kê số hội viên, điểm phát hành, điểm quy đổi, số voucher active/quy đổi của nhãn hàng.
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandKPICardsRes} "Lấy thẻ KPI nhãn hàng thành công"
// @Router /api/v1/brand/dashboard/kpi-cards [get]
func (h *BrandDashboardHandler) GetKPICards(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetBrandKPICards(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thẻ KPI nhãn hàng thành công", res)
	return nil
}

// GetPointsLiability phân tích rủi ro nghĩa vụ tài chính nợ điểm quy đổi (Column-Level RBAC)
// @Summary Phân tích nợ điểm hết hạn trong 30 ngày (Column-Level RBAC)
// @Description Thống kê số điểm hết hạn trong 30 ngày tới & giá trị quy đổi VND tương ứng. Giá trị VND sẽ bị ẩn (redacted) đối với tài khoản cấp Staff.
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandPointsLiabilityRes} "Lấy báo cáo nghĩa vụ điểm quy đổi thành công"
// @Router /api/v1/brand/dashboard/financials/points-liability [get]
func (h *BrandDashboardHandler) GetPointsLiability(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetPointsLiability(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy báo cáo nghĩa vụ điểm quy đổi thành công", res)
	return nil
}

// GetCustomerAcquisition lấy thống kê kênh thu hút khách hàng
// @Summary Thống kê kênh thu hút khách hàng
// @Description Thống kê số lượng khách hàng và tỷ lệ phần trăm theo từng kênh thu hút (joined_source)
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CustomerAcquisitionRes} "Lấy thống kê kênh thu hút khách hàng thành công"
// @Router /api/v1/brand/dashboard/customers/acquisition [get]
func (h *BrandDashboardHandler) GetCustomerAcquisition(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetCustomerAcquisition(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thống kê kênh thu hút khách hàng thành công", res)
	return nil
}

// GetLoyaltyTierDistribution lấy thống kê phân bổ hạng loyalty
// @Summary Phân bổ hạng loyalty thành viên
// @Description Thống kê số lượng thành viên theo từng hạng loyalty (Bronze, Silver, Gold, v.v.)
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.TierDistributionRes} "Lấy phân bổ hạng loyalty thành công"
// @Router /api/v1/brand/dashboard/loyalty/tier-distribution [get]
func (h *BrandDashboardHandler) GetLoyaltyTierDistribution(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetLoyaltyTierDistribution(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy phân bổ hạng loyalty thành công", res)
	return nil
}

// GetPointExpiryForecast lấy dự báo điểm sắp hết hạn
// @Summary Dự báo điểm loyalty sắp hết hạn
// @Description Dự báo số điểm sắp hết hạn theo từng tháng dựa trên dữ liệu loyalty_point_lots
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PointExpiryForecastRes} "Lấy dự báo điểm hết hạn thành công"
// @Router /api/v1/brand/dashboard/loyalty/point-expiry-forecast [get]
func (h *BrandDashboardHandler) GetPointExpiryForecast(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetPointExpiryForecast(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy dự báo điểm hết hạn thành công", res)
	return nil
}

// GetBenefitRedemptionAnalytics lấy thống kê đổi benefit
// @Summary Thống kê đổi benefit
// @Description Thống kê top benefit được đổi nhiều nhất và xu hướng đổi benefit theo thời gian
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BenefitRedemptionAnalyticsRes} "Lấy thống kê đổi benefit thành công"
// @Router /api/v1/brand/dashboard/benefits/redemption-analytics [get]
func (h *BrandDashboardHandler) GetBenefitRedemptionAnalytics(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetBenefitRedemptionAnalytics(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thống kê đổi benefit thành công", res)
	return nil
}

// GetCatalogStats lấy thống kê sản phẩm trong catalog
// @Summary Thống kê catalog sản phẩm
// @Description Thống kê tổng số sản phẩm trong catalog, phân bổ theo trạng thái và loại sản phẩm
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CatalogStatsRes} "Lấy thống kê catalog thành công"
// @Router /api/v1/brand/dashboard/operations/catalog-stats [get]
func (h *BrandDashboardHandler) GetCatalogStats(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetCatalogStats(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thống kê catalog thành công", res)
	return nil
}

// GetCustomerSpendSegmentation lấy phân khúc chi tiêu khách hàng
// @Summary Phân khúc chi tiêu khách hàng
// @Description Thống kê số lượng khách hàng theo các nhóm chi tiêu (low, medium, high, premium)
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.SpendSegmentationRes} "Lấy phân khúc chi tiêu thành công"
// @Router /api/v1/brand/dashboard/customers/spend-segmentation [get]
func (h *BrandDashboardHandler) GetCustomerSpendSegmentation(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetCustomerSpendSegmentation(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy phân khúc chi tiêu thành công", res)
	return nil
}

// GetDigitalSampleLabAnalytics phân tích phản hồi sản phẩm mẫu thử Digital Sample Lab
// @Summary Phân tích đánh giá mẫu thử Digital Sample Lab
// @Description Thống kê số lượt bình chọn, rating trung bình, sentiment phản hồi và danh sách danh mục trang phục thường phối kèm nhiều nhất.
// @Tags Dashboard Brand
// @Produce json
// @Param brandId query string true "ID Nhãn hàng"
// @Param itemId path string true "ID Sản phẩm mẫu thử"
// @Success 200 {object} shared_pres.APIResponse{data=dto.DigitalSampleLabAnalyticsRes} "Lấy phân tích mẫu thử thành công"
// @Router /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId} [get]
func (h *BrandDashboardHandler) GetDigitalSampleLabAnalytics(c *gin.Context) error {
	var req dto.GetBrandDashboardQueryReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		return err
	}

	res, err := h.brandUC.GetDigitalSampleLabAnalytics(c.Request.Context(), brandID, itemID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy phân tích mẫu thử thành công", res)
	return nil
}
