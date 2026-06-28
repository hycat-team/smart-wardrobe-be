package worker

import (
	"context"
	"sync"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// VisionCategorySnapshot stores the prompt and category lookups needed by wardrobe image analysis.
type VisionCategorySnapshot struct {
	Prompt          string
	CategoryMap     map[string]uuid.UUID
	CategoryNameMap map[uuid.UUID]string
	OtherCategoryID uuid.UUID
	LoadedAt        time.Time
	ExpiresAt       time.Time
}

// VisionCategoryCache keeps a short-lived in-memory snapshot of wardrobe categories for worker jobs.
type VisionCategoryCache struct {
	categoryRepo repositories.ICategoryRepository
	logger       logger.Interface
	ttl          time.Duration

	cacheMu sync.RWMutex
	cached  *VisionCategorySnapshot
}

// NewVisionCategoryCache creates the in-memory wardrobe category cache used by vision workers.
func NewVisionCategoryCache(
	cfg *config.Config,
	categoryRepo repositories.ICategoryRepository,
	logger logger.Interface,
) *VisionCategoryCache {
	return &VisionCategoryCache{
		categoryRepo: categoryRepo,
		logger:       logger,
		ttl:          time.Duration(cfg.Wardrobe.CategoryCacheTTLSeconds) * time.Second,
	}
}

// Get returns a cached category snapshot or reloads it when the cache expires.
func (c *VisionCategoryCache) Get(ctx context.Context) (*VisionCategorySnapshot, error) {
	now := time.Now().UTC()
	c.cacheMu.RLock()
	if snapshot := c.cached; snapshot != nil && snapshot.ExpiresAt.After(now) {
		c.cacheMu.RUnlock()
		return snapshot, nil
	}
	c.cacheMu.RUnlock()

	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	now = time.Now().UTC()
	if snapshot := c.cached; snapshot != nil && snapshot.ExpiresAt.After(now) {
		return snapshot, nil
	}

	snapshot, err := c.reload(ctx, now)
	if err != nil {
		if c.cached != nil {
			c.logger.Warn("[VisionCategoryCache] Failed to refresh category snapshot, reusing stale cache",
				zap.Error(err),
				zap.Time("expires_at", c.cached.ExpiresAt),
			)
			return c.cached, nil
		}
		return nil, err
	}

	c.cached = snapshot
	return snapshot, nil
}

func (c *VisionCategoryCache) reload(ctx context.Context, now time.Time) (*VisionCategorySnapshot, error) {
	categories, err := c.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	aiCatRefs := make([]dto.AICategoryRef, len(categories))
	categoryMap := make(map[string]uuid.UUID, len(categories))
	categoryNameMap := make(map[uuid.UUID]string, len(categories))
	var otherCategoryID uuid.UUID

	for i, cat := range categories {
		aiCatRefs[i] = dto.AICategoryRef{
			Name: cat.Name,
			Slug: cat.Slug,
		}
		categoryMap[cat.Slug] = cat.ID
		categoryNameMap[cat.ID] = cat.Name
		if cat.Slug == "other" {
			otherCategoryID = cat.ID
		}
	}

	return &VisionCategorySnapshot{
		Prompt:          getVisionSystemPrompt(aiCatRefs),
		CategoryMap:     categoryMap,
		CategoryNameMap: categoryNameMap,
		OtherCategoryID: otherCategoryID,
		LoadedAt:        now,
		ExpiresAt:       now.Add(c.ttl),
	}, nil
}
