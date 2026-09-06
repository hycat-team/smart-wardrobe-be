package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type RecordAuditLogInput struct {
	ActorUserID uuid.UUID      `json:"actorUserId"`
	Action      string         `json:"action"`
	TargetType  string         `json:"targetType"`
	TargetID    *uuid.UUID     `json:"targetId,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	RequestIP   *string        `json:"requestIp,omitempty"`
}

type AdminAuditLogDTO struct {
	ID          uuid.UUID      `json:"id"`
	ActorUserID uuid.UUID      `json:"actorUserId"`
	Action      string         `json:"action"`
	TargetType  string         `json:"targetType"`
	TargetID    *uuid.UUID     `json:"targetId,omitempty"`
	Payload     map[string]any `json:"payload"`
	RequestIP   *string        `json:"requestIp,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type GetAuditLogsQueryReq struct {
	shared_dto.PaginationQuery
	ActorUserID *string `form:"actorUserId"`
	Action      *string `form:"action"`
	TargetType  *string `form:"targetType"`
	TargetID    *string `form:"targetId"`
	FromDate    *string `form:"fromDate"`
	ToDate      *string `form:"toDate"`
}

type AdminAuditLogListRes = shared_dto.PaginationResult[*AdminAuditLogDTO]

// AuditLogFilter is the query filter for listing audit logs.
// Defined here (application layer) so presentation and application layers
// can reference it without importing domain/repositories.
type AuditLogFilter struct {
	ActorUserID *uuid.UUID
	Action      *string
	TargetType  *string
	TargetID    *uuid.UUID
	FromDate    *time.Time
	ToDate      *time.Time
}
