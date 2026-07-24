package dto

import "time"

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
	CreatedAt    time.Time `json:"createdAt"`
}
