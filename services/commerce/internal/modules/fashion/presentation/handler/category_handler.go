package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	usecase_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
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
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.CategoryAPIRes} "Danh sách danh mục"
// @Router /api/v1/categories [get]
func (h *CategoryHandler) GetCategories(c *gin.Context) error {
	categories, err := h.categoryUseCase.GetAllCategories(c.Request.Context())
	if err != nil {
		return err
	}

	result := make([]*dto.CategoryAPIRes, len(categories))
	for i, cat := range categories {
		result[i] = &dto.CategoryAPIRes{
			ID:        cat.ID,
			Name:      cat.Name,
			Slug:      cat.Slug,
			SortOrder: cat.SortOrder,
		}
	}

	shared_pres.Success(c, msgCategoryGetCategoriesSuccess, result)
	return nil
}

// GetCategoriesAdmin get all categories for admin
// @Summary Lấy danh sách danh mục trang phục (Admin)
// @Description Cho phép Admin lấy danh sách toàn bộ danh mục trang phục trong hệ thống để quản trị
// @Tags Admin Category
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.CategoryAPIRes} "Danh sách danh mục"
// @Router /api/v1/admin/categories [get]
func (h *CategoryHandler) GetCategoriesAdmin(c *gin.Context) error {
	return h.GetCategories(c)
}

// GetCategoryByIDAdmin get category details for admin
// @Summary Lấy chi tiết danh mục trang phục (Admin)
// @Description Cho phép Admin lấy thông tin chi tiết của một danh mục trang phục theo ID
// @Tags Admin Category
// @Produce json
// @Param id path string true "ID danh mục"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CategoryAPIRes} "Thông tin danh mục"
// @Router /api/v1/admin/categories/{id} [get]
func (h *CategoryHandler) GetCategoryByIDAdmin(c *gin.Context) error {
	categoryID, err := validation.GetUUIDParam(c, "id")
	if err != nil {
		return err
	}

	category, err := h.categoryUseCase.GetCategory(c.Request.Context(), categoryID)
	if err != nil {
		return err
	}

	result := &dto.CategoryAPIRes{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		SortOrder: category.SortOrder,
	}

	shared_pres.Success(c, msgCategoryGetCategorySuccess, result)
	return nil
}

// CreateCategoryAdmin create a new category
// @Summary Tạo danh mục trang phục (Admin)
// @Description Cho phép Admin tạo mới một danh mục trang phục trong hệ thống
// @Tags Admin Category
// @Accept json
// @Produce json
// @Param body body dto.CreateCategoryReq true "Thông tin danh mục cần tạo"
// @Success 201 {object} shared_pres.APIResponse{data=dto.CategoryAPIRes} "Thông tin danh mục sau khi tạo"
// @Router /api/v1/admin/categories [post]
func (h *CategoryHandler) CreateCategoryAdmin(c *gin.Context) error {
	var input dto.CreateCategoryReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	category, err := h.categoryUseCase.CreateCategory(c.Request.Context(), dto.CreateCategoryInput{
		Name:      input.Name,
		Slug:      input.Slug,
		SortOrder: sortOrder,
	})
	if err != nil {
		return err
	}

	result := &dto.CategoryAPIRes{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		SortOrder: category.SortOrder,
	}

	shared_pres.Created(c, msgCategoryCreateSuccess, result)
	return nil
}

// UpdateCategoryAdmin update a category
// @Summary Cập nhật danh mục trang phục (Admin)
// @Description Cho phép Admin cập nhật thông tin của một danh mục trang phục theo ID
// @Tags Admin Category
// @Accept json
// @Produce json
// @Param id path string true "ID danh mục"
// @Param body body dto.UpdateCategoryReq true "Thông tin danh mục cần cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CategoryAPIRes} "Thông tin danh mục sau khi cập nhật"
// @Router /api/v1/admin/categories/{id} [put]
func (h *CategoryHandler) UpdateCategoryAdmin(c *gin.Context) error {
	categoryID, err := validation.GetUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var input dto.UpdateCategoryReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	name := input.Name
	slug := input.Slug
	category, err := h.categoryUseCase.UpdateCategory(c.Request.Context(), categoryID, dto.UpdateCategoryInput{
		Name:      &name,
		Slug:      &slug,
		SortOrder: input.SortOrder,
	})
	if err != nil {
		return err
	}

	result := &dto.CategoryAPIRes{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		SortOrder: category.SortOrder,
	}

	shared_pres.Success(c, msgCategoryUpdateSuccess, result)
	return nil
}

// DeleteCategoryAdmin delete a category
// @Summary Xóa danh mục trang phục (Admin)
// @Description Cho phép Admin xóa một danh mục trang phục nếu không còn trang phục người dùng liên kết
// @Tags Admin Category
// @Produce json
// @Param id path string true "ID danh mục"
// @Success 200 {object} shared_pres.APIResponse "Xóa danh mục thành công"
// @Router /api/v1/admin/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategoryAdmin(c *gin.Context) error {
	categoryID, err := validation.GetUUIDParam(c, "id")
	if err != nil {
		return err
	}

	if err := h.categoryUseCase.DeleteCategory(c.Request.Context(), categoryID); err != nil {
		return err
	}

	shared_pres.Success(c, msgCategoryDeleteSuccess, nil)
	return nil
}
