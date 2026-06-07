package handler

import (
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
	msgInteractionToggleLikeSuccess     = "Cập nhật like thành công"
	msgInteractionAddCommentSuccess     = "Thêm bình luận thành công"
	msgInteractionUpdateCommentSuccess  = "Cập nhật bình luận thành công"
	msgInteractionDeleteCommentSuccess  = "Xóa bình luận thành công"
)

type PostInteractionHandler struct {
	interactionUC usecase_interfaces.IPostInteractionUseCase
}

func NewPostInteractionHandler(interactionUC usecase_interfaces.IPostInteractionUseCase) *PostInteractionHandler {
	return &PostInteractionHandler{interactionUC: interactionUC}
}

// TogglePostLike like or unlike post
// @Summary Thích / Bỏ thích bài đăng
// @Description Like hoặc unlike một bài viết trên cộng đồng bằng cách gửi trạng thái rõ ràng
// @Tags Community
// @Accept json
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Param body body dto.LikePostReq true "Trạng thái thích"
// @Success 200 {object} shared_pres.APIResponse "Cập nhật like thành công"
// @Router /api/v1/posts/{postID}/like [put]
func (h *PostInteractionHandler) TogglePostLike(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return communityerrors.ErrInvalidPostIDFormat
	}

	var input dto.LikePostReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.interactionUC.TogglePostLike(c.Request.Context(), userID, postID, *input.IsLiked); err != nil {
		return err
	}

	shared_pres.Success(c, msgInteractionToggleLikeSuccess, nil)
	return nil
}

// AddComment add comment to post
// @Summary Thêm bình luận vào bài viết
// @Description Tạo bình luận mới dưới bài viết cộng đồng
// @Tags Community
// @Accept json
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Param body body dto.AddCommentReq true "Nội dung bình luận"
// @Success 201 {object} shared_pres.APIResponse{data=dto.CommentRes} "Thêm bình luận thành công"
// @Router /api/v1/posts/{postID}/comments [post]
func (h *PostInteractionHandler) AddComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return communityerrors.ErrInvalidPostIDFormat
	}

	var input dto.AddCommentReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.interactionUC.AddComment(c.Request.Context(), userID, postID, input.Content)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgInteractionAddCommentSuccess, response)
	return nil
}

// UpdateComment update comment of post
// @Summary Cập nhật bình luận của bài viết
// @Description Chỉnh sửa nội dung bình luận thuộc bài viết cộng đồng
// @Tags Community
// @Accept json
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Param commentID path string true "ID bình luận"
// @Param body body dto.UpdateCommentReq true "Nội dung bình luận mới"
// @Success 200 {object} shared_pres.APIResponse{data=dto.CommentRes} "Cập nhật bình luận thành công"
// @Router /api/v1/posts/{postID}/comments/{commentID} [put]
func (h *PostInteractionHandler) UpdateComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return communityerrors.ErrInvalidPostIDFormat
	}
	commentID, err := uuid.Parse(c.Param("commentID"))
	if err != nil {
		return communityerrors.ErrInvalidCommentIDFormat
	}

	var input dto.UpdateCommentReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.interactionUC.UpdateComment(c.Request.Context(), userID, postID, commentID, input.Content)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgInteractionUpdateCommentSuccess, response)
	return nil
}

// DeleteComment delete comment of post
// @Summary Xóa bình luận của bài viết
// @Description Xóa bình luận thuộc bài viết cộng đồng của chính người dùng hiện tại
// @Tags Community
// @Accept json
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Xóa bình luận thành công"
// @Router /api/v1/posts/{postID}/comments/{commentID} [delete]
func (h *PostInteractionHandler) DeleteComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return communityerrors.ErrInvalidPostIDFormat
	}
	commentID, err := uuid.Parse(c.Param("commentID"))
	if err != nil {
		return communityerrors.ErrInvalidCommentIDFormat
	}

	if err := h.interactionUC.DeleteComment(c.Request.Context(), userID, postID, commentID); err != nil {
		return err
	}

	shared_pres.Success(c, msgInteractionDeleteCommentSuccess, nil)
	return nil
}
