package worker

import (
	"context"
	"math"
	"time"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const hotnessGravity = 1.5

type IPostHotnessWorker interface {
	Start()
	Stop()
}

type PostHotnessWorker struct {
	postScoreRepo repositories.IPostScoreRepository
	cronEngine    *cron.Cron
	log           logger.Interface
}

func NewPostHotnessWorker(
	postScoreRepo repositories.IPostScoreRepository,
	log logger.Interface,
) IPostHotnessWorker {
	return &PostHotnessWorker{
		postScoreRepo: postScoreRepo,
		cronEngine:    cron.New(cron.WithSeconds()),
		log:           log,
	}
}

func (w *PostHotnessWorker) Start() {
	go w.executeRefresh()

	_, err := w.cronEngine.AddFunc("0 */10 * * * *", func() {
		w.log.Info("Community hotness worker tick triggered")
		w.executeRefresh()
	})
	if err != nil {
		w.log.Error("Failed to register community hotness worker", zap.Error(err))
		return
	}

	w.cronEngine.Start()
	w.log.Info("Community hotness worker started successfully")
}

func (w *PostHotnessWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
		w.log.Info("Community hotness worker stopped safely")
	}
}

func (w *PostHotnessWorker) executeRefresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	metrics, err := w.postScoreRepo.ListScoreMetrics(ctx)
	if err != nil {
		w.log.Error("Failed to load post score metrics", zap.Error(err))
		return
	}

	now := time.Now()
	snapshots := make([]*entities.PostScoreSnapshot, 0, len(metrics))
	for _, metric := range metrics {
		createdAt := time.Unix(metric.CreatedAtUnix, 0)
		ageInHours := now.Sub(createdAt).Hours()
		score := ((float64(metric.LikeCount) * 1) + (float64(metric.CommentCount) * 2) - 1) / math.Pow(ageInHours+2, hotnessGravity)

		snapshots = append(snapshots, &entities.PostScoreSnapshot{
			PostID:             metric.PostID,
			GlobalHotnessScore: score,
			LastCalculatedAt:   now,
		})
	}

	if err := w.postScoreRepo.UpsertScores(ctx, snapshots); err != nil {
		w.log.Error("Failed to upsert post hotness scores", zap.Error(err))
		return
	}

	w.log.Info("Community hotness worker finished refresh cycle", zap.Int("posts", len(snapshots)))
}
