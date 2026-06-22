package mapper

import (
	"sort"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/community/application/dto"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// MapToTransferBuyerSummaryRes maps an identity UserRes DTO into a TransferBuyerSummaryRes DTO.
func MapToTransferBuyerSummaryRes(user *identity_dto.UserRes) *dto.TransferBuyerSummaryRes {
	if user == nil {
		return nil
	}
	return &dto.TransferBuyerSummaryRes{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarUrl,
	}
}

// MapToSellerTransferPostResList maps a slice of PostItem entities into a sorted slice of SellerTransferPostRes DTOs.
func MapToSellerTransferPostResList(items []*entities.PostItem, buyersByID map[uuid.UUID]*dto.TransferBuyerSummaryRes) []*dto.SellerTransferPostRes {
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
			Item:          MapWardrobeItem(item.WardrobeItem),
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

	return result
}

// MapToTransferRequestResList maps a slice of TransferRequest entities into a slice of TransferRequestRes DTOs.
func MapToTransferRequestResList(reqs []*entities.TransferRequest, buyerMap map[uuid.UUID]*identity_dto.UserRes) []*dto.TransferRequestRes {
	result := make([]*dto.TransferRequestRes, 0, len(reqs))
	for _, r := range reqs {
		if r == nil {
			continue
		}
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
	return result
}

// MapToPendingTransferResList maps a slice of PostItem entities into a slice of PendingTransferRes DTOs for the buyer.
func MapToPendingTransferResList(items []*entities.PostItem, sellerNamesByID map[uuid.UUID]string) []*dto.PendingTransferRes {
	result := make([]*dto.PendingTransferRes, 0, len(items))
	for _, item := range items {
		if item == nil || item.Post == nil {
			continue
		}

		sellerName := "Người dùng ẩn danh"
		if name, exists := sellerNamesByID[item.Post.UserID]; exists && name != "" {
			sellerName = name
		}

		result = append(result, &dto.PendingTransferRes{
			PostItemID: item.ID,
			Item:       MapWardrobeItem(item.WardrobeItem),
			SellerName: sellerName,
		})
	}
	return result
}
