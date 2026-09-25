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

// OutfitEventPayload is published when an outfit is deleted
// (topic "wardrobe.outfit.deleted") so consumers (community) can cascade.
type OutfitEventPayload struct {
	OutfitID uuid.UUID `json:"outfitId"`
	UserID   uuid.UUID `json:"userId"`
}
