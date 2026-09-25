package dto

import "github.com/google/uuid"

// OutfitDeletedEventPayload mirrors the wardrobe event payload published on
// topic "wardrobe.outfit.deleted".
type OutfitDeletedEventPayload struct {
	OutfitID uuid.UUID `json:"outfitId"`
}

// UserStatusChangedEventPayload mirrors the identity event published when an
// admin activates/deactivates a user.
type UserStatusChangedEventPayload struct {
	UserID uuid.UUID `json:"userId"`
	Active bool      `json:"active"`
}
