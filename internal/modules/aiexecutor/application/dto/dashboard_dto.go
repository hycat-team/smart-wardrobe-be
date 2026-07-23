package dto

import "time"

type AIUsageAdminStatsDTO struct {
	SnapshotDate         time.Time `json:"snapshot_date"`
	AIRequestsCount      int       `json:"ai_requests_count"`
	AIErrorCount         int       `json:"ai_error_count"`
	AIPaidTokens         int64     `json:"ai_paid_tokens"`
	AIActualCostMicroVND int64     `json:"ai_actual_cost_micro_vnd"`
}
