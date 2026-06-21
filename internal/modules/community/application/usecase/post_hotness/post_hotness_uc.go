package post_hotness

import (
	"context"
	"math"
	"time"

	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	hotnessGravity          = 1.5
	batchSize               = 500
	decayWindow             = 3 * 24 * time.Hour
	highScoreStaleThreshold = 0.10
)

type PostHotnessUseCase struct {
	postRepo      repositories.IPostRepository
	postScoreRepo repositories.IPostScoreRepository
	log           logger.Interface
}

func NewPostHotnessUseCase(
	postRepo repositories.IPostRepository,
	postScoreRepo repositories.IPostScoreRepository,
	log logger.Interface,
) usecase_interfaces.IPostHotnessUseCase {
	return &PostHotnessUseCase{
		postRepo:      postRepo,
		postScoreRepo: postScoreRepo,
		log:           log,
	}
}

func (uc *PostHotnessUseCase) RefreshHotness(ctx context.Context) error {
	startedAt := time.Now()
	recentSince := startedAt.Add(-decayWindow)
	staleBefore := startedAt.Add(-decayWindow)

	dirtyIDs, err := uc.postRepo.GetDirtyPostIDs(ctx, batchSize)
	if err != nil {
		uc.log.Error("Failed to load dirty post IDs", zap.Error(err))
		return err
	}

	recentDecayIDs, err := uc.postRepo.GetDecayRefreshPostIDs(ctx, recentSince, batchSize)
	if err != nil {
		uc.log.Error("Failed to load recent decay post IDs", zap.Error(err))
		return err
	}

	highScoreStaleIDs, err := uc.postRepo.GetHighScoreStalePostIDs(ctx, staleBefore, highScoreStaleThreshold, batchSize)
	if err != nil {
		uc.log.Error("Failed to load high-score stale post IDs", zap.Error(err))
		return err
	}

	dedupedIDs := dedupePostIDs(dirtyIDs, recentDecayIDs, highScoreStaleIDs)
	if len(dedupedIDs) == 0 {
		uc.log.Info("[PostHotnessUseCase] Job succeeded",
			zap.Int("dirty_selected_count", len(dirtyIDs)),
			zap.Int("recent_decay_selected_count", len(recentDecayIDs)),
			zap.Int("high_score_stale_selected_count", len(highScoreStaleIDs)),
			zap.Int("processed_count", 0),
			zap.Duration("duration", time.Since(startedAt)),
		)
		return nil
	}

	metrics, err := uc.postScoreRepo.ListScoreMetricsByPostIDs(ctx, dedupedIDs)
	if err != nil {
		uc.log.Error("Failed to load post score metrics by post IDs", zap.Error(err))
		return err
	}

	snapshots := make([]*entities.PostScoreSnapshot, 0, len(metrics))
	for _, metric := range metrics {
		createdAt := time.Unix(metric.CreatedAtUnix, 0)
		ageInHours := startedAt.Sub(createdAt).Hours()
		score := ((float64(metric.LikeCount) * 1) + (float64(metric.CommentCount) * 2) - 1) / math.Pow(ageInHours+2, hotnessGravity)

		snapshots = append(snapshots, &entities.PostScoreSnapshot{
			PostID:             metric.PostID,
			GlobalHotnessScore: score,
			LastCalculatedAt:   startedAt,
		})
	}

	if err := uc.postScoreRepo.UpsertScores(ctx, snapshots); err != nil {
		uc.log.Error("Failed to upsert post hotness scores", zap.Error(err))
		return err
	}

	if err := uc.postRepo.ClearHotnessDirty(ctx, dirtyIDs); err != nil {
		uc.log.Error("Failed to clear hotness dirty flags", zap.Error(err))
		return err
	}

	uc.log.Info(
		"[PostHotnessUseCase] Job succeeded",
		zap.Int("dirty_selected_count", len(dirtyIDs)),
		zap.Int("recent_decay_selected_count", len(recentDecayIDs)),
		zap.Int("high_score_stale_selected_count", len(highScoreStaleIDs)),
		zap.Int("processed_count", len(metrics)),
		zap.Int("upserted_count", len(snapshots)),
		zap.Int("dirty_cleared_count", len(dirtyIDs)),
		zap.Duration("duration", time.Since(startedAt)),
	)
	return nil
}

func dedupePostIDs(groups ...[]uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	result := make([]uuid.UUID, 0)
	for _, group := range groups {
		for _, postID := range group {
			if _, exists := seen[postID]; exists {
				continue
			}
			seen[postID] = struct{}{}
			result = append(result, postID)
		}
	}
	return result
}
