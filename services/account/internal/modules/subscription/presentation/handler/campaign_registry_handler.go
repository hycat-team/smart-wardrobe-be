package handler

import (
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 026 T034 (FR-052, FR-060, FR-063): the WRITE face of the campaign registry.
//
// THIN BY CONSTRUCTION, like `CampaignAdminHandler` and for the same reason
// (constitution §IV): bind, delegate, wrap. Every rule about what a campaign may look
// like lives in the use case and its validator, so a second copy of one here would be a
// second answer to the same question — and this is the layer that is unreachable from a
// test that exercises the rules, which is how the two copies would drift.
//
// IT IS A SEPARATE HANDLER FROM `CampaignAdminHandler`, NOT ANOTHER METHOD ON IT. That
// handler's constructor takes `ICampaignQueryUseCase`, an interface with NO write
// method behind it, and FR-069's read-only guarantee is that fact rather than a habit.
// Adding a unit of work and a registry writer to that type would delete the guarantee
// in one edit, and 025 built the separation precisely so that "the look-up cannot spend
// a slot" would be visible in a constructor's parameter list.
const (
	// successAdminCreateCampaign says the REQUEST WAS PROCESSED rather than that
	// everything is fine. "Tạo chiến dịch thành công" placed next to a response whose
	// `status` is `not_started` reads as a contradiction; this wording does not.
	successAdminCreateCampaign = "Đã xử lý yêu cầu tạo chiến dịch."
)

// CampaignRegistryHandler serves the three campaign write endpoints.
//
// The type is introduced by US1 with `Create` alone; `Update` (T053) and `Close` (T043)
// add their methods here, and their routes in `router.go`, in the same order the
// features depend on each other.
type CampaignRegistryHandler struct {
	logger   logger.Interface
	registry uc_interfaces.ICampaignRegistryUseCase
}

// NewCampaignRegistryHandler builds the registry writer handler.
func NewCampaignRegistryHandler(
	log logger.Interface,
	registry uc_interfaces.ICampaignRegistryUseCase,
) *CampaignRegistryHandler {
	return &CampaignRegistryHandler{logger: log, registry: registry}
}

// Create
// @Summary Mở một chiến dịch mới
// @Description Ghi một dòng vào sổ đăng ký chiến dịch và ghi nhật ký kiểm toán trong cùng một giao dịch. Trả 409 khi mã đã tồn tại, 400 khi yêu cầu hỏng hình dạng (mã sai định dạng hoặc thời gian không phải RFC3339 có múi giờ) và 422 khi yêu cầu đúng hình dạng nhưng sai nghĩa (quota dưới 1, reserve âm, endsAt không sau startsAt, planSlug rỗng).
// @Tags Admin Campaign
// @Accept json
// @Produce json
// @Param body body sub_dto.CampaignCreateReq true "Thông tin chiến dịch cần mở"
// @Success 201 {object} shared_pres.APIResponse{data=sub_dto.CampaignSummaryRes}
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 403 {object} shared_pres.APIResponse
// @Failure 409 {object} shared_pres.APIResponse
// @Failure 422 {object} shared_pres.APIResponse
// @Failure 429 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns [post]
func (h *CampaignRegistryHandler) Create(c *gin.Context) error {
	var req sub_dto.CampaignCreateReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	// FR-060: the audit row names WHO did this, so an unidentified administrator cannot
	// be recorded — it would leave a campaign with a nil actor and no way to answer
	// "who opened this". Refused rather than defaulted, for the same reason the batch
	// grant refuses an unidentified caller (024 FR-082).
	actorAdminID, err := contextutils.GetUserId(c)
	if err != nil || actorAdminID == uuid.Nil {
		return apperror.NewUnauthorized("Vui lòng đăng nhập để thực hiện thao tác quản lý chiến dịch.")
	}

	// FR-060: the request IP is recorded so a disputed change can be traced to a
	// connection and not only to an account.
	clientIP := c.ClientIP()

	summary, err := h.registry.Create(c.Request.Context(), actorAdminID, &req, &clientIP)
	if err != nil {
		return err
	}

	// 201, not 200: a campaign now EXISTS in the registry, which is what the whole
	// feature is for (contract §3.2). `shared_pres.Created` exists for exactly this and
	// answers 200 for everything else, so the distinction survives the refactor that
	// would otherwise replace it.
	shared_pres.Created(c, successAdminCreateCampaign, summary)
	return nil
}
