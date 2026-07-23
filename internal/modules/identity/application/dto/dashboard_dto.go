package dto

import "time"

type UserAdminStatsDTO struct {
	SnapshotDate   time.Time `json:"snapshot_date"`
	TotalUsers     int       `json:"total_users"`
	NewUsers       int       `json:"new_users"`
	ActiveUsersDau int       `json:"active_users_dau"`
}
