package dto

import (
	"time"

	"github.com/google/uuid"
)

type BrandAdminStatsDTO struct {
	SnapshotDate        time.Time `json:"snapshotDate"`
	TotalBrands         int       `json:"totalBrands"`
	PendingReviewBrands int       `json:"pendingReviewBrands"`
	ActiveCampaigns     int       `json:"activeCampaigns"`
}

type BrandAnalyticsStatsDTO struct {
	BrandID                   uuid.UUID `json:"brandId"`
	SnapshotDate              time.Time `json:"snapshotDate"`
	TotalMembers              int       `json:"totalMembers"`
	PointsIssued              int       `json:"pointsIssued"`
	PointsRedeemed            int       `json:"pointsRedeemed"`
	PointsExpiring30d         int       `json:"pointsExpiring30d"`
	PointsLiabilityValueVND   float64   `json:"pointsLiabilityValueVnd"`
	ActiveVouchers            int       `json:"activeVouchers"`
	VouchersRedeemed          int       `json:"vouchersRedeemed"`
	DigitalSampleVotesCount   int       `json:"digitalSampleVotesCount"`
	DigitalSampleAvgRating    float64   `json:"digitalSampleAvgRating"`
	OpenCSTickets             int       `json:"openCsTickets"`
	CSFirstResponseAvgSeconds int       `json:"csFirstResponseAvgSeconds"`
}
