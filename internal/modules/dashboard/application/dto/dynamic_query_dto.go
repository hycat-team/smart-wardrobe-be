package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/granularity"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/metrickey"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/targetdashboard"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

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
	Granularity         granularity.Granularity         `json:"granularity"` // "hour" | "day" | "week" | "month"
	Filters             map[string]string               `json:"filters,omitempty"`
	GroupBy             []string                        `json:"groupBy,omitempty"`
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
	CreatedAt    time.Time `json:"createdAt"`
}

type DrillDownAIEventListRes = shared_dto.PaginationResult[DrillDownAIEventDTO]

type DrillDownSampleFeedbackQueryReq struct {
	shared_dto.PaginationQuery
	ItemID string `form:"itemId" json:"itemId"`
}

type DrillDownSampleFeedbackDTO struct {
	FeedbackID string    `json:"feedbackId"`
	ItemID     uuid.UUID `json:"itemId"`
	UserID     uuid.UUID `json:"userId"`
	Rating     float64   `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"createdAt"`
}

type DrillDownSampleFeedbackListRes = shared_dto.PaginationResult[DrillDownSampleFeedbackDTO]
