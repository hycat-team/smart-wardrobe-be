package handler

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

// GetBrandItemFeedbacks retrieves feedbacks for a specific brand sample.
// @Summary [Staff] Lấy phản hồi/đóng góp ý kiến mẫu thử kỹ thuật số
// @Tags Brand Item
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param itemId path string true "ID sản phẩm mẫu thử"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.DigitalSampleResponseRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items/{itemId}/feedbacks [get]
func (h *BrandPortalHandler) GetBrandItemFeedbacks(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	itemID, err := validation.GetUUIDParam(c, "itemId")
	if err != nil {
		return err
	}
	res, err := h.itemUC.GetBrandItemFeedbacks(c.Request.Context(), userID, brandID, itemID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemFeedbackSuccess, res)
	return nil
}

// ListBrandProducts lists active products for a brand (public, paginated).
// @Summary Lấy danh sách sản phẩm brand (public)
// @Tags Brand
// @Produce json
// @Param brandId path string true "ID brand"
// @Param page query int false "Trang"
// @Param limit query int false "Số lượng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemListBriefRes}
// @Router /api/v1/brands/{brandId}/items [get]
func (h *BrandPortalHandler) ListBrandProducts(c *gin.Context) error {
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var query dto.GetBrandItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.itemUC.ListBrandProductsForCustomer(c.Request.Context(), brandID, query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemListSuccess, res)
	return nil
}

// ListBrandSamples lists active samples for a brand customer (requires sample_mix_access).
// @Summary Lấy danh sách mẫu thử brand
// @Tags Brand
// @Produce json
// @Param brandId path string true "ID brand"
// @Param page query int false "Trang"
// @Param limit query int false "Số lượng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemListBriefRes}
// @Router /api/v1/brands/{brandId}/items/samples [get]
func (h *BrandPortalHandler) ListBrandSamples(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var query dto.GetBrandItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.itemUC.ListBrandSamplesForCustomer(c.Request.Context(), userID, brandID, query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemListSuccess, res)
	return nil
}

// GetBrandItemForUser gets active brand item detail. Product is public, sample requires auth + access check.
// @Summary Lấy chi tiết sản phẩm/mẫu thử brand (product public, sample yêu cầu auth)
// @Description Product public hoàn toàn (không cần đăng nhập). Sample yêu cầu user đã đăng nhập và có quyền sample_mix_access, nếu không đủ quyền trả về 403.
// @Tags Brand
// @Produce json
// @Param itemId path string true "ID item"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-items/{itemId} [get]
func (h *BrandPortalHandler) GetBrandItemForUser(c *gin.Context) error {
	itemID, err := validation.GetUUIDParam(c, "itemId")
	if err != nil {
		return err
	}
	userID := contextutils.GetUserIdOptional(c)
	res, err := h.itemUC.GetBrandItemForUser(c.Request.Context(), userID, itemID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandDetailSuccess, res)
	return nil
}

// SubmitSampleFeedback submits user feedback on a brand sample.
// @Summary [User] Gửi phản hồi, đánh giá mẫu thử kỹ thuật số
// @Tags Brand
// @Accept json
// @Produce json
// @Param itemId path string true "ID sản phẩm mẫu thử"
// @Param body body dto.SubmitSampleFeedbackReq true "Nội dung phản hồi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.DigitalSampleResponseRes}
// @Router /api/v1/brand-items/{itemId}/feedbacks [post]
func (h *BrandPortalHandler) SubmitSampleFeedback(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	itemID, err := validation.GetUUIDParam(c, "itemId")
	if err != nil {
		return err
	}
	var input dto.SubmitSampleFeedbackReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.itemUC.SubmitSampleFeedback(c.Request.Context(), userID, itemID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandFeedbackSubmitSuccess, res)
	return nil
}
