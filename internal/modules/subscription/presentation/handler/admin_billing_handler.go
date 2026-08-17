package handler

import (
	"strconv"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	successAdminListTransactions        = "Lấy danh sách giao dịch thành công"
	successAdminGetTransaction          = "Lấy chi tiết giao dịch thành công"
	successAdminRetryInvestigation      = "Gửi lại yêu cầu đối soát thành công"
	successAdminResolveAsPaid           = "Xác nhận giao dịch đã thanh toán thành công"
	successAdminResolveAsCancelled      = "Xác nhận hủy giao dịch thành công"
	successAdminResolveAsCreationFailed = "Đánh dấu tạo link thất bại thành công"
)

type AdminBillingHandler struct {
	logger       logger.Interface
	adminBilling uc_interfaces.IAdminBillingUseCase
	resolution   uc_interfaces.IPaymentResolutionUseCase
}

func NewAdminBillingHandler(
	logger logger.Interface,
	adminBilling uc_interfaces.IAdminBillingUseCase,
	resolution uc_interfaces.IPaymentResolutionUseCase,
) *AdminBillingHandler {
	return &AdminBillingHandler{
		logger:       logger,
		adminBilling: adminBilling,
		resolution:   resolution,
	}
}

type adminListTransactionsReq struct {
	Status   string `form:"status"`
	FromDate string `form:"fromDate"`
	ToDate   string `form:"toDate"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
}

// ListTransactions
// @Summary Lấy danh sách giao dịch thanh toán
// @Description Trả về danh sách giao dịch nạp tiền/đăng ký (lọc theo trạng thái, thời gian).
// @Tags Admin Billing
// @Accept json
// @Produce json
// @Param status query string false "Trạng thái giao dịch"
// @Param fromDate query string false "Ngày bắt đầu (YYYY-MM-DD)"
// @Param toDate query string false "Ngày kết thúc (YYYY-MM-DD)"
// @Param page query int false "Trang"
// @Param limit query int false "Số bản ghi/trang"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/transactions [get]
func (h *AdminBillingHandler) ListTransactions(c *gin.Context) error {
	var req adminListTransactionsReq
	if err := validation.BindQuery(c, &req); err != nil {
		return err
	}
	filter := uc_interfaces.AdminBillingFilter{}
	if req.Status != "" {
		if st, ok := parseDepositStatus(req.Status); ok {
			filter.Status = &st
		}
	}
	if req.FromDate != "" {
		if t, err := time.Parse("2006-01-02", req.FromDate); err == nil {
			filter.FromDate = &t
		}
	}
	if req.ToDate != "" {
		if t, err := time.Parse("2006-01-02", req.ToDate); err == nil {
			filter.ToDate = &t
		}
	}
	items, total, err := h.adminBilling.ListTransactions(c.Request.Context(), filter, req.Page, req.Limit)
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListTransactions, gin.H{
		"items": items,
		"total": total,
	})
	return nil
}

type adminGetTransactionParams struct {
	ID string `uri:"id" binding:"required"`
}

// GetTransaction
// @Summary Lấy chi tiết giao dịch thanh toán
// @Description Trả về chi tiết một giao dịch thanh toán theo ID.
// @Tags Admin Billing
// @Produce json
// @Param id path string true "ID giao dịch"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/transactions/{id} [get]
func (h *AdminBillingHandler) GetTransaction(c *gin.Context) error {
	var params adminGetTransactionParams
	if err := c.ShouldBindUri(&params); err != nil {
		return err
	}
	id, err := uuid.Parse(params.ID)
	if err != nil {
		return err
	}
	item, err := h.adminBilling.GetTransaction(c.Request.Context(), id)
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminGetTransaction, item)
	return nil
}

type resolveRequest struct {
	Reason        string `json:"reason"`
	Reference     string `json:"reference"`
	PaymentLinkID string `json:"paymentLinkId"`
}

// buildResolutionInput resolves the orderCode + current status of the
// transaction so the resolution use case can validate the expected state.
func (h *AdminBillingHandler) buildResolutionInput(c *gin.Context, req resolveRequest) (uc_interfaces.ManualPaymentResolutionInput, error) {
	adminID, err := contextutils.GetUserId(c)
	if err != nil {
		return uc_interfaces.ManualPaymentResolutionInput{}, err
	}
	orderCode, err := strconv.ParseInt(c.Param("orderCode"), 10, 64)
	if err != nil {
		return uc_interfaces.ManualPaymentResolutionInput{}, err
	}
	tx, err := h.adminBilling.GetTransactionByOrderCode(c.Request.Context(), orderCode)
	if err != nil {
		return uc_interfaces.ManualPaymentResolutionInput{}, err
	}
	if tx == nil {
		return uc_interfaces.ManualPaymentResolutionInput{}, nil
	}
	return uc_interfaces.ManualPaymentResolutionInput{
		OrderCode:      orderCode,
		ExpectedStatus: tx.Status,
		ActorAdminID:   adminID.String(),
		Reason:         req.Reason,
		Evidence: &payment.PaymentLinkInfo{
			OrderCode:     orderCode,
			PaymentLinkID: req.PaymentLinkID,
			Reference:     req.Reference,
			Amount:        tx.Amount,
			AmountPaid:    tx.Amount,
			Currency:      "VND",
		},
	}, nil
}

// RetryInvestigation
// @Summary Gửi lại yêu cầu đối soát giao dịch
// @Description Chuyển giao dịch về trạng thái cần đối soát lại.
// @Tags Admin Billing
// @Accept json
// @Produce json
// @Param orderCode path int true "Mã đơn hàng"
// @Param body body resolveRequest false "Lý do"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/transactions/{orderCode}/retry-investigation [post]
func (h *AdminBillingHandler) RetryInvestigation(c *gin.Context) error {
	var req resolveRequest
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	input, err := h.buildResolutionInput(c, req)
	if err != nil {
		return err
	}
	if err := h.resolution.RetryInvestigation(c.Request.Context(), input); err != nil {
		return err
	}
	shared_pres.Success(c, successAdminRetryInvestigation, nil)
	return nil
}

// ResolveAsPaid
// @Summary Xác nhận giao dịch đã thanh toán
// @Description Xác nhận đã thanh toán. Bắt buộc có bằng chứng (reference + paymentLinkId).
// @Tags Admin Billing
// @Accept json
// @Produce json
// @Param orderCode path int true "Mã đơn hàng"
// @Param body body resolveRequest true "Bằng chứng + lý do"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-paid [post]
func (h *AdminBillingHandler) ResolveAsPaid(c *gin.Context) error {
	var req resolveRequest
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	input, err := h.buildResolutionInput(c, req)
	if err != nil {
		return err
	}
	if err := h.resolution.ResolveAsPaid(c.Request.Context(), input); err != nil {
		return err
	}
	shared_pres.Success(c, successAdminResolveAsPaid, nil)
	return nil
}

// ResolveAsCancelled
// @Summary Xác nhận hủy giao dịch
// @Description Xác nhận giao dịch đã bị hủy.
// @Tags Admin Billing
// @Accept json
// @Produce json
// @Param orderCode path int true "Mã đơn hàng"
// @Param body body resolveRequest false "Lý do"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-cancelled [post]
func (h *AdminBillingHandler) ResolveAsCancelled(c *gin.Context) error {
	var req resolveRequest
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	input, err := h.buildResolutionInput(c, req)
	if err != nil {
		return err
	}
	if err := h.resolution.ResolveAsCancelled(c.Request.Context(), input); err != nil {
		return err
	}
	shared_pres.Success(c, successAdminResolveAsCancelled, nil)
	return nil
}

// ResolveAsCreationFailed
// @Summary Đánh dấu tạo link thất bại
// @Description Đánh dấu giao dịch là tạo link thanh toán thất bại.
// @Tags Admin Billing
// @Accept json
// @Produce json
// @Param orderCode path int true "Mã đơn hàng"
// @Param body body resolveRequest false "Lý do"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-creation-failed [post]
func (h *AdminBillingHandler) ResolveAsCreationFailed(c *gin.Context) error {
	var req resolveRequest
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}
	input, err := h.buildResolutionInput(c, req)
	if err != nil {
		return err
	}
	if err := h.resolution.ResolveAsCreationFailed(c.Request.Context(), input); err != nil {
		return err
	}
	shared_pres.Success(c, successAdminResolveAsCreationFailed, nil)
	return nil
}

// parseDepositStatus converts a status name (lowercase) into a DepositStatus value.
func parseDepositStatus(name string) (depositstatus.DepositStatus, bool) {
	switch name {
	case "pending":
		return depositstatus.Pending, true
	case "success":
		return depositstatus.Success, true
	case "creating":
		return depositstatus.Creating, true
	case "reconciliation_required":
		return depositstatus.ReconciliationRequired, true
	case "reconciling":
		return depositstatus.Reconciling, true
	case "creation_failed":
		return depositstatus.CreationFailed, true
	case "cancelled":
		return depositstatus.Cancelled, true
	case "expired":
		return depositstatus.Expired, true
	case "investigation_required":
		return depositstatus.InvestigationRequired, true
	case "refunded":
		return depositstatus.Refunded, true
	default:
		return 0, false
	}
}
