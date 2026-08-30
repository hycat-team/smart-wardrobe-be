package dto

import (
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

// Admin stats read models are owned by the domain repository layer; aliased
// here so handlers keep using the dto package (feature 017).
type BrandAdminStatsDTO = repositories.BrandAdminStatsDTO

type BrandAnalyticsStatsDTO = repositories.BrandAnalyticsStatsDTO

type SampleFeedbackDrillDownDTO = repositories.SampleFeedbackDrillDownDTO

type ItemFeedbackSummaryDTO struct {
	VotesCount        int
	AvgRating         float64
	FeedbackSentiment string
	OutfitIDs         []uuid.UUID
}
