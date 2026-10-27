package dto

import (
	"time"

	"github.com/google/uuid"
)

// StyleSignalInput is the input of IStyleLearningUseCase.ApplySignal (feature
// 033, D7). It is the common shape of every taste signal, so the outfit-save
// worker and the feedback endpoint go through the exact same learning path.
//
// Source carries the signal vocabulary instead of the database code:
// "outfit_saved", "feedback_like", "feedback_dislike" or "feedback_saved". The
// learning use case maps it to the smallint stored in
// fashion.style_signals.source and to the weight used by the decay formula
// (data-model §3).
//
// SignalID is the idempotency key. The outfit outbox event generates it once and
// reuses it on redelivery; the feedback endpoint derives it deterministically
// (D8), so a redelivered message can never be learned twice.
type StyleSignalInput struct {
	SignalID         uuid.UUID
	UserID           uuid.UUID
	Source           string
	OutfitID         *uuid.UUID
	RecommendationID *uuid.UUID
	FashionItemIDs   []uuid.UUID
	OccurredAt       time.Time
}
