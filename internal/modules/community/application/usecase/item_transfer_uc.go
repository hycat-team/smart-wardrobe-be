package usecase

import (
	"context"
	"sort"
	"time"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/contract"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
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
	markSold := func(txCtx context.Context) error {
		postItem, err := uc.postItemRepo.GetByID(txCtx, postItemID)
		if err != nil {
			return err
		}
		if postItem == nil {
			return apperror.NewNotFound("Không tìm thấy sản phẩm được yêu cầu.")
		}

		post, err := uc.postRepo.GetByID(txCtx, postItem.PostID)
		if err != nil {
			return err
		}
		if post == nil || post.UserID != userID {
			return apperror.NewForbidden("Bạn không được phép thực hiện thao tác này.")
		}

		hasOtherActiveTransfer, err := uc.postItemRepo.HasActiveTransfer(txCtx, postItem.ItemID, &postItemID)
		if err != nil {
			return err
		}
		if hasOtherActiveTransfer {
			return apperror.NewBadRequest("Trang phục này đang nằm trong một giao dịch khác.")
		}

		postItem.BuyerUserID = &buyerUserID
		postItem.TransferState = transferstate.Pending
		postItem.Status = postitemstatus.Sold
		now := time.Now()
		postItem.SoldAt = &now
		postItem.DeclinedAt = nil
		if err := uc.postItemRepo.Update(txCtx, postItem); err != nil {
			return err
		}

		siblings, err := uc.postItemRepo.GetSiblingItems(txCtx, postItem.ItemID, postItem.ID)
		if err != nil {
			return err
		}
		for _, sibling := range siblings {
			if sibling.TransferState == transferstate.Accepted {
				continue
			}
			sibling.Status = postitemstatus.Hidden
			if err := uc.postItemRepo.Update(txCtx, sibling); err != nil {
				return err
			}
		}

		return uc.wardrobeCtr.UpdateItemStatus(txCtx, postItem.ItemID, wardrobestatus.Selling)
	}

	return uc.uow.Execute(ctx, markSold)
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

func (uc *ItemTransferUseCase) GetSellerTransferPosts(ctx context.Context, sellerUserID uuid.UUID) ([]*dto.SellerTransferPostRes, error) {
	items, err := uc.postItemRepo.GetTransferItemsBySellerID(ctx, sellerUserID)
	if err != nil {
		return nil, err
	}

	buyersByID := make(map[uuid.UUID]*dto.TransferBuyerSummaryRes)
	for _, item := range items {
		if item == nil || item.BuyerUserID == nil {
			continue
		}
		if _, exists := buyersByID[*item.BuyerUserID]; exists {
			continue
		}

		userRes, err := uc.identityCtr.GetByID(ctx, *item.BuyerUserID)
		if err != nil || userRes == nil {
			continue
		}
		buyersByID[*item.BuyerUserID] = dto.NewTransferBuyerSummary(userRes)
	}

	postsByID := make(map[uuid.UUID]*dto.SellerTransferPostRes)
	postOrder := make([]uuid.UUID, 0)
	for _, item := range items {
		if item == nil || item.Post == nil {
			continue
		}

		postRes, exists := postsByID[item.PostID]
		if !exists {
			postRes = &dto.SellerTransferPostRes{
				PostID:    item.Post.ID,
				Title:     item.Post.Title,
				PostType:  string(item.Post.PostType),
				CreatedAt: item.Post.CreatedAt,
				UpdatedAt: item.Post.UpdatedAt,
				Items:     make([]*dto.SellerTransferPostItemRes, 0),
			}
			postsByID[item.PostID] = postRes
			postOrder = append(postOrder, item.PostID)
		}

		var buyer *dto.TransferBuyerSummaryRes
		if item.BuyerUserID != nil {
			buyer = buyersByID[*item.BuyerUserID]
		}

		postRes.Items = append(postRes.Items, &dto.SellerTransferPostItemRes{
			PostItemID:    item.ID,
			Item:          mapper.MapWardrobeItem(item.WardrobeItem),
			Price:         item.Price,
			ItemCondition: int16(item.ItemCondition),
			Status:        int16(item.Status),
			TransferState: int16(item.TransferState),
			SoldAt:        item.SoldAt,
			DeclinedAt:    item.DeclinedAt,
			Buyer:         buyer,
		})
	}

	result := make([]*dto.SellerTransferPostRes, 0, len(postOrder))
	for _, postID := range postOrder {
		postRes := postsByID[postID]
		sort.SliceStable(postRes.Items, func(i, j int) bool {
			left := postRes.Items[i]
			right := postRes.Items[j]
			if left.SoldAt == nil || right.SoldAt == nil || left.SoldAt.Equal(*right.SoldAt) {
				return left.PostItemID.String() < right.PostItemID.String()
			}
			return left.SoldAt.After(*right.SoldAt)
		})
		result = append(result, postRes)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

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
			return apperror.NewNotFound("Không tìm thấy sản phẩm được yêu cầu.")
		}

		if item.TransferState != transferstate.Pending {
			return apperror.NewBadRequest("Yêu cầu chuyển nhượng này đã được xử lý hoặc không còn hiệu lực.")
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

		siblings, err := uc.postItemRepo.GetSiblingItems(txCtx, item.ItemID, item.ID)
		if err != nil {
			return err
		}
		for _, sibling := range siblings {
			sibling.Status = postitemstatus.Hidden
			if sibling.TransferState != transferstate.Accepted {
				sibling.TransferState = transferstate.None
				sibling.BuyerUserID = nil
			}
			if err := uc.postItemRepo.Update(txCtx, sibling); err != nil {
				return err
			}
		}

		return nil
	}

	if err := uc.uow.Execute(ctx, acceptTransfer); err != nil {
		return nil, err
	}

	return cloned, nil
}

func (uc *ItemTransferUseCase) DeclineTransfer(ctx context.Context, buyerUserID uuid.UUID, postItemID uuid.UUID) error {
	declineTransfer := func(txCtx context.Context) error {
		postItem, err := uc.postItemRepo.GetByID(txCtx, postItemID)
		if err != nil {
			return err
		}
		if postItem == nil || postItem.BuyerUserID == nil || *postItem.BuyerUserID != buyerUserID {
			return apperror.NewNotFound("Không tìm thấy sản phẩm được yêu cầu.")
		}

		if postItem.TransferState != transferstate.Pending {
			return apperror.NewBadRequest("Yêu cầu chuyển nhượng này đã được xử lý hoặc không còn hiệu lực.")
		}

		postItem.TransferState = transferstate.Declined
		postItem.Status = postitemstatus.Available
		now := time.Now()
		postItem.DeclinedAt = &now
		if err := uc.postItemRepo.Update(txCtx, postItem); err != nil {
			return err
		}

		hasOtherActiveTransfer, err := uc.postItemRepo.HasActiveTransfer(txCtx, postItem.ItemID, &postItem.ID)
		if err != nil {
			return err
		}
		if !hasOtherActiveTransfer {
			siblings, err := uc.postItemRepo.GetSiblingItems(txCtx, postItem.ItemID, postItem.ID)
			if err != nil {
				return err
			}
			for _, sibling := range siblings {
				if sibling.TransferState == transferstate.Accepted {
					continue
				}
				sibling.Status = postitemstatus.Available
				if err := uc.postItemRepo.Update(txCtx, sibling); err != nil {
					return err
				}
			}
			if err := uc.wardrobeCtr.UpdateItemStatus(txCtx, postItem.ItemID, wardrobestatus.Selling); err != nil {
				return err
			}
		}

		return nil
	}

	return uc.uow.Execute(ctx, declineTransfer)
}

var _ uc_interfaces.IItemTransferUseCase = (*ItemTransferUseCase)(nil)
