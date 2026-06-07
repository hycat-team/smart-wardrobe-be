package handler

import (
	"strings"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	"smart-wardrobe-be/internal/modules/community/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgPostCreateSuccess              = "Tạo bài đăng thành công"
	msgPostGetFeedSuccess             = "Lấy feed thành công"
	msgPostGetUploadSignatureSuccess = "Lấy chữ ký tải media bài đăng thành công"
	msgPostGetDetailSuccess           = "Lấy chi tiết bài đăng thành công"
	msgPostDeleteSuccess              = "Xóa bài đăng thành công"
	msgPostRemoveItemsSuccess         = "Gỡ món khỏi bài đăng thành công"
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

	shared_pres.Created(c, msgPostCreateSuccess, response)
	return nil
}

// GetFeed list community posts feed
// @Summary Lấy danh sách bài đăng cộng đồng
// @Description Lấy feed danh sách bài đăng của cộng đồng sắp xếp theo thứ tự mới nhất/hot nhất
// @Tags Community
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.GetFeedRes} "Lấy feed thành công"
// @Router /api/v1/posts [get]
func (h *PostHandler) GetFeed(c *gin.Context) error {
	var query dto.GetFeedQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	var viewerUserID *uuid.UUID
	if userID, err := contextutils.GetUserId(c); err == nil {
		viewerUserID = &userID
	}

	query.Sort = strings.TrimSpace(strings.ToLower(query.Sort))
	query.PostType = strings.TrimSpace(strings.ToUpper(query.PostType))

	response, err := h.postUC.GetFeed(c.Request.Context(), viewerUserID, query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetFeedSuccess, response)
	return nil
}

// GetUploadSignature get secure cloudinary signature for post media upload
// @Summary Lấy chữ ký tải media bài đăng
// @Description Lấy chữ ký bảo mật từ Cloudinary để client tải trực tiếp media bài đăng cộng đồng lên
// @Tags Community
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult} "Chữ ký và thông tin upload"
// @Router /api/v1/posts/upload-signature [get]
func (h *PostHandler) GetUploadSignature(c *gin.Context) error {
	response, err := h.postUC.GetUploadSignature(c.Request.Context())
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetUploadSignatureSuccess, response)
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
		return communityerrors.ErrInvalidPostIDFormat
	}

	var viewerUserID *uuid.UUID
	if userID, err := contextutils.GetUserId(c); err == nil {
		viewerUserID = &userID
	}

	response, err := h.postUC.GetPostDetail(c.Request.Context(), postID, viewerUserID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetDetailSuccess, response)
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
		return communityerrors.ErrInvalidPostIDFormat
	}

	if err := h.postUC.DeletePost(c.Request.Context(), userID, postID); err != nil {
		return err
	}

	shared_pres.Success(c, msgPostDeleteSuccess, nil)
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
		return communityerrors.ErrInvalidPostIDFormat
	}

	var input dto.RemovePostItemsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.postUC.RemovePostItems(c.Request.Context(), userID, postID, input.PostItemIDs); err != nil {
		return err
	}

	shared_pres.Success(c, msgPostRemoveItemsSuccess, nil)
	return nil
}
