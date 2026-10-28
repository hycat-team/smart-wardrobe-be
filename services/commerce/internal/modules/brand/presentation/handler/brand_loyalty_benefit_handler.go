package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateBrandBenefit creates a new benefit for a brand.
// @Summary Tạo quyền lợi cho brand (Staff)
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param body body dto.CreateBrandBenefitReq true "Thông tin quyền lợi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits [post]
func (h *BrandLoyaltyHandler) CreateBrandBenefit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
// @Param brandId path string true "ID hoặc slug brand"
// @Param page query int false "Số trang"
// @Param limit query int false "Số lượng mỗi trang"
// @Param unlockType query string false "Loại mở khóa (tier_privilege, point_redemption, manual_grant)"
// @Param benefitType query string false "Loại quyền lợi (voucher, discount, gift, free_shipping, early_access, feature_access)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandBenefitListRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits [get]
func (h *BrandLoyaltyHandler) ListBrandBenefitsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var query dto.GetBrandBenefitsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.benefitUC.ListBrandBenefitsForStaff(c.Request.Context(), userID, brandID, query)
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
// @Param brandId path string true "ID hoặc slug brand"
// @Param benefitId path string true "ID quyền lợi"
// @Param body body dto.UpdateBenefitStatusReq true "Trạng thái mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandBenefitRes}
// @Router /api/v1/brand-portal/brands/{brandId}/benefits/{benefitId}/status [patch]
func (h *BrandLoyaltyHandler) UpdateBenefitStatus(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	benefitID, err := validation.GetUUIDParam(c, "benefitId")
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
// @Summary Lấy danh sách quyền lợi đang hoạt động của brand (Public)
// @Tags Brand Benefit
// @Accept json
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param page query int false "Số trang"
// @Param limit query int false "Số lượng mỗi trang"
// @Param unlockType query string false "Loại mở khóa (tier_privilege, point_redemption, manual_grant)"
// @Param benefitType query string false "Loại quyền lợi (voucher, discount, gift, free_shipping, early_access, feature_access)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandBenefitListRes}
// @Router /api/v1/brands/{brandId}/benefits [get]
func (h *BrandLoyaltyHandler) ListActiveBenefitsForUser(c *gin.Context) error {
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var query dto.GetBrandBenefitsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.benefitUC.ListActiveBenefitsForUser(c.Request.Context(), brandID, query)
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
	benefitID, err := validation.GetUUIDParam(c, "benefitId")
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
	benefitID, err := validation.GetUUIDParam(c, "benefitId")
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

// ListMyBenefits returns every active benefit of a brand together with the
// caller's unlock and redemption state.
// @Summary Lấy danh sách quyền lợi của tôi kèm trạng thái
// @Description Trả về các quyền lợi đang active của brand kèm trạng thái mở khoá theo hạng thành viên, điểm hiện có và lượt đổi gần nhất của người dùng hiện tại.
// @Tags Brand Benefit
// @Produce json
// @Param brandId path string true "ID hoặc slug brand"
// @Param page query int false "Số trang"
// @Param limit query int false "Số lượng mỗi trang"
// @Param unlockType query string false "Loại mở khoá (tier_privilege, point_redemption, manual_grant)"
// @Param benefitType query string false "Loại quyền lợi (voucher, discount, gift, free_shipping, early_access, feature_access)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.MyBrandBenefitListRes}
// @Router /api/v1/me/brand-loyalties/{brandId}/benefits [get]
func (h *BrandLoyaltyHandler) ListMyBenefits(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}

	var query dto.GetBrandBenefitsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	res, err := h.benefitUC.ListMyBenefits(c.Request.Context(), userID, brandID, query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListMyBenefitsSuccess, res)
	return nil
}

// ListBenefitRedemptionsForUser lists current user's benefit redemptions.
// @Summary Lấy danh sách quyền lợi đã nhận của tôi
// @Description Lấy danh sách quyền lợi đã nhận của người dùng hiện tại, hỗ trợ lọc theo brand
// @Tags Brand Benefit
// @Produce json
// @Param brandId query string false "Lọc theo Brand ID"
// @Param page query int false "Số trang"
// @Param limit query int false "Số lượng mỗi trang"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BenefitRedemptionListRes}
// @Router /api/v1/me/benefit-redemptions [get]
func (h *BrandLoyaltyHandler) ListBenefitRedemptionsForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var query dto.GetBenefitRedemptionsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	var brandID *uuid.UUID
	if query.BrandID != nil {
		id, err := uuid.Parse(*query.BrandID)
		if err != nil {
			return apperror.NewBadRequest("brandId không hợp lệ")
		}
		brandID = &id
	}

	normalized := query.PaginationQuery.Normalize()

	res, err := h.benefitUC.ListBenefitRedemptionsForUser(c.Request.Context(), userID, brandID, normalized.Page, normalized.Limit)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyListRedeemedBenefitsSuccess, res)
	return nil
}
