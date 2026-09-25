package handler

import (
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	msgPostCreateSuccess             = "Tạo bài đăng thành công"
	msgPostUpdateSuccess             = "Cập nhật bài đăng thành công"
	msgPostDeleteSuccess             = "Xóa bài đăng thành công"
	msgPostGetDetailSuccess          = "Lấy chi tiết bài đăng thành công"
	msgPostGetFeedSuccess            = "Lấy bảng tin thành công"
	msgPostGetUploadSignatureSuccess = "Lấy chữ ký tải media bài đăng thành công"
)

type PostHandler struct {
	postUC uc_interfaces.IPostUseCase
}

func NewPostHandler(postUC uc_interfaces.IPostUseCase) *PostHandler {
	return &PostHandler{postUC: postUC}
}

// CreatePost creates a community post.
// @Summary Tạo bài đăng cộng đồng
// @Description Tạo bài đăng loại outfit (gắn outfit thuộc sở hữu) hoặc media (ảnh/video/không media)
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
	res, err := h.postUC.CreatePost(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgPostCreateSuccess, res)
	return nil
}

// UpdatePost updates a post owned by the caller.
// @Summary Cập nhật bài đăng cộng đồng
// @Description Cập nhật tiêu đề, nội dung, outfit hoặc media của bài đăng thuộc sở hữu (không đổi loại bài)
// @Tags Community
// @Accept json
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Param body body dto.UpdatePostReq true "Nội dung cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostRes} "Cập nhật bài đăng thành công"
// @Router /api/v1/posts/{postPublicID} [put]
func (h *PostHandler) UpdatePost(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.UpdatePostReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.postUC.UpdatePost(c.Request.Context(), userID, c.Param("postPublicID"), input)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgPostUpdateSuccess, res)
	return nil
}

// DeletePost deletes a post owned by the caller.
// @Summary Xóa bài đăng cộng đồng
// @Description Xóa mềm bài đăng của chính người dùng hiện tại (không thể khôi phục)
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

// GetPostDetail returns a post detail.
// @Summary Lấy chi tiết bài đăng
// @Description Lấy chi tiết một bài đăng công khai (tác giả vẫn xem được bài đang bị ẩn của mình)
// @Tags Community
// @Produce json
// @Param postPublicID path string true "Mã công khai bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostRes} "Lấy chi tiết bài đăng thành công"
// @Router /api/v1/posts/{postPublicID} [get]
func (h *PostHandler) GetPostDetail(c *gin.Context) error {
	viewer := optionalViewer(c)
	res, err := h.postUC.GetPostDetail(c.Request.Context(), c.Param("postPublicID"), viewer)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgPostGetDetailSuccess, res)
	return nil
}

// GetFeed lists the community feed.
// @Summary Lấy bảng tin cộng đồng
// @Description Lấy bảng tin explore/following, sắp xếp hot/latest, hỗ trợ lọc loại bài và theo username
// @Tags Community
// @Produce json
// @Param query query dto.GetFeedQueryReq false "Bộ lọc bảng tin"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostListRes} "Lấy bảng tin thành công"
// @Router /api/v1/posts [get]
func (h *PostHandler) GetFeed(c *gin.Context) error {
	var query dto.GetFeedQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	query.Type = strings.TrimSpace(query.Type)
	query.Sort = strings.TrimSpace(query.Sort)
	query.PostType = strings.TrimSpace(query.PostType)
	query.Username = strings.TrimSpace(query.Username)

	res, err := h.postUC.GetFeed(c.Request.Context(), optionalViewer(c), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgPostGetFeedSuccess, res)
	return nil
}

// GetUploadSignature returns a Cloudinary upload signature for post media.
// @Summary Lấy chữ ký tải media bài đăng
// @Description Lấy chữ ký Cloudinary để client tải trực tiếp ảnh hoặc video lên
// @Tags Community
// @Produce json
// @Param resourceType query string false "Loại media (image|video)" Enums(image, video)
// @Success 200 {object} shared_pres.APIResponse{data=dto.UploadSignatureResult} "Lấy chữ ký thành công"
// @Router /api/v1/posts/upload-signature [get]
func (h *PostHandler) GetUploadSignature(c *gin.Context) error {
	res, err := h.postUC.GetUploadSignature(c.Request.Context(), c.Query("resourceType"))
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgPostGetUploadSignatureSuccess, res)
	return nil
}

func optionalViewer(c *gin.Context) *uuid.UUID {
	id := contextutils.GetUserIdOptional(c)
	if id == uuid.Nil {
		return nil
	}
	return &id
}
