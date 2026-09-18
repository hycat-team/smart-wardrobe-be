package dto

import "github.com/google/uuid"

// WardrobeEventPayload is the RabbitMQ message payload published on wardrobe
// item lifecycle changes ("created", "updated", "deleted").
type WardrobeEventPayload struct {
	ItemID        uuid.UUID `json:"item_id"`
	UserID        uuid.UUID `json:"user_id"`
	FashionItemID uuid.UUID `json:"fashion_item_id,omitempty"`
	ItemType      int       `json:"item_type,omitempty"`
	Action        string    `json:"action"`
}
