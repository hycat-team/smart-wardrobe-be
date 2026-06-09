package usecase

import (
	"context"
	"sort"
	"time"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/requeststatus"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"
	"smart-wardrobe-be/internal/shared/domain/entities"

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

		// Batch fetch all Posts
		postIDs := make([]uuid.UUID, 0, len(postItems))
		for _, pi := range postItems {
			postIDs = append(postIDs, pi.PostID)
		}
		posts, err := uc.postRepo.GetByIDs(txCtx, postIDs)
		if err != nil {
			return err
		}
		postMap := make(map[uuid.UUID]*entities.Post)
		for _, p := range posts {
			if p != nil {
				postMap[p.ID] = p
			}
		}

		// Batch fetch Active Transfers
		itemIDs := make([]uuid.UUID, 0, len(postItems))
		for _, pi := range postItems {
			itemIDs = append(itemIDs, pi.ItemID)
		}
		activeTransfers, err := uc.postItemRepo.GetActiveTransfersByItemIDs(txCtx, itemIDs)
		if err != nil {
			return err
		}
		activeTransferMap := make(map[uuid.UUID]map[uuid.UUID]bool)
		for _, at := range activeTransfers {
			if at != nil {
				if _, ok := activeTransferMap[at.ItemID]; !ok {
					activeTransferMap[at.ItemID] = make(map[uuid.UUID]bool)
				}
				activeTransferMap[at.ItemID][at.ID] = true
			}
		}

		// Batch fetch Buyer Pending Requests
		buyerReqs, err := uc.transferRequestRepo.GetByBuyerAndPostItems(txCtx, buyerID, postItemIDs)
		if err != nil {
			return err
		}
		buyerReqMap := make(map[uuid.UUID]*entities.TransferRequest)
		for _, br := range buyerReqs {
			if br != nil {
				buyerReqMap[br.PostItemID] = br
			}
		}

		// Batch fetch Other Requests to reject
		otherReqs, err := uc.transferRequestRepo.GetByPostItemIDs(txCtx, postItemIDs)
		if err != nil {
			return err
		}
		otherReqsMap := make(map[uuid.UUID][]*entities.TransferRequest)
		for _, r := range otherReqs {
			if r != nil {
				otherReqsMap[r.PostItemID] = append(otherReqsMap[r.PostItemID], r)
			}
		}

		// Batch fetch Sibling Items
		siblings, err := uc.postItemRepo.GetSiblingItemsByItemIDs(txCtx, itemIDs, postItemIDs)
		if err != nil {
			return err
		}
		siblingsMap := make(map[uuid.UUID][]*entities.PostItem)
		for _, sib := range siblings {
			if sib != nil {
				siblingsMap[sib.ItemID] = append(siblingsMap[sib.ItemID], sib)
			}
		}

		for _, postItem := range postItems {
			// Validate quyền sở hữu bài đăng của seller
			post := postMap[postItem.PostID]
			if post == nil || post.UserID != sellerUserID {
				return communityerrors.ErrTransferForbidden
			}

			// Kiểm tra xem món đồ này có giao dịch active nào khác không
			hasOtherActiveTransfer := false
			if trans, ok := activeTransferMap[postItem.ItemID]; ok {
				for atID := range trans {
					if atID != postItem.ID {
						hasOtherActiveTransfer = true
						break
					}
				}
			}
			if hasOtherActiveTransfer {
				return communityerrors.ErrItemInAnotherTransfer
			}

			// Kiểm tra yêu cầu xin mua (TransferRequest) của buyer này đối với post item
			buyerReq := buyerReqMap[postItem.ID]
			if buyerReq == nil || buyerReq.Status != requeststatus.Pending {
				return communityerrors.ErrNoPendingRequest
			}

			// Đánh dấu yêu cầu xin mua này thành Accepted
			buyerReq.Status = requeststatus.Accepted
			if err := uc.transferRequestRepo.Update(txCtx, buyerReq); err != nil {
				return err
			}

			// Từ chối các yêu cầu xin mua khác của món đồ này
			for _, otherReq := range otherReqsMap[postItem.ID] {
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
			for _, sibling := range siblingsMap[postItem.ItemID] {
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

	var buyerIDsToFetch []uuid.UUID
	for _, r := range reqs {
		if r.Buyer == nil {
			buyerIDsToFetch = append(buyerIDsToFetch, r.BuyerID)
		}
	}

	buyerMap := make(map[uuid.UUID]*identity_dto.UserRes)
	if len(buyerIDsToFetch) > 0 {
		buyers, err := uc.identityCtr.GetByIDs(ctx, buyerIDsToFetch)
		if err == nil {
			for _, b := range buyers {
				if b != nil {
					buyerMap[b.ID] = b
				}
			}
		}
	}

	result := make([]*dto.TransferRequestRes, 0, len(reqs))
	for _, r := range reqs {
		username := "Người dùng ẩn danh"
		var avatarURL *string
		if r.Buyer != nil {
			username = r.Buyer.Username
			avatarURL = r.Buyer.AvatarUrl
		} else if b, exists := buyerMap[r.BuyerID]; exists {
			username = b.Username
			avatarURL = b.AvatarUrl
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
