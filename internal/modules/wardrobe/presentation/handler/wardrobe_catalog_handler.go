package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/pkg/utils/validation"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

// GetSystemCatalogWardrobeItems get all system catalog wardrobe items
// @Summary Lấy danh sách trang phục hệ thống
// @Description Lấy danh sách trang phục mẫu của hệ thống, hỗ trợ tìm kiếm thông minh bằng Elasticsearch và fallback sang database khi cần
// @Tags Wardrobe
// @Produce json
// @Param query query dto.SearchWardrobeItemsQueryReq false "Bộ lọc tìm kiếm trang phục hệ thống"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.SearchWardrobeItemRes]} "Danh sách trang phục tìm thấy"
// @Router /api/v1/system-catalog/wardrobe-items [get]
func (h *WardrobeItemHandler) GetSystemCatalogWardrobeItems(c *gin.Context) error {
	var query dto.SearchWardrobeItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.itemUseCase.GetSystemCatalogWardrobeItems(c.Request.Context(), query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetSystemCatalogItemsSuccess, response)
	return nil
}

// GetCatalogItemsAdmin gets all system catalog templates (Admin)
// @Summary Lấy danh sách trang phục mẫu (Admin)
// @Description Cho phép Admin lấy danh sách trang phục mẫu hệ thống để quản lý
// @Tags Admin Wardrobe
// @Produce json
// @Param query query dto.GetSystemCatalogItemsQueryReq false "Bộ lọc danh sách trang phục mẫu"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.WardrobeItemRes]} "Danh sách trang phục mẫu"
// @Router /api/v1/admin/wardrobe-items [get]
func (h *WardrobeItemHandler) GetCatalogItemsAdmin(c *gin.Context) error {
	var query dto.GetSystemCatalogItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.catalogUseCase.GetSystemCatalogItems(c.Request.Context(), query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetCatalogItemsSuccess, response)
	return nil
}

// UpdateCatalogItemAdmin updates a system catalog item (Admin)
// @Summary Cập nhật trang phục mẫu (Admin)
// @Description Cho phép Admin cập nhật thông tin thuộc tính của một trang phục mẫu hệ thống
// @Tags Admin Wardrobe
// @Accept json
// @Produce json
// @Param id path string true "ID trang phục mẫu"
// @Param body body dto.UpdateSystemCatalogItemReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Thông tin trang phục mẫu sau cập nhật"
// @Router /api/v1/admin/wardrobe-items/{id} [put]
func (h *WardrobeItemHandler) UpdateCatalogItemAdmin(c *gin.Context) error {
	itemID, err := validation.GetUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var input dto.UpdateSystemCatalogItemReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.catalogUseCase.UpdateSystemCatalogItem(c.Request.Context(), itemID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeUpdateCatalogItemSuccess, response)
	return nil
}

// DeleteCatalogItemAdmin deletes a system catalog item (Admin)
// @Summary Xóa trang phục mẫu (Admin)
// @Description Cho phép Admin xóa một trang phục mẫu hệ thống
// @Tags Admin Wardrobe
// @Produce json
// @Param id path string true "ID trang phục mẫu"
// @Success 200 {object} shared_pres.APIResponse "Xóa thành công"
// @Router /api/v1/admin/wardrobe-items/{id} [delete]
func (h *WardrobeItemHandler) DeleteCatalogItemAdmin(c *gin.Context) error {
	itemID, err := validation.GetUUIDParam(c, "id")
	if err != nil {
		return err
	}

	if err := h.catalogUseCase.DeleteSystemCatalogItem(c.Request.Context(), itemID); err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeDeleteCatalogItemSuccess, nil)
	return nil
}

// compile-time guard keeps the swagger annotation import valid after the
// catalog handlers were split out of wardrobe_item_handler.go.
var _ = shared_dto.PaginationResult[*dto.SearchWardrobeItemRes]{}
