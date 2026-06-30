package item_transfer

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/application/usecase/post"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/community/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/community/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/requeststatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

// TransferSaleContext groups the preloaded records required to mark post items as sold.
type TransferSaleContext struct {
	PostItems         []*entities.PostItem
	PostsByID         map[uuid.UUID]*entities.Post
	ActiveTransferMap map[uuid.UUID]map[uuid.UUID]bool
	BuyerRequestsByID map[uuid.UUID]*entities.TransferRequest
	OtherRequestsByID map[uuid.UUID][]*entities.TransferRequest
	SiblingItemsByID  map[uuid.UUID][]*entities.PostItem
	AffectedPostIDs   map[uuid.UUID]bool
}

// MarkPostItemsSold accepts the buyer requests and moves the selected items into pending transfer.
func (uc *ItemTransferUseCase) MarkPostItemsSold(ctx context.Context, sellerUserID uuid.UUID, postItemIDs []uuid.UUID, buyerID uuid.UUID) error {
	if len(postItemIDs) == 0 {
		return nil
	}

	now := time.Now()
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		saleCtx, err := uc.loadTransferSaleContext(txCtx, postItemIDs, buyerID)
		if err != nil {
			return err
		}

		for _, postItem := range saleCtx.PostItems {
			if err := uc.validateTransferSale(postItem, saleCtx, sellerUserID); err != nil {
				return err
			}
			if err := uc.acceptBuyerRequest(txCtx, saleCtx.BuyerRequestsByID[postItem.ID]); err != nil {
				return err
			}
			if err := uc.rejectCompetingRequests(txCtx, saleCtx.OtherRequestsByID[postItem.ID], buyerID); err != nil {
				return err
			}
			if err := uc.markItemPendingTransfer(txCtx, postItem, buyerID, now); err != nil {
				return err
			}
			saleCtx.AffectedPostIDs[postItem.PostID] = true

			if err := uc.hideSiblingItems(txCtx, saleCtx.SiblingItemsByID[postItem.ItemID], saleCtx.AffectedPostIDs); err != nil {
				return err
			}
		}

		return uc.syncAffectedTransferPosts(txCtx, saleCtx.AffectedPostIDs)
	})
}

// GetSellerTransferPosts returns grouped transfer items for a seller ordered by recency.
func (uc *ItemTransferUseCase) GetSellerTransferPosts(ctx context.Context, sellerUserID uuid.UUID) ([]*dto.SellerTransferPostRes, error) {
	items, err := uc.postItemRepo.GetTransferItemsBySellerID(ctx, sellerUserID)
	if err != nil {
		return nil, err
	}

	buyerIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item == nil || item.BuyerUserID == nil {
			continue
		}
		buyerIDs = append(buyerIDs, *item.BuyerUserID)
	}

	buyerUsers, err := uc.identityCtr.GetByIDs(ctx, sliceutils.UniqueUUIDs(buyerIDs))
	if err != nil {
		buyerUsers = nil
	}

	buyersByID := make(map[uuid.UUID]*dto.TransferBuyerSummaryRes)
	for _, userRes := range buyerUsers {
		if userRes == nil {
			continue
		}
		buyersByID[userRes.ID] = mapper.MapToTransferBuyerSummaryRes(userRes)
	}

	return mapper.MapToSellerTransferPostResList(items, buyersByID), nil
}

// GetTransferRequestsForSeller returns buyer requests for one seller-owned post item.
func (uc *ItemTransferUseCase) GetTransferRequestsForSeller(ctx context.Context, sellerUserID uuid.UUID, postItemID uuid.UUID) ([]*dto.TransferRequestRes, error) {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return nil, err
	}
	if postItem == nil {
		return nil, communityerrors.ErrRequestedProductNotFound()
	}

	postEntity, err := uc.postRepo.GetByID(ctx, postItem.PostID)
	if err != nil {
		return nil, err
	}
	if postEntity == nil || postEntity.UserID != sellerUserID {
		return nil, communityerrors.ErrTransferForbidden()
	}

	reqs, err := uc.transferRequestRepo.GetByPostItemID(ctx, postItemID)
	if err != nil {
		return nil, err
	}

	buyerMap := uc.loadTransferRequestBuyers(ctx, reqs)
	return mapper.MapToTransferRequestResList(reqs, buyerMap), nil
}

// loadTransferSaleContext preloads all records needed to process a sold-item workflow in one transaction.
func (uc *ItemTransferUseCase) loadTransferSaleContext(txCtx context.Context, postItemIDs []uuid.UUID, buyerID uuid.UUID) (*TransferSaleContext, error) {
	postItems, err := uc.postItemRepo.GetByIDs(txCtx, postItemIDs)
	if err != nil {
		return nil, err
	}
	if len(postItems) != len(postItemIDs) {
		return nil, communityerrors.ErrRequestedProductNotFound()
	}

	postIDs := make([]uuid.UUID, 0, len(postItems))
	itemIDs := make([]uuid.UUID, 0, len(postItems))
	for _, pi := range postItems {
		postIDs = append(postIDs, pi.PostID)
		itemIDs = append(itemIDs, pi.ItemID)
	}

	posts, err := uc.postRepo.GetByIDs(txCtx, postIDs)
	if err != nil {
		return nil, err
	}
	postsByID := make(map[uuid.UUID]*entities.Post, len(posts))
	for _, p := range posts {
		if p != nil {
			postsByID[p.ID] = p
		}
	}

	activeTransfers, err := uc.postItemRepo.GetActiveTransfersByItemIDs(txCtx, itemIDs)
	if err != nil {
		return nil, err
	}
	activeTransferMap := make(map[uuid.UUID]map[uuid.UUID]bool)
	for _, at := range activeTransfers {
		if at == nil {
			continue
		}
		if _, ok := activeTransferMap[at.ItemID]; !ok {
			activeTransferMap[at.ItemID] = make(map[uuid.UUID]bool)
		}
		activeTransferMap[at.ItemID][at.ID] = true
	}

	buyerReqs, err := uc.transferRequestRepo.GetByBuyerAndPostItems(txCtx, buyerID, postItemIDs)
	if err != nil {
		return nil, err
	}
	buyerReqMap := make(map[uuid.UUID]*entities.TransferRequest, len(buyerReqs))
	for _, br := range buyerReqs {
		if br != nil {
			buyerReqMap[br.PostItemID] = br
		}
	}

	otherReqs, err := uc.transferRequestRepo.GetByPostItemIDs(txCtx, postItemIDs)
	if err != nil {
		return nil, err
	}
	otherReqMap := make(map[uuid.UUID][]*entities.TransferRequest)
	for _, req := range otherReqs {
		if req != nil {
			otherReqMap[req.PostItemID] = append(otherReqMap[req.PostItemID], req)
		}
	}

	siblings, err := uc.postItemRepo.GetSiblingItemsByItemIDs(txCtx, itemIDs, postItemIDs)
	if err != nil {
		return nil, err
	}
	siblingsMap := make(map[uuid.UUID][]*entities.PostItem)
	for _, sibling := range siblings {
		if sibling != nil {
			siblingsMap[sibling.ItemID] = append(siblingsMap[sibling.ItemID], sibling)
		}
	}

	return &TransferSaleContext{
		PostItems:         postItems,
		PostsByID:         postsByID,
		ActiveTransferMap: activeTransferMap,
		BuyerRequestsByID: buyerReqMap,
		OtherRequestsByID: otherReqMap,
		SiblingItemsByID:  siblingsMap,
		AffectedPostIDs:   map[uuid.UUID]bool{},
	}, nil
}

// validateTransferSale checks seller ownership, transfer conflicts, and buyer request state.
func (uc *ItemTransferUseCase) validateTransferSale(postItem *entities.PostItem, saleCtx *TransferSaleContext, sellerUserID uuid.UUID) error {
	postEntity := saleCtx.PostsByID[postItem.PostID]
	if postEntity == nil || postEntity.UserID != sellerUserID {
		return communityerrors.ErrTransferForbidden()
	}

	if trans, ok := saleCtx.ActiveTransferMap[postItem.ItemID]; ok {
		for activeTransferID := range trans {
			if activeTransferID != postItem.ID {
				return communityerrors.ErrItemInAnotherTransfer()
			}
		}
	}

	buyerReq := saleCtx.BuyerRequestsByID[postItem.ID]
	if buyerReq == nil || buyerReq.Status != requeststatus.Pending {
		return communityerrors.ErrNoPendingRequest()
	}

	return nil
}

// acceptBuyerRequest marks the target buyer request as accepted.
func (uc *ItemTransferUseCase) acceptBuyerRequest(txCtx context.Context, buyerReq *entities.TransferRequest) error {
	buyerReq.Status = requeststatus.Accepted
	return uc.transferRequestRepo.Update(txCtx, buyerReq)
}

// rejectCompetingRequests rejects all other pending requests for the same post item.
func (uc *ItemTransferUseCase) rejectCompetingRequests(txCtx context.Context, requests []*entities.TransferRequest, buyerID uuid.UUID) error {
	for _, req := range requests {
		if req.BuyerID == buyerID || req.Status != requeststatus.Pending {
			continue
		}
		req.Status = requeststatus.Rejected
		if err := uc.transferRequestRepo.Update(txCtx, req); err != nil {
			return err
		}
	}
	return nil
}

// markItemPendingTransfer updates the sold item record to the pending-transfer state.
func (uc *ItemTransferUseCase) markItemPendingTransfer(txCtx context.Context, postItem *entities.PostItem, buyerID uuid.UUID, now time.Time) error {
	postItem.BuyerUserID = &buyerID
	postItem.TransferState = transferstate.Pending
	postItem.Status = postitemstatus.Sold
	postItem.SoldAt = &now
	postItem.DeclinedAt = nil
	return uc.postItemRepo.Update(txCtx, postItem)
}

// hideSiblingItems hides duplicate listings of the same wardrobe item across other posts.
func (uc *ItemTransferUseCase) hideSiblingItems(txCtx context.Context, siblings []*entities.PostItem, affectedPostIDs map[uuid.UUID]bool) error {
	for _, sibling := range siblings {
		if sibling.TransferState == transferstate.Accepted {
			continue
		}
		sibling.Status = postitemstatus.Hidden
		if err := uc.postItemRepo.Update(txCtx, sibling); err != nil {
			return err
		}
		affectedPostIDs[sibling.PostID] = true
	}
	return nil
}

// syncAffectedTransferPosts recalculates post totals for all posts touched by a transfer workflow.
func (uc *ItemTransferUseCase) syncAffectedTransferPosts(txCtx context.Context, affectedPostIDs map[uuid.UUID]bool) error {
	for postID := range affectedPostIDs {
		if err := post.SyncPostTotalPrice(txCtx, uc.postRepo, uc.postItemRepo, postID); err != nil {
			return err
		}
	}
	return nil
}

// loadTransferRequestBuyers fetches missing buyer profile data for transfer requests.
func (uc *ItemTransferUseCase) loadTransferRequestBuyers(ctx context.Context, reqs []*entities.TransferRequest) map[uuid.UUID]*identity_dto.UserRes {
	var buyerIDsToFetch []uuid.UUID
	for _, req := range reqs {
		if req.Buyer == nil {
			buyerIDsToFetch = append(buyerIDsToFetch, req.BuyerID)
		}
	}

	buyerMap := make(map[uuid.UUID]*identity_dto.UserRes)
	if len(buyerIDsToFetch) == 0 {
		return buyerMap
	}

	buyers, err := uc.identityCtr.GetByIDs(ctx, buyerIDsToFetch)
	if err != nil {
		return buyerMap
	}
	for _, buyer := range buyers {
		if buyer != nil {
			buyerMap[buyer.ID] = buyer
		}
	}
	return buyerMap
}
