package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

// IRecommendationFeedbackUseCase accepts the feedback of one recommendation
// (feature 033, US6).
//
// The client only sends the recommendation id and an action. Ownership and the
// item ids come from the server-side snapshot (plan D8), so a client can never
// feed arbitrary items into the learning pipeline.
type IRecommendationFeedbackUseCase interface {
	// SubmitFeedback authenticates the snapshot, maps the action to a learning
	// source and applies the signal. It is idempotent by a deterministic signal
	// id, so submitting the same action twice learns it once (FR-037).
	//
	// There is deliberately no entitlement check here: the signal is stored for
	// every user, while the recommendation pipeline only applies learned data
	// for entitled users (FR-042/FR-053).
	SubmitFeedback(
		ctx context.Context,
		userID uuid.UUID,
		input dto.SubmitRecommendationFeedbackReq,
	) (*dto.SubmitRecommendationFeedbackRes, error)
}
