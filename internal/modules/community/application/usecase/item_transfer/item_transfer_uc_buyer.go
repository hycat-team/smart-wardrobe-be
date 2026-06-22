package item_transfer

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/application/usecase/post"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/requeststatus"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func (uc *ItemTransferUseCase) CreateTransferRequests(ctx context.Context, buyerUserID uuid.UUID, postItemIDs []uuid.UUID) error {
	if len(postItemIDs) == 0 {
		return nil
	}

	createRequests := func(txCtx context.Context) error {
		uniquePostItemIDs := uniqueUUIDs(postItemIDs)
		postItems, err := uc.postItemRepo.GetByIDs(txCtx, uniquePostItemIDs)
		if err != nil {
			return err
		}

		postItemsByID := make(map[uuid.UUID]*entities.PostItem, len(postItems))
		postIDs := make([]uuid.UUID, 0, len(postItems))
		for _, postItem := range postItems {
			if postItem == nil {
				continue
			}
			postItemsByID[postItem.ID] = postItem
			postIDs = append(postIDs, postItem.PostID)
		}

		posts, err := uc.postRepo.GetByIDs(txCtx, uniqueUUIDs(postIDs))
		if err != nil {
			return err
		}
		postsByID := make(map[uuid.UUID]*entities.Post, len(posts))
		for _, post := range posts {
			if post == nil {
				continue
			}
			postsByID[post.ID] = post
		}

		existingRequests, err := uc.transferRequestRepo.GetByBuyerAndPostItems(txCtx, buyerUserID, uniquePostItemIDs)
		if err != nil {
			return err
		}
		existingRequestsByPostItemID := make(map[uuid.UUID]*entities.TransferRequest, len(existingRequests))
		for _, existing := range existingRequests {
			if existing == nil {
				continue
			}
			existingRequestsByPostItemID[existing.PostItemID] = existing
		}

		for _, postItemID := range postItemIDs {
			postItem := postItemsByID[postItemID]
			if postItem == nil {
				return communityerrors.ErrRequestedProductNotFound()
			}

			if postItem.Status != postitemstatus.Available || postItem.TransferState != transferstate.None {
				return communityerrors.ErrTransferRequestInvalid()
			}

			post := postsByID[postItem.PostID]
			if post == nil {
				return communityerrors.ErrPostNotFound()
			}

			if post.UserID == buyerUserID {
				return communityerrors.ErrBuyerSelfRequest()
			}

			existing := existingRequestsByPostItemID[postItemID]
			if existing != nil {
				if existing.Status == requeststatus.Pending {
					continue
				}

				existing.Status = requeststatus.Pending
				if err := uc.transferRequestRepo.Update(txCtx, existing); err != nil {
					return err
				}
				continue
			}

			req := &entities.TransferRequest{
				PostItemID: postItemID,
				BuyerID:    buyerUserID,
				Status:     requeststatus.Pending,
			}
			if err := uc.transferRequestRepo.Create(txCtx, req); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.uow.Execute(ctx, createRequests)
}

func (uc *ItemTransferUseCase) GetPendingTransfers(ctx context.Context, buyerUserID uuid.UUID) ([]*dto.PendingTransferRes, error) {
	items, err := uc.postItemRepo.GetPendingByBuyerID(ctx, buyerUserID)
	if err != nil {
		return nil, err
	}

	sellerIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item == nil || item.Post == nil {
			continue
		}
		sellerIDs = append(sellerIDs, item.Post.UserID)
	}

	sellerUsers, err := uc.identityCtr.GetByIDs(ctx, uniqueUUIDs(sellerIDs))
	if err != nil {
		sellerUsers = nil
	}

	sellerNamesByID := make(map[uuid.UUID]string, len(sellerUsers))
	for _, user := range sellerUsers {
		if user == nil {
			continue
		}
		sellerNamesByID[user.ID] = user.Username
	}

	return mapper.MapToPendingTransferResList(items, sellerNamesByID), nil
}

func (uc *ItemTransferUseCase) AcceptTransfers(ctx context.Context, buyerUserID uuid.UUID, postItemIDs []uuid.UUID) ([]*wardrobe_dto.WardrobeItemRes, error) {
	if len(postItemIDs) == 0 {
		return nil, nil
	}

	var clonedItems []*wardrobe_dto.WardrobeItemRes

	acceptTransfer := func(txCtx context.Context) error {
		postItems, err := uc.postItemRepo.GetByIDs(txCtx, postItemIDs)
		if err != nil {
			return err
		}
		if len(postItems) != len(postItemIDs) {
			return communityerrors.ErrRequestedProductNotFound()
		}

		postsToSync := make(map[uuid.UUID]bool)

		// Batch fetch Sibling Items
		itemIDs := make([]uuid.UUID, 0, len(postItems))
		for _, pi := range postItems {
			itemIDs = append(itemIDs, pi.ItemID)
		}
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

		for _, item := range postItems {
			if item.BuyerUserID == nil || *item.BuyerUserID != buyerUserID {
				return communityerrors.ErrRequestedProductNotFound()
			}
			if item.TransferState != transferstate.Pending {
				return communityerrors.ErrTransferRequestInvalid()
			}

			cloned, err := uc.wardrobeCtr.CopyItemToUser(txCtx, item.ItemID, buyerUserID)
			if err != nil {
				return err
			}
			clonedItems = append(clonedItems, cloned)

			if err := uc.wardrobeCtr.UpdateItemStatus(txCtx, item.ItemID, wardrobestatus.Sold); err != nil {
				return err
			}

			item.TransferState = transferstate.Accepted
			if err := uc.postItemRepo.Update(txCtx, item); err != nil {
				return err
			}

			postsToSync[item.PostID] = true

			for _, sibling := range siblingsMap[item.ItemID] {
				sibling.Status = postitemstatus.Hidden
				if sibling.TransferState != transferstate.Accepted {
					sibling.TransferState = transferstate.None
					sibling.BuyerUserID = nil
				}
				if err := uc.postItemRepo.Update(txCtx, sibling); err != nil {
					return err
				}
				postsToSync[sibling.PostID] = true
			}
		}

		for postID := range postsToSync {
			if err := post.SyncPostTotalPrice(txCtx, uc.postRepo, uc.postItemRepo, postID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := uc.uow.Execute(ctx, acceptTransfer); err != nil {
		return nil, err
	}

	return clonedItems, nil
}

func (uc *ItemTransferUseCase) DeclineTransfers(ctx context.Context, buyerUserID uuid.UUID, postItemIDs []uuid.UUID) error {
	if len(postItemIDs) == 0 {
		return nil
	}

	declineTransfer := func(txCtx context.Context) error {
		postItems, err := uc.postItemRepo.GetByIDs(txCtx, postItemIDs)
		if err != nil {
			return err
		}
		if len(postItems) != len(postItemIDs) {
			return communityerrors.ErrRequestedProductNotFound()
		}

		postsToSync := make(map[uuid.UUID]bool)

		// Batch fetch Buyer Pending Requests
		buyerReqs, err := uc.transferRequestRepo.GetByBuyerAndPostItems(txCtx, buyerUserID, postItemIDs)
		if err != nil {
			return err
		}
		buyerReqMap := make(map[uuid.UUID]*entities.TransferRequest)
		for _, br := range buyerReqs {
			if br != nil {
				buyerReqMap[br.PostItemID] = br
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
			if postItem.BuyerUserID == nil || *postItem.BuyerUserID != buyerUserID {
				return communityerrors.ErrRequestedProductNotFound()
			}
			if postItem.TransferState != transferstate.Pending {
				return communityerrors.ErrTransferRequestInvalid()
			}

			postItem.TransferState = transferstate.Declined
			postItem.Status = postitemstatus.Available
			now := time.Now()
			postItem.DeclinedAt = &now
			if err := uc.postItemRepo.Update(txCtx, postItem); err != nil {
				return err
			}

			postsToSync[postItem.PostID] = true

			buyerReq := buyerReqMap[postItem.ID]
			if buyerReq != nil {
				buyerReq.Status = requeststatus.Canceled
				if err := uc.transferRequestRepo.Update(txCtx, buyerReq); err != nil {
					return err
				}
			}

			hasOtherActiveTransfer := false
			if trans, ok := activeTransferMap[postItem.ItemID]; ok {
				for atID := range trans {
					if atID != postItem.ID {
						hasOtherActiveTransfer = true
						break
					}
				}
			}

			if !hasOtherActiveTransfer {
				for _, sibling := range siblingsMap[postItem.ItemID] {
					if sibling.TransferState == transferstate.Accepted {
						continue
					}
					sibling.Status = postitemstatus.Available
					if err := uc.postItemRepo.Update(txCtx, sibling); err != nil {
						return err
					}
					postsToSync[sibling.PostID] = true
				}
				if err := uc.wardrobeCtr.UpdateItemStatus(txCtx, postItem.ItemID, wardrobestatus.Selling); err != nil {
					return err
				}
			}
		}

		for postID := range postsToSync {
			if err := post.SyncPostTotalPrice(txCtx, uc.postRepo, uc.postItemRepo, postID); err != nil {
				return err
			}
		}

		return nil
	}

	return uc.uow.Execute(ctx, declineTransfer)
}
