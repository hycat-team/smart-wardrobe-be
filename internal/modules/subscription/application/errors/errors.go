package subscriptionerrors

import (
	"fmt"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
)

// Lỗi động (cần định dạng tham số)
func ErrPayosMinAmount(minAmount int64) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Số tiền thanh toán tối thiểu qua cổng PayOS là %d VNĐ.", minAmount))
}

func ErrTransactionNotFound(orderCode int64) *apperror.Error {
	return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy giao dịch thanh toán với mã đơn hàng %d.", orderCode))
}

// Lỗi tĩnh
var (
	// PayOS & Gateway Errors
	ErrPayosMustBeInteger         = apperror.NewBadRequest("Số tiền thanh toán qua cổng PayOS phải là số tiền nguyên theo đơn vị VND.")
	ErrInvalidAmount              = apperror.NewBadRequest(sharedmoney.ErrInvalidAmount)
	ErrVerifySignatureFailed      = apperror.NewBadRequest("Không thể xác thực chữ ký giao dịch.")
	ErrWebhookPayloadMalformed    = apperror.NewBadRequest("Dữ liệu thông báo giao dịch không đúng định dạng.")
	ErrPaymentAmountMismatch      = apperror.NewBadRequest("Số tiền thanh toán không khớp với số tiền của giao dịch.")
	ErrProcessPaymentGatewayFailed = apperror.NewInternalError("Không thể xử lý thông tin cổng thanh toán.")
	ErrLockTransactionFailed      = apperror.NewInternalError("Không thể khóa dữ liệu giao dịch.")
	ErrCompletePaymentRecordFailed = apperror.NewInternalError("Không thể hoàn tất hồ sơ giao dịch thanh toán.")
	ErrQueryTransactionFailed      = apperror.NewInternalError("Không thể truy vấn thông tin giao dịch thanh toán.")

	// Wallet Errors
	ErrWalletNotFound             = apperror.NewNotFound("Không tìm thấy thông tin ví của tài khoản.")
	ErrInvalidDepositAmount       = apperror.NewBadRequest("Số tiền nạp không hợp lệ.")
	ErrDepositInitFailed          = apperror.NewInternalError("Không thể khởi tạo giao dịch nạp tiền.")
	ErrDepositMustBeInteger       = apperror.NewBadRequest("Số tiền nạp vào ví phải là số tiền nguyên theo đơn vị VND.")
	ErrPaymentLinkUpdateFailed    = apperror.NewInternalError("Không thể cập nhật thông tin liên kết thanh toán.")
	ErrQueryWalletBalanceFailed   = apperror.NewInternalError("Không thể truy vấn thông tin số dư ví.")
	ErrWalletInsufficientBalance  = apperror.NewBadRequest("Số dư ví không đủ để thực hiện giao dịch.")
	ErrWalletCreateFailed         = apperror.NewInternalError("Không thể khởi tạo ví mới.")
	ErrWalletBalanceUpdateFailed  = apperror.NewInternalError("Không thể cập nhật số dư ví.")
	ErrWalletStatementSaveFailed  = apperror.NewInternalError("Không thể lưu lịch sử giao dịch ví.")

	// Subscription & Plan Errors
	ErrUserSubscriptionNotFound       = apperror.NewNotFound("Không tìm thấy gói hội viên của tài khoản.")
	ErrSubscriptionExpiredAutoRenew   = apperror.NewBadRequest("Gói hội viên của bạn đã hết hạn, không thể thiết lập tự động gia hạn.")
	ErrSubscriptionPlanNotFound       = apperror.NewNotFound("Không tìm thấy thông tin gói hội viên.")
	ErrDefaultPlanConfigNotFound      = apperror.NewNotFound("Không tìm thấy cấu hình cho gói hội viên mặc định.")
	ErrDefaultPlanLoadFailed          = apperror.NewInternalError("Không thể tải thông tin cấu hình gói hội viên mặc định.")
	ErrQueryExpiredSubscriptionsFailed = apperror.NewInternalError("Không thể truy vấn danh sách gói hội viên hết hạn.")
	ErrRequestedPlanNotFound          = apperror.NewNotFound("Không tìm thấy gói hội viên được yêu cầu.")
	ErrFreePlanDirectPurchase         = apperror.NewBadRequest("Không thể đăng ký trực tiếp gói hội viên miễn phí")
	ErrDirectPurchaseCreateFailed     = apperror.NewInternalError("Không thể tạo giao dịch thanh toán trực tiếp.")
	ErrPaymentLinkCreateFailed        = apperror.NewInternalError("Không thể liên kết địa chỉ thanh toán.")
	ErrSearchPlanFailed               = apperror.NewInternalError("Không thể tìm kiếm thông tin gói cước.")
	ErrCurrentSubscriptionLoadFailed  = apperror.NewInternalError("Không thể tải thông tin gói hội viên hiện tại.")
	ErrCurrentSubscriptionNotFound    = apperror.NewNotFound("Không tìm thấy gói hội viên hiện tại.")
	ErrActivateNewSubscriptionFailed  = apperror.NewInternalError("Không thể kích hoạt gói hội viên mới.")
	ErrUpdateSubscriptionExpiryFailed = apperror.NewInternalError("Không thể cập nhật thời hạn gói hội viên.")
	ErrPlanInactive                   = apperror.NewBadRequest("Gói hội viên này hiện đang tạm dừng hoạt động.")
	ErrPendingPaymentExists           = apperror.NewConflict("Bạn đang có giao dịch thanh toán chờ xử lý. Vui lòng hoàn tất giao dịch cũ hoặc đợi cho đến khi hết hạn.")
	ErrAlreadyRegisteredUnlimitedPlan = apperror.NewConflict("Bạn đã đăng ký gói hội viên không giới hạn thời gian này.")
	ErrSubscriptionStillActive        = apperror.NewBadRequest("Gói hội viên hiện tại của bạn vẫn còn hiệu lực. Vui lòng tắt chế độ tự động gia hạn nếu không có nhu cầu gia hạn tiếp.")
	ErrLinkedPlanNotFound             = apperror.NewBadRequest("Không tìm thấy thông tin gói hội viên liên kết với giao dịch.")
	ErrLoadPlanFailed                 = apperror.NewInternalError("Không thể tải thông tin gói hội viên.")

	// AI Quota Errors
	ErrAiOutfitQuotaExceeded = apperror.NewBadRequest("Bạn đã dùng hết lượt tạo trang phục bằng AI trong hôm nay.")
	ErrAiChatQuotaExceeded   = apperror.NewBadRequest("Bạn đã dùng hết lượt trò chuyện với AI Chatbot trong hôm nay.")
)
