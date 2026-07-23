package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/feedbacksentiment"

	"github.com/google/uuid"
)

type BrandKPICardsRes struct {
	BrandID          uuid.UUID `json:"brandId"`
	TotalMembers     int       `json:"totalMembers"`
	PointsIssued     int       `json:"pointsIssued"`
	PointsRedeemed   int       `json:"pointsRedeemed"`
	ActiveVouchers   int       `json:"activeVouchers"`
	VouchersRedeemed int       `json:"vouchersRedeemed"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type BrandPointsLiabilityRes struct {
	BrandID                 uuid.UUID `json:"brandId"`
	PointsExpiring30d       int       `json:"pointsExpiring30d"`
	PointsLiabilityValueVND float64   `json:"pointsLiabilityValueVnd"`
}

type PairedCategoryStat struct {
	CategoryName string `json:"categoryName"`
	PairCount    int    `json:"pairCount"`
}

type DigitalSampleLabAnalyticsRes struct {
	ItemID              uuid.UUID                           `json:"itemId"`
	VotesCount          int                                 `json:"votesCount"`
	AvgRating           float64                             `json:"avgRating"`
	FeedbackSentiment   feedbacksentiment.FeedbackSentiment `json:"feedbackSentiment"`
	TopPairedCategories []PairedCategoryStat                `json:"topPairedCategories"`
}
