package dto

import (
	"time"

	"github.com/google/uuid"
)

type BrandAdminStatsDTO struct {
	SnapshotDate        time.Time `json:"snapshot_date"`
	TotalBrands         int       `json:"total_brands"`
	PendingReviewBrands int       `json:"pending_review_brands"`
	ActiveCampaigns     int       `json:"active_campaigns"`
}

type BrandAnalyticsStatsDTO struct {
	BrandID                   uuid.UUID `json:"brand_id"`
	SnapshotDate              time.Time `json:"snapshot_date"`
	TotalMembers              int       `json:"total_members"`
	PointsIssued              int       `json:"points_issued"`
	PointsRedeemed            int       `json:"points_redeemed"`
	PointsExpiring30d         int       `json:"points_expiring_30d"`
	PointsLiabilityValueVND   float64   `json:"points_liability_value_vnd"`
	ActiveVouchers            int       `json:"active_vouchers"`
	VouchersRedeemed          int       `json:"vouchers_redeemed"`
	DigitalSampleVotesCount   int       `json:"digital_sample_votes_count"`
	DigitalSampleAvgRating    float64   `json:"digital_sample_avg_rating"`
	OpenCSTickets             int       `json:"open_cs_tickets"`
	CSFirstResponseAvgSeconds int       `json:"cs_first_response_avg_seconds"`
}
