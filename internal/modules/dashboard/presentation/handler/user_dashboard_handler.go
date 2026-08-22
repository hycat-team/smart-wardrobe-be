package handler

import (
	_ "smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	_ "smart-wardrobe-be/internal/shared/presentation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgUserGetWardrobeCategoryDistributionSuccess = "Lấy phân bổ danh mục tủ đồ thành công"
	msgUserGetWardrobeInsightsSuccess             = "Lấy phân tích tủ đồ cá nhân thành công"
)

type UserDashboardHandler struct {
	userUC uc_interfaces.IUserDashboardUseCase
}

func NewUserDashboardHandler(userUC uc_interfaces.IUserDashboardUseCase) *UserDashboardHandler {
	return &UserDashboardHandler{userUC: userUC}
}

// GetWardrobeCategoryDistribution lấy phân bổ danh mục tủ đồ
// @Summary Phân bổ danh mục tủ đồ
// @Description Thống kê số lượng và tỷ lệ phần trăm item trong tủ đồ theo từng danh mục thời trang
// @Tags Dashboard User
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeCategoryDistributionRes} "Lấy phân bổ danh mục tủ đồ thành công"
// @Router /api/v1/me/dashboard/wardrobe/category-distribution [get]
func (h *UserDashboardHandler) GetWardrobeCategoryDistribution(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	res, err := h.userUC.GetWardrobeCategoryDistribution(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgUserGetWardrobeCategoryDistributionSuccess, res)
	return nil
}

// GetWardrobeInsights lấy thống kê phân tích tủ đồ thông minh cho End User
// @Summary Thống kê phân tích tủ đồ cá nhân
// @Description Lấy tổng giá trị tài chính tủ đồ, giá trị trung bình mỗi món đồ (tổng giá trị / số món), tổng số đồ lãng phí chưa mặc > 30 ngày và danh sách đề xuất.
// @Tags Dashboard User
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UserWardrobeInsightsRes} "Lấy phân tích tủ đồ cá nhân thành công"
// @Router /api/v1/me/dashboard/wardrobe-insights [get]
func (h *UserDashboardHandler) GetWardrobeInsights(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	res, err := h.userUC.GetWardrobeInsights(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgUserGetWardrobeInsightsSuccess, res)
	return nil
}
