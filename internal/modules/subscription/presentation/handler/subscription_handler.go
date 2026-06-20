package handler

import (
	_ "smart-wardrobe-be/internal/modules/subscription/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	_ "smart-wardrobe-be/internal/modules/subscription/contract"
	req_dto "smart-wardrobe-be/internal/modules/subscription/presentation/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
)

const (
	successGetUserSubscriptionOverview = "Lấy thông tin gói hội viên hiện tại thành công"
	successGetDailyQuota               = "Lấy hạn ngạch sử dụng thành công"
	successSetAutoRenewStatus          = "Thay đổi trạng thái tự động gia hạn thành công"
	successGetPlans                    = "Lấy danh sách gói Premium thành công"
)

type SubscriptionHandler struct {
	subscriptionUseCase usecase_interfaces.ISubscriptionUseCase
}

func NewSubscriptionHandler(subUseCase usecase_interfaces.ISubscriptionUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionUseCase: subUseCase,
	}
}

// GetUserSubscriptionOverview retrieves user current subscription details
// @Summary Lấy thông tin gói hội viên hiện tại
// @Description Lấy thông tin chi tiết gói hội viên đang kích hoạt của người dùng hiện tại.
// @Description Định nghĩa enum PlanKind:
// @Description - 0: DefaultFree (Gói miễn phí mặc định)
// @Description - 1: Finite (Gói giới hạn số ngày sử dụng)
// @Description - 2: Lifetime (Gói trọn đời không giới hạn thời gian)
// @Description Định nghĩa TierRank (Cấp độ gói):
// @Description - 0: Cấp Free
// @Description - 1: Cấp Premium (Gói có cấp Premium lớn hơn sẽ được ưu tiên kích hoạt trước)
// @Tags Subscription
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=contract.UserSubscriptionOverviewDTO} "Thông tin gói hội viên hiện tại"
// @Router /api/v1/subscriptions/me [get]
// @Security BearerAuth
func (h *SubscriptionHandler) GetUserSubscriptionOverview(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	overviewDTO, err := h.subscriptionUseCase.GetUserSubscriptionOverview(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetUserSubscriptionOverview, overviewDTO)
	return nil
}

// GetDailyQuota retrieves user daily quota and performs lazy reset if outdated
// @Summary Lấy hạn ngạch sử dụng hàng ngày
// @Description Lấy hạn ngạch chi tiết và trạng thái sử dụng của người dùng trong ngày.
// @Description Định nghĩa enum PlanKind:
// @Description - 0: DefaultFree (Gói miễn phí mặc định)
// @Description - 1: Finite (Gói giới hạn số ngày sử dụng)
// @Description - 2: Lifetime (Gói trọn đời không giới hạn thời gian)
// @Description Định nghĩa TierRank (Cấp độ gói):
// @Description - 0: Cấp Free
// @Description - 1: Cấp Premium (Gói có cấp Premium lớn hơn sẽ được ưu tiên kích hoạt trước)
// @Tags Subscription
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=contract.UserSubscriptionDTO} "Hạn ngạch sử dụng và thông tin gói hiện tại"
// @Router /api/v1/subscriptions/me/daily-quota [get]
// @Security BearerAuth
func (h *SubscriptionHandler) GetDailyQuota(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	quotaDTO, err := h.subscriptionUseCase.GetDailyQuota(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetDailyQuota, quotaDTO)
	return nil
}

// SetAutoRenewStatus updates the automatic subscription renewal setting
// @Summary Thiết lập tự động gia hạn gói cước
// @Description Thiết lập bật hoặc tắt tính năng tự động gia hạn gói cước qua ví nội bộ khi hết hạn
// @Tags Subscription
// @Accept json
// @Produce json
// @Param body body req_dto.SetAutoRenewReq true "Trạng thái thiết lập tự động gia hạn"
// @Success 200 {object} shared_pres.APIResponse "Trạng thái tự động gia hạn mới"
// @Router /api/v1/subscriptions/me/auto-renew [put]
// @Security BearerAuth
func (h *SubscriptionHandler) SetAutoRenewStatus(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req req_dto.SetAutoRenewReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	isEnabled, err := h.subscriptionUseCase.SetAutoRenewStatus(c.Request.Context(), userID, *req.Enabled)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successSetAutoRenewStatus, gin.H{
		"isAutoRenewEnabled": isEnabled,
	})
	return nil
}

// GetPlans retrieves all subscription plans
// @Summary Lấy danh sách các gói Premium
// @Description Lấy danh sách tất cả các gói đăng ký Premium hiện có.
// @Description Định nghĩa enum PlanKind:
// @Description - 0: DefaultFree (Gói miễn phí mặc định)
// @Description - 1: Finite (Gói giới hạn số ngày sử dụng)
// @Description - 2: Lifetime (Gói trọn đời không giới hạn thời gian)
// @Description Định nghĩa TierRank (Cấp độ gói):
// @Description - 0: Cấp Free
// @Description - 1: Cấp Premium (Gói có cấp Premium lớn hơn sẽ được ưu tiên kích hoạt trước)
// @Tags Subscription
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.SubscriptionPlanDTO} "Danh sách gói cước"
// @Router /api/v1/subscriptions/plans [get]
func (h *SubscriptionHandler) GetPlans(c *gin.Context) error {
	plans, err := h.subscriptionUseCase.GetPlans(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetPlans, plans)
	return nil
}
