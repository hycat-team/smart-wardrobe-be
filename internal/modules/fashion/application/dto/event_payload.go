package dto

import (
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"

	"github.com/google/uuid"
)

// SSEEventPayload is the Server-Sent Events frame streamed to clients during
// batch fashion analysis.
type SSEEventPayload struct {
	ItemID uuid.UUID                           `json:"itemId"`
	Status analyzetaskstatus.AnalyzeTaskStatus `json:"status"` // "completed" | "failed" | "needs_review"
	Total  int                                 `json:"total"`
	Index  int                                 `json:"index"`
	Data   any                                 `json:"data,omitempty"`
	Error  string                              `json:"error,omitempty"`
}

// FashionAnalyzedEventPayload is the RabbitMQ message payload published when a
// fashion item finishes AI analysis (consumed by wardrobe/brand workers).
type FashionAnalyzedEventPayload struct {
	TaskID        string                              `json:"taskId"`
	ItemID        uuid.UUID                           `json:"itemId"`
	ItemType      string                              `json:"itemType"` // "wardrobe" | "brand"
	FashionItemID *uuid.UUID                          `json:"fashionItemId,omitempty"`
	Status        analyzetaskstatus.AnalyzeTaskStatus `json:"status"` // "completed" | "needs_review" | "failed"
	Total         int                                 `json:"total"`
	Index         int                                 `json:"index"`
	FashionBrief  *FashionItemBriefRes                `json:"fashionBrief,omitempty"`
	Error         string                              `json:"error,omitempty"`
}

// FashionItemCreatedEventPayload is the RabbitMQ message payload published when
// a fashion item record is created (consumed by wardrobe/brand bind workers).
type FashionItemCreatedEventPayload struct {
	ItemID        uuid.UUID `json:"itemId"`
	FashionItemID uuid.UUID `json:"fashionItemId"`
	ItemType      string    `json:"itemType"` // "wardrobe" | "brand"
	UserID        uuid.UUID `json:"userId,omitempty"`
}

// FashionProcessingExhaustedEventPayload is the RabbitMQ message payload
// published when AI analysis retries are exhausted (consumed by wardrobe/brand
// processing-exhausted workers).
type FashionProcessingExhaustedEventPayload struct {
	WardrobeItemID uuid.UUID `json:"wardrobeItemId"`
	FashionItemID  uuid.UUID `json:"fashionItemId"`
	Reason         string    `json:"reason"`
	UserID         uuid.UUID `json:"userId,omitempty"`
}
