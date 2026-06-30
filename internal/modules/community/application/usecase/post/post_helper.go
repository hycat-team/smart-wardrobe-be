package post

import (
	"context"
	"crypto/rand"
	"strings"

	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/community/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/community/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/community/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/itemcondition"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

const postPublicIDPrefix = "p_"

var base62Alphabet = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

type resolvedPostItemInput struct {
	ItemID        uuid.UUID
	Price         float64
	ItemCondition itemcondition.ItemCondition
}

func newPostPublicID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	chars := make([]byte, 16)
	for i, b := range buf {
		chars[i] = base62Alphabet[int(b)%len(base62Alphabet)]
	}
	return postPublicIDPrefix + string(chars), nil
}

func (uc *UserPostUseCase) generateUniquePostPublicID(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		publicID, err := newPostPublicID()
		if err != nil {
			return "", err
		}
		post, err := uc.publishing.postRepo.GetByPublicID(ctx, publicID)
		if err != nil {
			return "", err
		}
		if post == nil {
			return publicID, nil
		}
	}
	return "", communityerrors.ErrInvalidPostPublicIDFormat()
}

func (uc *UserPostUseCase) resolvePostItems(ctx context.Context, postType posttype.PostType, items []community_dto.PostItemInputReq) ([]resolvedPostItemInput, error) {
	result := make([]resolvedPostItemInput, 0, len(items))
	if len(items) == 0 {
		return result, nil
	}

	itemIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ItemID)
	}

	wardrobeItems, err := uc.publishing.wardrobeCtr.GetItemsByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	wardrobeByID := make(map[uuid.UUID]*wardrobe_dto.WardrobeItemRes, len(wardrobeItems))
	for _, item := range wardrobeItems {
		wardrobeByID[item.ID] = item
	}

	for _, item := range items {
		resolved := resolvedPostItemInput{
			ItemID:        item.ItemID,
			ItemCondition: itemcondition.Standard,
		}
		if item.ItemCondition != nil {
			resolved.ItemCondition = *item.ItemCondition
		}

		if item.Price != nil {
			resolved.Price = *item.Price
		} else if wardrobeItem := wardrobeByID[item.ItemID]; wardrobeItem != nil && wardrobeItem.Price != nil {
			resolved.Price = *wardrobeItem.Price
		} else if postType == posttype.Sale {
			return nil, communityerrors.ErrPostItemPriceRequired()
		}

		result = append(result, resolved)
	}

	return result, nil
}

// SyncPostTotalPrice recalculates the total visible price of a post from its current items.
func SyncPostTotalPrice(ctx context.Context, postRepo repositories.IPostRepository, postItemRepo repositories.IPostItemRepository, postID uuid.UUID) error {
	total, err := postItemRepo.SumVisiblePriceByPostID(ctx, postID)
	if err != nil {
		return err
	}

	post, err := postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil {
		return nil
	}

	post.TotalPrice = total
	return postRepo.Update(ctx, post)
}

func toPostSharePath(publicID string) string {
	return "/posts/" + publicID
}

// NormalizePostPublicID validates and normalizes the public identifier used by post routes.
func NormalizePostPublicID(raw string) (string, error) {
	publicID := strings.TrimSpace(raw)
	if !strings.HasPrefix(publicID, postPublicIDPrefix) || len(publicID) != len(postPublicIDPrefix)+16 {
		return "", communityerrors.ErrInvalidPostPublicIDFormat()
	}
	return publicID, nil
}

func isVisiblePostItem(item *entities.PostItem) bool {
	return item != nil && item.Status != postitemstatus.Hidden
}

func getVisiblePostItems(post *entities.Post) []*entities.PostItem {
	if post == nil || len(post.Items) == 0 {
		return nil
	}
	visibleItems := make([]*entities.PostItem, 0, len(post.Items))
	for _, item := range post.Items {
		if isVisiblePostItem(item) {
			visibleItems = append(visibleItems, item)
		}
	}
	return visibleItems
}

// SyncWardrobeStatusByItem derives the wardrobe item status from all active marketplace listings.
func SyncWardrobeStatusByItem(
	ctx context.Context,
	postItemRepo repositories.IPostItemRepository,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	itemID uuid.UUID,
) error {
	postItems, err := postItemRepo.GetActiveByItemID(ctx, itemID)
	if err != nil {
		return err
	}

	nextStatus := wardrobestatus.InWardrobe
	for _, item := range postItems {
		if item.TransferState == transferstate.Accepted || item.Status == postitemstatus.Sold {
			nextStatus = wardrobestatus.Sold
			break
		}
		if item.Status == postitemstatus.Available || item.TransferState == transferstate.Pending {
			nextStatus = wardrobestatus.Selling
		}
	}

	return wardrobeCtr.UpdateItemStatus(ctx, itemID, nextStatus)
}

// UniqueItemIDs extracts distinct wardrobe item IDs from post item records.
func UniqueItemIDs(items []*entities.PostItem) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(items))
	result := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.ItemID]; ok {
			continue
		}
		seen[item.ItemID] = struct{}{}
		result = append(result, item.ItemID)
	}
	return result
}
