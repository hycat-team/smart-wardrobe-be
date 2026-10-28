package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

// GetLoyaltyProgramForStaff gets active loyalty program for a brand.
// @Summary Lấy chương trình loyalty hoạt động của brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyProgramRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/program [get]
func (h *BrandLoyaltyHandler) GetLoyaltyProgramForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
// @Param brandId path string true "ID hoặc slug brand"
// @Param body body dto.UpsertLoyaltyProgramReq true "Thông tin cấu hình"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyProgramRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/program [put]
func (h *BrandLoyaltyHandler) UpsertLoyaltyProgram(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
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

// CreateLoyaltyTier creates a new loyalty tier for a brand.
// @Summary Tạo hạng thành viên cho brand (Owner)
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param body body dto.CreateLoyaltyTierReq true "Thông tin hạng thành viên"
// @Success 201 {object} shared_pres.APIResponse{data=dto.LoyaltyTierRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/tiers [post]
func (h *BrandLoyaltyHandler) CreateLoyaltyTier(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var input dto.CreateLoyaltyTierReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.loyaltyUC.CreateLoyaltyTier(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandLoyaltyCreateTierSuccess, res)
	return nil
}

// UpdateLoyaltyTier updates a loyalty tier.
// @Summary Cập nhật hạng thành viên (Owner)
// @Tags Brand Loyalty
// @Accept json
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param tierId path string true "ID hạng thành viên"
// @Param body body dto.UpdateLoyaltyTierReq true "Thông tin cập nhật hạng thành viên"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyTierRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/tiers/{tierId} [put]
func (h *BrandLoyaltyHandler) UpdateLoyaltyTier(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	tierID, err := validation.GetUUIDParam(c, "tierId")
	if err != nil {
		return err
	}
	var input dto.UpdateLoyaltyTierReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.loyaltyUC.UpdateLoyaltyTier(c.Request.Context(), userID, brandID, tierID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyUpdateTierSuccess, res)
	return nil
}

// GetLoyaltyTiersForStaff lists loyalty tiers for a brand.
// @Summary Lấy danh sách hạng loyalty của brand
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyTierRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/tiers [get]
func (h *BrandLoyaltyHandler) GetLoyaltyTiersForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
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

// GetLoyaltyTierDetailForStaff gets tier detail with benefits for staff.
// @Summary Lấy chi tiết hạng loyalty kèm quyền lợi (Staff)
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param tierId path string true "ID hạng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyTierDetailRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/tiers/{tierId} [get]
func (h *BrandLoyaltyHandler) GetLoyaltyTierDetailForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	tierID, err := validation.GetUUIDParam(c, "tierId")
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyTierDetailForStaff(c.Request.Context(), userID, brandID, tierID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetTiersSuccess, res)
	return nil
}

// GetLoyaltyTierDetailForUser gets tier detail with benefits for user.
// @Summary Lấy chi tiết hạng loyalty kèm quyền lợi (User)
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param tierId path string true "ID hạng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyTierDetailRes}
// @Router /api/v1/brands/{brandId}/loyalty/tiers/{tierId} [get]
func (h *BrandLoyaltyHandler) GetLoyaltyTierDetailForUser(c *gin.Context) error {
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	tierID, err := validation.GetUUIDParam(c, "tierId")
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyTierDetailForUser(c.Request.Context(), brandID, tierID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetTiersSuccess, res)
	return nil
}

// GetLoyaltyTiersForUser lists loyalty tiers with cumulative tier privileges.
// @Summary Lấy danh sách hạng loyalty kèm quyền lợi (User)
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyTierDetailRes}
// @Router /api/v1/brands/{brandId}/loyalty/tiers [get]
func (h *BrandLoyaltyHandler) GetLoyaltyTiersForUser(c *gin.Context) error {
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	res, err := h.loyaltyUC.GetLoyaltyTiersForUser(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetTiersSuccess, res)
	return nil
}
