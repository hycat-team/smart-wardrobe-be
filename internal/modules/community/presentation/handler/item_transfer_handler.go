package handler

import (
	communitydto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	_ "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
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
	msgTransferRequestSuccess        = "Gửi yêu cầu xin mua thành công"
	msgTransferGetRequestsSuccess    = "Lấy danh sách người xin mua thành công"
)

type ItemTransferHandler struct {
	transferUC usecase_interfaces.IItemTransferUseCase
}

func NewItemTransferHandler(transferUC usecase_interfaces.IItemTransferUseCase) *ItemTransferHandler {
	return &ItemTransferHandler{transferUC: transferUC}
}

// MarkPostItemsSold marks multiple post items as sold to a buyer.
// @Summary Đánh dấu các món đồ đã bán (Bulk)
// @Description Đánh dấu danh sách các trang phục đã được bán cho một người dùng khác (qua buyerId) và kích hoạt trạng thái bàn giao
// @Tags Transfers
// @Accept json
// @Produce json
// @Param body body communitydto.MarkPostItemsSoldReq true "Thông tin người mua và danh sách sản phẩm"
// @Success 200 {object} shared_pres.APIResponse "Đánh dấu đã bán thành công"
// @Router /api/v1/transfers/mark-sold [post]
func (h *ItemTransferHandler) MarkPostItemsSold(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input communitydto.MarkPostItemsSoldReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.transferUC.MarkPostItemsSold(c.Request.Context(), userID, input.PostItemIDs, input.BuyerID); err != nil {
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
// @Success 200 {object} shared_pres.APIResponse{data=[]communitydto.PendingTransferRes} "Lấy danh sách đang chờ nhận thành công"
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
// @Success 200 {object} shared_pres.APIResponse{data=[]communitydto.SellerTransferPostRes} "Lấy danh sách bài đăng bàn giao thành công"
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

// AcceptTransfers accepts item transfers into the buyer wardrobe (Bulk).
// @Summary Chấp nhận nhận bàn giao danh sách trang phục
// @Description Đồng ý nhận danh sách trang phục đã mua về tủ đồ cá nhân
// @Tags Transfers
// @Accept json
// @Produce json
// @Param body body communitydto.AcceptTransfersReq true "Danh sách sản phẩm bàn giao"
// @Success 200 {object} shared_pres.APIResponse{data=[]communitydto.WardrobeItemRes} "Nhận món đồ vào tủ thành công"
// @Router /api/v1/transfers/accept [post]
func (h *ItemTransferHandler) AcceptTransfers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input communitydto.AcceptTransfersReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.transferUC.AcceptTransfers(c.Request.Context(), userID, input.PostItemIDs)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferAcceptSuccess, response)
	return nil
}

// DeclineTransfers declines item transfers (Bulk).
// @Summary Từ chối nhận bàn giao danh sách trang phục
// @Description Từ chối nhận bàn giao danh sách trang phục mua từ bài đăng cộng đồng
// @Tags Transfers
// @Accept json
// @Produce json
// @Param body body communitydto.AcceptTransfersReq true "Danh sách sản phẩm bàn giao"
// @Success 200 {object} shared_pres.APIResponse "Từ chối nhận món đồ thành công"
// @Router /api/v1/transfers/decline [post]
func (h *ItemTransferHandler) DeclineTransfers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input communitydto.AcceptTransfersReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.transferUC.DeclineTransfers(c.Request.Context(), userID, input.PostItemIDs); err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferDeclineSuccess, nil)
	return nil
}

// CreateTransferRequests creates requests to buy items.
// @Summary Gửi yêu cầu xin mua trang phục (Bulk)
// @Description Người mua đăng ký muốn mua một hoặc nhiều món đồ trong bài đăng của người bán
// @Tags Transfers
// @Accept json
// @Produce json
// @Param body body communitydto.CreateTransferRequestsReq true "Danh sách sản phẩm xin mua"
// @Success 200 {object} shared_pres.APIResponse "Gửi yêu cầu xin mua thành công"
// @Router /api/v1/transfers/requests [post]
func (h *ItemTransferHandler) CreateTransferRequests(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input communitydto.CreateTransferRequestsReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.transferUC.CreateTransferRequests(c.Request.Context(), userID, input.PostItemIDs); err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferRequestSuccess, nil)
	return nil
}

// GetTransferRequestsForSeller gets list of requests for a post item.
// @Summary Lấy danh sách người xin mua của một sản phẩm
// @Description Người bán xem danh sách những người mua đã gửi yêu cầu xin mua cho món đồ cụ thể
// @Tags Transfers
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài đăng"
// @Success 200 {object} shared_pres.APIResponse{data=[]communitydto.TransferRequestRes} "Lấy danh sách người xin mua thành công"
// @Router /api/v1/transfers/items/{postItemID}/requests [get]
func (h *ItemTransferHandler) GetTransferRequestsForSeller(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return communityerrors.ErrInvalidPostItemIDFormat()
	}

	res, err := h.transferUC.GetTransferRequestsForSeller(c.Request.Context(), userID, postItemID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgTransferGetRequestsSuccess, res)
	return nil
}
