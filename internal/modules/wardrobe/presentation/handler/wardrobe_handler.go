package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	_ "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgWardrobeGetUploadSignatureSuccess    = "Lấy chữ ký tải ảnh trang phục thành công"
	msgWardrobeGetItemsSuccess              = "Lấy danh sách trang phục thành công"
	msgWardrobeGetItemByIDSuccess           = "Lấy thông tin chi tiết trang phục thành công"
	msgWardrobeCloneItemSuccess             = "Nhân bản trang phục thành công"
	msgWardrobeInitClosetFromCatalogSuccess = "Khởi tạo nhanh tủ đồ thành công"
	msgWardrobeBatchUploadItemsSuccess      = "Tải lên và bắt đầu phân tích hàng loạt thành công"
	msgWardrobeSearchItemsSuccess           = "Tìm kiếm trang phục thành công"
	msgWardrobeManualClassifySuccess        = "Tự phân loại trang phục thủ công thành công"
)

type WardrobeHandler struct {
	wardrobeUseCase usecase_interfaces.IWardrobeUseCase
}

func NewWardrobeHandler(uc usecase_interfaces.IWardrobeUseCase) *WardrobeHandler {
	return &WardrobeHandler{
		wardrobeUseCase: uc,
	}
}

// GetUploadSignature get secure cloudinary signature for item upload
// @Summary Lấy chữ ký tải ảnh trang phục
// @Description Lấy chữ ký bảo mật từ Cloudinary để client tải trực tiếp ảnh trang phục lên
// @Tags Wardrobe
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult} "Chữ ký và thông tin upload"
// @Router /api/v1/wardrobe-items/upload-signature [get]
func (h *WardrobeHandler) GetUploadSignature(c *gin.Context) error {
	signatureRes, err := h.wardrobeUseCase.GetUploadSignature(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetUploadSignatureSuccess, signatureRes)
	return nil
}

// GetWardrobeItems get all active wardrobe items with lock status
// @Summary Lấy danh sách trang phục
// @Description Lấy toàn bộ danh sách trang phục của người dùng, phân tích và áp dụng trạng thái khóa động nếu hạ cấp gói
// @Tags Wardrobe
// @Produce json
// @Param category_slug query string false "Slug danh mục cần lọc"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục"
// @Router /api/v1/me/wardrobe-items [get]
func (h *WardrobeHandler) GetWardrobeItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var query dto.GetWardrobeItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.GetWardrobeItems(c.Request.Context(), userID, query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetItemsSuccess, response)
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
func (h *WardrobeHandler) GetWardrobeItemByID(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.GetWardrobeItemByID(c.Request.Context(), userID, itemID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeGetItemByIDSuccess, response)
	return nil
}

// CloneWardrobeItem clone an existing wardrobe item
// @Summary Nhân bản trang phục
// @Description Sao chép nhanh một trang phục có sẵn trong tủ đồ (tái sử dụng nguyên bản AI metadata & ảnh), tối đa 5 bản sao
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param id path string true "ID trang phục gốc"
// @Param body body dto.CloneWardrobeItemReq true "Số lượng nhân bản"
// @Success 201 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục được nhân bản"
// @Router /api/v1/wardrobe-items/{id}/clone [post]
func (h *WardrobeHandler) CloneWardrobeItem(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	var input dto.CloneWardrobeItemReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.CloneWardrobeItem(c.Request.Context(), userID, itemID, input.Quantity)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgWardrobeCloneItemSuccess, response)
	return nil
}

// InitClosetFromCatalog initialize user's wardrobe using pre-analyzed global catalog templates
// @Summary Khởi tạo nhanh tủ đồ cá nhân
// @Description Sao chép hàng loạt các trang phục mẫu (Global Catalog) từ hệ thống sang tủ đồ cá nhân của user, không tốn quota AI
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param body body dto.InitClosetFromCatalogReq true "Danh sách ID trang phục mẫu"
// @Success 201 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục cá nhân được tạo"
// @Router /api/v1/wardrobe-items/catalog-init [post]
func (h *WardrobeHandler) InitClosetFromCatalog(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.InitClosetFromCatalogReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.InitClosetFromCatalog(c.Request.Context(), userID, input.CatalogItemIDs)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgWardrobeInitClosetFromCatalogSuccess, response)
	return nil
}

// BatchUploadWardrobeItems upload and process background AI analysis for multiple cropped wardrobe accessories
// @Summary Số hóa trang phục hàng loạt
// @Description Hỗ trợ upload hàng loạt trang phục đã cắt (phụ kiện, áo quần), hệ thống sẽ tạo các ô đồ ở trạng thái Đang xử lý (Processing) và tự động gọi AI phân tích ngầm
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param body body dto.BatchUploadWardrobeItemsReq true "Danh sách ảnh trang phục cắt"
// @Success 201 {object} shared_pres.APIResponse{data=[]dto.WardrobeItemRes} "Danh sách trang phục đang được xử lý ngầm"
// @Router /api/v1/wardrobe-items/batch-upload [post]
func (h *WardrobeHandler) BatchUploadWardrobeItems(c *gin.Context) error {
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

	response, err := h.wardrobeUseCase.BatchUploadWardrobeItems(c.Request.Context(), userID, roleSlug, input)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgWardrobeBatchUploadItemsSuccess, response)
	return nil
}

// SearchWardrobeItems searches wardrobe items using Elasticsearch CQRS
// @Summary Tìm kiếm trang phục có sẵn của hệ thống (Elasticsearch CQRS)
// @Description Hỗ trợ tìm kiếm thông minh đa thuộc tính, fuzzy gõ sai chính tả bằng bộ lọc Elasticsearch tốc độ mili-giây.
// @Tags Wardrobe
// @Produce json
// @Param q query string false "Từ khóa tìm kiếm (Ví dụ: áo sơ mi cotton mát mẻ)"
// @Param category_slug query string false "Slug danh mục cần lọc"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.SearchWardrobeItemRes} "Danh sách trang phục tìm thấy"
// @Router /api/v1/wardrobe-items/search [get]
func (h *WardrobeHandler) SearchWardrobeItems(c *gin.Context) error {
	var query dto.SearchWardrobeItemsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.SearchWardrobeItems(c.Request.Context(), query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeSearchItemsSuccess, response)
	return nil
}

// ManualClassify manual classification fallback for a failed wardrobe item
// @Summary Tự phân loại trang phục thủ công
// @Description Cho phép người dùng tự điền tay thông tin cho trang phục phân tích lỗi, hệ thống dùng Text Embedding cập nhật vector và duyệt vào tủ đồ
// @Tags Wardrobe
// @Accept json
// @Produce json
// @Param id path string true "ID trang phục"
// @Param body body dto.ManualClassifyReq true "Thông tin phân loại thủ công"
// @Success 200 {object} shared_pres.APIResponse{data=dto.WardrobeItemRes} "Chi tiết trang phục sau khi cập nhật"
// @Router /api/v1/wardrobe-items/{id}/manual-classify [put]
func (h *WardrobeHandler) ManualClassify(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	var input dto.ManualClassifyReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.ManualClassify(c.Request.Context(), userID, itemID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgWardrobeManualClassifySuccess, response)
	return nil
}
