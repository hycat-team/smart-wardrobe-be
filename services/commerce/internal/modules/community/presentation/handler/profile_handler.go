package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgGetPublicProfileSuccess = "Lấy trang cá nhân thành công"
	msgGetUserPostsSuccess     = "Lấy bài đăng của người dùng thành công"
)

type ProfileHandler struct {
	profileUC uc_interfaces.IProfileUseCase
	postUC    uc_interfaces.IPostUseCase
}

func NewProfileHandler(profileUC uc_interfaces.IProfileUseCase, postUC uc_interfaces.IPostUseCase) *ProfileHandler {
	return &ProfileHandler{profileUC: profileUC, postUC: postUC}
}

// GetPublicProfile returns a user's public profile.
// @Summary Lấy trang cá nhân công khai
// @Description Lấy thông tin công khai (tên, giới tính, avatar) kèm số follower/following/bài đăng và trạng thái theo dõi
// @Tags Community
// @Produce json
// @Param username path string true "Tên người dùng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PublicProfileRes} "Lấy trang cá nhân thành công"
// @Router /api/v1/users/{username} [get]
func (h *ProfileHandler) GetPublicProfile(c *gin.Context) error {
	res, err := h.profileUC.GetPublicProfile(c.Request.Context(), c.Param("username"), optionalViewer(c))
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgGetPublicProfileSuccess, res)
	return nil
}

// GetUserPosts lists a user's published posts.
// @Summary Lấy bài đăng của người dùng
// @Description Lấy danh sách bài đăng đã công khai của người dùng (mới nhất trước, phân trang)
// @Tags Community
// @Produce json
// @Param username path string true "Tên người dùng"
// @Param query query dto.GetUserPostsQueryReq false "Phân trang"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostListRes} "Lấy bài đăng của người dùng thành công"
// @Router /api/v1/users/{username}/posts [get]
func (h *ProfileHandler) GetUserPosts(c *gin.Context) error {
	var query dto.GetUserPostsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.postUC.GetUserPosts(c.Request.Context(), c.Param("username"), optionalViewer(c), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgGetUserPostsSuccess, res)
	return nil
}
