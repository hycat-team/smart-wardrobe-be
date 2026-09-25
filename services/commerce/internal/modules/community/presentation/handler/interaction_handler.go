package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgToggleLikeSuccess    = "Cập nhật like thành công"
	msgAddCommentSuccess    = "Thêm bình luận thành công"
	msgUpdateCommentSuccess = "Cập nhật bình luận thành công"
	msgDeleteCommentSuccess = "Xóa bình luận thành công"
	msgGetCommentsSuccess   = "Lấy danh sách bình luận thành công"
	msgGetRepliesSuccess    = "Lấy danh sách phản hồi thành công"
	msgGetLikesSuccess      = "Lấy danh sách người thích thành công"
)

type InteractionHandler struct {
	interactionUC uc_interfaces.IPostInteractionUseCase
}

func NewInteractionHandler(interactionUC uc_interfaces.IPostInteractionUseCase) *InteractionHandler {
	return &InteractionHandler{interactionUC: interactionUC}
}

// TogglePostLike likes or unlikes a post.
// @Summary Thích / Bỏ thích bài đăng
// @Description Bật/tắt like bài đăng (idempotent), bộ đếm like cập nhật tương ứng
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body dto.LikePostReq true "Trạng thái thích"
// @Success 200 {object} shared_pres.APIResponse "Cập nhật like thành công"
// @Router /api/v1/posts/{postPublicID}/like [put]
func (h *InteractionHandler) TogglePostLike(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.LikePostReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	if err := h.interactionUC.TogglePostLike(c.Request.Context(), userID, c.Param("postPublicID"), *input.IsLiked); err != nil {
		return err
	}
	shared_pres.Success(c, msgToggleLikeSuccess, nil)
	return nil
}

// AddComment adds a comment or reply.
// @Summary Thêm bình luận
// @Description Thêm bình luận gốc hoặc phản hồi; phản hồi vào cấp 2 sẽ tự động gắn lên bình luận gốc
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body dto.AddCommentReq true "Nội dung bình luận"
// @Success 201 {object} shared_pres.APIResponse{data=dto.CommentRes} "Thêm bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments [post]
func (h *InteractionHandler) AddComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.AddCommentReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.interactionUC.AddComment(c.Request.Context(), userID, c.Param("postPublicID"), input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgAddCommentSuccess, res)
	return nil
}

// UpdateComment updates a comment owned by the caller.
// @Summary Cập nhật bình luận
// @Description Cập nhật nội dung bình luận của chính người dùng hiện tại
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param commentID path string true "ID bình luận"
// @Param body body dto.UpdateCommentReq true "Nội dung bình luận mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CommentRes} "Cập nhật bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments/{commentID} [put]
func (h *InteractionHandler) UpdateComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	commentID, err := validation.GetUUIDParam(c, "commentID")
	if err != nil {
		return err
	}
	var input dto.UpdateCommentReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.interactionUC.UpdateComment(c.Request.Context(), userID, c.Param("postPublicID"), commentID, input.Content)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgUpdateCommentSuccess, res)
	return nil
}

// DeleteComment deletes a comment owned by the caller.
// @Summary Xóa bình luận
// @Description Xóa mềm bình luận của chính người dùng hiện tại; bình luận gốc còn phản hồi sẽ hiển thị dấu đã xóa
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Xóa bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments/{commentID} [delete]
func (h *InteractionHandler) DeleteComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	commentID, err := validation.GetUUIDParam(c, "commentID")
	if err != nil {
		return err
	}
	if err := h.interactionUC.DeleteComment(c.Request.Context(), userID, c.Param("postPublicID"), commentID); err != nil {
		return err
	}
	shared_pres.Success(c, msgDeleteCommentSuccess, nil)
	return nil
}

// GetPostComments lists root comments of a post.
// @Summary Lấy bình luận gốc của bài đăng
// @Description Lấy danh sách bình luận gốc (mới nhất trước) kèm số lượng phản hồi
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.CommentRes} "Lấy danh sách bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments [get]
func (h *InteractionHandler) GetPostComments(c *gin.Context) error {
	res, err := h.interactionUC.GetPostComments(c.Request.Context(), c.Param("postPublicID"))
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgGetCommentsSuccess, res)
	return nil
}

// GetCommentReplies lists replies of a root comment.
// @Summary Lấy phản hồi của bình luận gốc
// @Description Lấy danh sách phản hồi (cũ nhất trước) của một bình luận gốc
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param commentID path string true "ID bình luận gốc"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.CommentRes} "Lấy danh sách phản hồi thành công"
// @Router /api/v1/posts/{postPublicID}/comments/{commentID}/replies [get]
func (h *InteractionHandler) GetCommentReplies(c *gin.Context) error {
	commentID, err := validation.GetUUIDParam(c, "commentID")
	if err != nil {
		return err
	}
	res, err := h.interactionUC.GetCommentReplies(c.Request.Context(), c.Param("postPublicID"), commentID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgGetRepliesSuccess, res)
	return nil
}

// GetPostLikes lists users who liked a post.
// @Summary Lấy danh sách người thích bài đăng
// @Description Lấy danh sách người dùng đã thích bài đăng (phân trang)
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param query query dto.GetLikesQueryReq false "Phân trang"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostLikeUserListRes} "Lấy danh sách người thích thành công"
// @Router /api/v1/posts/{postPublicID}/likes [get]
func (h *InteractionHandler) GetPostLikes(c *gin.Context) error {
	var query dto.GetLikesQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.interactionUC.GetPostLikes(c.Request.Context(), c.Param("postPublicID"), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgGetLikesSuccess, res)
	return nil
}
