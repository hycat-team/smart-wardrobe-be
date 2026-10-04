package handler

import (
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

// Thông điệp thành công (tiếng Việt — hiến phán: mọi thông điệp cho client phải
// viết bằng tiếng Việt, và hợp đồng §6 yêu cầu khai báo hằng số ở đầu tệp handler).
//
// Chúng nói REQUEST ĐÃ ĐƯỢC XỬ LÝ, không phải rằng mọi thứ đều ổn. Một thông điệp
// kiểu "Lấy danh sách chiến dịch thành công" đặt cạnh `eligibility:
// "eligible_but_exhausted"` sẽ đọc như một mâu thuẫn; "Đã xử lý yêu cầu tra cứu"
// thì không.
const (
	successAdminListCampaigns         = "Đã xử lý yêu cầu tra cứu danh sách chiến dịch."
	successAdminGetCampaign           = "Đã xử lý yêu cầu tra cứu trạng thái chiến dịch."
	successAdminListCampaignClaims    = "Đã xử lý yêu cầu tra cứu danh sách tài khoản được cấp gói."
	successAdminGetAccountEligibility = "Đã xử lý yêu cầu tra cứu trạng thái tài khoản trong chiến dịch."
	// successAdminGetCampaignAudit says the REQUEST WAS READ, which is a different claim
	// from "the trail is complete" and is the only one this endpoint can make.
	//
	// Nhật ký kiểm toán được ghi BẤT ĐỒNG BỘ từ dòng outbox (FR-060), nên một thao tác vừa
	// thành công có thể chưa xuất hiện. Một thông điệp khẳng định "đã xem đủ nhật ký" sẽ
	// khiến người vận hành tưởng thao tác vừa rồi chưa được ghi lại — đúng nhầm lẫn mà
	// FR-060 đã cảnh báo. Cùng cách viết với ba thông điệp trên: yêu cầu đã được xử lý.
	successAdminGetCampaignAudit = "Đã xử lý yêu cầu tra cứu nhật ký chiến dịch."
)

// CampaignAdminHandler serves the read-only operator look-up (025 T081, FR-065 to
// FR-072).
//
// THIN BY CONSTRUCTION (constitution §IV): every method binds, delegates and
// wraps. There is no rule here, no arithmetic and no query — which is what makes
// FR-069 checkable. The handler holds an interface that has no write method on it,
// so there is nothing for a handler to call that could spend a slot, grant a plan
// or send an email.
type CampaignAdminHandler struct {
	logger   logger.Interface
	campaign uc_interfaces.ICampaignQueryUseCase
}

func NewCampaignAdminHandler(logger logger.Interface, campaign uc_interfaces.ICampaignQueryUseCase) *CampaignAdminHandler {
	return &CampaignAdminHandler{logger: logger, campaign: campaign}
}

// campaignClaimsQuery is the query string of the claims endpoint (FR-067).
//
// `From` and `To` are STRINGS, not times, and they are parsed by the use case
// rather than bound here. Binding a `time.Time` with a struct tag would hand the
// parse to Gin's default time decoder, which accepts a format the operator's own
// documentation does not show — so the accepted inputs would differ from the
// documented ones with nothing to say so.
type campaignClaimsQuery struct {
	shared_dto.PaginationQuery
	From string `form:"from"`
	To   string `form:"to"`
}

// campaignPathParams is the `:code` of every campaign-scoped route.
//
// `binding:"required"` rather than a hand-written emptiness check: the contract
// (§4) says a missing required query parameter is a 400 from
// `validation.BindQuery`, and letting the shared binder produce it keeps the
// message consistent with every other endpoint in the service instead of
// inventing a sixth wording for the same problem.
type campaignPathParams struct {
	Code string `uri:"code" binding:"required"`
}

// campaignEligibilityPathParams adds `:userId` for the single-account route.
//
// The two are separate types rather than one type with an optional field because
// `:userId` is REQUIRED on exactly one route and must be on no other: an optional
// tag would let the claims route accept a stray `:userId` and silently ignore it.
type campaignEligibilityPathParams struct {
	Code   string `uri:"code" binding:"required"`
	UserID string `uri:"userId" binding:"required"`
}

// ListCampaigns
// @Summary Tra cứu danh sách chiến dịch
// @Description Trả về trạng thái dẫn xuất của mọi chiến dịch: số lượt cấp, số suất còn lại, trần cứng, trạng thái vòng đời và cờ sức khỏe của lần quét bù gần nhất.
// @Tags Admin Campaign
// @Produce json
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[sub_dto.CampaignSummaryRes]}
// @Router /api/v1/admin/campaigns [get]
func (h *CampaignAdminHandler) ListCampaigns(c *gin.Context) error {
	var req shared_dto.PaginationQuery
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	items, total, err := h.campaign.ListCampaigns(c.Request.Context(), req.Page, req.Limit)
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListCampaigns, &shared_dto.PaginationResult[*sub_dto.CampaignSummaryRes]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(req, total),
	})
	return nil
}

// GetCampaign
// @Summary Tra cứu trạng thái một chiến dịch
// @Description Trả về trạng thái dẫn xuất của một chiến dịch theo mã. Trả về 404 kèm thông điệp tiếng Việt khi mã không tồn tại.
// @Tags Admin Campaign
// @Produce json
// @Param code path string true "Mã chiến dịch"
// @Success 200 {object} shared_pres.APIResponse{data=sub_dto.CampaignSummaryRes}
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 403 {object} shared_pres.APIResponse
// @Failure 404 {object} shared_pres.APIResponse
// @Failure 429 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns/{code} [get]
func (h *CampaignAdminHandler) GetCampaign(c *gin.Context) error {
	var params campaignPathParams
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	summary, err := h.campaign.GetCampaign(c.Request.Context(), params.Code)
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminGetCampaign, summary)
	return nil
}

// ListClaims
// @Summary Tra cứu danh sách tài khoản đã được cấp gói
// @Description Trả về danh sách tài khoản đã được cấp gói theo chiến dịch, có phân trang và lọc theo khoảng thời điểm cấp.
// @Tags Admin Campaign
// @Produce json
// @Param code path string true "Mã chiến dịch"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Param from query string false "Thời điểm cấp bắt đầu (RFC3339)"
// @Param to query string false "Thời điểm cấp kết thúc, không bao gồm (RFC3339)"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[sub_dto.CampaignClaimRes]}
// @Failure 404 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns/{code}/claims [get]
func (h *CampaignAdminHandler) ListClaims(c *gin.Context) error {
	var params campaignPathParams
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	var req campaignClaimsQuery
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	filter, err := parseClaimsWindow(req)
	if err != nil {
		return err
	}
	items, total, err := h.campaign.ListClaims(c.Request.Context(), params.Code, filter, req.Page, req.Limit)
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListCampaignClaims, &shared_dto.PaginationResult[*sub_dto.CampaignClaimRes]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(req.PaginationQuery, total),
	})
	return nil
}

// GetAccountEligibility
// @Summary Tra cứu trạng thái một tài khoản trong chiến dịch
// @Description Trả về kết luận cho MỘT tài khoản: đã được cấp, đủ điều kiện nhưng hết suất, hay không đủ điều kiện. Đây là câu trả lời trực tiếp cho câu hỏi của khách hàng.
// @Tags Admin Campaign
// @Produce json
// @Param code path string true "Mã chiến dịch"
// @Param userId path string true "ID tài khoản"
// @Success 200 {object} shared_pres.APIResponse{data=sub_dto.CampaignAccountEligibilityRes}
// @Failure 404 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns/{code}/eligibility/{userId} [get]
func (h *CampaignAdminHandler) GetAccountEligibility(c *gin.Context) error {
	var params campaignEligibilityPathParams
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	userID, err := validation.GetUUIDParam(c, "userId")
	if err != nil {
		return err
	}
	result, err := h.campaign.GetAccountEligibility(c.Request.Context(), params.Code, userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminGetAccountEligibility, result)
	return nil
}

// GetCampaignAudit
// @Summary Tra cứu nhật ký kiểm toán của một chiến dịch
// @Description Trả về nhật ký ai/lúc nào/đổi gì cho đúng một chiến dịch, có phân trang. Mọi bản ghi có targetType là campaign và targetId là null; mã chiến dịch nằm trong payload.campaignCode. Trả 404 khi mã không tồn tại, và nhật ký rỗng không phải câu trả lời cho một mã lạ.
// @Tags Admin Campaign
// @Produce json
// @Param code path string true "Mã chiến dịch"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[sub_dto.CampaignAuditLogRes]}
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 403 {object} shared_pres.APIResponse
// @Failure 404 {object} shared_pres.APIResponse
// @Failure 429 {object} shared_pres.APIResponse
// @Router /api/v1/admin/campaigns/{code}/audit [get]
//
// IT IS HERE, ON THE READ-ONLY LOOK-UP, AND NOT ON `CampaignRegistryHandler`, because
// nothing about it writes. The trail is written by the three write operations through the
// audit outbox; this one only reads what a consumer already wrote (FR-060), so putting it
// on the writer would give the write side a reason to grow a read method and would move an
// endpoint that grants nobody anything out of the type whose whole purpose is that it does
// not (FR-069).
//
// THE 404 IS NOT DECIDED HERE, and that is the part worth reading. The trail lives in the
// `admin` module, which cannot know which campaigns exist, so it answers an unknown code
// with an EMPTY PAGE. This handler must therefore ask the REGISTRY first — and it must do
// so through the use case, not through the repository: constitution §IV forbids a handler
// reaching a repository, and §II forbids this module reading `shared.admin_audit_logs`
// directly. `CampaignQueryUseCase.ListCampaignAudit` makes that ordering, and the comment
// on the interface method says why it is the requirement rather than a detail.
func (h *CampaignAdminHandler) GetCampaignAudit(c *gin.Context) error {
	// THE SAME `campaignPathParams` the other three reads bind: `:code` is required
	// identically on all of them, and a second declaration would be a second answer to
	// "what does a missing code look like".
	var params campaignPathParams
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	var req shared_dto.PaginationQuery
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	items, total, err := h.campaign.ListCampaignAudit(c.Request.Context(), params.Code, req.Page, req.Limit)
	if err != nil {
		return err
	}
	// THE PAGE IS BUILT HERE AND NOT IN THE USE CASE, so the use case answers the
	// QUESTION (this campaign's trail) and the handler answers the ENVELOPE. That split is
	// why the three other paginated reads on this handler look identical.
	shared_pres.Success(c, successAdminGetCampaignAudit, &shared_dto.PaginationResult[*sub_dto.CampaignAuditLogRes]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(req, total),
	})
	return nil
}

// parseClaimsWindow turns the two optional RFC3339 strings into a filter.
//
// AN EMPTY STRING MEANS UNBOUNDED, not "the year 1" or "now". An operator who
// exports a whole month passes only one bound, and treating the missing one as the
// zero time would silently exclude every grant before 1970 — an empty export that
// looks like a campaign that granted nobody.
//
// RFC3339 with an EXPLICIT ZONE is required, matching the campaign configuration
// itself (data-model.md §4.2). A zone-less timestamp here would be read in the
// server's zone, seven hours off in production, and the filter would then exclude
// the grants the operator was looking for — silently, because the response would
// simply be shorter than expected.
func parseClaimsWindow(req campaignClaimsQuery) (repositories.CampaignClaimsFilter, error) {
	filter := repositories.CampaignClaimsFilter{}
	from, err := parseOptionalRFC3339(req.From, "from")
	if err != nil {
		return filter, err
	}
	to, err := parseOptionalRFC3339(req.To, "to")
	if err != nil {
		return filter, err
	}
	filter.From = from
	filter.To = to
	return filter, nil
}

// parseOptionalRFC3339 parses one optional timestamp bound.
//
// RFC3339 with an EXPLICIT ZONE is required, matching the campaign configuration
// itself (data-model.md §4.2). A zone-less timestamp would be read in the server's
// zone — seven hours off in production — and the filter would then exclude the very
// grants the operator was looking for, silently, because the response would simply
// be shorter than expected.
//
// A MALFORMED BOUND IS REFUSED, not ignored. Ignoring it would answer a different
// question from the one that was asked, and an operator who typed the wrong format
// would receive a plausible-looking page and conclude the campaign granted fewer
// plans than it did.
func parseOptionalRFC3339(raw, field string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignClaimWindowMalformed(field)
	}
	return &parsed, nil
}
