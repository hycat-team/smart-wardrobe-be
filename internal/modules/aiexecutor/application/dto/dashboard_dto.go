package dto

import "time"

type AIUsageAdminStatsDTO struct {
	SnapshotDate         time.Time `json:"snapshotDate"`
	AIRequestsCount      int       `json:"aiRequestsCount"`
	AIErrorCount         int       `json:"aiErrorCount"`
	AIPaidTokens         int64     `json:"aiPaidTokens"`
	AIActualCostMicroVND int64     `json:"aiActualCostMicroVnd"`
}
