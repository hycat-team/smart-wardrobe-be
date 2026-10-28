package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandItemFilter struct {
	BrandID  uuid.UUID
	ItemType *branditemtype.BrandItemType
	Status   *branditemstatus.BrandItemStatus
	Page     int
	Limit    int
}

type BrandItemListResult struct {
	Items      []*entities.BrandItem
	TotalCount int64
}

type IBrandItemRepository interface {
	shared_repos.IGenericRepository[entities.BrandItem, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error)
	GetByBrandIDFiltered(ctx context.Context, filter BrandItemFilter) ([]*entities.BrandItem, error)
	GetByTaskID(ctx context.Context, taskID uuid.UUID) (*entities.BrandItem, error)
	GetByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) ([]*entities.BrandItem, error)
	GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error)
	GetByFashionItemID(ctx context.Context, fashionItemID uuid.UUID) (*entities.BrandItem, error)
	GetByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*entities.BrandItem, error)
	GetByBrandIDPaginated(ctx context.Context, filter BrandItemFilter) (*BrandItemListResult, error)
	CountByStatus(ctx context.Context, brandID uuid.UUID) ([]BrandItemStatusCount, error)
	CountByType(ctx context.Context, brandID uuid.UUID) ([]BrandItemTypeCount, error)
	CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error)
	// ClaimManualAnalysisRetry atomically claims a manual retry for a brand item,
	// transitioning it to draft and assigning a new task id. expectedTaskID is a
	// compare-and-swap guard so concurrent retries cannot create duplicate
	// sessions. Returns the reloaded item and whether the claim succeeded.
	ClaimManualAnalysisRetry(ctx context.Context, itemID uuid.UUID, expectedTaskID *uuid.UUID, newTaskID uuid.UUID, now time.Time) (*entities.BrandItem, bool, error)
}

type BrandItemStatusCount struct {
	Status string
	Count  int64
}

type BrandItemTypeCount struct {
	ItemType string
	Count    int64
}

type IDigitalSampleResponseRepository interface {
	shared_repos.IGenericRepository[entities.DigitalSampleResponse, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.DigitalSampleResponse, error)
	GetByBrandItemID(ctx context.Context, brandItemID uuid.UUID) ([]*entities.DigitalSampleResponse, error)
}
