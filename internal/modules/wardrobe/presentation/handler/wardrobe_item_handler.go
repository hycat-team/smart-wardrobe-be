package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var _ shared_dto.PaginationQuery

const (
	msgWardrobeGetUploadSignatureSuccess    = "Lấy chữ ký tải ảnh trang phục thành công"
	msgWardrobeGetItemsSuccess              = "Lấy danh sách trang phục thành công"
	msgWardrobeGetPendingItemsSuccess       = "Lấy danh sách trang phục đang xử lý thành công"
	msgWardrobeGetItemByIDSuccess           = "Lấy thông tin chi tiết trang phục thành công"
	msgWardrobeCloneItemSuccess             = "Nhân bản trang phục thành công"
	msgWardrobeInitClosetFromCatalogSuccess = "Khởi tạo nhanh tủ đồ thành công"
	msgWardrobeBatchUploadItemsSuccess      = "Tải lên và bắt đầu phân tích hàng loạt thành công"
	msgWardrobeGetSystemCatalogItemsSuccess = "Lấy danh sách trang phục hệ thống thành công"
	msgWardrobeManualClassifySuccess        = "Tự phân loại trang phục thủ công thành công"
	msgWardrobeRetryAnalysisSuccess         = "Yêu cầu phân tích lại trang phục đã được gửi thành công"
	msgWardrobeGetCatalogItemsSuccess       = "Lấy danh sách trang phục mẫu thành công"
	msgWardrobeUpdateCatalogItemSuccess     = "Cập nhật trang phục mẫu thành công"
	msgWardrobeDeleteCatalogItemSuccess     = "Xóa trang phục mẫu thành công"
	msgWardrobeDeleteItemsBulkSuccess       = "Xóa hàng loạt trang phục thành công"
	msgWardrobeDeleteLockedItemsSuccess     = "Xóa toàn bộ trang phục bị khóa thành công"
)

// WardrobeItemHandler handles wardrobe item HTTP endpoints.
type WardrobeItemHandler struct {
	itemUseCase    usecase_interfaces.IWardrobeItemUseCase
	catalogUseCase usecase_interfaces.IWardrobeCatalogUseCase
	workerUseCase  usecase_interfaces.IWardrobeWorkerUseCase
}

// NewWardrobeItemHandler builds the wardrobe item presentation handler.
func NewWardrobeItemHandler(
	itemUseCase usecase_interfaces.IWardrobeItemUseCase,
	catalogUseCase usecase_interfaces.IWardrobeCatalogUseCase,
	workerUseCase usecase_interfaces.IWardrobeWorkerUseCase,
) *WardrobeItemHandler {
	return &WardrobeItemHandler{
		itemUseCase:    itemUseCase,
		catalogUseCase: catalogUseCase,
		workerUseCase:  workerUseCase,
	}
}

// GetUploadSignature get secure cloudinary signature for item upload
// @Summary Lấy chữ ký tải ảnh trang phục
// @Description Lấy chữ ký bảo mật từ Cloudinary để client tải trực tiếp ảnh trang phục lên
// @Tags Wardrobe
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.UploadSignatureResult} "Chữ ký và thông tin upload"
// @Router /api/v1/wardrobe-items/upload-signature [get]
func (h *WardrobeItemHandler) GetUploadSignature(c *gin.Context) error {
	signatureRes, err := h.itemUseCase.GetUploadSignature(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetUploadSignatureSuccess, signatureRes)
	return nil
}

// GetWardrobeItems get all usable wardrobe items with lock status
// @Summary Lấy danh sách trang phục usable
// @Description Lấy danh sách trang phục usable trong tủ đồ của người dùng và áp dụng trạng thái khóa động nếu hạ cấp gói
// @Tags Wardrobe
// @Produce json
// @Param page query int false "Số trang (mặc định: 1)"
// @Param limit query int false "Số lượng phần tử trên trang (mặc định: 20)"
// @Param category_slug query string false "Slug danh mục cần lọc"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.WardrobeItemRes]} "Danh sách trang phục usable"
// @Router /api/v1/me/wardrobe-items [get]
func (h *WardrobeItemHandler) GetWardrobeItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var query dto.GetWardrobeItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.itemUseCase.GetWardrobeItems(c.Request.Context(), userID, query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetItemsSuccess, response)
	return nil
}

// GetPendingWardrobeItems get non-usable wardrobe items for processing and review UX
// @Summary Lấy danh sách trang phục đang xử lý hoặc cần rà soát
// @Description Lấy các trang phục chưa usable trong tủ đồ, gồm Đang xử lý, Lỗi phân tích, hoặc Cần người dùng rà soát
// @Tags Wardrobe
// @Produce json
// @Param page query int false "Số trang (mặc định: 1)"
// @Param limit query int false "Số lượng phần tử trên trang (mặc định: 20)"
// @Param status query int false "Lọc theo status cụ thể"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.WardrobeItemRes]} "Danh sách trang phục non-usable"
// @Router /api/v1/me/wardrobe-items/pending [get]
func (h *WardrobeItemHandler) GetPendingWardrobeItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var query dto.GetPendingWardrobeItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.itemUseCase.GetPendingWardrobeItems(c.Request.Context(), userID, query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetPendingItemsSuccess, response)
	return nil
}

// GetWardrobeItemByID get details of a specific wardrobe item
// @Summary Xem chi tiết trang phục
// @Description Lấy thông tin chi tiết của một trang phục theo ID, tự động chặn nếu trang phục nằm trong vùng bị khóa
// @Tags Wardrobe
// @Produce json
// @Param id path string true "ID trang phục"
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Chi tiết trang phục"
// @Router /api/v1/wardrobe-items/{id} [get]
func (h *WardrobeItemHandler) GetWardrobeItemByID(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	response, err := h.itemUseCase.GetWardrobeItemByID(c.Request.Context(), userID, itemID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetItemByIDSuccess, response)
	return nil
}

// CloneWardrobeItem clone an existing wardrobe item
// @Summary Nhân bản trang phục
// @Description Sao chép nhanh một trang phục có sẵn trong tủ đồ, tối đa 5 bản sao
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param id path string true "ID trang phục gốc"
// @Param body body dto.CloneWardrobeItemReq true "Số lượng nhân bản"
// @Success 201 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục được nhân bản"
// @Router /api/v1/wardrobe-items/{id}/clone [post]
func (h *WardrobeItemHandler) CloneWardrobeItem(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	var input dto.CloneWardrobeItemReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.itemUseCase.CloneWardrobeItem(c.Request.Context(), userID, itemID, input.Quantity)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgWardrobeCloneItemSuccess, response)
	return nil
}

// InitClosetFromCatalog initialize user's wardrobe using pre-analyzed global catalog templates
// @Summary Khởi tạo nhanh tủ đồ cá nhân
// @Description Sao chép hàng loạt các trang phục mẫu từ hệ thống sang tủ đồ cá nhân của user, không tốn quota AI
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param body body dto.InitClosetFromCatalogReq true "Danh sách ID trang phục mẫu"
// @Success 201 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục cá nhân được tạo"
// @Router /api/v1/wardrobe-items/catalog-init [post]
func (h *WardrobeItemHandler) InitClosetFromCatalog(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.InitClosetFromCatalogReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.catalogUseCase.InitClosetFromCatalog(c.Request.Context(), userID, input.CatalogItemIDs)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgWardrobeInitClosetFromCatalogSuccess, response)
	return nil
}

// BatchUploadWardrobeItems upload and process background AI analysis for wardrobe images
// @Summary Số hóa trang phục hàng loạt
// @Description Hỗ trợ upload hàng loạt ảnh trang phục, hệ thống sẽ tạo các item ở trạng thái Đang xử lý và tự động gọi AI phân tích ngầm
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param body body dto.BatchUploadWardrobeItemsReq true "Danh sách ảnh trang phục"
// @Success 201 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục đang được xử lý ngầm"
// @Router /api/v1/wardrobe-items/batch-upload [post]
func (h *WardrobeItemHandler) BatchUploadWardrobeItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	roleSlug, err := contextutils.GetRoleSlug(c)
	if err != nil {
		return err
	}

	var input dto.BatchUploadWardrobeItemsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.itemUseCase.BatchUploadWardrobeItems(c.Request.Context(), userID, roleSlug, input)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgWardrobeBatchUploadItemsSuccess, response)
	return nil
}

// GetSystemCatalogWardrobeItems get all system catalog wardrobe items
// @Summary Lấy danh sách trang phục hệ thống
// @Description Lấy danh sách trang phục mẫu của hệ thống, hỗ trợ tìm kiếm thông minh bằng Elasticsearch và fallback sang database khi cần
// @Tags Wardrobe
// @Produce json
// @Param page query int false "Số trang (mặc định: 1)"
// @Param limit query int false "Số lượng phần tử trên trang (mặc định: 20)"
// @Param q query string false "Từ khóa tìm kiếm"
// @Param category_slug query string false "Slug danh mục cần lọc"
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

// ManualClassify manual classification fallback for a failed or review-required wardrobe item
// @Summary Tự phân loại trang phục thủ công
// @Description Cho phép người dùng tự điền tay thông tin cho trang phục phân tích lỗi hoặc cần rà soát, hệ thống dùng Text Embedding cập nhật vector và duyệt vào tủ đồ
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param id path string true "ID trang phục"
// @Param body body dto.ManualClassifyReq true "Thông tin phân loại thủ công"
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Chi tiết trang phục sau khi cập nhật"
// @Router /api/v1/wardrobe-items/{id}/manual-classify [put]
func (h *WardrobeItemHandler) ManualClassify(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	var input dto.ManualClassifyReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.itemUseCase.ManualClassify(c.Request.Context(), userID, itemID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeManualClassifySuccess, response)
	return nil
}

// RetryWardrobeAnalysis requeues AI analysis for a failed or review-required wardrobe item
// @Summary Thử phân tích lại trang phục
// @Description Cho phép người dùng gửi lại yêu cầu phân tích AI cho trang phục đang lỗi hoặc cần rà soát
// @Tags Wardrobe
// @Produce json
// @Param id path string true "ID trang phục"
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Chi tiết trang phục sau khi đưa lại vào hàng đợi xử lý"
// @Router /api/v1/wardrobe-items/{id}/retry-analysis [post]
func (h *WardrobeItemHandler) RetryWardrobeAnalysis(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	response, err := h.itemUseCase.RetryWardrobeAnalysis(c.Request.Context(), userID, itemID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeRetryAnalysisSuccess, response)
	return nil
}

// GetCatalogItemsAdmin gets all system catalog templates (Admin)
// @Summary Lấy danh sách trang phục mẫu (Admin)
// @Description Cho phép Admin lấy danh sách trang phục mẫu hệ thống để quản lý
// @Tags Admin
// @Produce json
// @Param page query int false "Số trang (mặc định: 1)"
// @Param limit query int false "Số lượng phần tử trên trang (mặc định: 20)"
// @Param q query string false "Từ khóa tìm kiếm"
// @Param category_slug query string false "Slug danh mục"
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
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "ID trang phục mẫu"
// @Param body body dto.UpdateSystemCatalogItemReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Thông tin trang phục mẫu sau cập nhật"
// @Router /api/v1/admin/wardrobe-items/{id} [put]
func (h *WardrobeItemHandler) UpdateCatalogItemAdmin(c *gin.Context) error {
	itemID, err := uuid.Parse(c.Param("id"))
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
// @Tags Admin
// @Produce json
// @Param id path string true "ID trang phục mẫu"
// @Success 200 {object} shared_pres.APIResponse "Xóa thành công"
// @Router /api/v1/admin/wardrobe-items/{id} [delete]
func (h *WardrobeItemHandler) DeleteCatalogItemAdmin(c *gin.Context) error {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	if err := h.catalogUseCase.DeleteSystemCatalogItem(c.Request.Context(), itemID); err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeDeleteCatalogItemSuccess, nil)
	return nil
}

// DeleteWardrobeItemsBulk delete multiple wardrobe items
// @Summary Xóa hàng loạt trang phục
// @Description Cho phép người dùng chọn và xóa mềm hàng loạt trang phục khỏi tủ đồ để giải phóng quota dung lượng
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param body body dto.BulkDeleteItemsReq true "Danh sách ID trang phục cần xóa"
// @Success 200 {object} shared_pres.APIResponse "Xóa hàng loạt trang phục thành công"
// @Router /api/v1/wardrobe-items/bulk [delete]
func (h *WardrobeItemHandler) DeleteWardrobeItemsBulk(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.BulkDeleteItemsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.itemUseCase.DeleteWardrobeItemsBulk(c.Request.Context(), userID, input.IDs); err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeDeleteItemsBulkSuccess, nil)
	return nil
}

// DeleteLockedWardrobeItems delete all locked wardrobe items due to quota
// @Summary Xóa toàn bộ trang phục bị khóa
// @Description Tự động quét và xóa mềm toàn bộ trang phục vượt quá hạn mức sử dụng bị khóa của người dùng để giải phóng quota
// @Tags Wardrobe
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Xóa toàn bộ trang phục bị khóa thành công"
// @Router /api/v1/wardrobe-items/locked [delete]
func (h *WardrobeItemHandler) DeleteLockedWardrobeItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	if err := h.itemUseCase.DeleteLockedWardrobeItems(c.Request.Context(), userID); err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeDeleteLockedItemsSuccess, nil)
	return nil
}
