package handler

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
// @Param page query int false "Trang hiện tại (mặc định: 1)"
// @Param limit query int false "Số lượng bản ghi trên mỗi trang (mặc định: 20)"
// @Param q query string false "Từ khóa tìm kiếm (tên, SĐT, mã KH)"
// @Param status query string false "Trạng thái khách hàng (ACTIVE, INACTIVE)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandCustomerListRes}
// @Router /api/v1/brand-portal/brands/{brandId}/customers [get]
func (h *BrandLoyaltyHandler) GetBrandCustomers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}

	var query dto.GetBrandCustomersQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	res, err := h.loyaltyUC.GetBrandCustomers(c.Request.Context(), userID, brandID, query)
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	customerID, err := validation.GetUUIDParam(c, "customerId")
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
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandCustomerBriefRes}
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
