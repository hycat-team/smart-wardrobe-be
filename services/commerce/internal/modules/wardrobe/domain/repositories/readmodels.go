package repositories

import "github.com/google/uuid"

// UserWardrobeSummaryStatsDTO is a repository read model owned by the domain
// layer (feature 017); application/dto aliases it for handler compatibility.
type UserWardrobeSummaryStatsDTO struct {
	UserID                  uuid.UUID `json:"userId"`
	TotalItems              int       `json:"totalItems"`
	TotalWardrobeValueVND   float64   `json:"totalWardrobeValueVnd"`
	AverageCostPerWearVND   float64   `json:"averageCostPerWearVnd"`
	UnderutilizedItemsCount int       `json:"underutilizedItemsCount"`
}
