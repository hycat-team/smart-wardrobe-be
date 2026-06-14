package worker

import (
	"context"
	"sync"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// VisionCategoryContext keeps the prebuilt vision prompt and category lookup maps for worker jobs.
type VisionCategoryContext struct {
	Prompt          string
	CategoryMap     map[string]uuid.UUID
	CategoryNameMap map[uuid.UUID]string
	OtherCategoryID uuid.UUID
	LoadedAt        time.Time
	ExpiresAt       time.Time
}

// VisionCategoryContextProvider caches wardrobe category context for vision worker jobs.
type VisionCategoryContextProvider struct {
	categoryRepo repositories.ICategoryRepository
	logger       logger.Interface
	ttl          time.Duration

	cacheMu sync.RWMutex
	cached  *VisionCategoryContext
}

// NewVisionCategoryContextProvider creates an in-memory category context cache for wardrobe workers.
func NewVisionCategoryContextProvider(
	cfg *config.Config,
	categoryRepo repositories.ICategoryRepository,
	logger logger.Interface,
) *VisionCategoryContextProvider {
	return &VisionCategoryContextProvider{
		categoryRepo: categoryRepo,
		logger:       logger,
		ttl:          time.Duration(cfg.Wardrobe.CategoryCacheTTLSeconds) * time.Second,
	}
}

// Get returns the cached category context or reloads it when the TTL has expired.
func (p *VisionCategoryContextProvider) Get(ctx context.Context) (*VisionCategoryContext, error) {
	now := time.Now().UTC()
	p.cacheMu.RLock()
	if snapshot := p.cached; snapshot != nil && snapshot.ExpiresAt.After(now) {
		p.cacheMu.RUnlock()
		return snapshot, nil
	}
	p.cacheMu.RUnlock()

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	now = time.Now().UTC()
	if snapshot := p.cached; snapshot != nil && snapshot.ExpiresAt.After(now) {
		return snapshot, nil
	}

	snapshot, err := p.reload(ctx, now)
	if err != nil {
		if p.cached != nil {
			p.logger.Warn("[VisionCategoryContextProvider] Failed to refresh category context, reusing stale cache",
				zap.Error(err),
				zap.Time("expires_at", p.cached.ExpiresAt),
			)
			return p.cached, nil
		}
		return nil, err
	}

	p.cached = snapshot
	return snapshot, nil
}

func (p *VisionCategoryContextProvider) reload(ctx context.Context, now time.Time) (*VisionCategoryContext, error) {
	categories, err := p.categoryRepo.GetAll(ctx)
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

	return &VisionCategoryContext{
		Prompt:          getVisionSystemPrompt(aiCatRefs),
		CategoryMap:     categoryMap,
		CategoryNameMap: categoryNameMap,
		OtherCategoryID: otherCategoryID,
		LoadedAt:        now,
		ExpiresAt:       now.Add(p.ttl),
	}, nil
}
