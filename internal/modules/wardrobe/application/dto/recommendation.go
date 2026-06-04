package dto

import "github.com/google/uuid"

type RecommendOutfitReq struct {
	Occasion    *string `json:"occasion"`
	StyleTarget *string `json:"styleTarget"`
	Season      *string `json:"season"`
	Weather     *string `json:"weather"`
	Details     *string `json:"details"`
}

type RecommendedOutfitRes struct {
	Title       string                  `json:"title"`
	Explanation string                  `json:"explanation"`
	Items       []*RecommendedItemGroup `json:"items"`
}

type RecommendedItemGroup struct {
	Role         string             `json:"role"`
	Primary      *WardrobeItemRes   `json:"primary"`
	Alternatives []*WardrobeItemRes `json:"alternatives"`
}

type PendingTransferRes struct {
	PostItemID uuid.UUID        `json:"postItemId"`
	Item       *WardrobeItemRes `json:"item"`
	SellerName string           `json:"sellerName"`
}
