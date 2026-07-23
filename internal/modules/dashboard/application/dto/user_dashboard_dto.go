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
	AverageItemValueVND     float64                `json:"averageItemValueVnd"`
	UnderutilizedItemsCount int                    `json:"underutilizedItemsCount"`
	UnderutilizedItems      []UnderutilizedItemDTO `json:"underutilizedItems"`
	AIQuotaRemaining        int                    `json:"aiQuotaRemaining"`
	ExpiringVouchersCount   int                    `json:"expiringVouchersCount"`
}
