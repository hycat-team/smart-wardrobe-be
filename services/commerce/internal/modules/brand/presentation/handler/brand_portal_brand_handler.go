package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

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
// @Tags Admin Brand
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
// @Tags Admin Brand
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
// @Tags Admin Brand
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
// @Description Supports pagination and search by brand name or slug.
// @Tags Brand
// @Produce json
// @Param query query dto.GetActiveBrandsQueryReq false "Active brand filters"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PublicBrandBriefListRes}
// @Router /api/v1/brands [get]
func (h *BrandPortalHandler) GetActiveBrands(c *gin.Context) error {
	var query dto.GetActiveBrandsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.brandUC.GetActiveBrands(c.Request.Context(), query)
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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

// GetBrandLogoUploadSignature gets Cloudinary signature for brand images upload (logo + background).
// @Summary Lấy chữ ký upload logo/ảnh nền brand
// @Tags Brand Portal
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult}
// @Router /api/v1/brand-portal/brands/profile-images/upload-signature [get]
func (h *BrandPortalHandler) GetBrandLogoUploadSignature(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetBrandLogoUploadSignature(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandGetUploadLogoSignatureSuccess, res)
	return nil
}

// UpdateBrandImages updates brand logo and/or background Cloudinary references.
// @Summary Cập nhật logo/ảnh nền brand
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.UpdateBrandImagesReq true "Thông tin logo và ảnh nền"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brand-portal/brands/{brandId}/profile-images [patch]
func (h *BrandPortalHandler) UpdateBrandImages(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var input dto.UpdateBrandImagesReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.UpdateBrandImages(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandUpdateLogoSuccess, res)
	return nil
}

// UpdateBrandProfile updates the brand name and description.
// @Summary Cập nhật thông tin brand (Owner)
// @Tags Brand Portal
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.UpdateBrandProfileReq true "Tên và mô tả thương hiệu"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandRes}
// @Router /api/v1/brand-portal/brands/{brandId}/profile [patch]
func (h *BrandPortalHandler) UpdateBrandProfile(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	var input dto.UpdateBrandProfileReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.UpdateBrandProfile(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandUpdateProfileSuccess, res)
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
