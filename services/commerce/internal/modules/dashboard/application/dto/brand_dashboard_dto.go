package dto

import (
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/feedbacksentiment"

	"github.com/google/uuid"
)

type GetBrandDashboardQueryReq struct {
	BrandID string `form:"brandId" json:"brandId" binding:"required"`
}

type BrandKPICardsRes struct {
	TotalMembers            int     `json:"totalMembers"`
	PointsIssued            int     `json:"pointsIssued"`
	PointsRedeemed          int     `json:"pointsRedeemed"`
	ActiveVouchers          int     `json:"activeVouchers"`
	VouchersRedeemed        int     `json:"vouchersRedeemed"`
	DigitalSampleVotesCount int     `json:"digitalSampleVotesCount"`
	DigitalSampleAvgRating  float64 `json:"digitalSampleAvgRating"`
	OpenCSTickets           int     `json:"openCsTickets"`
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
	BrandItemID         *uuid.UUID                          `json:"brandItemId,omitempty"`
	BrandItemName       *string                             `json:"brandItemName,omitempty"`
	BrandItemImageURL   *string                             `json:"brandItemImageUrl,omitempty"`
	VotesCount          int                                 `json:"votesCount"`
	AvgRating           float64                             `json:"avgRating"`
	FeedbackSentiment   feedbacksentiment.FeedbackSentiment `json:"feedbackSentiment"`
	TopPairedCategories []PairedCategoryStat                `json:"topPairedCategories"`
}

// Customer Acquisition

type CustomerAcquisitionDTO struct {
	Source        string  `json:"source"`
	CustomerCount int     `json:"customerCount"`
	Percentage    float64 `json:"percentage"`
}

type CustomerAcquisitionRes struct {
	BrandID        uuid.UUID                `json:"brandId"`
	TotalCustomers int                      `json:"totalCustomers"`
	Sources        []CustomerAcquisitionDTO `json:"sources"`
}

// Loyalty Tier Distribution

type TierDistributionDTO struct {
	TierID      uuid.UUID `json:"tierId"`
	TierName    string    `json:"tierName"`
	TierRank    int       `json:"tierRank"`
	MemberCount int       `json:"memberCount"`
	Percentage  float64   `json:"percentage"`
}

type TierDistributionRes struct {
	BrandID      uuid.UUID             `json:"brandId"`
	TotalMembers int                   `json:"totalMembers"`
	Tiers        []TierDistributionDTO `json:"tiers"`
}

// Point Expiry Forecast

type PointExpiryForecastDTO struct {
	Month          string `json:"month"`
	PointsExpiring int    `json:"pointsExpiring"`
}

type PointExpiryForecastRes struct {
	BrandID  uuid.UUID                `json:"brandId"`
	Forecast []PointExpiryForecastDTO `json:"forecast"`
}

// Benefit Redemption Analytics

type RedemptionTrendDTO struct {
	Month           string `json:"month"`
	RedemptionCount int    `json:"redemptionCount"`
}

type BenefitRedemptionDTO struct {
	BenefitID       uuid.UUID `json:"benefitId"`
	BenefitName     string    `json:"benefitName"`
	BenefitType     string    `json:"benefitType"`
	RedemptionCount int       `json:"redemptionCount"`
	Percentage      float64   `json:"percentage"`
}

type BenefitRedemptionAnalyticsRes struct {
	BrandID          uuid.UUID              `json:"brandId"`
	TotalRedemptions int                    `json:"totalRedemptions"`
	Trend            []RedemptionTrendDTO   `json:"trend"`
	TopBenefits      []BenefitRedemptionDTO `json:"topBenefits"`
}

// Catalog Stats

type CatalogStatGroupDTO struct {
	Label      string  `json:"label"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CatalogStatsRes struct {
	BrandID    uuid.UUID             `json:"brandId"`
	TotalItems int                   `json:"totalItems"`
	ByStatus   []CatalogStatGroupDTO `json:"byStatus"`
	ByType     []CatalogStatGroupDTO `json:"byType"`
}

// Customer Spend Segmentation

type SpendSegmentDTO struct {
	Segment       string   `json:"segment"`
	MinVND        float64  `json:"minVnd"`
	MaxVND        *float64 `json:"maxVnd,omitempty"`
	CustomerCount int      `json:"customerCount"`
	Percentage    float64  `json:"percentage"`
}

type SpendSegmentationRes struct {
	BrandID        uuid.UUID         `json:"brandId"`
	TotalCustomers int               `json:"totalCustomers"`
	Segments       []SpendSegmentDTO `json:"segments"`
}
