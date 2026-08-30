package handler

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.LoyaltyPointTransactionBriefRes}
// @Router /api/v1/me/brand-loyalties/{brandId}/transactions [get]
func (h *BrandLoyaltyHandler) GetUserBrandLoyaltyTransactions(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
// @Param page query int false "Trang hiện tại (mặc định: 1)"
// @Param limit query int false "Số lượng bản ghi trên mỗi trang (mặc định: 20)"
// @Tags Brand Loyalty
// @Produce json
// @Param brandId path string true "ID brand"
// @Param accountId path string true "ID loyalty account"
// @Success 200 {object} shared_pres.APIResponse{data=dto.LoyaltyTransactionListRes}
// @Router /api/v1/brand-portal/brands/{brandId}/loyalty/accounts/{accountId}/transactions [get]
func (h *BrandLoyaltyHandler) GetLoyaltyAccountTransactionsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	accountID, err := validation.GetUUIDParam(c, "accountId")
	if err != nil {
		return err
	}

	var query dto.GetLoyaltyTransactionsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	res, err := h.loyaltyUC.GetLoyaltyAccountTransactionsForStaff(c.Request.Context(), userID, brandID, accountID, query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandLoyaltyGetAccountHistorySuccess, res)
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	accountID, err := validation.GetUUIDParam(c, "accountId")
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
