package dto

// SetAutoRenewReq represents the request to update auto renew status
type SetAutoRenewReq struct {
	Enabled *bool `json:"enabled" binding:"required" label:"trạng thái tự động gia hạn"`
}
