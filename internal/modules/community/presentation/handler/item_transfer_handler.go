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
	msgTransferMarkSoldSuccess       = "Đánh dấu đã bán thành công"
	msgTransferGetPendingSuccess     = "Lấy danh sách đang chờ nhận thành công"
	msgTransferGetSellerPostsSuccess = "Lấy danh sách bài đăng bàn giao thành công"
	msgTransferAcceptSuccess         = "Nhận món đồ vào tủ thành công"
	msgTransferDeclineSuccess        = "Từ chối nhận món đồ thành công"
)

type ItemTransferHandler struct {
	transferUC usecase_interfaces.IItemTransferUseCase
}

func NewItemTransferHandler(transferUC usecase_interfaces.IItemTransferUseCase) *ItemTransferHandler {
	return &ItemTransferHandler{transferUC: transferUC}
}

// MarkPostItemSold marks a post item as sold to another user.
// @Summary Đánh dấu món đồ đã bán
// @Description Đánh dấu trang phục đã được bán cho một người dùng khác và kích hoạt trạng thái bàn giao
// @Tags Transfers
// @Accept json
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài đăng"
// @Param body body dto.UpdatePostItemsBuyerReq true "Thông tin người mua"
// @Success 200 {object} shared_pres.APIResponse "Đánh dấu đã bán thành công"
// @Router /api/v1/transfers/items/{postItemID}/mark-sold [post]
func (h *ItemTransferHandler) MarkPostItemSold(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return communityerrors.ErrInvalidPostItemIDFormat
	}

	var input dto.UpdatePostItemsBuyerReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.transferUC.MarkPostItemSold(c.Request.Context(), userID, postItemID, input.BuyerUserID); err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferMarkSoldSuccess, nil)
	return nil
}

// GetPendingTransfers gets pending transfer items of the current buyer.
// @Summary Danh sách trang phục đang chờ nhận bàn giao
// @Description Lấy danh sách các trang phục do người khác đánh dấu bán cho bạn đang chờ xác nhận
// @Tags Transfers
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.PendingTransferRes} "Lấy danh sách đang chờ nhận thành công"
// @Router /api/v1/transfers/me/pending [get]
func (h *ItemTransferHandler) GetPendingTransfers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.transferUC.GetPendingTransfers(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferGetPendingSuccess, response)
	return nil
}

// GetSellerTransferPosts gets transfer-related posts of the current seller.
// @Summary Danh sách bài đăng bàn giao của người bán
// @Description Lấy danh sách các bài đăng của người bán có món đồ đang chờ, được chấp nhận, bị từ chối hoặc đã bán trong luồng bàn giao
// @Tags Transfers
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.SellerTransferPostRes} "Lấy danh sách bài đăng bàn giao thành công"
// @Router /api/v1/transfers/me/posts [get]
func (h *ItemTransferHandler) GetSellerTransferPosts(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.transferUC.GetSellerTransferPosts(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferGetSellerPostsSuccess, response)
	return nil
}

// AcceptTransfer accepts item transfer into the buyer wardrobe.
// @Summary Chấp nhận nhận bàn giao trang phục
// @Description Đồng ý nhận trang phục đã mua về tủ đồ cá nhân
// @Tags Transfers
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostItemRes} "Nhận món đồ vào tủ thành công"
// @Router /api/v1/transfers/items/{postItemID}/accept [post]
func (h *ItemTransferHandler) AcceptTransfer(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return communityerrors.ErrInvalidPostItemIDFormat
	}

	response, err := h.transferUC.AcceptTransfer(c.Request.Context(), userID, postItemID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferAcceptSuccess, response)
	return nil
}

// DeclineTransfer declines item transfer.
// @Summary Từ chối nhận bàn giao trang phục
// @Description Từ chối nhận bàn giao trang phục mua từ bài đăng cộng đồng
// @Tags Transfers
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài đăng"
// @Success 200 {object} shared_pres.APIResponse "Từ chối nhận món đồ thành công"
// @Router /api/v1/transfers/items/{postItemID}/decline [post]
func (h *ItemTransferHandler) DeclineTransfer(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return communityerrors.ErrInvalidPostItemIDFormat
	}

	if err := h.transferUC.DeclineTransfer(c.Request.Context(), userID, postItemID); err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferDeclineSuccess, nil)
	return nil
}
