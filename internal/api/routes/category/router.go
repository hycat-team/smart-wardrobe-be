package category

import (
	category_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type CategoryRouter struct {
	categoryHandler *category_handler.CategoryHandler
}

func NewRouter(h *category_handler.CategoryHandler) *CategoryRouter {
	return &CategoryRouter{
		categoryHandler: h,
	}
}

func (r *CategoryRouter) Init(group *gin.RouterGroup) {
	categoryApi := group.Group("/categories")
	{
		categoryApi.GET("", shared_pres.WrapHandler(r.categoryHandler.GetCategories))
	}
}
