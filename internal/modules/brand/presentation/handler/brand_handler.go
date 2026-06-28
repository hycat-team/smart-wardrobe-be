package handler

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgBrandCreateRequestSuccess   = "Gửi yêu cầu tạo brand thành công"
	msgBrandCreateAdminSuccess     = "Tạo brand thành công"
	msgBrandUpdateStatusSuccess    = "Cập nhật trạng thái brand thành công"
	msgBrandListSuccess            = "Lấy danh sách brand thành công"
	msgBrandDetailSuccess          = "Lấy thông tin brand thành công"
	msgBrandMemberCreateSuccess    = "Thêm thành viên brand thành công"
	msgBrandMemberListSuccess      = "Lấy danh sách thành viên brand thành công"
	msgBrandCustomerListSuccess    = "Lấy danh sách khách hàng brand thành công"
	msgBrandJoinLoyaltySuccess     = "Tham gia loyalty brand thành công"
	msgBrandOfflineCustomerSuccess = "Tạo khách hàng offline thành công"
	msgBrandGrantPointsSuccess     = "Ghi nhận giao dịch điểm loyalty thành công"
)

type BrandHandler struct {
	brandUC usecase_interfaces.IBrandCoreUseCase
}

func NewBrandHandler(brandUC usecase_interfaces.IBrandCoreUseCase) *BrandHandler {
	return &BrandHandler{brandUC: brandUC}
}

// CreateBrandRequest creates a pending brand request.
// @Summary Gửi yêu cầu tạo brand
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param body body dto.CreateBrandReq true "Thông tin brand"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brand-portal/brands [post]
func (h *BrandHandler) CreateBrandRequest(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.CreateBrandReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.CreateBrandRequest(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandCreateRequestSuccess, res)
	return nil
}

// CreateBrandAdmin creates an active brand directly.
// @Summary Tạo brand active trực tiếp (Admin)
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body dto.CreateBrandReq true "Thông tin brand"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/admin/brands [post]
func (h *BrandHandler) CreateBrandAdmin(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.CreateBrandReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.CreateBrandByAdmin(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandCreateAdminSuccess, res)
	return nil
}

// UpdateBrandStatusAdmin updates brand status.
// @Summary Cập nhật trạng thái brand (Admin)
// @Tags Admin
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.UpdateBrandStatusReq true "Trạng thái mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/admin/brands/{brandId}/status [patch]
func (h *BrandHandler) UpdateBrandStatusAdmin(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.UpdateBrandStatusReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.UpdateBrandStatus(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandUpdateStatusSuccess, res)
	return nil
}

// GetActiveBrands lists public active brands.
// @Summary Lấy danh sách brand đang active
// @Tags Brand
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandRes}
// @Router /api/v1/brands [get]
func (h *BrandHandler) GetActiveBrands(c *gin.Context) error {
	res, err := h.brandUC.GetActiveBrands(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandListSuccess, res)
	return nil
}

// GetBrandForPortal gets brand detail for active brand members.
// @Summary Lấy thông tin brand portal
// @Tags Brand Portal
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brand-portal/brands/{brandId} [get]
func (h *BrandHandler) GetBrandForPortal(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandForPortal(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandDetailSuccess, res)
	return nil
}

// AddBrandMember adds a member to a brand.
// @Summary Thêm thành viên vào brand
// @Description Cho phép chủ sở hữu brand thêm thành viên mới vào thương hiệu của mình
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.AddBrandMemberReq true "Thông tin thành viên mới"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandMemberRes}
// @Router /api/v1/brand-portal/brands/{brandId}/members [post]
func (h *BrandHandler) AddBrandMember(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.AddBrandMemberReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.AddBrandMember(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandMemberCreateSuccess, res)
	return nil
}

// GetBrandMembers retrieves members of a brand.
// @Summary Lấy danh sách thành viên của brand
// @Description Lấy danh sách tất cả các thành viên trực thuộc brand này
// @Tags Brand Portal
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandMemberRes}
// @Router /api/v1/brand-portal/brands/{brandId}/members [get]
func (h *BrandHandler) GetBrandMembers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandMembers(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandMemberListSuccess, res)
	return nil
}

// GetBrandCustomers retrieves loyalty customers of a brand.
// @Summary Lấy danh sách khách hàng của brand
// @Description Lấy danh sách các khách hàng đã liên kết với brand
// @Tags Brand Portal
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandCustomerRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers [get]
func (h *BrandHandler) GetBrandCustomers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandCustomers(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandCustomerListSuccess, res)
	return nil
}

// JoinLoyalty registers a user as a loyalty customer of a brand.
// @Summary Tham gia chương trình khách hàng thân thiết
// @Description Đăng ký người dùng hiện tại tham gia chương trình loyalty của brand
// @Tags Brand
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandCustomerRes}
// @Router /api/v1/brands/{brandId}/join-loyalty [post]
func (h *BrandHandler) JoinLoyalty(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	role, err := contextutils.GetRoleSlug(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.JoinLoyalty(c.Request.Context(), userID, role, brandID)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandJoinLoyaltySuccess, res)
	return nil
}

// CreateOfflineCustomer creates an offline customer record under a brand.
// @Summary Tạo khách hàng offline cho brand
// @Description Cho phép nhân viên/chủ brand ghi nhận thông tin khách hàng mua hàng offline
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.CreateOfflineBrandCustomerReq true "Thông tin khách hàng mua offline"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandCustomerRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers/offline-purchase [post]
func (h *BrandHandler) CreateOfflineCustomer(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.CreateOfflineBrandCustomerReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.CreateOfflineCustomer(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandOfflineCustomerSuccess, res)
	return nil
}

// GrantLoyaltyPoints records a loyalty point transaction.
// @Summary Ghi nhận cộng/trừ điểm loyalty cho brand customer
// @Description API thống nhất để brand staff ghi nhận điểm bằng userId, phone hoặc externalCustomerCode
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.GrantLoyaltyPointsReq true "Thông tin giao dịch điểm"
// @Success 201 {object} shared_pres.APIResponse{data=dto.LoyaltyPointsTransactionRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/points [post]
func (h *BrandHandler) GrantLoyaltyPoints(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.GrantLoyaltyPointsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.GrantLoyaltyPoints(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandGrantPointsSuccess, res)
	return nil
}

// CreateBrandBenefit creates a new benefit for a brand.
// @Summary Tạo quyền lợi cho brand (Staff)
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.CreateBrandBenefitReq true "Thông tin quyền lợi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits [post]
func (h *BrandHandler) CreateBrandBenefit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.CreateBrandBenefitReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.CreateBrandBenefit(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, "Tạo quyền lợi brand thành công", res)
	return nil
}

// ListBrandBenefitsForStaff lists all benefits for staff.
// @Summary Lấy danh sách quyền lợi cho brand staff
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits [get]
func (h *BrandHandler) ListBrandBenefitsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.ListBrandBenefitsForStaff(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách quyền lợi brand thành công", res)
	return nil
}

// UpdateBenefitStatus updates the status of a benefit.
// @Summary Cập nhật trạng thái quyền lợi (Staff)
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param benefitId path string true "ID quyền lợi"
// @Param body body dto.UpdateBenefitStatusReq true "Trạng thái mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits/{benefitId}/status [patch]
func (h *BrandHandler) UpdateBenefitStatus(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	benefitID, err := uuid.Parse(c.Param("benefitId"))
	if err != nil {
		return err
	}
	var input dto.UpdateBenefitStatusReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.UpdateBenefitStatus(c.Request.Context(), userID, brandID, benefitID, input.Status)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Cập nhật trạng thái quyền lợi thành công", res)
	return nil
}

// ListActiveBenefitsForUser lists active benefits for a user.
// @Summary Lấy danh sách quyền lợi đang hoạt động của brand (User)
// @Tags Brand
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandBenefitRes}
// @Router /api/v1/brands/{brandId}/benefits [get]
func (h *BrandHandler) ListActiveBenefitsForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.ListActiveBenefitsForUser(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách quyền lợi brand thành công", res)
	return nil
}

// RedeemBenefit allows a user to redeem a benefit.
// @Summary Đổi quyền lợi của brand (User)
// @Tags Brand
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param benefitId path string true "ID quyền lợi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BenefitRedemptionRes}
// @Router /api/v1/brands/{brandId}/benefits/{benefitId}/redeem [post]
func (h *BrandHandler) RedeemBenefit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	benefitID, err := uuid.Parse(c.Param("benefitId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.RedeemBenefit(c.Request.Context(), userID, brandID, benefitID)
	if err != nil {
		return err
	}
	shared_pres.Created(c, "Đổi quyền lợi thành công", res)
	return nil
}

// GetUserConversation returns the user's conversation with a brand.
// @Summary Lấy cuộc hội thoại hiện tại (User)
// @Tags Brand
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandConversationRes}
// @Router /api/v1/brands/{brandId}/conversation [get]
func (h *BrandHandler) GetUserConversation(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetUserConversation(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thông tin hội thoại thành công", res)
	return nil
}

// SendUserMessage sends a message from a user to a brand.
// @Summary Gửi tin nhắn đến brand (User)
// @Tags Brand
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param brandId path string true "ID brand"
// @Param body body dto.SendBrandChatMessageReq true "Nội dung tin nhắn"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandConversationMessageRes}
// @Router /api/v1/brands/{brandId}/conversation/messages [post]
func (h *BrandHandler) SendUserMessage(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.SendBrandChatMessageReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.SendUserMessage(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, "Gửi tin nhắn thành công", res)
	return nil
}

// ListBrandConversations lists conversations for a brand portal.
// @Summary Lấy danh sách các cuộc hội thoại của brand (Staff)
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandConversationRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations [get]
func (h *BrandHandler) ListBrandConversations(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.ListBrandConversations(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách hội thoại thành công", res)
	return nil
}

// ListConversationMessages lists messages in a conversation.
// @Summary Lấy danh sách tin nhắn trong cuộc hội thoại (Staff)
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID cuộc hội thoại"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandConversationMessageRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages [get]
func (h *BrandHandler) ListConversationMessages(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.ListConversationMessages(c.Request.Context(), userID, brandID, conversationID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách tin nhắn thành công", res)
	return nil
}

// SendStaffMessage sends a message from a staff member to a conversation.
// @Summary Gửi phản hồi của brand staff (Staff)
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID cuộc hội thoại"
// @Param body body dto.SendBrandChatMessageReq true "Nội dung phản hồi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandConversationMessageRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages [post]
func (h *BrandHandler) SendStaffMessage(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		return err
	}
	var input dto.SendBrandChatMessageReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.SendStaffMessage(c.Request.Context(), userID, brandID, conversationID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, "Gửi phản hồi thành công", res)
	return nil
}
