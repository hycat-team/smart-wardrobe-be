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
	// successAdminCloseCampaign is worded the same way for the same reason: the
	// response beside it says `status: closed`, and a message claiming the campaign
	// is now closed would be a claim about the FUTURE of a one-way operation whose
	// whole meaning is that it cannot be taken back (FR-034).
	successAdminCloseCampaign = "Đã xử lý yêu cầu đóng chiến dịch."
	// successAdminUpdateCampaign is worded like the other two, and for the same reason:
	// the response beside it reports a `version` that just moved, so a message
	// claiming "the campaign has been updated" would describe an instant the client
	// has not seen yet. It says the REQUEST was processed, which is what happened.
	successAdminUpdateCampaign = "Đã xử lý yêu cầu cập nhật chiến dịch."
)

// CampaignRegistryHandler serves the three campaign write endpoints.
//
// The type is introduced by US1 with `Create` alone; US2 adds `Close` (T043) and US3
// adds `Update` (T053), each with its route in `router.go`, in the order the features
// depend on each other.
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

// Update
// @Summary Sửa một chiến dịch đang tồn tại
// @Description Sửa planSlug/quota/reserve khi chiến dịch chưa cấp gói cho ai, và sửa startsAt/endsAt bất kể đã cấp hay chưa. Trả 404 khi mã không tồn tại (trước cả khi so phiên bản), 422 khi chiến dịch đã cấp mà thân có đổi planSlug/quota/reserve, 412 khi phiên bản lệch, và 200 no-op (không ghi, không tăng phiên bản, không ghi nhật ký) khi không trường nào đổi.
// @Tags Admin Campaign
// @Accept json
// @Produce json
// @Param code path string true "Mã chiến dịch"
// @Param body body sub_dto.CampaignUpdateReq true "Phiên bản mong đợi và các trường cần sửa"
// @Success 200 {object} shared_pres.APIResponse{data=sub_dto.CampaignSummaryRes}
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 403 {object} shared_pres.APIResponse
// @Failure 404 {object} shared_pres.APIResponse
// @Failure 412 {object} shared_pres.APIResponse
// @Failure 422 {object} shared_pres.APIResponse
// @Failure 429 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns/{code} [patch]
func (h *CampaignRegistryHandler) Update(c *gin.Context) error {
	// The SAME `campaignPathParams` the read routes and `Close` bind: `Code` is
	// required identically on all of them, and a second declaration would be a second
	// answer to "what does a missing code look like".
	//
	// AND IT IS THE ONLY PLACE THE CODE EXISTS. `code` is immutable after creation
	// (FR-024), so `CampaignUpdateReq` has no such field: a body that could contradict
	// the URL would have ambiguous authority, and the body value would be silently
	// ignored here.
	var params campaignPathParams
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}

	var req sub_dto.CampaignUpdateReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	// FR-060, for the same reason `Create` and `Close` do it: the audit row has to
	// name WHO changed the campaign, and an unidentified administrator would leave an
	// edit nobody can be shown to have performed.
	actorAdminID, err := contextutils.GetUserId(c)
	if err != nil || actorAdminID == uuid.Nil {
		return apperror.NewUnauthorized("Vui lòng đăng nhập để thực hiện thao tác quản lý chiến dịch.")
	}

	// FR-060: the request IP makes a disputed edit traceable to a connection and not
	// only to an account.
	clientIP := c.ClientIP()

	summary, err := h.registry.Update(c.Request.Context(), actorAdminID, params.Code, &req, &clientIP)
	if err != nil {
		return err
	}

	// 200, INCLUDING for the no-op (FR-055's exception, contract §4.4). The handler
	// cannot tell a no-op from a real edit and does not need to: the state the
	// operator asked for is the state the response describes, and the body carries the
	// version the row actually has so the next edit from this screen is not stale.
	shared_pres.Success(c, successAdminUpdateCampaign, summary)
	return nil
}

// Close
// @Summary Đóng cưỡng bức một chiến dịch
// @Description Ghi closed_at và tăng phiên bản trong cùng giao dịch với dòng nhật ký kiểm toán campaign.close. Thao tác là một chiều: không có API mở lại. Trả 404 khi mã không tồn tại, 422 khi lý do đóng rỗng, và 412 khi phiên bản lệch với một chiến dịch chưa đóng. Gọi lại trên chiến dịch đã đóng trả 200 idempotent và không ghi thêm bản ghi kiểm toán.
// @Tags Admin Campaign
// @Accept json
// @Produce json
// @Param code path string true "Mã chiến dịch"
// @Param body body sub_dto.CampaignCloseReq true "Phiên bản mong đợi và lý do đóng"
// @Success 200 {object} shared_pres.APIResponse{data=sub_dto.CampaignSummaryRes}
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 403 {object} shared_pres.APIResponse
// @Failure 404 {object} shared_pres.APIResponse
// @Failure 412 {object} shared_pres.APIResponse
// @Failure 422 {object} shared_pres.APIResponse
// @Failure 429 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns/{code}/close [post]
func (h *CampaignRegistryHandler) Close(c *gin.Context) error {
	// The SAME `campaignPathParams` the read routes bind, rather than a type of its
	// own: `Code` is required identically on all of them, and a second declaration
	// would be a second answer to "what does a missing code look like" (contract §4).
	var params campaignPathParams
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}

	var req sub_dto.CampaignCloseReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	// FR-060, for the same reason `Create` does it: the audit row has to name WHO
	// closed the campaign, and an unidentified administrator would leave a closure
	// nobody can be shown to have performed.
	actorAdminID, err := contextutils.GetUserId(c)
	if err != nil || actorAdminID == uuid.Nil {
		return apperror.NewUnauthorized("Vui lòng đăng nhập để thực hiện thao tác quản lý chiến dịch.")
	}

	// FR-060: the request IP is recorded so a disputed closure can be traced to a
	// connection and not only to an account. A closure is a one-way operation, so a
	// dispute about it has no later correction to appeal to — the trail is all there is.
	clientIP := c.ClientIP()

	summary, err := h.registry.Close(c.Request.Context(), actorAdminID, params.Code, &req, &clientIP)
	if err != nil {
		return err
	}

	// 200, INCLUDING for the idempotent no-op on an already-closed campaign
	// (contract §5.3). The handler cannot tell the two apart and does not need to:
	// both are "the campaign is closed", which is the state the operator asked for.
	shared_pres.Success(c, successAdminCloseCampaign, summary)
	return nil
}
