package usecase

import (
	"context"
	"sort"
	"time"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/requeststatus"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"

	"github.com/google/uuid"
)

func (uc *ItemTransferUseCase) MarkPostItemsSold(ctx context.Context, sellerUserID uuid.UUID, postItemIDs []uuid.UUID, buyerID uuid.UUID) error {
	if len(postItemIDs) == 0 {
		return nil
	}

	now := time.Now()

	markSold := func(txCtx context.Context) error {
		postItems, err := uc.postItemRepo.GetByIDs(txCtx, postItemIDs)
		if err != nil {
			return err
		}
		if len(postItems) != len(postItemIDs) {
			return communityerrors.ErrRequestedProductNotFound
		}

		postsToSync := make(map[uuid.UUID]bool)

		for _, postItem := range postItems {
			// Validate quyền sở hữu bài đăng của seller
			post, err := uc.postRepo.GetByID(txCtx, postItem.PostID)
			if err != nil {
				return err
			}
			if post == nil || post.UserID != sellerUserID {
				return communityerrors.ErrTransferForbidden
			}

			// Kiểm tra xem món đồ này có giao dịch active nào khác không
			hasOtherActiveTransfer, err := uc.postItemRepo.HasActiveTransfer(txCtx, postItem.ItemID, &postItem.ID)
			if err != nil {
				return err
			}
			if hasOtherActiveTransfer {
				return communityerrors.ErrItemInAnotherTransfer
			}

			// Kiểm tra yêu cầu xin mua (TransferRequest) của buyer này đối với post item
			buyerReq, err := uc.transferRequestRepo.GetByBuyerAndPostItem(txCtx, buyerID, postItem.ID)
			if err != nil {
				return err
			}
			if buyerReq == nil || buyerReq.Status != requeststatus.Pending {
				return communityerrors.ErrNoPendingRequest
			}

			// Đánh dấu yêu cầu xin mua này thành Accepted
			buyerReq.Status = requeststatus.Accepted
			if err := uc.transferRequestRepo.Update(txCtx, buyerReq); err != nil {
				return err
			}

			// Từ chối các yêu cầu xin mua khác của món đồ này
			otherReqs, err := uc.transferRequestRepo.GetByPostItemID(txCtx, postItem.ID)
			if err != nil {
				return err
			}
			for _, otherReq := range otherReqs {
				if otherReq.BuyerID != buyerID && otherReq.Status == requeststatus.Pending {
					otherReq.Status = requeststatus.Rejected
					if err := uc.transferRequestRepo.Update(txCtx, otherReq); err != nil {
						return err
					}
				}
			}

			// Cập nhật post item sang trạng thái Pending transfer
			postItem.BuyerUserID = &buyerID
			postItem.TransferState = transferstate.Pending
			postItem.Status = postitemstatus.Sold
			postItem.SoldAt = &now
			postItem.DeclinedAt = nil
			if err := uc.postItemRepo.Update(txCtx, postItem); err != nil {
				return err
			}

			postsToSync[postItem.PostID] = true

			// Tìm các sibling items (món đồ đăng ở bài khác) và ẩn đi
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
				postsToSync[sibling.PostID] = true
			}
		}

		// Đồng bộ hóa tổng giá tiền các bài đăng bị ảnh hưởng
		for postID := range postsToSync {
			if err := syncPostTotalPrice(txCtx, uc.postRepo, uc.postItemRepo, postID); err != nil {
				return err
			}
		}

		return nil
	}

	return uc.uow.Execute(ctx, markSold)
}

func (uc *ItemTransferUseCase) GetSellerTransferPosts(ctx context.Context, sellerUserID uuid.UUID) ([]*dto.SellerTransferPostRes, error) {
	items, err := uc.postItemRepo.GetTransferItemsBySellerID(ctx, sellerUserID)
	if err != nil {
		return nil, err
	}

	buyersByID := make(map[uuid.UUID]*dto.TransferBuyerSummaryRes)
	buyerIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item == nil || item.BuyerUserID == nil {
			continue
		}
		buyerIDs = append(buyerIDs, *item.BuyerUserID)
	}

	buyerUsers, err := uc.identityCtr.GetByIDs(ctx, uniqueUUIDs(buyerIDs))
	if err != nil {
		buyerUsers = nil
	}
	for _, userRes := range buyerUsers {
		if userRes == nil {
			continue
		}
		buyersByID[userRes.ID] = &dto.TransferBuyerSummaryRes{
			ID:        userRes.ID,
			Username:  userRes.Username,
			AvatarURL: userRes.AvatarUrl,
		}
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
				PostType:  item.Post.PostType,
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
			ItemCondition: item.ItemCondition,
			Status:        item.Status,
			TransferState: item.TransferState,
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

func (uc *ItemTransferUseCase) GetTransferRequestsForSeller(ctx context.Context, sellerUserID uuid.UUID, postItemID uuid.UUID) ([]*dto.TransferRequestRes, error) {
	// Kiểm tra xem post item có tồn tại không và người gọi có phải là chủ bài đăng không
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return nil, err
	}
	if postItem == nil {
		return nil, communityerrors.ErrRequestedProductNotFound
	}

	post, err := uc.postRepo.GetByID(ctx, postItem.PostID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.UserID != sellerUserID {
		return nil, communityerrors.ErrTransferForbidden
	}

	reqs, err := uc.transferRequestRepo.GetByPostItemID(ctx, postItemID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.TransferRequestRes, 0, len(reqs))
	for _, r := range reqs {
		username := "Người dùng ẩn danh"
		var avatarURL *string
		if r.Buyer != nil {
			username = r.Buyer.Username
			avatarURL = r.Buyer.AvatarUrl
		} else {
			userRes, err := uc.identityCtr.GetByID(ctx, r.BuyerID)
			if err == nil && userRes != nil {
				username = userRes.Username
				avatarURL = userRes.AvatarUrl
			}
		}

		result = append(result, &dto.TransferRequestRes{
			ID:        r.ID,
			BuyerID:   r.BuyerID,
			Username:  username,
			AvatarURL: avatarURL,
			Status:    r.Status,
			CreatedAt: r.CreatedAt,
		})
	}

	return result, nil
}
