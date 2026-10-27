package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
)

// IStyleLearningUseCase applies one taste signal to the derived style profile
// (feature 033, US4/US6). The outfit-save worker and the feedback endpoint share
// this use case so a "saved" feedback is learned through the same decay path as
// an outfit save (D7).
type IStyleLearningUseCase interface {
	// ApplySignal records the signal once and updates learned_styles/learned_at.
	//
	// It is idempotent by SignalID: a redelivered outbox message is dropped
	// before the profile is touched, so the decay is never applied twice
	// (FR-037). The signal row and the profile update commit in the same
	// transaction, so a failure leaves no half-learned state.
	//
	// When personalization.enabled is false the call is a no-op that returns
	// nil, which lets the worker ack the message without writing anything
	// (kill switch, contracts §8). Explicitly declared preferences
	// (styles/excluded_styles/enabled) are never modified.
	ApplySignal(ctx context.Context, input dto.StyleSignalInput) error
}
