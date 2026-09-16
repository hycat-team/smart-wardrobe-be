package repositories

import (
	"time"

	"github.com/google/uuid"
)

// Admin read models returned by repository queries. They live in the domain
// layer (as repository read models) so the repository interfaces never
// depend on application/dto (constitution §IV); application/dto aliases
// these types for handler compatibility.
// See feature 017 (codebase hygiene refactor).

type BrandAdminStatsDTO struct {
	SnapshotDate        time.Time `json:"snapshotDate"`
	TotalBrands         int       `json:"totalBrands"`
	PendingReviewBrands int       `json:"pendingReviewBrands"`
}

type BrandAnalyticsStatsDTO struct {
	BrandID                 uuid.UUID `json:"brandId"`
	SnapshotDate            time.Time `json:"snapshotDate"`
	TotalMembers            int       `json:"totalMembers"`
	PointsIssued            int       `json:"pointsIssued"`
	PointsRedeemed          int       `json:"pointsRedeemed"`
	PointsExpiring30d       int       `json:"pointsExpiring30d"`
	PointsLiabilityValueVND float64   `json:"pointsLiabilityValueVnd"`
	ActiveVouchers          int       `json:"activeVouchers"`
	VouchersRedeemed        int       `json:"vouchersRedeemed"`
	DigitalSampleVotesCount int       `json:"digitalSampleVotesCount"`
	DigitalSampleAvgRating  float64   `json:"digitalSampleAvgRating"`
	OpenCSTickets           int       `json:"openCsTickets"`
}

type SampleFeedbackDrillDownDTO struct {
	FeedbackID string    `json:"feedbackId"`
	ItemID     uuid.UUID `json:"itemId"`
	UserID     uuid.UUID `json:"userId"`
	Rating     float64   `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"createdAt"`
}
