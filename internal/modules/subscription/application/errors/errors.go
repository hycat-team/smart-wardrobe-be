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
func ErrPayosMustBeInteger() *apperror.Error {
	return apperror.NewBadRequest("Số tiền thanh toán qua cổng PayOS phải là số tiền nguyên theo đơn vị VND.")
}

func ErrInvalidAmount() *apperror.Error {
	return apperror.NewBadRequest(sharedmoney.ErrInvalidAmount)
}

func ErrVerifySignatureFailed() *apperror.Error {
	return apperror.NewBadRequest("Không thể xác thực chữ ký giao dịch.")
}

func ErrWebhookPayloadMalformed() *apperror.Error {
	return apperror.NewBadRequest("Dữ liệu thông báo giao dịch không đúng định dạng.")
}

func ErrPaymentAmountMismatch() *apperror.Error {
	return apperror.NewBadRequest("Số tiền thanh toán không khớp với số tiền của giao dịch.")
}

func ErrProcessPaymentGatewayFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể xử lý thông tin cổng thanh toán.")
}

func ErrLockTransactionFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể khóa dữ liệu giao dịch.")
}

func ErrCompletePaymentRecordFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể hoàn tất hồ sơ giao dịch thanh toán.")
}

func ErrQueryTransactionFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể truy vấn thông tin giao dịch thanh toán.")
}

func ErrWalletNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thông tin ví của tài khoản.")
}

func ErrInvalidDepositAmount() *apperror.Error {
	return apperror.NewBadRequest("Số tiền nạp không hợp lệ.")
}

func ErrDepositInitFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể khởi tạo giao dịch nạp tiền.")
}

func ErrDepositMustBeInteger() *apperror.Error {
	return apperror.NewBadRequest("Số tiền nạp vào ví phải là số tiền nguyên theo đơn vị VND.")
}

func ErrPaymentLinkUpdateFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể cập nhật thông tin liên kết thanh toán.")
}

func ErrQueryWalletBalanceFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể truy vấn thông tin số dư ví.")
}

func ErrWalletInsufficientBalance() *apperror.Error {
	return apperror.NewBadRequest("Số dư ví không đủ để thực hiện giao dịch.")
}

func ErrWalletCreateFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể khởi tạo ví mới.")
}

func ErrWalletBalanceUpdateFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể cập nhật số dư ví.")
}

func ErrWalletStatementSaveFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể lưu lịch sử giao dịch ví.")
}

func ErrUserSubscriptionNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói hội viên của tài khoản.")
}

func ErrSubscriptionExpiredAutoRenew() *apperror.Error {
	return apperror.NewBadRequest("Gói hội viên của bạn đã hết hạn, không thể thiết lập tự động gia hạn.")
}

func ErrSubscriptionPlanNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thông tin gói hội viên.")
}

func ErrDefaultPlanConfigNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cấu hình cho gói hội viên mặc định.")
}

func ErrDefaultPlanLoadFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tải thông tin cấu hình gói hội viên mặc định.")
}

func ErrQueryExpiredSubscriptionsFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể truy vấn danh sách gói hội viên hết hạn.")
}

func ErrRequestedPlanNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói hội viên được yêu cầu.")
}

func ErrFreePlanDirectPurchase() *apperror.Error {
	return apperror.NewBadRequest("Không thể đăng ký trực tiếp gói hội viên miễn phí")
}

func ErrDirectPurchaseCreateFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tạo giao dịch thanh toán trực tiếp.")
}

func ErrPaymentLinkCreateFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể liên kết địa chỉ thanh toán.")
}

func ErrSearchPlanFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tìm kiếm thông tin gói cước.")
}

func ErrCurrentSubscriptionLoadFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tải thông tin gói hội viên hiện tại.")
}

func ErrCurrentSubscriptionNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói hội viên hiện tại.")
}

func ErrActivateNewSubscriptionFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể kích hoạt gói hội viên mới.")
}

func ErrUpdateSubscriptionExpiryFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể cập nhật thời hạn gói hội viên.")
}

func ErrPlanInactive() *apperror.Error {
	return apperror.NewBadRequest("Gói hội viên này hiện đang tạm dừng hoạt động.")
}

func ErrPendingPaymentExists() *apperror.Error {
	return apperror.NewConflict("Bạn đang có giao dịch thanh toán chờ xử lý. Vui lòng hoàn tất giao dịch cũ hoặc đợi cho đến khi hết hạn.")
}

func ErrAlreadyRegisteredUnlimitedPlan() *apperror.Error {
	return apperror.NewConflict("Bạn đã đăng ký gói hội viên không giới hạn thời gian này.")
}

func ErrSubscriptionStillActive() *apperror.Error {
	return apperror.NewBadRequest("Gói hội viên hiện tại của bạn vẫn còn hiệu lực. Vui lòng tắt chế độ tự động gia hạn nếu không có nhu cầu gia hạn tiếp.")
}

func ErrLinkedPlanNotFound() *apperror.Error {
	return apperror.NewBadRequest("Không tìm thấy thông tin gói hội viên liên kết với giao dịch.")
}

func ErrLoadPlanFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tải thông tin gói hội viên.")
}

func ErrInvalidManualPaymentResolution() *apperror.Error {
	return apperror.NewBadRequest("Không thể áp dụng quyết định xử lý thủ công vì trạng thái hoặc bằng chứng thanh toán không hợp lệ.")
}

func ErrAiOutfitQuotaExceeded() *apperror.Error {
	return apperror.NewBadRequest("Bạn đã dùng hết lượt tạo trang phục bằng AI trong hôm nay.")
}

func ErrAiChatQuotaExceeded() *apperror.Error {
	return apperror.NewBadRequest("Bạn đã dùng hết lượt trò chuyện với AI Chatbot trong hôm nay.")
}
