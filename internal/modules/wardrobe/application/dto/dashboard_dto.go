package dto

import (
	"github.com/google/uuid"
)

type UnderutilizedItemInfoDTO struct {
	ItemID           uuid.UUID `json:"itemId"`
	Name             string    `json:"name"`
	CategoryName     string    `json:"categoryName"`
	PurchasePriceVND float64   `json:"purchasePriceVnd"`
	WearCount        int       `json:"wearCount"`
	CostPerWearVND   float64   `json:"costPerWearVnd"`
	LastWornDaysAgo  int       `json:"lastWornDaysAgo"`
	ImageURL         string    `json:"imageUrl"`
}

type UserWardrobeSummaryStatsDTO struct {
	UserID                  uuid.UUID `json:"userId"`
	TotalItems              int       `json:"totalItems"`
	TotalWardrobeValueVND   float64   `json:"totalWardrobeValueVnd"`
	AverageCostPerWearVND   float64   `json:"averageCostPerWearVnd"`
	UnderutilizedItemsCount int       `json:"underutilizedItemsCount"`
}
