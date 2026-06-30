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

type BrandPortalHandler struct {
	brandUC usecase_interfaces.IBrandUseCase
	itemUC  usecase_interfaces.IBrandItemUseCase
}

func NewBrandPortalHandler(brandUC usecase_interfaces.IBrandUseCase, itemUC usecase_interfaces.IBrandItemUseCase) *BrandPortalHandler {
	return &BrandPortalHandler{
		brandUC: brandUC,
		itemUC:  itemUC,
	}
}

// CreateBrandRequest creates a pending brand request.
// @Summary Gửi yêu cầu tạo brand
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param body body dto.CreateBrandReq true "Thông tin brand"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brand-portal/brands [post]
func (h *BrandPortalHandler) CreateBrandRequest(c *gin.Context) error {
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
func (h *BrandPortalHandler) CreateBrandAdmin(c *gin.Context) error {
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
func (h *BrandPortalHandler) UpdateBrandStatusAdmin(c *gin.Context) error {
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

// GetBrandsAdmin lấy danh sách brand cho Admin
// @Summary Lấy danh sách brand (Admin)
// @Description Cho phép admin lấy danh sách brand phân trang, tìm kiếm theo tên/slug và lọc theo trạng thái của brand.
// @Tags Admin
// @Accept json
// @Produce json
// @Param query query dto.GetBrandsAdminQueryReq false "Bộ lọc danh sách brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminBrandListRes} "Lấy danh sách brand thành công"
// @Router /api/v1/admin/brands [get]
func (h *BrandPortalHandler) GetBrandsAdmin(c *gin.Context) error {
	var query dto.GetBrandsAdminQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandsForAdmin(c.Request.Context(), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandListSuccess, res)
	return nil
}

// GetActiveBrands lists public active brands.
// @Summary Lấy danh sách brand đang active
// @Tags Brand
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandRes}
// @Router /api/v1/brands [get]
func (h *BrandPortalHandler) GetActiveBrands(c *gin.Context) error {
	res, err := h.brandUC.GetActiveBrands(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandListSuccess, res)
	return nil
}

// GetActiveBrand gets public active brand detail.
// @Summary Lấy chi tiết brand active
// @Tags Brand
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brands/{brandId} [get]
func (h *BrandPortalHandler) GetActiveBrand(c *gin.Context) error {
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetActiveBrand(c.Request.Context(), brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandDetailSuccess, res)
	return nil
}

// GetMyPortalBrands lists brands where the current user is an active member.
// @Summary Lấy danh sách brand của staff hiện tại
// @Tags Brand Portal
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.PortalBrandRes}
// @Router /api/v1/brand-portal/me/brands [get]
func (h *BrandPortalHandler) GetMyPortalBrands(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandsForPortalUser(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandListSuccess, res)
	return nil
}

// GetBrandLogoUploadSignature gets Cloudinary signature for brand logo upload.
// @Summary Lấy chữ ký upload logo brand
// @Tags Brand Portal
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult}
// @Router /api/v1/brand-portal/brands/logo-upload-signature [get]
func (h *BrandPortalHandler) GetBrandLogoUploadSignature(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandLogoUploadSignature(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy chữ ký upload logo brand thành công", res)
	return nil
}

// UpdateBrandLogo updates brand logo Cloudinary references.
// @Summary Cập nhật logo brand
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.UpdateBrandLogoReq true "Thông tin logo"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brand-portal/brands/{brandId}/logo [patch]
func (h *BrandPortalHandler) UpdateBrandLogo(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.UpdateBrandLogoReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.UpdateBrandLogo(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Cập nhật logo brand thành công", res)
	return nil
}

// GetBrandForPortal gets brand detail for active brand members.
// @Summary Lấy thông tin brand portal
// @Tags Brand Portal
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PortalBrandRes}
// @Router /api/v1/brand-portal/brands/{brandId} [get]
func (h *BrandPortalHandler) GetBrandForPortal(c *gin.Context) error {
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

// CreateBrandItem creates a new brand item/sample.
// @Summary [Staff] Tạo sản phẩm hoặc mẫu thử của Brand
// @Tags Brand Item
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.CreateBrandItemReq true "Thông tin sản phẩm"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items [post]
func (h *BrandPortalHandler) CreateBrandItem(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.CreateBrandItemReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.itemUC.CreateBrandItem(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandItemCreateSuccess, res)
	return nil
}

// GetBrandItemsForStaff retrieves brand items.
// @Summary [Staff] Lấy danh sách sản phẩm hoặc mẫu thử của Brand
// @Tags Brand Item
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandItemRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items [get]
func (h *BrandPortalHandler) GetBrandItemsForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.itemUC.GetBrandItemsForStaff(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemListSuccess, res)
	return nil
}

// GetBrandItemUploadSignature gets Cloudinary signature for brand item upload.
// @Summary Lấy chữ ký upload ảnh sản phẩm brand
// @Tags Brand Item
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult}
// @Router /api/v1/brand-portal/brands/{brandId}/items/upload-signature [get]
func (h *BrandPortalHandler) GetBrandItemUploadSignature(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.itemUC.GetBrandItemUploadSignature(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy chữ ký upload ảnh sản phẩm brand thành công", res)
	return nil
}

// GetBrandItemForStaff retrieves one brand item for staff.
// @Summary Lấy chi tiết sản phẩm brand cho staff
// @Tags Brand Item
// @Produce json
// @Param brandId path string true "ID brand"
// @Param itemId path string true "ID item"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items/{itemId} [get]
func (h *BrandPortalHandler) GetBrandItemForStaff(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		return err
	}
	res, err := h.itemUC.GetBrandItemForStaff(c.Request.Context(), userID, brandID, itemID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemGetSuccess, res)
	return nil
}

// UpdateBrandItem updates an existing brand item.
// @Summary [Staff] Cập nhật sản phẩm hoặc mẫu thử của Brand
// @Tags Brand Item
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param itemId path string true "ID sản phẩm"
// @Param body body dto.UpdateBrandItemReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items/{itemId} [put]
func (h *BrandPortalHandler) UpdateBrandItem(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		return err
	}
	var input dto.UpdateBrandItemReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.itemUC.UpdateBrandItem(c.Request.Context(), userID, brandID, itemID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemUpdateSuccess, res)
	return nil
}

// UpdateBrandItemStatus updates brand item status only.
// @Summary Cập nhật trạng thái sản phẩm brand
// @Tags Brand Item
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param itemId path string true "ID item"
// @Param body body dto.UpdateBrandItemStatusReq true "Trang thai moi"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items/{itemId}/status [patch]
func (h *BrandPortalHandler) UpdateBrandItemStatus(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		return err
	}
	var input dto.UpdateBrandItemStatusReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.itemUC.UpdateBrandItemStatus(c.Request.Context(), userID, brandID, itemID, input.Status)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemUpdateSuccess, res)
	return nil
}

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
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
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

// ListBrandItemsForUser lists active brand items for consumers.
// @Summary [User] Lấy danh sách sản phẩm hoặc mẫu thử hoạt động của Brand
// @Tags Brand
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandItemRes}
// @Router /api/v1/brands/{brandId}/items [get]
func (h *BrandPortalHandler) ListBrandItemsForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.itemUC.ListBrandItemsForUser(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandItemListSuccess, res)
	return nil
}

// GetBrandItemForUser gets active brand item detail.
// @Summary Lấy chi tiết sản phẩm brand đang hoạt động
// @Tags Brand
// @Produce json
// @Param itemId path string true "ID item"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-items/{itemId} [get]
func (h *BrandPortalHandler) GetBrandItemForUser(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		return err
	}
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
	itemID, err := uuid.Parse(c.Param("itemId"))
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
