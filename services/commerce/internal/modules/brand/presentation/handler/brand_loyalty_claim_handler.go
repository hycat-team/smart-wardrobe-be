package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	customerID, err := validation.GetUUIDParam(c, "customerId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	customerID, err := validation.GetUUIDParam(c, "customerId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	customerID, err := validation.GetUUIDParam(c, "customerId")
	if err != nil {
		return err
	}
	claimID, err := validation.GetUUIDParam(c, "claimId")
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
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandCustomerBriefRes}
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
