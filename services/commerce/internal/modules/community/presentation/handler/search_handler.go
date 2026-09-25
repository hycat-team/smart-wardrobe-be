package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const msgSearchSuccess = "Tìm kiếm thành công"

type SearchHandler struct {
	searchUC uc_interfaces.ISearchUseCase
}

func NewSearchHandler(searchUC uc_interfaces.ISearchUseCase) *SearchHandler {
	return &SearchHandler{searchUC: searchUC}
}

// Search searches users and posts.
// @Summary Tìm kiếm người dùng và bài đăng
// @Description Tìm theo username/họ tên và nội dung bài đăng (200 ký tự đầu), trả 2 nhóm phân trang độc lập
// @Tags Community
// @Produce json
// @Param q query string false "Từ khóa"
// @Param type query string false "Loại kết quả" Enums(all, users, posts)
// @Param postType query string false "Lọc loại bài đăng" Enums(outfit, media)
// @Success 200 {object} shared_pres.APIResponse{data=dto.SearchRes} "Tìm kiếm thành công"
// @Router /api/v1/search [get]
func (h *SearchHandler) Search(c *gin.Context) error {
	var query dto.SearchQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.searchUC.Search(c.Request.Context(), query, optionalViewer(c))
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgSearchSuccess, res)
	return nil
}
