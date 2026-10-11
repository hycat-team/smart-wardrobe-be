package repositories

import "time"

// Admin AI usage read models owned by the domain layer (feature 017);
// application/dto aliases them for handler compatibility.
type AIUsageAdminStatsDTO struct {
	SnapshotDate         time.Time `json:"snapshotDate"`
	AIRequestsCount      int       `json:"aiRequestsCount"`
	AIErrorCount         int       `json:"aiErrorCount"`
	AIPaidTokens         int64     `json:"aiPaidTokens"`
	AIActualCostMicroVND int64     `json:"aiActualCostMicroVnd"`
}

type AIEventDrillDownDTO struct {
	RequestID    string    `json:"requestId"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	StatusCode   int       `json:"statusCode"`
	ErrorMessage string    `json:"errorMessage"`
	ErrorCode    string    `json:"errorCode"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AIUsageByOperationDTO struct {
	Operation         string `json:"operation"`
	LogicalRoute      string `json:"logicalRoute"`
	RequestCount      int64  `json:"requestCount"`
	TotalCostMicroVND int64  `json:"totalCostMicroVnd"`
	TotalPromptTokens int64  `json:"totalPromptTokens"`
	TotalOutputTokens int64  `json:"totalOutputTokens"`
}

type AIErrorBreakdownDTO struct {
	ErrorCode  string `json:"errorCode"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	ErrorCount int64  `json:"errorCount"`
}
