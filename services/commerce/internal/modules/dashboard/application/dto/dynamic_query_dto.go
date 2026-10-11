package dto

import (
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/granularity"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/metrickey"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/targetdashboard"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type DynamicQueryTimeframeDTO struct {
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

type DynamicQueryReq struct {
	TargetDashboard     targetdashboard.TargetDashboard `json:"targetDashboard"` // "admin" | "brand" | "user"
	BrandID             *uuid.UUID                      `json:"brandId,omitempty"`
	Metrics             []metrickey.MetricKey           `json:"metrics"`
	Timeframe           DynamicQueryTimeframeDTO        `json:"timeframe"`
	Granularity         granularity.Granularity         `json:"granularity"` // "day" | "week" | "month" | "year"
	CompareWithPrevious bool                            `json:"compareWithPrevious"`
}

type TimeSeriesPointDTO struct {
	Timestamp time.Time                       `json:"timestamp"`
	Metrics   map[metrickey.MetricKey]float64 `json:"metrics"`
}

type ComparisonDTO struct {
	PreviousPeriodFrom     string                          `json:"previousPeriodFrom"`
	PreviousPeriodTo       string                          `json:"previousPeriodTo"`
	MetricDeltaPercentages map[metrickey.MetricKey]float64 `json:"metricDeltaPercentages"`
}

type DynamicQueryRes struct {
	TargetDashboard targetdashboard.TargetDashboard `json:"targetDashboard"`
	Granularity     granularity.Granularity         `json:"granularity"`
	Series          []TimeSeriesPointDTO            `json:"series"`
	Comparison      *ComparisonDTO                  `json:"comparison,omitempty"`
}

// Drill-Down DTOs

type DrillDownAIEventQueryReq struct {
	shared_dto.PaginationQuery
	Status string `form:"status" json:"status"`
}

type DrillDownAIEventDTO struct {
	RequestID    string    `json:"requestId"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	StatusCode   int       `json:"statusCode"`
	ErrorMessage string    `json:"errorMessage"`
	ErrorCode    string    `json:"errorCode"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DrillDownAIEventListRes = shared_dto.PaginationResult[DrillDownAIEventDTO]

type DrillDownSampleFeedbackQueryReq struct {
	shared_dto.PaginationQuery
	ItemID string `form:"itemId" json:"itemId"`
}

type DrillDownSampleFeedbackDTO struct {
	FeedbackID        string                     `json:"feedbackId"`
	ItemID            uuid.UUID                  `json:"itemId"`
	UserID            uuid.UUID                  `json:"userId"`
	Rating            float64                    `json:"rating"`
	Comment           string                     `json:"comment"`
	CreatedAt         time.Time                  `json:"createdAt"`
	User              *identity_dto.UserBriefRes `json:"user,omitempty"`
	BrandItemName     *string                    `json:"brandItemName,omitempty"`
	BrandItemImageURL *string                    `json:"brandItemImageUrl,omitempty"`
}

type DrillDownSampleFeedbackListRes = shared_dto.PaginationResult[DrillDownSampleFeedbackDTO]
