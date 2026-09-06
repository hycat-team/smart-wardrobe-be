package dto

import (
	"time"

	"github.com/google/uuid"
)

type CustomerAcquisitionDTO struct {
	Source        string `json:"source"`
	CustomerCount int64  `json:"customerCount"`
}

type LoyaltyTierDistributionDTO struct {
	TierID      uuid.UUID `json:"tierId"`
	TierName    string    `json:"tierName"`
	TierRank    int       `json:"tierRank"`
	MemberCount int64     `json:"memberCount"`
}

type PointExpiryForecastDTO struct {
	Month          time.Time `json:"month"`
	PointsExpiring int64     `json:"pointsExpiring"`
}

type BenefitRedemptionDTO struct {
	BenefitID       uuid.UUID `json:"benefitId"`
	BenefitName     string    `json:"benefitName"`
	BenefitType     string    `json:"benefitType"`
	RedemptionCount int64     `json:"redemptionCount"`
	Month           time.Time `json:"month"`
}

type CatalogStatsDTO struct {
	TotalCount int64                 `json:"totalCount"`
	ByStatus   []CatalogStatGroupDTO `json:"byStatus"`
	ByType     []CatalogStatGroupDTO `json:"byType"`
}

type CatalogStatGroupDTO struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type SpendSegmentDTO struct {
	Segment       string `json:"segment"`
	CustomerCount int64  `json:"customerCount"`
}
