package mapper

import (
	dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
)

// MapToActivityEventRes maps one admin activity read-model row onto the HTTP
// response item (029 R3). Nullable identity fields stay pointers, so the JSON
// omits them exactly when the account never set them.
func MapToActivityEventRes(row repositories.AdminActivityEventRow) *dto.ActivityEventRes {
	return &dto.ActivityEventRes{
		ID:            row.ID,
		UserID:        row.UserID,
		ActivityType:  row.ActivityType,
		OccurredAt:    row.OccurredAt,
		CreatedAt:     row.CreatedAt,
		UserEmail:     row.Email,
		UserFirstName: row.FirstName,
		UserLastName:  row.LastName,
	}
}
