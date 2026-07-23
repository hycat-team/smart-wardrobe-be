package dto

import "time"

type UserAdminStatsDTO struct {
	SnapshotDate   time.Time `json:"snapshotDate"`
	TotalUsers     int       `json:"totalUsers"`
	NewUsers       int       `json:"newUsers"`
	ActiveUsersDau int       `json:"activeUsersDau"`
}
