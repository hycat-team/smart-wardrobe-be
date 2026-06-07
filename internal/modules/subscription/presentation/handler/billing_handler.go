package handler

import (
	"encoding/json"
	"io"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	successGetWallet              = "Lấy thông tin ví thành công"
	successGetWalletStatements    = "Lấy lịch sử ví thành công"
	successCreateWalletTopUp      = "Khởi tạo giao dịch nạp tiền thành công"
	successCreateDirectPurchase   = "Khởi tạo giao dịch đăng ký thành công"
	successPurchasePlanWithWallet = "Đăng ký gói hội viên thành công"
	successProcessPayOSWebhook    = "Xử lý thông báo thanh toán thành công"
)

type BillingHandler struct {
	logger                logger.Interface
	walletUseCase         usecase_interfaces.IWalletUseCase
	subPurchaseUseCase    usecase_interfaces.ISubscriptionPurchaseUseCase
	paymentWebhookUseCase usecase_interfaces.IPaymentWebhookUseCase
}

func NewBillingHandler(
	logger logger.Interface,
	walletUseCase usecase_interfaces.IWalletUseCase,
	subPurchaseUseCase usecase_interfaces.ISubscriptionPurchaseUseCase,
	paymentWebhookUseCase usecase_interfaces.IPaymentWebhookUseCase,
) *BillingHandler {
	return &BillingHandler{
		logger:                logger,
		walletUseCase:         walletUseCase,
		subPurchaseUseCase:    subPurchaseUseCase,
		paymentWebhookUseCase: paymentWebhookUseCase,
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
// func (h *BillingHandler) GetWallet(c *gin.Context) error {
func (h *BillingHandler) GetWallet(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	walletDTO, err := h.walletUseCase.GetWallet(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetWallet, walletDTO)
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

	statements, err := h.walletUseCase.GetWalletStatements(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successGetWalletStatements, statements)
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

	paymentLink, err := h.walletUseCase.CreateWalletTopUp(c.Request.Context(), userID, &req)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successCreateWalletTopUp, paymentLink)
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
func (h *BillingHandler) CreateDirectPurchase(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req dto.DirectPurchaseReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	paymentLink, err := h.subPurchaseUseCase.CreateDirectPurchase(c.Request.Context(), userID, &req)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successCreateDirectPurchase, paymentLink)
	return nil
}

// PurchasePlanWithWallet purchases a subscription plan using internal wallet balance
// @Summary Đăng ký mua gói cước bằng ví nội bộ
// @Description Thực hiện mua gói cước bằng cách trừ số dư ví nội bộ của người dùng
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.DirectPurchaseReq false "Thông tin gói cước (chỉ cần planSlug)"
// @Success 200 {object} shared_pres.APIResponse "Kết quả đăng ký"
// @Router /api/v1/subscriptions/me/purchase-with-wallet [post]
func (h *BillingHandler) PurchasePlanWithWallet(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var req struct {
		PlanSlug string `json:"planSlug" binding:"required" label:"gói hội viên"`
	}
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	err = h.subPurchaseUseCase.PurchasePlanWithWallet(c.Request.Context(), userID, req.PlanSlug)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successPurchasePlanWithWallet, nil)
	return nil
}

// ProcessPayOSWebhook processes incoming webhook payloads from PayOS
// @Summary Xử lý Webhook thông báo thanh toán từ PayOS
// @Description Tiếp nhận và xác thực thông báo IPN từ PayOS khi người dùng thanh toán thành công
// @Tags Billing
// @Accept json
// @Produce json
// @Param body body dto.PayOSWebhookReq true "Dữ liệu Webhook"
// @Success 200 {object} shared_pres.APIResponse "Kết quả xử lý"
// @Router /api/v1/subscriptions/payos-webhook [post]
func (h *BillingHandler) ProcessPayOSWebhook(c *gin.Context) error {
	rawBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	var req struct {
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(rawBytes, &req); err != nil {
		return err
	}

	err = h.paymentWebhookUseCase.ProcessWebhook(c.Request.Context(), rawBytes, req.Signature)
	if err != nil {
		h.logger.Error("Error verifying webhook signature", zap.Error(err))
		return err
	}

	shared_pres.Success(c, successProcessPayOSWebhook, nil)
	return nil
}
