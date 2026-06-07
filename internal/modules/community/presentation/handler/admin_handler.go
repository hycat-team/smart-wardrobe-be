package handler

import (
	"smart-wardrobe-be/internal/modules/community/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgAdminDeletePostSuccess    = "Xóa bài đăng thành công"
	msgAdminDeleteCommentSuccess = "Xóa bình luận thành công"
	msgAdminHidePostItemSuccess   = "Ẩn listing thành công"
)

type AdminHandler struct {
	postUC        usecase_interfaces.IPostUseCase
	interactionUC usecase_interfaces.IPostInteractionUseCase
}

func NewAdminHandler(
	postUC usecase_interfaces.IPostUseCase,
	interactionUC usecase_interfaces.IPostInteractionUseCase,
) *AdminHandler {
	return &AdminHandler{
		postUC:        postUC,
		interactionUC: interactionUC,
	}
}

// DeletePost xóa bài đăng community bằng quyền admin
// @Summary Xóa bài đăng community
// @Description Cho phép admin xóa bài đăng community vi phạm
// @Tags Admin
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Xóa bài đăng thành công"
// @Router /api/v1/admin/community/posts/{postID} [delete]
func (h *AdminHandler) DeletePost(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(c.Param("postID"))
	if err != nil {
		return communityerrors.ErrInvalidPostIDFormat
	}

	if err := h.postUC.AdminDeletePost(c.Request.Context(), adminUserID, postID); err != nil {
		return err
	}

	shared_pres.Success(c, msgAdminDeletePostSuccess, nil)
	return nil
}

// DeleteComment xóa bình luận community bằng quyền admin
// @Summary Xóa bình luận community
// @Description Cho phép admin xóa bình luận community vi phạm
// @Tags Admin
// @Produce json
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Xóa bình luận thành công"
// @Router /api/v1/admin/community/comments/{commentID} [delete]
func (h *AdminHandler) DeleteComment(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	commentID, err := uuid.Parse(c.Param("commentID"))
	if err != nil {
		return communityerrors.ErrInvalidCommentIDFormat
	}

	if err := h.interactionUC.AdminDeleteComment(c.Request.Context(), adminUserID, commentID); err != nil {
		return err
	}

	shared_pres.Success(c, msgAdminDeleteCommentSuccess, nil)
	return nil
}

// HidePostItem ẩn listing community bằng quyền admin
// @Summary Ẩn listing community
// @Description Cho phép admin ẩn listing hoặc post item vi phạm khỏi community
// @Tags Admin
// @Produce json
// @Param postItemID path string true "ID post item"
// @Success 200 {object} shared_pres.APIResponse "Ẩn listing thành công"
// @Router /api/v1/admin/community/post-items/{postItemID} [delete]
func (h *AdminHandler) HidePostItem(c *gin.Context) error {
	adminUserID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return communityerrors.ErrInvalidPostItemIDFormat
	}

	if err := h.postUC.AdminHidePostItem(c.Request.Context(), adminUserID, postItemID); err != nil {
		return err
	}

	shared_pres.Success(c, msgAdminHidePostItemSuccess, nil)
	return nil
}
