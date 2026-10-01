package handler

import (
	"strings"
	"unicode/utf8"

	admin_contract "smart-wardrobe-be/services/account/contract/admin"
	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	"smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 024 T088: FR-062 fixes the batch at 1..500 items. The size is a constant
// rather than configuration because it is part of the API contract — a
// configurable limit would let a deployment accept a request shape the web
// console cannot render. The RATE tiers come from config (FR-078), because how
// much an administrator may do per minute is an operational decision, not an API
// promise.
const (
	adminGrantMaxItems   = 500
	headerIdempotencyKey = "Idempotency-Key"

	// 024: adminGrantMaxIdempotencyKeyLength bounds the caller's key so the event key
	// it becomes still fits the column.
	//
	// The event key is built as `ADMIN_GRANT:{key}:{itemIndex}` and lands in
	// `user_subscription_events.event_key`, which is `character varying(255) NOT NULL
	// UNIQUE`. So the usable length is 255 minus the 12 characters of the prefix, minus
	// the colon, minus the widest item index (3 digits, for 500 items): 239.
	//
	// The limit is counted in CHARACTERS, not bytes, because that is what the column
	// counts. `varchar(255)` in PostgreSQL is 255 characters, so a Vietnamese key of
	// 200 ký tự occupies 600 bytes and still fits — counting bytes would refuse it for
	// no reason. This is measured with utf8.RuneCountInString to match the column
	// exactly, rather than being "close enough".
	//
	// Without this check an over-long key is accepted here and only fails later, on
	// INSERT. The failure then surfaces per item as `grant_failed`, a reason code that
	// says "the infrastructure failed" — for a request the administrator got exactly
	// right. Refusing it at the edge names the real cause, which is what FR-085 asks
	// for and what T129 established when it stopped reporting caller mistakes as
	// infrastructure failures.
	adminGrantMaxIdempotencyKeyLength = 239
)

const (
	successAdminListPlans         = "Lấy danh sách gói cước thành công"
	successAdminCreatePlan        = "Tạo gói cước mới thành công"
	successAdminUpdatePlan        = "Cập nhật gói cước thành công"
	successAdminSetPlanActive     = "Cập nhật trạng thái hoạt động của gói cước thành công"
	successAdminListSubscriptions = "Lấy danh sách đăng ký hội viên thành công"
	successAdminCancelSub         = "Hủy gói hội viên của người dùng thành công"
	// 024 T088: the batch answers with a per-item outcome, so this message says
	// the REQUEST was processed, not that every item succeeded. "Cấp gói thành
	// công" would contradict the results array beside it.
	successAdminGrantSub = "Đã xử lý yêu cầu cấp gói."
)

type AdminSubscriptionHandler struct {
	logger           logger.Interface
	planUseCase      uc_interfaces.ISubscriptionPlanUseCase
	subscriptionUC   uc_interfaces.ISubscriptionUseCase
	auditLogContract admin_contract.IAuditLogContract
}

func NewAdminSubscriptionHandler(
	logger logger.Interface,
	planUseCase uc_interfaces.ISubscriptionPlanUseCase,
	subscriptionUC uc_interfaces.ISubscriptionUseCase,
	auditLogContract admin_contract.IAuditLogContract,
) *AdminSubscriptionHandler {
	return &AdminSubscriptionHandler{
		logger:           logger,
		planUseCase:      planUseCase,
		subscriptionUC:   subscriptionUC,
		auditLogContract: auditLogContract,
	}
}

// ListPlans
// @Summary Danh sách tất cả gói cước
// @Description Lấy toàn bộ danh sách gói cước trong hệ thống (bao gồm cả gói đang tắt).
// @Tags Admin Subscription
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans [get]
func (h *AdminSubscriptionHandler) ListPlans(c *gin.Context) error {
	plans, err := h.planUseCase.ListAdminPlans(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListPlans, plans)
	return nil
}

// CreatePlan
// @Summary Tạo gói cước mới
// @Description Tạo một gói cước mới trong hệ thống.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param body body dto.CreateSubscriptionPlanReq true "Thông tin gói cước"
// @Success 201 {object} shared_pres.APIResponse{data=dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans [post]
func (h *AdminSubscriptionHandler) CreatePlan(c *gin.Context) error {
	var req dto.CreateSubscriptionPlanReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	created, err := h.planUseCase.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionPlanCreate),
		TargetType:  string(adminaudit.SubscriptionPlan),
		TargetID:    &created.ID,
		Payload: map[string]any{
			"planCode": created.Slug,
			"name":     created.Name,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Created(c, successAdminCreatePlan, created)
	return nil
}

type adminPlanIDParam struct {
	ID string `uri:"id" binding:"required"`
}

// UpdatePlan
// @Summary Cập nhật thông tin gói cước
// @Description Cập nhật thông tin chi tiết một gói cước theo ID.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param id path string true "ID gói cước"
// @Param body body dto.UpdateSubscriptionPlanReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans/{id} [put]
func (h *AdminSubscriptionHandler) UpdatePlan(c *gin.Context) error {
	var params adminPlanIDParam
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	id, err := uuid.Parse(params.ID)
	if err != nil {
		return err
	}
	var req dto.UpdateSubscriptionPlanReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	updated, err := h.planUseCase.UpdatePlan(c.Request.Context(), id, &req)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionPlanUpdate),
		TargetType:  string(adminaudit.SubscriptionPlan),
		TargetID:    &id,
		RequestIP:   &clientIP,
	})

	shared_pres.Success(c, successAdminUpdatePlan, updated)
	return nil
}

// SetPlanActive
// @Summary Bật hoặc tắt trạng thái hoạt động gói cước
// @Description Cập nhật trạng thái isActive của gói cước.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param id path string true "ID gói cước"
// @Param body body dto.SetPlanActiveReq true "Trạng thái hoạt động"
// @Success 200 {object} shared_pres.APIResponse{data=dto.SubscriptionPlanDTO}
// @Router /api/v1/admin/subscriptions/plans/{id}/status [patch]
func (h *AdminSubscriptionHandler) SetPlanActive(c *gin.Context) error {
	var params adminPlanIDParam
	if err := validation.BindURI(c, &params); err != nil {
		return err
	}
	id, err := uuid.Parse(params.ID)
	if err != nil {
		return err
	}
	var req dto.SetPlanActiveReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	updated, err := h.planUseCase.SetPlanActive(c.Request.Context(), id, req.IsActive)
	if err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionPlanDeactivate),
		TargetType:  string(adminaudit.SubscriptionPlan),
		TargetID:    &id,
		Payload: map[string]any{
			"isActive": req.IsActive,
		},
		RequestIP: &clientIP,
	})

	shared_pres.Success(c, successAdminSetPlanActive, updated)
	return nil
}

type adminListSubscriptionsQuery struct {
	shared_dto.PaginationQuery
	PlanCode *string `form:"planCode"`
	Status   *string `form:"status"`
}

// ListSubscriptions
// @Summary Danh sách đăng ký gói của người dùng
// @Description Lấy danh sách đăng ký gói hội viên của người dùng (lọc theo planCode, status).
// @Tags Admin Subscription
// @Produce json
// @Param planCode query string false "Mã slug gói"
// @Param status query string false "Trạng thái (active, expired)"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.AdminUserSubscriptionItemDTO]}
// @Router /api/v1/admin/subscriptions [get]
func (h *AdminSubscriptionHandler) ListSubscriptions(c *gin.Context) error {
	var req adminListSubscriptionsQuery
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	var statusEnum *usersubscriptionstatus.UserSubscriptionStatus
	if req.Status != nil && *req.Status != "" {
		if st, ok := usersubscriptionstatus.ParseUserSubscriptionStatus(*req.Status); ok {
			statusEnum = &st
		}
	}
	items, total, err := h.subscriptionUC.AdminList(c.Request.Context(), req.PlanCode, statusEnum, req.Page, req.Limit)
	if err != nil {
		return err
	}

	response := &shared_dto.PaginationResult[*dto.AdminUserSubscriptionItemDTO]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(req.PaginationQuery, total),
	}

	shared_pres.Success(c, successAdminListSubscriptions, response)
	return nil
}

type adminUserIDParam struct {
	UserID string `uri:"userId" binding:"required"`
}

// CancelSubscription
// @Summary Hủy gói hội viên của người dùng
// @Description Hủy gói hội viên của người dùng và chuyển về gói miễn phí (Free).
// @Tags Admin Subscription
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/{userId}/cancel [post]
func (h *AdminSubscriptionHandler) CancelSubscription(c *gin.Context) error {
	var params adminUserIDParam
	if err := c.ShouldBindUri(&params); err != nil {
		return err
	}
	userID, err := uuid.Parse(params.UserID)
	if err != nil {
		return err
	}
	if err := h.subscriptionUC.AdminCancel(c.Request.Context(), userID); err != nil {
		return err
	}

	adminID, _ := contextutils.GetUserId(c)
	clientIP := c.ClientIP()
	_ = h.auditLogContract.Record(c.Request.Context(), admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      string(adminaudit.SubscriptionCancel),
		TargetType:  string(adminaudit.UserSubscription),
		TargetID:    &userID,
		RequestIP:   &clientIP,
	})

	shared_pres.Success(c, successAdminCancelSub, nil)
	return nil
}

// 024 T178: `adminGrantPlanReq` is GONE. The batch endpoint takes
// `dto.AdminGrantBatchReq`; a single-account grant is a list of one, which is the whole
// point of FR-060 ("AdminGrantBatch is the ONE way an administrator grants a plan").
// This request struct was the leftover of the separate single-account route that
// existed before that consolidation, and no handler read it.

// GrantSubscriptionsBatch
// @Summary Cấp gói hội viên cho danh sách tài khoản
// @Description Cấp gói cho một danh sách tài khoản. Cấp cho một tài khoản là một danh sách một phần tử, dùng chung giao diện này. Mỗi phần tử độc lập: một phần tử lỗi không ảnh hưởng các phần tử còn lại.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param Idempotency-Key header string true "Khoá chống gửi lại, do quản trị viên sinh ra"
// @Param body body dto.AdminGrantBatchReq true "Danh sách tài khoản cần cấp gói"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminGrantBatchRes}
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 429 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscription-grants [post]
func (h *AdminSubscriptionHandler) GrantSubscriptionsBatch(c *gin.Context) error {
	var req dto.AdminGrantBatchReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	// FR-062/FR-063 are checked HERE, not by `binding`, because both outcomes
	// reject the WHOLE request: an empty or oversized list is not a per-item
	// failure, and reporting it as one would produce a 200 with a line of
	// errors, which reads as success.
	if len(req.Items) == 0 {
		return subscriptionerrors.ErrAdminGrantEmptyBatch()
	}
	if len(req.Items) > adminGrantMaxItems {
		return subscriptionerrors.ErrAdminGrantTooManyItems(adminGrantMaxItems)
	}

	// FR-068: the key is required, and it is the only thing that distinguishes a
	// retry from a second deliberate grant.
	identifierKey := c.GetHeader(headerIdempotencyKey)
	// Trim before use, not just before the emptiness test. The header may legitimately
	// arrive with padding, and a key that keeps its spaces is a DIFFERENT key from the
	// same key without them: `" abc "` and `"abc"` would produce two distinct event
	// keys, and a network retry that picked up the padding back would silently
	// re-grant instead of being recognised. Trimming makes the key the administrator
	// meant, which is the whole point of a key chosen to be stable.
	identifierKey = strings.TrimSpace(identifierKey)
	if identifierKey == "" {
		return subscriptionerrors.ErrAdminGrantIdempotencyKeyRequired()
	}
	if keyLength := utf8.RuneCountInString(identifierKey); keyLength > adminGrantMaxIdempotencyKeyLength {
		return subscriptionerrors.ErrAdminGrantIdempotencyKeyTooLong(adminGrantMaxIdempotencyKeyLength, keyLength)
	}
	adminID, _ := contextutils.GetUserId(c)
	if adminID == uuid.Nil {
		// The audit record requires an actor. An unidentified administrator is
		// refused rather than recorded as a nil actor, because a grant that
		// cannot be attributed is exactly what FR-081 exists to prevent.
		return apperror.NewUnauthorized("Vui lòng đăng nhập để thực hiện thao tác cấp gói.")
	}

	// FR-082: every audit row carries the administrator's network address, so a
	// disputed grant can be traced to a connection rather than only to an account.
	//
	// FR-075: the per-item outcome is returned in the SAME request. The
	// administrator does not poll, and a 202 would leave them unable to tell
	// which accounts were actually granted.
	//
	// Rate limiting is NOT done here (024 T099): it lives in
	// AdminGrantRateLimitMiddleware, on the route, where it also covers requests
	// that never reach this function.
	clientIP := c.ClientIP()
	res, err := h.subscriptionUC.AdminGrantBatch(c.Request.Context(), adminID, req.Items, identifierKey, &clientIP)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successAdminGrantSub, res)
	return nil
}

// RestorePolicyGrants
// @Summary Khôi phục mức quyền tính phí AI cho tài khoản thiếu bản ghi
// @Description Quét mọi tài khoản có bản ghi đăng ký và phát lại tín hiệu để dịch vụ AI dựng lại mức quyền. Mặc định dryRun=true. Chạy lại nhiều lần không gây tác dụng phụ.
// @Tags Admin Subscription
// @Accept json
// @Produce json
// @Param body body dto.RestorePolicyGrantsReq false "Tuỳ chọn chạy thử hoặc thực thi"
// @Success 200 {object} shared_pres.APIResponse{data=dto.RestorePolicyGrantsRes}
// @Router /api/v1/admin/subscription-grants/restore [post]
func (h *AdminSubscriptionHandler) RestorePolicyGrants(c *gin.Context) error {
	var req dto.RestorePolicyGrantsReq
	// The body is optional: an operator who omits it gets the safe default. A
	// malformed body is still rejected, so a typo is not mistaken for "no body".
	if c.Request.ContentLength > 0 {
		if err := validation.BindJSON(c, &req); err != nil {
			return err
		}
	}

	// ⚠️ Dry run is the DEFAULT, not the alternative (FR-088). Repairing
	// entitlements across every account is exactly the kind of bulk write whose
	// outcome an operator should see before it happens.
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	adminID, _ := contextutils.GetUserId(c)
	if adminID == uuid.Nil {
		return apperror.NewUnauthorized("Vui lòng đăng nhập để thực hiện khôi phục mức quyền.")
	}

	// SC-021: the repair is traceable to the administrator AND their connection,
	// exactly like the batch grant path. Without this it would be the one
	// administrative write on this feature that cannot be attributed to a source
	// address.
	clientIP := c.ClientIP()
	res, err := h.subscriptionUC.RestorePolicyGrants(c.Request.Context(), adminID, dryRun, &clientIP)
	if err != nil {
		return err
	}

	message := mapper.MsgRestoreApplied
	if dryRun {
		message = mapper.MsgRestoreDryRun
	}
	shared_pres.Success(c, message, res)
	return nil
}
