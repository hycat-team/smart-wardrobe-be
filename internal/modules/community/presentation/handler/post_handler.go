package handler

import (
	"strings"

	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	_ "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/community/posttype"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgPostCreateSuccess             = "Tạo bài đăng thành công"
	msgPostUpdateSuccess             = "Cập nhật bài đăng thành công"
	msgPostGetFeedSuccess            = "Lấy feed thành công"
	msgPostGetUploadSignatureSuccess = "Lấy chữ ký tải media bài đăng thành công"
	msgPostGetDetailSuccess          = "Lấy chi tiết bài đăng thành công"
	msgPostGetCommentsSuccess        = "Lấy danh sách bình luận thành công"
	msgPostGetRepliesSuccess         = "Lấy danh sách phản hồi thành công"
	msgPostGetLikesSuccess           = "Lấy danh sách người thích thành công"
	msgPostDeleteSuccess             = "Xóa bài đăng thành công"
	msgPostRemoveItemsSuccess        = "Gỡ món khỏi bài đăng thành công"
)

type PostHandler struct {
	postUC usecase_interfaces.IUserPostUseCase
}

func NewPostHandler(postUC usecase_interfaces.IUserPostUseCase) *PostHandler {
	return &PostHandler{postUC: postUC}
}

// CreatePost create a new community post
// @Summary Tạo bài đăng cộng đồng mới
// @Description Đăng bài bán đồ hoặc khoe outfit lên bảng tin cộng đồng
// @Tags Community
// @Accept json
// @Produce json
// @Param body body community_dto.CreatePostReq true "Nội dung bài đăng"
// @Success 201 {object} shared_pres.APIResponse{data=community_dto.PostRes} "Tạo bài đăng thành công"
// @Router /api/v1/posts [post]
func (h *PostHandler) CreatePost(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input community_dto.CreatePostReq
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

// UpdatePost update a community post
// @Summary Cập nhật bài đăng cộng đồng
// @Description Cập nhật nội dung, media và danh sách món đồ của bài đăng
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body community_dto.UpdatePostReq true "Nội dung bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=community_dto.PostRes} "Cập nhật bài đăng thành công"
// @Router /api/v1/posts/{postPublicID} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input community_dto.UpdatePostReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.postUC.UpdatePost(c.Request.Context(), userID, c.Param("postPublicID"), input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostUpdateSuccess, response)
	return nil
}

// GetFeed list community posts feed
// @Summary Lấy danh sách bài đăng cộng đồng
// @Description Lấy feed danh sách bài đăng của cộng đồng sắp xếp theo thứ tự mới nhất hoặc hot nhất
// @Tags Community
// @Produce json
// @Param query query community_dto.GetFeedQueryReq false "Bộ lọc feed bài đăng cộng đồng"
// @Success 200 {object} shared_pres.APIResponse{data=community_dto.GetFeedRes} "Lấy feed thành công"
// @Router /api/v1/posts [get]
func (h *PostHandler) GetFeed(c *gin.Context) error {
	var query community_dto.GetFeedQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	var viewerUserID *uuid.UUID
	if userID, err := contextutils.GetUserId(c); err == nil {
		viewerUserID = &userID
	}

	query.Sort = strings.TrimSpace(strings.ToLower(query.Sort))
	query.PostType = posttype.PostType(strings.TrimSpace(strings.ToLower(string(query.PostType))))

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

// GetPostDetail get post detail
// @Summary Lấy chi tiết bài đăng
// @Description Lấy thông tin chi tiết của một bài đăng cụ thể không bao gồm danh sách bình luận và danh sách người thích
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=community_dto.PostRes} "Lấy chi tiết bài đăng thành công"
// @Router /api/v1/posts/{postPublicID} [get]
func (h *PostHandler) GetPostDetail(c *gin.Context) error {
	var viewerUserID *uuid.UUID
	if userID, err := contextutils.GetUserId(c); err == nil {
		viewerUserID = &userID
	}

	response, err := h.postUC.GetPostDetail(c.Request.Context(), c.Param("postPublicID"), viewerUserID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetDetailSuccess, response)
	return nil
}

// GetPostComments get top level comments of a post
// @Summary Lấy bình luận cấp đầu của bài đăng
// @Description Lấy danh sách bình luận F0 của bài đăng cụ thể
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=[]community_dto.CommentRes} "Lấy danh sách bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments [get]
func (h *PostHandler) GetPostComments(c *gin.Context) error {
	response, err := h.postUC.GetPostComments(c.Request.Context(), c.Param("postPublicID"))
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetCommentsSuccess, response)
	return nil
}

// GetCommentReplies get replies of a top-level comment
// @Summary Lấy phản hồi của một bình luận cấp đầu
// @Description Lấy danh sách bình luận F1 của một bình luận F0 cụ thể
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param commentID path string true "ID bình luận cấp đầu"
// @Success 200 {object} shared_pres.APIResponse{data=[]community_dto.CommentRes} "Lấy danh sách phản hồi thành công"
// @Router /api/v1/posts/{postPublicID}/comments/{commentID}/replies [get]
func (h *PostHandler) GetCommentReplies(c *gin.Context) error {
	commentID, err := uuid.Parse(c.Param("commentID"))
	if err != nil {
		return communityerrors.ErrInvalidCommentIDFormat()
	}

	response, err := h.postUC.GetCommentReplies(c.Request.Context(), c.Param("postPublicID"), commentID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetRepliesSuccess, response)
	return nil
}

// GetPostLikes get users who liked a post
// @Summary Lấy danh sách người thích bài đăng
// @Description Lấy danh sách người dùng đã thích bài đăng để hiển thị kiểu Facebook
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=[]community_dto.PostLikeUserRes} "Lấy danh sách người thích thành công"
// @Router /api/v1/posts/{postPublicID}/likes [get]
func (h *PostHandler) GetPostLikes(c *gin.Context) error {
	response, err := h.postUC.GetPostLikes(c.Request.Context(), c.Param("postPublicID"))
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgPostGetLikesSuccess, response)
	return nil
}

// DeletePost delete post
// @Summary Xóa bài đăng
// @Description Xóa bài đăng của chính người dùng hiện tại
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Xóa bài đăng thành công"
// @Router /api/v1/posts/{postPublicID} [delete]
func (h *PostHandler) DeletePost(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	if err := h.postUC.DeletePost(c.Request.Context(), userID, c.Param("postPublicID")); err != nil {
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
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body community_dto.RemovePostItemsReq true "Danh sách ID các món đồ cần gỡ"
// @Success 200 {object} shared_pres.APIResponse "Gỡ món đồ thành công"
// @Router /api/v1/posts/{postPublicID}/items [delete]
func (h *PostHandler) RemovePostItems(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input community_dto.RemovePostItemsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.postUC.RemovePostItems(c.Request.Context(), userID, c.Param("postPublicID"), input.PostItemIDs); err != nil {
		return err
	}

	shared_pres.Success(c, msgPostRemoveItemsSuccess, nil)
	return nil
}
