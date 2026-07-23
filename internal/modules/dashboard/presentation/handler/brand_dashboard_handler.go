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
// @Tags Brand Dashboard
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
// @Tags Brand Dashboard
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

// GetDigitalSampleLabAnalytics phân tích phản hồi sản phẩm mẫu thử Digital Sample Lab
// @Summary Phân tích đánh giá mẫu thử Digital Sample Lab
// @Description Thống kê số lượt bình chọn, rating trung bình, sentiment phản hồi và danh sách danh mục trang phục thường phối kèm nhiều nhất.
// @Tags Brand Dashboard
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
