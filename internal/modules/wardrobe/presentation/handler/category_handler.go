package handler

import (
	_ "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	_ "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
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

	shared_pres.Success(c, "Lấy danh sách danh mục thành công", response)
	return nil
}
