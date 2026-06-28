package handler

const (
	msgBrandCreateRequestSuccess    = "Gửi yêu cầu tạo brand thành công"
	msgBrandCreateAdminSuccess      = "Tạo brand thành công"
	msgBrandUpdateStatusSuccess     = "Cập nhật trạng thái brand thành công"
	msgBrandListSuccess             = "Lấy danh sách brand thành công"
	msgBrandDetailSuccess           = "Lấy thông tin brand thành công"
	msgBrandMemberCreateSuccess     = "Thêm thành viên brand thành công"
	msgBrandMemberListSuccess       = "Lấy danh sách thành viên brand thành công"
	msgBrandCustomerListSuccess     = "Lấy danh sách khách hàng brand thành công"
	msgBrandJoinLoyaltySuccess      = "Tham gia loyalty brand thành công"
	msgBrandOfflineCustomerSuccess  = "Tạo khách hàng offline thành công"
	msgBrandGrantPointsSuccess      = "Ghi nhận giao dịch điểm loyalty thành công"
	msgBrandItemCreateSuccess       = "Tạo sản phẩm brand thành công"
	msgBrandItemListSuccess         = "Lấy danh sách sản phẩm brand thành công"
	msgBrandItemUpdateSuccess       = "Cập nhật sản phẩm brand thành công"
	msgBrandItemFeedbackSuccess     = "Lấy danh sách feedback sản phẩm thành công"
	msgBrandFeedbackSubmitSuccess   = "Gửi feedback sản phẩm thành công"
	msgBrandCreateClaimTokenSuccess = "Tạo mã claim liên kết tài khoản thành công"
	msgBrandClaimCustomerSuccess    = "Liên kết tài khoản loyalty thành công"
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
