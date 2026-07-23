package dto

import (
	sharedEntities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type UserWardrobeSummaryStatsDTO struct {
	UserID                  uuid.UUID                   `json:"user_id"`
	TotalItems              int                         `json:"total_items"`
	TotalWardrobeValueVND   float64                     `json:"total_wardrobe_value_vnd"`
	AverageCostPerWearVND   float64                     `json:"average_cost_per_wear_vnd"`
	UnderutilizedItemsCount int                         `json:"underutilized_items_count"`
	CategoryDistribution    sharedEntities.JSONDocument `json:"category_distribution"`
	ColorDistribution       sharedEntities.JSONDocument `json:"color_distribution"`
}
