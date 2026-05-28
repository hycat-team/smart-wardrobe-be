package handler

import (
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
)

type DailyQuotaHandler struct {
	subscriptionUseCase usecase_interfaces.ISubscriptionUseCase
}

func NewDailyQuotaHandler(subUseCase usecase_interfaces.ISubscriptionUseCase) *DailyQuotaHandler {
	return &DailyQuotaHandler{
		subscriptionUseCase: subUseCase,
	}
}

// GetDailyQuota retrieves user daily quota and performs lazy reset if outdated
// @Summary Lấy hạn ngạch sử dụng hàng ngày
// @Description Lấy hạn ngạch chi tiết và trạng thái sử dụng của người dùng trong ngày
// @Tags Subscription
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Hạn ngạch sử dụng"
// @Router /api/v1/subscriptions/daily-quota [get]
func (h *DailyQuotaHandler) GetDailyQuota(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	quotaDTO, err := h.subscriptionUseCase.GetDailyQuota(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy hạn ngạch sử dụng thành công", quotaDTO)
	return nil
}
