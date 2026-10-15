package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

// GetActivityEventsQueryReq is the query string of
// `GET /api/v1/admin/activity-events` (contracts/admin-api-delta.md §1).
//
// The identifiers and dates stay strings here on purpose: turning them into a
// filter (UUID, the two allowed activity types, the VN calendar range) is the
// use case's job, next to the query it constrains, so a non-HTTP caller gets
// the same 400s without going through Gin's binder.
type GetActivityEventsQueryReq struct {
	shared_dto.PaginationQuery
	UserID       string `form:"userId"`
	ActivityType string `form:"activityType"`
	FromDate     string `form:"fromDate"`
	ToDate       string `form:"toDate"`
}

// ActivityEventRes is one activity event with the account identity the admin
// list shows. Field names follow the UserRes convention (dto/identity/profile.go):
// `email`, `firstName`, `lastName` become `userEmail`, `userFirstName`,
// `userLastName` so the flat item stays unambiguous next to the event itself.
type ActivityEventRes struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"userId"`
	ActivityType string    `json:"activityType"`
	OccurredAt   time.Time `json:"occurredAt"`
	CreatedAt    time.Time `json:"createdAt"`
	UserEmail    string    `json:"userEmail"`
	// UserFirstName/UserLastName are absent when the account never set them
	// (NULL), while UserEmail is always present — the LEFT JOIN keeps the
	// event visible even for a soft-deleted account.
	UserFirstName *string `json:"userFirstName,omitempty"`
	UserLastName  *string `json:"userLastName,omitempty"`
}

// ActivityEventListRes is the paginated envelope: same shape as every other
// admin list (items + metadata).
type ActivityEventListRes = shared_dto.PaginationResult[*ActivityEventRes]
