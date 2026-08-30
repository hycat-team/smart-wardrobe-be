package dto

import (
	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
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

type UserWardrobeSummaryStatsDTO = repositories.UserWardrobeSummaryStatsDTO
