package handler

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgAdminGetPostsSuccess       = "Lấy danh sách bài đăng thành công"
	msgAdminHidePostSuccess       = "Ẩn bài đăng thành công"
	msgAdminRestorePostSuccess    = "Khôi phục bài đăng thành công"
	msgAdminDeletePostSuccess     = "Xóa bài đăng thành công"
	msgAdminGetCommentsSuccess    = "Lấy danh sách bình luận thành công"
	msgAdminHideCommentSuccess    = "Ẩn bình luận thành công"
	msgAdminRestoreCommentSuccess = "Khôi phục bình luận thành công"
	msgAdminDeleteCommentSuccess  = "Xóa bình luận thành công"
)

type AdminHandler struct {
	adminUC uc_interfaces.IAdminUseCase
}

func NewAdminHandler(adminUC uc_interfaces.IAdminUseCase) *AdminHandler {
	return &AdminHandler{adminUC: adminUC}
}

// ListPosts lists posts for moderation.
// @Summary Danh sách bài đăng (quản trị)
// @Description Lấy danh sách bài đăng để kiểm duyệt, lọc theo từ khóa và trạng thái
// @Tags Community Admin
// @Produce json
// @Param query query dto.GetAdminPostsQueryReq false "Bộ lọc kiểm duyệt"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminPostListRes} "Lấy danh sách bài đăng thành công"
// @Router /api/v1/admin/posts [get]
func (h *AdminHandler) ListPosts(c *gin.Context) error {
	var query dto.GetAdminPostsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.adminUC.ListPosts(c.Request.Context(), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetPostsSuccess, res)
	return nil
}

// HidePost hides a post.
// @Summary Ẩn bài đăng (quản trị)
// @Description Ẩn bài đăng vi phạm (có thể khôi phục); nội dung bị ẩn không hiển thị với người dùng thường
// @Tags Community Admin
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Ẩn bài đăng thành công"
// @Router /api/v1/admin/posts/{postID}/hide [patch]
func (h *AdminHandler) HidePost(c *gin.Context) error {
	return h.moderatePost(c, msgAdminHidePostSuccess, h.adminUC.HidePost)
}

// RestorePost restores a post hidden/deleted by an admin.
// @Summary Khôi phục bài đăng (quản trị)
// @Description Khôi phục bài đăng do admin ẩn/xóa; không khôi phục được bài tác giả tự xóa
// @Tags Community Admin
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Khôi phục bài đăng thành công"
// @Router /api/v1/admin/posts/{postID}/restore [patch]
func (h *AdminHandler) RestorePost(c *gin.Context) error {
	return h.moderatePost(c, msgAdminRestorePostSuccess, h.adminUC.RestorePost)
}

// DeletePost soft-deletes a post.
// @Summary Xóa bài đăng (quản trị)
// @Description Xóa mềm bài đăng vi phạm (có thể khôi phục bởi admin)
// @Tags Community Admin
// @Produce json
// @Param postID path string true "ID bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Xóa bài đăng thành công"
// @Router /api/v1/admin/posts/{postID} [delete]
func (h *AdminHandler) DeletePost(c *gin.Context) error {
	return h.moderatePost(c, msgAdminDeletePostSuccess, h.adminUC.DeletePost)
}

// ListComments lists comments for moderation.
// @Summary Danh sách bình luận (quản trị)
// @Description Lấy danh sách bình luận để kiểm duyệt, lọc theo từ khóa
// @Tags Community Admin
// @Produce json
// @Param query query dto.GetAdminCommentsQueryReq false "Bộ lọc kiểm duyệt"
// @Success 200 {object} shared_pres.APIResponse{data=dto.AdminCommentListRes} "Lấy danh sách bình luận thành công"
// @Router /api/v1/admin/comments [get]
func (h *AdminHandler) ListComments(c *gin.Context) error {
	var query dto.GetAdminCommentsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.adminUC.ListComments(c.Request.Context(), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgAdminGetCommentsSuccess, res)
	return nil
}

// HideComment hides a comment (cascades to replies).
// @Summary Ẩn bình luận (quản trị)
// @Description Ẩn bình luận vi phạm; ẩn bình luận gốc sẽ ẩn luôn các phản hồi con
// @Tags Community Admin
// @Produce json
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Ẩn bình luận thành công"
// @Router /api/v1/admin/comments/{commentID}/hide [patch]
func (h *AdminHandler) HideComment(c *gin.Context) error {
	return h.moderateComment(c, msgAdminHideCommentSuccess, h.adminUC.HideComment)
}

// RestoreComment restores a comment.
// @Summary Khôi phục bình luận (quản trị)
// @Description Khôi phục bình luận do admin ẩn/xóa; không khôi phục được bình luận tác giả tự xóa
// @Tags Community Admin
// @Produce json
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Khôi phục bình luận thành công"
// @Router /api/v1/admin/comments/{commentID}/restore [patch]
func (h *AdminHandler) RestoreComment(c *gin.Context) error {
	return h.moderateComment(c, msgAdminRestoreCommentSuccess, h.adminUC.RestoreComment)
}

// DeleteComment soft-deletes a comment (cascades to replies).
// @Summary Xóa bình luận (quản trị)
// @Description Xóa mềm bình luận vi phạm; xóa bình luận gốc sẽ xóa luôn các phản hồi con
// @Tags Community Admin
// @Produce json
// @Param commentID path string true "ID bình luận"
// @Success 200 {object} shared_pres.APIResponse "Xóa bình luận thành công"
// @Router /api/v1/admin/comments/{commentID} [delete]
func (h *AdminHandler) DeleteComment(c *gin.Context) error {
	return h.moderateComment(c, msgAdminDeleteCommentSuccess, h.adminUC.DeleteComment)
}

type moderationAction func(ctx context.Context, adminID, targetID uuid.UUID) error

func (h *AdminHandler) moderatePost(c *gin.Context, message string, action moderationAction) error {
	adminID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postID, err := validation.GetUUIDParam(c, "postID")
	if err != nil {
		return err
	}
	if err := action(c.Request.Context(), adminID, postID); err != nil {
		return err
	}
	shared_pres.Success(c, message, nil)
	return nil
}

func (h *AdminHandler) moderateComment(c *gin.Context, message string, action moderationAction) error {
	adminID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	commentID, err := validation.GetUUIDParam(c, "commentID")
	if err != nil {
		return err
	}
	if err := action(c.Request.Context(), adminID, commentID); err != nil {
		return err
	}
	shared_pres.Success(c, message, nil)
	return nil
}
