package handler

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgCategoryGetCategoriesSuccess = "Lấy danh sách danh mục thành công"
	msgCategoryGetCategorySuccess   = "Lấy thông tin danh mục thành công"
	msgCategoryCreateSuccess        = "Tạo danh mục thành công"
	msgCategoryUpdateSuccess        = "Cập nhật danh mục thành công"
	msgCategoryDeleteSuccess        = "Xóa danh mục thành công"
)

type CategoryHandler struct {
	categoryUseCase usecase_interfaces.ICategoryUseCase
}

func NewCategoryHandler(uc usecase_interfaces.ICategoryUseCase) *CategoryHandler {
	return &CategoryHandler{
		categoryUseCase: uc,
	}
}

// GetCategories get all categories
// @Summary Lấy tất cả danh mục trang phục
// @Description Lấy danh sách toàn bộ danh mục trang phục trong hệ thống
// @Tags Category
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.CategoryRes} "Danh sách danh mục"
// @Router /api/v1/categories [get]
func (h *CategoryHandler) GetCategories(c *gin.Context) error {
	response, err := h.categoryUseCase.GetCategories(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgCategoryGetCategoriesSuccess, response)
	return nil
}

// GetCategoriesAdmin get all categories for admin
// @Summary Lấy danh sách danh mục trang phục (Admin)
// @Description Cho phép Admin lấy danh sách toàn bộ danh mục trang phục trong hệ thống để quản trị
// @Tags Admin
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.CategoryRes} "Danh sách danh mục"
// @Router /api/v1/admin/categories [get]
func (h *CategoryHandler) GetCategoriesAdmin(c *gin.Context) error {
	return h.GetCategories(c)
}

// GetCategoryByIDAdmin get category details for admin
// @Summary Lấy chi tiết danh mục trang phục (Admin)
// @Description Cho phép Admin lấy thông tin chi tiết của một danh mục trang phục theo ID
// @Tags Admin
// @Produce json
// @Param id path string true "ID danh mục"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CategoryRes} "Thông tin danh mục"
// @Router /api/v1/admin/categories/{id} [get]
func (h *CategoryHandler) GetCategoryByIDAdmin(c *gin.Context) error {
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	response, err := h.categoryUseCase.GetCategoryByID(c.Request.Context(), categoryID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgCategoryGetCategorySuccess, response)
	return nil
}

// CreateCategoryAdmin create a new category
// @Summary Tạo danh mục trang phục (Admin)
// @Description Cho phép Admin tạo mới một danh mục trang phục trong hệ thống
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body dto.CreateCategoryReq true "Thông tin danh mục cần tạo"
// @Success 201 {object} shared_pres.APIResponse{data=dto.CategoryRes} "Thông tin danh mục sau khi tạo"
// @Router /api/v1/admin/categories [post]
func (h *CategoryHandler) CreateCategoryAdmin(c *gin.Context) error {
	var input dto.CreateCategoryReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.categoryUseCase.CreateCategory(c.Request.Context(), input)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgCategoryCreateSuccess, response)
	return nil
}

// UpdateCategoryAdmin update a category
// @Summary Cập nhật danh mục trang phục (Admin)
// @Description Cho phép Admin cập nhật thông tin của một danh mục trang phục theo ID
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "ID danh mục"
// @Param body body dto.UpdateCategoryReq true "Thông tin danh mục cần cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CategoryRes} "Thông tin danh mục sau khi cập nhật"
// @Router /api/v1/admin/categories/{id} [put]
func (h *CategoryHandler) UpdateCategoryAdmin(c *gin.Context) error {
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	var input dto.UpdateCategoryReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.categoryUseCase.UpdateCategory(c.Request.Context(), categoryID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgCategoryUpdateSuccess, response)
	return nil
}

// DeleteCategoryAdmin delete a category
// @Summary Xóa danh mục trang phục (Admin)
// @Description Cho phép Admin xóa một danh mục trang phục nếu không còn trang phục người dùng liên kết
// @Tags Admin
// @Produce json
// @Param id path string true "ID danh mục"
// @Success 200 {object} shared_pres.APIResponse "Xóa danh mục thành công"
// @Router /api/v1/admin/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategoryAdmin(c *gin.Context) error {
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	if err := h.categoryUseCase.DeleteCategory(c.Request.Context(), categoryID); err != nil {
		return err
	}

	shared_pres.Success(c, msgCategoryDeleteSuccess, nil)
	return nil
}
