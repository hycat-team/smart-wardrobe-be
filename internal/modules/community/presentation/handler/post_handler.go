package handler

import (
	"smart-wardrobe-be/internal/modules/community/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PostHandler struct {
	postUC usecase_interfaces.IPostUseCase
}

func NewPostHandler(postUC usecase_interfaces.IPostUseCase) *PostHandler {
	return &PostHandler{postUC: postUC}
}

// CreatePost create a new community post
// @Summary Tạo bài đăng cộng đồng mới
// @Description Đăng bài bán đồ hoặc khoe outfit lên bảng tin cộng đồng
// @Tags Community
// @Accept json
// @Produce json
// @Param body body dto.CreatePostReq true "Nội dung bài đăng"
// @Success 201 {object} shared_pres.APIResponse{data=dto.PostRes} "Tạo bài đăng thành công"
// @Router /api/v1/posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.CreatePostReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.postUC.CreatePost(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Created(c, "Tạo bài đăng thành công", response)
	return nil
}

// GetFeed list community posts feed
// @Summary Lấy danh sách bài đăng cộng đồng
// @Description Lấy feed danh sách bài đăng của cộng đồng sắp xếp theo thứ tự mới nhất/hot nhất
// @Tags Community
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.PostRes} "Lấy feed thành công"
// @Router /api/v1/posts [get]
func (h *PostHandler) GetFeed(c *gin.Context) error {
	response, err := h.postUC.GetFeed(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy feed thành công", response)
	return nil
}

// GetPostDetail get post detail with comments and likes
// @Summary Lấy chi tiết bài đăng
// @Description Lấy thông tin chi tiết của một bài đăng cụ thể kèm danh sách bình luận
// @Tags Community
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostRes} "Lấy chi tiết bài đăng thành công"
// @Router /api/v1/posts/{postID} [get]
func (h *PostHandler) GetPostDetail(c *gin.Context) error {
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return err
	}

	response, err := h.postUC.GetPostDetail(c.Request.Context(), postID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy chi tiết bài đăng thành công", response)
	return nil
}

// DeletePost delete post
// @Summary Xóa bài đăng
// @Description Xóa bài đăng của chính người dùng hiện tại
// @Tags Community
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Xóa bài đăng thành công"
// @Router /api/v1/posts/{postID} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return err
	}

	if err := h.postUC.DeletePost(c.Request.Context(), userID, postID); err != nil {
		return err
	}

	shared_pres.Success(c, "Xóa bài đăng thành công", nil)
	return nil
}

// RemovePostItems remove specific wardrobe items from post
// @Summary Gỡ món đồ khỏi bài đăng
// @Description Gỡ một hoặc nhiều món đồ ra khỏi danh sách bán trong bài đăng
// @Tags Community
// @Accept json
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Param body body dto.RemovePostItemsReq true "Danh sách ID các món đồ cần gỡ"
// @Success 200 {object} shared_pres.APIResponse "Gỡ món đồ thành công"
// @Router /api/v1/posts/{postID}/items [delete]
func (h *PostHandler) RemovePostItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return err
	}

	var input dto.RemovePostItemsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.postUC.RemovePostItems(c.Request.Context(), userID, postID, input.PostItemIDs); err != nil {
		return err
	}

	shared_pres.Success(c, "Gỡ món khỏi bài đăng thành công", nil)
	return nil
}
