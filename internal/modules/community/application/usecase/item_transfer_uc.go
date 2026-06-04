package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/contract"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type ItemTransferUseCase struct {
	postRepo     repositories.IPostRepository
	postItemRepo repositories.IPostItemRepository
	identityCtr  identity_contract.IUserContract
	wardrobeCtr  wardrobe_contract.IWardrobeContract
	uow          shared_repos.IUnitOfWork
}

func NewItemTransferUseCase(
	postRepo repositories.IPostRepository,
	postItemRepo repositories.IPostItemRepository,
	identityCtr identity_contract.IUserContract,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IItemTransferUseCase {
	return &ItemTransferUseCase{
		postRepo:     postRepo,
		postItemRepo: postItemRepo,
		identityCtr:  identityCtr,
		wardrobeCtr:  wardrobeCtr,
		uow:          uow,
	}
}

func (uc *ItemTransferUseCase) MarkPostItemSold(ctx context.Context, userID uuid.UUID, postItemID uuid.UUID, buyerUserID uuid.UUID) error {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return err
	}
	if postItem == nil {
		return errorcode.NewNotFound("Không tìm thấy món đồ đăng bán.")
	}

	post, err := uc.postRepo.GetByID(ctx, postItem.PostID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return errorcode.NewForbidden("Bạn không có quyền thực hiện hành động này.")
	}

	if postItem.Status == postitemstatus.Sold {
		return errorcode.NewBadRequest("Món đồ này đã được đánh dấu bán trước đó.")
	}

	postItem.BuyerUserID = &buyerUserID
	postItem.TransferState = transferstate.Pending
	postItem.Status = postitemstatus.Sold
	now := time.Now()
	postItem.SoldAt = &now

	return uc.postItemRepo.Update(ctx, postItem)
}

func (uc *ItemTransferUseCase) GetPendingTransfers(ctx context.Context, buyerUserID uuid.UUID) ([]*dto.PendingTransferRes, error) {
	items, err := uc.postItemRepo.GetPendingByBuyerID(ctx, buyerUserID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.PendingTransferRes, 0, len(items))
	for _, item := range items {
		if item.Post == nil {
			continue
		}
		userRes, err := uc.identityCtr.GetByID(ctx, item.Post.UserID)
		sellerName := "Người dùng ẩn danh"
		if err == nil && userRes != nil {
			sellerName = userRes.Username
		}
		result = append(result, &dto.PendingTransferRes{
			PostItemID: item.ID,
			Item:       mapper.MapWardrobeItem(item.WardrobeItem),
			SellerName: sellerName,
		})
	}
	return result, nil
}

func (uc *ItemTransferUseCase) AcceptTransfer(ctx context.Context, buyerUserID uuid.UUID, postItemID uuid.UUID) (*wardrobe_dto.WardrobeItemRes, error) {
	var cloned *wardrobe_dto.WardrobeItemRes

	acceptTransfer := func(txCtx context.Context) error {
		item, err := uc.postItemRepo.GetByID(txCtx, postItemID)
		if err != nil {
			return err
		}
		if item == nil || item.BuyerUserID == nil || *item.BuyerUserID != buyerUserID {
			return errorcode.NewNotFound("Không tìm thấy món đồ đang chờ nhận.")
		}

		if item.TransferState != transferstate.Pending {
			return errorcode.NewBadRequest("Yêu cầu bàn giao này đã được xử lý hoặc không còn hiệu lực.")
		}

		c, err := uc.wardrobeCtr.CopyItemToUser(txCtx, item.ItemID, buyerUserID)
		if err != nil {
			return err
		}
		cloned = c

		if err := uc.wardrobeCtr.UpdateItemStatus(txCtx, item.ItemID, wardrobestatus.Sold); err != nil {
			return err
		}

		item.TransferState = transferstate.Accepted
		if err := uc.postItemRepo.Update(txCtx, item); err != nil {
			return err
		}

		return nil
	}

	if err := uc.uow.Execute(ctx, acceptTransfer); err != nil {
		return nil, err
	}

	return cloned, nil
}

func (uc *ItemTransferUseCase) DeclineTransfer(ctx context.Context, buyerUserID uuid.UUID, postItemID uuid.UUID) error {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return err
	}
	if postItem == nil || postItem.BuyerUserID == nil || *postItem.BuyerUserID != buyerUserID {
		return errorcode.NewNotFound("Không tìm thấy món đồ đang chờ nhận.")
	}

	if postItem.TransferState != transferstate.Pending {
		return errorcode.NewBadRequest("Yêu cầu bàn giao này đã được xử lý hoặc không còn hiệu lực.")
	}

	postItem.TransferState = transferstate.Declined
	return uc.postItemRepo.Update(ctx, postItem)
}

var _ uc_interfaces.IItemTransferUseCase = (*ItemTransferUseCase)(nil)
