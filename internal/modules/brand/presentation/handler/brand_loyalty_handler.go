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

type BrandLoyaltyHandler struct {
	brandUC   usecase_interfaces.IBrandUseCase
	loyaltyUC usecase_interfaces.IBrandLoyaltyUseCase
	benefitUC usecase_interfaces.IBrandBenefitUseCase
	claimUC   usecase_interfaces.IBrandClaimUseCase
}

func NewBrandLoyaltyHandler(
	brandUC usecase_interfaces.IBrandUseCase,
	loyaltyUC usecase_interfaces.IBrandLoyaltyUseCase,
	benefitUC usecase_interfaces.IBrandBenefitUseCase,
	claimUC usecase_interfaces.IBrandClaimUseCase,
) *BrandLoyaltyHandler {
	return &BrandLoyaltyHandler{
		brandUC:   brandUC,
		loyaltyUC: loyaltyUC,
		benefitUC: benefitUC,
		claimUC:   claimUC,
	}
}

// AddBrandMember adds members to a brand.
// @Summary Thêm thành viên vào brand
// @Description Cho phép owner thêm nhiều thành viên với vai trò staff bằng email hoặc tên đăng nhập. API này không tạo owner mới.
// @Tags Brand Member
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.AddBrandMembersReq true "Danh sách thành viên cần thêm"
// @Success 201 {object} shared_pres.APIResponse{data=dto.AddBrandMembersRes}
// @Router /api/v1/brand-portal/brands/{brandId}/members [post]
func (h *BrandLoyaltyHandler) AddBrandMember(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.AddBrandMembersReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.AddBrandMembers(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandMemberCreateSuccess, res)
	return nil
}

// GetBrandMembers retrieves members of a brand.
// @Summary Lấy danh sách thành viên của brand
// @Description Lấy danh sách tất cả các thành viên trực thuộc brand này
// @Tags Brand Member
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandMemberRes}
// @Router /api/v1/brand-portal/brands/{brandId}/members [get]
func (h *BrandLoyaltyHandler) GetBrandMembers(c *gin.Context) error {
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
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandCustomerRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers [get]
func (h *BrandLoyaltyHandler) GetBrandCustomers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetBrandCustomers(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandCustomerListSuccess, res)
	return nil
}

// GetBrandCustomer retrieves one brand customer.
// @Summary Lấy chi tiết khách hàng brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Param customerId path string true "ID customer"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandCustomerRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers/{customerId} [get]
func (h *BrandLoyaltyHandler) GetBrandCustomer(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	customerID, err := uuid.Parse(c.Param("customerId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetBrandCustomer(c.Request.Context(), userID, brandID, customerID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandCustomerDetailSuccess, res)
	return nil
}

// JoinLoyalty registers a user as a loyalty customer of a brand.
// @Summary Tham gia chương trình khách hàng thân thiết
// @Description Đăng ký người dùng hiện tại tham gia chương trình loyalty của brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandCustomerRes}
// @Router /api/v1/brands/{brandId}/join-loyalty [post]
func (h *BrandLoyaltyHandler) JoinLoyalty(c *gin.Context) error {
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
	res, err := h.loyaltyUC.JoinLoyalty(c.Request.Context(), userID, role, brandID)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandJoinLoyaltySuccess, res)
	return nil
}

// CreateOfflineCustomer creates an offline customer record under a brand.
// @Summary Tạo khách hàng offline cho brand
// @Description Cho phép nhân viên/chủ brand ghi nhận thông tin khách hàng mua hàng offline
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.CreateOfflineBrandCustomerReq true "Thông tin khách hàng mua offline"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandCustomerRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers/offline-purchase [post]
func (h *BrandLoyaltyHandler) CreateOfflineCustomer(c *gin.Context) error {
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
	res, err := h.loyaltyUC.CreateOfflineCustomer(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandOfflineCustomerSuccess, res)
	return nil
}

// GrantLoyaltyPoints records a loyalty point transaction.
// @Summary Ghi nhận cộng/trừ điểm loyalty cho brand customer
// @Description API thống nhất để brand staff ghi nhận điểm bằng userId, phone hoặc externalCustomerCode
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.GrantLoyaltyPointsReq true "Thông tin giao dịch điểm"
// @Success 201 {object} shared_pres.APIResponse{data=dto.LoyaltyPointsTransactionRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/points [post]
func (h *BrandLoyaltyHandler) GrantLoyaltyPoints(c *gin.Context) error {
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
	res, err := h.loyaltyUC.GrantLoyaltyPoints(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandGrantPointsSuccess, res)
	return nil
}

// ListUserBrandLoyalties lists current user's brand loyalty accounts.
// @Summary Lấy danh sách loyalty brand của tôi
// @Tags Brand Loyalty
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandLoyaltyRes}
// @Router /api/v1/me/brand-loyalties [get]
func (h *BrandLoyaltyHandler) ListUserBrandLoyalties(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.ListUserBrandLoyalties(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListSuccess, res)
	return nil
}

// GetUserBrandLoyalty gets current user's loyalty detail for a brand.
// @Summary Lấy chi tiết điểm loyalty của tôi theo brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandLoyaltyRes}
// @Router /api/v1/me/brand-loyalties/{brandId} [get]
func (h *BrandLoyaltyHandler) GetUserBrandLoyalty(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetUserBrandLoyalty(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyDetailSuccess, res)
	return nil
}

// GetUserBrandLoyaltyTransactions lists current user's loyalty transactions for a brand.
// @Summary Lấy lịch sử điểm loyalty của tôi theo brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyPointTransactionDetailRes}
// @Router /api/v1/me/brand-loyalties/{brandId}/transactions [get]
func (h *BrandLoyaltyHandler) GetUserBrandLoyaltyTransactions(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetUserBrandLoyaltyTransactions(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetPointsHistorySuccess, res)
	return nil
}

// GetLoyaltyAccountTransactionsForStaff lists transactions for a loyalty account.
// @Summary Lấy lịch sử điểm của loyalty account
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Param accountId path string true "ID loyalty account"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyPointTransactionDetailRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/accounts/{accountId}/transactions [get]
func (h *BrandLoyaltyHandler) GetLoyaltyAccountTransactionsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	accountID, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyAccountTransactionsForStaff(c.Request.Context(), userID, brandID, accountID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetAccountHistorySuccess, res)
	return nil
}

// GetLoyaltyProgramForStaff gets active loyalty program for a brand.
// @Summary Lấy chương trình loyalty hoạt động của brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyProgramRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/program [get]
func (h *BrandLoyaltyHandler) GetLoyaltyProgramForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyProgramForStaff(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetProgramSuccess, res)
	return nil
}

// UpsertLoyaltyProgram creates or updates the loyalty program for a brand.
// @Summary Tạo/Cập nhật chương trình loyalty (Owner)
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.UpsertLoyaltyProgramReq true "Thông tin cấu hình"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyProgramRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/program [put]
func (h *BrandLoyaltyHandler) UpsertLoyaltyProgram(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.UpsertLoyaltyProgramReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.loyaltyUC.UpsertLoyaltyProgram(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyUpdateProgramSuccess, res)
	return nil
}

// GetLoyaltyTiersForStaff lists loyalty tiers for a brand.
// @Summary Lấy danh sách hạng loyalty của brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyTierRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/tiers [get]
func (h *BrandLoyaltyHandler) GetLoyaltyTiersForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyTiersForStaff(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetTiersSuccess, res)
	return nil
}

// CreateBrandBenefit creates a new benefit for a brand.
// @Summary Tạo quyền lợi cho brand (Staff)
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.CreateBrandBenefitReq true "Thông tin quyền lợi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits [post]
func (h *BrandLoyaltyHandler) CreateBrandBenefit(c *gin.Context) error {
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
	res, err := h.benefitUC.CreateBrandBenefit(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandLoyaltyCreateBenefitSuccess, res)
	return nil
}

// ListBrandBenefitsForStaff lists all benefits for staff.
// @Summary Lấy danh sách quyền lợi cho brand staff
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits [get]
func (h *BrandLoyaltyHandler) ListBrandBenefitsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.benefitUC.ListBrandBenefitsForStaff(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListBenefitsSuccess, res)
	return nil
}

// UpdateBenefitStatus updates the status of a benefit.
// @Summary Cập nhật trạng thái quyền lợi (Staff)
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param benefitId path string true "ID quyền lợi"
// @Param body body dto.UpdateBenefitStatusReq true "Trạng thái mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits/{benefitId}/status [patch]
func (h *BrandLoyaltyHandler) UpdateBenefitStatus(c *gin.Context) error {
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
	res, err := h.benefitUC.UpdateBenefitStatus(c.Request.Context(), userID, brandID, benefitID, input.Status)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyUpdateBenefitStatusSuccess, res)
	return nil
}

// ListActiveBenefitsForUser lists active benefits for a user.
// @Summary Lấy danh sách quyền lợi đang hoạt động của brand (User)
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandBenefitRes}
// @Router /api/v1/brands/{brandId}/benefits [get]
func (h *BrandLoyaltyHandler) ListActiveBenefitsForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.benefitUC.ListActiveBenefitsForUser(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListBenefitsSuccess, res)
	return nil
}

// RedeemBenefit allows a user to redeem a benefit.
// @Summary Đổi quyền lợi brand (User)
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param benefitId path string true "ID quyền lợi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BenefitRedemptionRes}
// @Router /api/v1/brand-benefits/{benefitId}/redeem [post]
func (h *BrandLoyaltyHandler) RedeemBenefit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	benefitID, err := uuid.Parse(c.Param("benefitId"))
	if err != nil {
		return err
	}
	res, err := h.benefitUC.RedeemBenefit(c.Request.Context(), userID, benefitID)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandLoyaltyRedeemBenefitSuccess, res)
	return nil
}

// GetActiveBenefitForUser gets active benefit detail for a user.
// @Summary Lấy chi tiết quyền lợi brand đang hoạt động
// @Tags Brand Benefit
// @Produce json
// @Param benefitId path string true "ID benefit"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-benefits/{benefitId} [get]
func (h *BrandLoyaltyHandler) GetActiveBenefitForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	benefitID, err := uuid.Parse(c.Param("benefitId"))
	if err != nil {
		return err
	}
	res, err := h.benefitUC.GetActiveBenefitForUser(c.Request.Context(), userID, benefitID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetBenefitDetailSuccess, res)
	return nil
}

// ListBenefitRedemptionsForUser lists current user's benefit redemptions.
// @Summary Lấy danh sách quyền lợi đã nhận của tôi
// @Tags Brand Benefit
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BenefitRedemptionRes}
// @Router /api/v1/me/benefit-redemptions [get]
func (h *BrandLoyaltyHandler) ListBenefitRedemptionsForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	res, err := h.benefitUC.ListBenefitRedemptionsForUser(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListRedeemedBenefitsSuccess, res)
	return nil
}

// CreateClaimToken generates a claim token for an offline customer.
// @Summary Tạo mã claim cho khách hàng offline
// @Description Tạo một mã claim ngẫu nhiên dùng để liên kết tài khoản offline của khách hàng với tài khoản online của người dùng. Hạn dùng 24 giờ.
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID của Brand"
// @Param customerId path string true "ID của khách hàng cần tạo mã claim"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CreateClaimTokenRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers/{customerId}/claim-token [post]
func (h *BrandLoyaltyHandler) CreateClaimToken(c *gin.Context) error {
	staffUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	customerID, err := uuid.Parse(c.Param("customerId"))
	if err != nil {
		return err
	}
	res, err := h.claimUC.CreateBrandCustomerClaim(c.Request.Context(), staffUserID, brandID, customerID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandCreateClaimTokenSuccess, res)
	return nil
}

// ListClaimTokens lists issued claim token metadata for an offline customer.
// @Summary Lấy danh sách mã claim của khách hàng offline
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID của Brand"
// @Param customerId path string true "ID của khách hàng"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.ClaimTokenRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers/{customerId}/claim-tokens [get]
func (h *BrandLoyaltyHandler) ListClaimTokens(c *gin.Context) error {
	staffUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	customerID, err := uuid.Parse(c.Param("customerId"))
	if err != nil {
		return err
	}
	res, err := h.claimUC.ListBrandCustomerClaims(c.Request.Context(), staffUserID, brandID, customerID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListClaimTokensSuccess, res)
	return nil
}

// RevokeClaimToken revokes one issued claim token.
// @Summary Thu hồi mã claim của khách hàng offline
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID của Brand"
// @Param customerId path string true "ID của khách hàng"
// @Param claimId path string true "ID của mã claim"
// @Param body body dto.RevokeClaimTokenReq true "Thông tin thu hồi"
// @Success 200 {object} shared_pres.APIResponse{data=dto.ClaimTokenRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers/{customerId}/claim-tokens/{claimId}/revoke [post]
func (h *BrandLoyaltyHandler) RevokeClaimToken(c *gin.Context) error {
	staffUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	customerID, err := uuid.Parse(c.Param("customerId"))
	if err != nil {
		return err
	}
	claimID, err := uuid.Parse(c.Param("claimId"))
	if err != nil {
		return err
	}
	var input dto.RevokeClaimTokenReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.claimUC.RevokeBrandCustomerClaim(c.Request.Context(), staffUserID, brandID, customerID, claimID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyRevokeClaimTokenSuccess, res)
	return nil
}

// ClaimOfflineAccount links an offline customer and loyalty account to the current user.
// @Summary Liên kết tài khoản khách hàng offline
// @Description Người dùng nhập mã claim nhận được để liên kết hồ sơ mua hàng offline của họ với tài khoản online.
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param body body dto.ClaimOfflineAccountReq true "Mã claim"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandCustomerRes}
// @Router /api/v1/brands/claim [post]
func (h *BrandLoyaltyHandler) ClaimOfflineAccount(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.ClaimOfflineAccountReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.claimUC.ClaimBrandCustomer(c.Request.Context(), userID, input.ClaimToken, c.ClientIP())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandClaimCustomerSuccess, res)
	return nil
}

// GetUserBrandLoyaltyLots lists current user's loyalty point lots for a brand.
// @Summary Lấy danh sách lô điểm loyalty của tôi theo brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Param status query string false "Trạng thái lô điểm"
// @Param expiresAt query string false "Ngày hết hạn tối đa"
// @Param page query int false "Trang"
// @Param limit query int false "Số lượng"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyPointLotRes}
// @Router /api/v1/me/brand-loyalties/{brandId}/lots [get]
func (h *BrandLoyaltyHandler) GetUserBrandLoyaltyLots(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var query dto.ListLoyaltyPointLotsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetUserBrandLoyaltyLots(c.Request.Context(), userID, brandID, query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListPointBatchesSuccess, res)
	return nil
}

// GetLoyaltyAccountLotsForStaff lists point lots for a loyalty account.
// @Summary Lấy danh sách lô điểm của loyalty account
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Param accountId path string true "ID loyalty account"
// @Param status query string false "Trạng thái lô điểm"
// @Param expiresAt query string false "Ngày hết hạn tối đa"
// @Param page query int false "Trang"
// @Param limit query int false "Số lượng"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyPointLotRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/accounts/{accountId}/lots [get]
func (h *BrandLoyaltyHandler) GetLoyaltyAccountLotsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	accountID, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		return err
	}
	var query dto.ListLoyaltyPointLotsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyAccountLotsForStaff(c.Request.Context(), userID, brandID, accountID, query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListAccountBatchesSuccess, res)
	return nil
}
