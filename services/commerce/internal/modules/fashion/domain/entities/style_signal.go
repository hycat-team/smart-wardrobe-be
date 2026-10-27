package entities

import (
	"time"

	"github.com/google/uuid"
)

// Source values of a style signal. They mirror the CHECK constraint on
// fashion.style_signals.source (1..4).
const (
	StyleSignalSourceOutfitSaved     int16 = 1
	StyleSignalSourceFeedbackLike    int16 = 2
	StyleSignalSourceFeedbackDislike int16 = 3
	StyleSignalSourceFeedbackSaved   int16 = 4
)

// StyleSignal is one append-only taste signal (feature 033) shared by the
// outfit-save pipeline and the like/dislike/saved feedback. ID is the
// idempotency key: the outbox retry and the feedback endpoint reuse it, and the
// row is inserted with ON CONFLICT (id) DO NOTHING.
//
// OutfitID and FashionItemIDs carry no FK on purpose: a signal is history and
// must survive deletion of the outfit or of a fashion item.
type StyleSignal struct {
	ID               uuid.UUID    `gorm:"column:id;type:uuid;primaryKey"`
	UserID           uuid.UUID    `gorm:"column:user_id;type:uuid;not null"`
	Source           int16        `gorm:"column:source;type:smallint;not null"`
	OutfitID         *uuid.UUID   `gorm:"column:outfit_id;type:uuid"`
	RecommendationID *uuid.UUID   `gorm:"column:recommendation_id;type:uuid"`
	FashionItemIDs   UUIDList     `gorm:"column:fashion_item_ids;type:uuid[];not null"`
	StyleSnapshot    JSONDocument `gorm:"column:style_snapshot;type:jsonb"`
	CreatedAt        time.Time    `gorm:"column:created_at;type:timestamp with time zone;not null;default:now()"`
}
