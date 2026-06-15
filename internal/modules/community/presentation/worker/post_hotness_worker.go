package worker

import (
	"context"
	"math"
	"time"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	hotnessGravity          = 1.5
	batchSize               = 500
	decayWindow             = 3 * 24 * time.Hour
	highScoreStaleThreshold = 0.10
)

type IPostHotnessWorker interface {
	Start()
	Stop()
}

type PostHotnessWorker struct {
	postRepo      repositories.IPostRepository
	postScoreRepo repositories.IPostScoreRepository
	cronEngine    *cron.Cron
	log           logger.Interface
}

func NewPostHotnessWorker(
	postRepo repositories.IPostRepository,
	postScoreRepo repositories.IPostScoreRepository,
	log logger.Interface,
) IPostHotnessWorker {
	return &PostHotnessWorker{
		postRepo:      postRepo,
		postScoreRepo: postScoreRepo,
		cronEngine:    cron.New(cron.WithSeconds()),
		log:           log,
	}
}

func (w *PostHotnessWorker) Start() {
	go w.executeRefresh()

	_, err := w.cronEngine.AddFunc("0 */10 * * * *", w.executeRefresh)
	if err != nil {
		w.log.Error("Failed to register community hotness worker", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

func (w *PostHotnessWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
}

func (w *PostHotnessWorker) executeRefresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startedAt := time.Now()
	recentSince := startedAt.Add(-decayWindow)
	staleBefore := startedAt.Add(-decayWindow)

	dirtyIDs, err := w.postRepo.GetDirtyPostIDs(ctx, batchSize)
	if err != nil {
		w.log.Error("Failed to load dirty post IDs", zap.Error(err))
		return
	}

	recentDecayIDs, err := w.postRepo.GetDecayRefreshPostIDs(ctx, recentSince, batchSize)
	if err != nil {
		w.log.Error("Failed to load recent decay post IDs", zap.Error(err))
		return
	}

	highScoreStaleIDs, err := w.postRepo.GetHighScoreStalePostIDs(ctx, staleBefore, highScoreStaleThreshold, batchSize)
	if err != nil {
		w.log.Error("Failed to load high-score stale post IDs", zap.Error(err))
		return
	}

	dedupedIDs := dedupePostIDs(dirtyIDs, recentDecayIDs, highScoreStaleIDs)
	if len(dedupedIDs) == 0 {
		w.log.Info("[PostHotnessWorker] Job succeeded",
			zap.Int("dirty_selected_count", len(dirtyIDs)),
			zap.Int("recent_decay_selected_count", len(recentDecayIDs)),
			zap.Int("high_score_stale_selected_count", len(highScoreStaleIDs)),
			zap.Int("processed_count", 0),
			zap.Duration("duration", time.Since(startedAt)),
		)
		return
	}

	metrics, err := w.postScoreRepo.ListScoreMetricsByPostIDs(ctx, dedupedIDs)
	if err != nil {
		w.log.Error("Failed to load post score metrics by post IDs", zap.Error(err))
		return
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

	if err := w.postScoreRepo.UpsertScores(ctx, snapshots); err != nil {
		w.log.Error("Failed to upsert post hotness scores", zap.Error(err))
		return
	}

	if err := w.postRepo.ClearHotnessDirty(ctx, dirtyIDs); err != nil {
		w.log.Error("Failed to clear hotness dirty flags", zap.Error(err))
		return
	}

	w.log.Info(
		"[PostHotnessWorker] Job succeeded",
		zap.Int("dirty_selected_count", len(dirtyIDs)),
		zap.Int("recent_decay_selected_count", len(recentDecayIDs)),
		zap.Int("high_score_stale_selected_count", len(highScoreStaleIDs)),
		zap.Int("processed_count", len(metrics)),
		zap.Int("upserted_count", len(snapshots)),
		zap.Int("dirty_cleared_count", len(dirtyIDs)),
		zap.Duration("duration", time.Since(startedAt)),
	)
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
