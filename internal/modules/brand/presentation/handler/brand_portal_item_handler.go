package handler

import (
	"io"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

// CreateBrandItem creates a new brand item/sample.
// @Summary [Staff] Tạo sản phẩm hoặc mẫu thử của Brand
// @Tags Brand Item
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.CreateBrandItemReq true "Thông tin sản phẩm"
// @Success 202 {object} shared_pres.APIResponse{data=dto.BrandItemRes}
// @Router /api/v1/brand-portal/brands/{brandId}/items [post]
func (h *BrandPortalHandler) CreateBrandItem(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
	shared_pres.Accepted(c, msgBrandItemCreateSuccess, res)
	return nil
}

// SubscribeBrandItemTaskSSE opens a Server-Sent Events stream for a specific brand item creation taskId
// @Summary Đăng ký nhận thông báo realtime cho Brand Item Task ID qua SSE
// @Description Mở kết nối SSE để lắng nghe các cập nhật trạng thái phân tích ảnh của Brand Item Task ID
// @Tags Brand Item
// @Param brandId path string true "ID brand"
// @Param taskId path string true "Mã Task ID"
// @Success 200 {string} string "SSE event stream"
// @Produce text/event-stream
// @Router /api/v1/brand-portal/brands/{brandId}/items/tasks/{taskId}/sse [get]
func (h *BrandPortalHandler) SubscribeBrandItemTaskSSE(c *gin.Context) error {
	taskId, err := validation.GetStringParam(c, "taskId")
	if err != nil {
		return err
	}

	shared_pres.InitSSE(c)

	ch := h.sseManager.Subscribe(taskId)
	defer h.sseManager.Unsubscribe(taskId, ch)

	c.Stream(func(w io.Writer) bool {
		// Keep long analysis streams alive past the server-wide WriteTimeout
		// (spec 019 FR-003).
		shared_pres.ExtendSSEWriteDeadline(c)
		select {
		case <-c.Request.Context().Done():
			return false
		case data, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("message", string(data))
			return true
		}
	})

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
	brandID, err := validation.GetUUIDParam(c, "brandId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	res, err := h.itemUC.GetBrandItemUploadSignature(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandGetUploadItemSignatureSuccess, res)
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	itemID, err := validation.GetUUIDParam(c, "itemId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	itemID, err := validation.GetUUIDParam(c, "itemId")
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
	brandID, err := validation.GetUUIDParam(c, "brandId")
	if err != nil {
		return err
	}
	itemID, err := validation.GetUUIDParam(c, "itemId")
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
