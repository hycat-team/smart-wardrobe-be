package handler

import (
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/presentation/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

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
// @Router /api/v1/subscriptions/me/daily-quota [get]
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

// SetAutoRenewStatus updates the automatic subscription renewal setting
// @Summary Thiết lập tự động gia hạn gói cước
// @Description Thiết lập bật hoặc tắt tính năng tự động gia hạn gói cước qua ví nội bộ khi hết hạn
// @Tags Subscription
// @Accept json
// @Produce json
// @Param body body dto.SetAutoRenewReq true "Trạng thái thiết lập tự động gia hạn"
// @Success 200 {object} shared_pres.APIResponse "Trạng thái tự động gia hạn mới"
// @Router /api/v1/subscriptions/me/toggle-auto-renew [patch]
func (h *DailyQuotaHandler) SetAutoRenewStatus(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req dto.SetAutoRenewReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	isEnabled, err := h.subscriptionUseCase.SetAutoRenewStatus(c.Request.Context(), userID, *req.Enabled)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Thay đổi trạng thái tự động gia hạn thành công", gin.H{
		"is_auto_renew_enabled": isEnabled,
	})
	return nil
}

// GetPlans retrieves all subscription plans
// @Summary Lấy danh sách các gói Premium
// @Description Lấy danh sách tất cả các gói đăng ký Premium hiện có
// @Tags Subscription
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Danh sách gói cước"
// @Router /api/v1/subscriptions/plans [get]
func (h *DailyQuotaHandler) GetPlans(c *gin.Context) error {
	plans, err := h.subscriptionUseCase.GetPlans(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy danh sách gói Premium thành công", plans)
	return nil
}
