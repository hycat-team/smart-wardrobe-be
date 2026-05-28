package handler

import (
	"encoding/json"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
)

type BillingHandler struct {
	billingUseCase usecase_interfaces.IBillingUseCase
}

func NewBillingHandler(billingUseCase usecase_interfaces.IBillingUseCase) *BillingHandler {
	return &BillingHandler{
		billingUseCase: billingUseCase,
	}
}

// GetWallet retrieves user internal wallet balance
// @Summary Lấy số dư ví người dùng
// @Description Lấy số dư hiện có trong ví nội bộ của người dùng
// @Tags Billing
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Số dư ví"
// @Router /api/v1/subscriptions/me/wallet [get]
func (h *BillingHandler) GetWallet(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	walletDTO, err := h.billingUseCase.GetWallet(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy thông tin ví thành công", walletDTO)
	return nil
}

// GetWalletStatements retrieves historical wallet statements
// @Summary Lấy lịch sử giao dịch ví nội bộ
// @Description Lấy nhật ký biến động số dư ví nội bộ của người dùng
// @Tags Billing
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Danh sách lịch sử giao dịch"
// @Router /api/v1/subscriptions/me/wallet/statements [get]
func (h *BillingHandler) GetWalletStatements(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	statements, err := h.billingUseCase.GetWalletStatements(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy lịch sử ví thành công", statements)
	return nil
}

// CreateWalletTopUp initiates a wallet topup link via PayOS
// @Summary Tạo yêu cầu nạp tiền vào ví nội bộ
// @Description Khởi tạo link thanh toán VietQR qua cổng PayOS để nạp tiền vào ví
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.WalletTopUpReq true "Thông tin nạp tiền"
// @Success 200 {object} shared_pres.APIResponse "Link thanh toán"
// @Router /api/v1/subscriptions/me/wallet/topup [post]
func (h *BillingHandler) CreateWalletTopUp(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req dto.WalletTopUpReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	paymentLink, err := h.billingUseCase.CreateWalletTopUp(c.Request.Context(), userID, &req)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Khởi tạo giao dịch nạp tiền thành công", paymentLink)
	return nil
}

// CreateDirectPurchase initiates a subscription plan purchase link via PayOS
// @Summary Đăng ký mua gói cước trực tiếp
// @Description Khởi tạo link thanh toán VietQR qua cổng PayOS để đăng ký gói cước trực tiếp
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.DirectPurchaseReq true "Thông tin gói cước"
// @Success 200 {object} shared_pres.APIResponse "Link thanh toán"
// @Router /api/v1/subscriptions/me/purchase [post]
// @Router /api/v1/subscriptions/me/purchase [post]
func (h *BillingHandler) CreateDirectPurchase(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req dto.DirectPurchaseReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	paymentLink, err := h.billingUseCase.CreateDirectPurchase(c.Request.Context(), userID, &req)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Khởi tạo giao dịch đăng ký thành công", paymentLink)
	return nil
}

// ProcessPayOSWebhook processes incoming webhook payloads from PayOS
// @Summary Xử lý Webhook thông báo thanh toán từ PayOS
// @Description Tiếp nhận và xác thực thông báo IPN từ PayOS khi người dùng thanh toán thành công
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.PayOSWebhookReq true "Webhook Payload"
// @Success 200 {object} shared_pres.APIResponse "Kết quả xử lý"
// @Router /api/v1/subscriptions/payos-webhook [post]
func (h *BillingHandler) ProcessPayOSWebhook(c *gin.Context) error {
	var req dto.PayOSWebhookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return err
	}

	rawBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	err = h.billingUseCase.ProcessWebhook(c.Request.Context(), rawBytes, req.Signature)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Xử lý thông báo thanh toán thành công", nil)
	return nil
}

// GetPlans retrieves all subscription plans
// @Summary Lấy danh sách các gói Premium
// @Description Lấy danh sách tất cả các gói đăng ký Premium hiện có
// @Tags Billing
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Danh sách gói cước"
// @Router /api/v1/subscriptions/plans [get]
func (h *BillingHandler) GetPlans(c *gin.Context) error {
	plans, err := h.billingUseCase.GetPlans(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy danh sách gói Premium thành công", plans)
	return nil
}
