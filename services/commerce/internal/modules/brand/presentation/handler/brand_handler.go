package handler

const (
	msgBrandCreateRequestSuccess    = "Gửi yêu cầu tạo brand thành công"
	msgBrandCreateAdminSuccess      = "Tạo brand thành công"
	msgBrandUpdateStatusSuccess     = "Cập nhật trạng thái brand thành công"
	msgBrandListSuccess             = "Lấy danh sách brand thành công"
	msgBrandDetailSuccess           = "Lấy thông tin brand thành công"
	msgBrandMemberCreateSuccess     = "Thêm thành viên brand thành công"
	msgBrandMemberListSuccess       = "Lấy danh sách thành viên brand thành công"
	msgBrandCustomerDetailSuccess   = "Lấy thông tin khách hàng brand thành công"
	msgBrandCustomerListSuccess     = "Lấy danh sách khách hàng brand thành công"
	msgBrandJoinLoyaltySuccess      = "Tham gia loyalty brand thành công"
	msgBrandOfflineCustomerSuccess  = "Tạo khách hàng offline thành công"
	msgBrandGrantPointsSuccess      = "Ghi nhận giao dịch điểm loyalty thành công"
	msgBrandItemCreateSuccess       = "Tạo sản phẩm brand thành công"
	msgBrandItemGetSuccess          = "Lấy thông tin sản phẩm brand thành công"
	msgBrandItemListSuccess         = "Lấy danh sách sản phẩm brand thành công"
	msgBrandItemUpdateSuccess       = "Cập nhật sản phẩm brand thành công"
	msgBrandItemFeedbackSuccess     = "Lấy danh sách feedback sản phẩm thành công"
	msgBrandFeedbackSubmitSuccess   = "Gửi feedback sản phẩm thành công"
	msgBrandCreateClaimTokenSuccess = "Tạo mã claim liên kết tài khoản thành công"
	msgBrandClaimCustomerSuccess    = "Liên kết tài khoản loyalty thành công"

	// Upload & Brand Portal
	msgBrandGetUploadLogoSignatureSuccess = "Lấy chữ ký upload logo brand thành công"
	msgBrandUpdateLogoSuccess             = "Cập nhật logo brand thành công"
	msgBrandGetUploadItemSignatureSuccess = "Lấy chữ ký upload ảnh sản phẩm brand thành công"

	// Brand Chat
	msgBrandChatGetUserConversationSuccess      = "Lấy thông tin hội thoại thành công"
	msgBrandChatSendUserMessageSuccess          = "Gửi tin nhắn thành công"
	msgBrandChatListBrandConversationsSuccess   = "Lấy danh sách hội thoại thành công"
	msgBrandChatListConversationMessagesSuccess = "Lấy danh sách tin nhắn thành công"
	msgBrandChatSendStaffMessageSuccess         = "Gửi phản hồi thành công"
	msgBrandChatMarkConversationReadSuccess     = "Đánh dấu đã đọc hội thoại thành công"
	msgBrandChatCloseConversationSuccess        = "Đóng hội thoại thành công"
	msgBrandChatReopenConversationSuccess       = "Mở lại hội thoại thành công"

	// Brand Loyalty 추가
	msgBrandLoyaltyListSuccess                 = "Lấy danh sách loyalty brand thành công"
	msgBrandLoyaltyDetailSuccess               = "Lấy chi tiết loyalty brand thành công"
	msgBrandLoyaltyGetPointsHistorySuccess     = "Lấy lịch sử điểm loyalty thành công"
	msgBrandLoyaltyGetAccountHistorySuccess    = "Lấy lịch sử điểm loyalty account thành công"
	msgBrandLoyaltyGetProgramSuccess           = "Lấy chương trình loyalty thành công"
	msgBrandLoyaltyUpdateProgramSuccess        = "Cập nhật chương trình loyalty thành công"
	msgBrandLoyaltyGetTiersSuccess             = "Lấy danh sách hạng loyalty thành công"
	msgBrandLoyaltyCreateTierSuccess           = "Tạo hạng thành viên thành công"
	msgBrandLoyaltyUpdateTierSuccess           = "Cập nhật hạng thành viên thành công"
	msgBrandLoyaltyCreateBenefitSuccess        = "Tạo quyền lợi brand thành công"
	msgBrandLoyaltyListBenefitsSuccess         = "Lấy danh sách quyền lợi brand thành công"
	msgBrandLoyaltyUpdateBenefitStatusSuccess  = "Cập nhật trạng thái quyền lợi thành công"
	msgBrandLoyaltyRedeemBenefitSuccess        = "Đổi quyền lợi thành công"
	msgBrandLoyaltyGetBenefitDetailSuccess     = "Lấy chi tiết quyền lợi brand thành công"
	msgBrandLoyaltyListRedeemedBenefitsSuccess = "Lấy danh sách quyền lợi đã nhận thành công"
	msgBrandLoyaltyListClaimTokensSuccess      = "Lấy danh sách mã claim thành công"
	msgBrandLoyaltyRevokeClaimTokenSuccess     = "Thu hồi mã claim thành công"
	msgBrandLoyaltyListPointBatchesSuccess     = "Lấy danh sách lô điểm loyalty thành công"
	msgBrandLoyaltyListAccountBatchesSuccess   = "Lấy danh sách lô điểm loyalty account thành công"
)

type BrandHandler struct {
	*BrandPortalHandler
	*BrandLoyaltyHandler
	*BrandChatHandler
}

func NewBrandHandler(
	brandPortal *BrandPortalHandler,
	brandLoyalty *BrandLoyaltyHandler,
	brandChat *BrandChatHandler,
) *BrandHandler {
	return &BrandHandler{
		BrandPortalHandler:  brandPortal,
		BrandLoyaltyHandler: brandLoyalty,
		BrandChatHandler:    brandChat,
	}
}
