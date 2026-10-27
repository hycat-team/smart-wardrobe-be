package dto

import (
	"time"

	"github.com/google/uuid"
)

// OutfitSavedEventPayload is the RabbitMQ payload published on topic
// "wardrobe.outfit.saved" (feature 033) when an outfit is created, or when
// UpdateOutfit adds items. It is consumed by the style learning worker.
//
// SignalID is the idempotency key: the outbox may deliver the message more than
// once, and the consumer inserts fashion.style_signals with id = SignalID using
// ON CONFLICT (id) DO NOTHING. Source is "outfit_saved" for this event.
type OutfitSavedEventPayload struct {
	SignalID       uuid.UUID   `json:"signalId"`
	UserID         uuid.UUID   `json:"userId"`
	OutfitID       uuid.UUID   `json:"outfitId"`
	FashionItemIDs []uuid.UUID `json:"fashionItemIds"`
	Source         string      `json:"source"`
	OccurredAt     time.Time   `json:"occurredAt"`
}
