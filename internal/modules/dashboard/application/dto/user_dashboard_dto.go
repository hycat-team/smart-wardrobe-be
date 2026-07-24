package dto

import (
	"github.com/google/uuid"
)

type UnderutilizedItemDTO struct {
	ItemID           uuid.UUID `json:"itemId"`
	Name             string    `json:"name"`
	CategoryName     string    `json:"categoryName"`
	PurchasePriceVND float64   `json:"purchasePriceVnd"`
	LastWornDaysAgo  int       `json:"lastWornDaysAgo"`
	ImageURL         string    `json:"imageUrl"`
}

type UserWardrobeInsightsRes struct {
	TotalItems              int                    `json:"totalItems"`
	TotalWardrobeValueVND   float64                `json:"totalWardrobeValueVnd"`
	UnderutilizedItemsCount int                    `json:"underutilizedItemsCount"`
	UnderutilizedItems      []UnderutilizedItemDTO `json:"underutilizedItems"`
	ExpiringVouchersCount   int                    `json:"expiringVouchersCount"`
}

// Wardrobe Category Distribution

type CategoryDistributionDTO struct {
	CategoryID   uuid.UUID `json:"categoryId"`
	CategoryName string    `json:"categoryName"`
	ItemCount    int       `json:"itemCount"`
	Percentage   float64   `json:"percentage"`
}

type WardrobeCategoryDistributionRes struct {
	TotalItems int                       `json:"totalItems"`
	Categories []CategoryDistributionDTO `json:"categories"`
}
