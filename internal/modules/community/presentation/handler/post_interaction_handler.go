package handler

import (
	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgInteractionToggleLikeSuccess    = "Cập nhật like thành công"
	msgInteractionAddCommentSuccess    = "Thêm bình luận thành công"
	msgInteractionUpdateCommentSuccess = "Cập nhật bình luận thành công"
	msgInteractionDeleteCommentSuccess = "Xóa bình luận thành công"
)

type PostInteractionHandler struct {
	interactionUC usecase_interfaces.IUserPostInteractionUseCase
}

func NewPostInteractionHandler(interactionUC usecase_interfaces.IUserPostInteractionUseCase) *PostInteractionHandler {
	return &PostInteractionHandler{interactionUC: interactionUC}
}

// TogglePostLike like or unlike post
// @Summary Thích / Bỏ thích bài đăng
// @Description Like hoặc unlike một bài viết trên cộng đồng bằng cách gửi trạng thái rõ ràng
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body community_dto.LikePostReq true "Trạng thái thích"
// @Success 200 {object} shared_pres.APIResponse "Cập nhật like thành công"
// @Router /api/v1/posts/{postPublicID}/like [put]
func (h *PostInteractionHandler) TogglePostLike(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input community_dto.LikePostReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.interactionUC.TogglePostLike(c.Request.Context(), userID, c.Param("postPublicID"), *input.IsLiked); err != nil {
		return err
	}

	shared_pres.Success(c, msgInteractionToggleLikeSuccess, nil)
	return nil
}

// AddComment add comment to post
// @Summary Thêm bình luận vào bài viết
// @Description Tạo bình luận mới hoặc phản hồi trực tiếp vào bình luận cấp đầu của bài viết cộng đồng
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body community_dto.AddCommentReq true "Nội dung bình luận"
// @Success 201 {object} shared_pres.APIResponse{data=community_dto.CommentRes} "Thêm bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments [post]
func (h *PostInteractionHandler) AddComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input community_dto.AddCommentReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.interactionUC.AddComment(c.Request.Context(), userID, c.Param("postPublicID"), input)
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
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param commentID path string true "ID bình luận"
// @Param body body community_dto.UpdateCommentReq true "Nội dung bình luận mới"
// @Success 200 {object} shared_pres.APIResponse{data=community_dto.CommentRes} "Cập nhật bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments/{commentID} [put]
func (h *PostInteractionHandler) UpdateComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	commentID, err := uuid.Parse(c.Param("commentID"))
	if err != nil {
		return communityerrors.ErrInvalidCommentIDFormat()
	}

	var input community_dto.UpdateCommentReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.interactionUC.UpdateComment(c.Request.Context(), userID, c.Param("postPublicID"), commentID, input.Content)
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
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Xóa bình luận thành công"
// @Router /api/v1/posts/{postPublicID}/comments/{commentID} [delete]
func (h *PostInteractionHandler) DeleteComment(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	commentID, err := uuid.Parse(c.Param("commentID"))
	if err != nil {
		return communityerrors.ErrInvalidCommentIDFormat()
	}

	if err := h.interactionUC.DeleteComment(c.Request.Context(), userID, c.Param("postPublicID"), commentID); err != nil {
		return err
	}

	shared_pres.Success(c, msgInteractionDeleteCommentSuccess, nil)
	return nil
}
