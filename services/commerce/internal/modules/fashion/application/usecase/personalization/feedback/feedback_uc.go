// Package feedback owns the recommendation feedback use case of feature 033
// (US6): it authenticates the recommendation snapshot, derives a deterministic
// signal id and hands the signal to the shared style learning use case.
//
// The endpoint deliberately trusts nothing but the recommendation id and the
// action: item ids and style labels come from the server-side snapshot, so a
// client cannot feed another user's items or arbitrary items into the learning
// pipeline (plan D8).
package feedback

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	uc_caching "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/learning"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Feedback actions carried by the request body (contracts §2.2).
const (
	feedbackActionLike    = "like"
	feedbackActionDislike = "dislike"
	feedbackActionSaved   = "saved"
)

// feedbackSignalSources maps an action to the learning source vocabulary. A
// "saved" feedback is a light signal (0.25) because the same outfit very likely
// already produced an "outfit_saved" one (data-model §3, plan D8).
var feedbackSignalSources = map[string]string{
	feedbackActionLike:    learning.SignalSourceFeedbackLike,
	feedbackActionDislike: learning.SignalSourceFeedbackDislike,
	feedbackActionSaved:   learning.SignalSourceFeedbackSaved,
}

// RecommendationFeedbackUseCase implements IRecommendationFeedbackUseCase.
type RecommendationFeedbackUseCase struct {
	logger          logger.Interface
	snapshotCache   uc_caching.IRecommendationSnapshotCache
	learningUseCase uc_interfaces.IStyleLearningUseCase
}

func NewRecommendationFeedbackUseCase(
	l logger.Interface,
	snapshotCache uc_caching.IRecommendationSnapshotCache,
	learningUseCase uc_interfaces.IStyleLearningUseCase,
) uc_interfaces.IRecommendationFeedbackUseCase {
	return &RecommendationFeedbackUseCase{
		logger:          l,
		snapshotCache:   snapshotCache,
		learningUseCase: learningUseCase,
	}
}

// SubmitFeedback authenticates the snapshot then applies one learning signal.
//
// Order of operations:
//  1. validate the action before touching the cache, so an invalid body is
//     rejected without a Redis round trip;
//  2. load the snapshot; a missing or expired snapshot answers 404, a backend
//     failure degrades to the same 404 (the port contract of plan D8);
//  3. refuse a snapshot owned by another user with 403;
//  4. derive the deterministic signal id - the same (user, recommendation,
//     action) always yields the same id, which is what makes a duplicate
//     submission learn once;
//  5. apply the signal through the shared learning use case. There is no
//     entitlement gate: the learning use case stores the signal for every user
//     and the recommendation helper only applies learned data when entitled
//     (FR-042/FR-053).
func (uc *RecommendationFeedbackUseCase) SubmitFeedback(
	ctx context.Context,
	userID uuid.UUID,
	input dto.SubmitRecommendationFeedbackReq,
) (*dto.SubmitRecommendationFeedbackRes, error) {
	action := strings.ToLower(strings.TrimSpace(input.Action))
	source, ok := feedbackSignalSources[action]
	if !ok {
		return nil, fashioerrors.ErrFeedbackActionInvalid()
	}
	if input.RecommendationID == uuid.Nil || uc.snapshotCache == nil {
		return nil, fashioerrors.ErrRecommendationSnapshotNotFound()
	}

	snapshot, err := uc.snapshotCache.Get(ctx, input.RecommendationID)
	if err != nil {
		// The Redis port already maps backend errors to a miss; a caller that
		// still returns an error is treated the same way so infrastructure
		// detail never leaks and a cache outage cannot 500 the endpoint.
		uc.logWarn("Load recommendation snapshot failed",
			zap.String("recommendation_id", input.RecommendationID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, fashioerrors.ErrRecommendationSnapshotNotFound()
	}
	if snapshot == nil || snapshot.UserID == uuid.Nil || len(snapshot.FashionItemIDs) == 0 {
		return nil, fashioerrors.ErrRecommendationSnapshotNotFound()
	}
	if snapshot.UserID != userID {
		return nil, fashioerrors.ErrFeedbackForbidden()
	}

	recommendationID := input.RecommendationID
	signal := dto.StyleSignalInput{
		SignalID:         deterministicFeedbackSignalID(userID, recommendationID, action),
		UserID:           userID,
		Source:           source,
		RecommendationID: &recommendationID,
		FashionItemIDs:   snapshot.FashionItemIDs,
		OccurredAt:       time.Now().UTC(),
	}
	if err := uc.learningUseCase.ApplySignal(ctx, signal); err != nil {
		return nil, err
	}

	return &dto.SubmitRecommendationFeedbackRes{Accepted: true}, nil
}

// deterministicFeedbackSignalID is the idempotency key of one feedback action
// (plan D8): hash(userId + recommendationId + action). The delimiter keeps the
// concatenation unambiguous.
func deterministicFeedbackSignalID(
	userID uuid.UUID,
	recommendationID uuid.UUID,
	action string,
) uuid.UUID {
	payload := userID.String() + "|" + recommendationID.String() + "|" + action
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(payload))
}

func (uc *RecommendationFeedbackUseCase) logWarn(message string, fields ...zap.Field) {
	if uc == nil || uc.logger == nil {
		return
	}
	uc.logger.Warn(message, fields...)
}
