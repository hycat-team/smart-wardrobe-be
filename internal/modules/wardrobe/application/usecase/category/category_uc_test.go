package category

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/itemtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedrepos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type fakeCategoryRepo struct{}

func (f fakeCategoryRepo) GetAll(ctx context.Context) ([]*entities.Category, error) { return nil, nil }
func (f fakeCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Category, error) {
	return nil, nil
}
func (f fakeCategoryRepo) GetByName(ctx context.Context, name string) (*entities.Category, error) {
	return nil, nil
}
func (f fakeCategoryRepo) GetBySlug(ctx context.Context, slug string) (*entities.Category, error) {
	return nil, nil
}
func (f fakeCategoryRepo) Create(ctx context.Context, entity *entities.Category) error { return nil }
func (f fakeCategoryRepo) CreateInBatches(ctx context.Context, entities []*entities.Category, batchSize int) error {
	return nil
}
func (f fakeCategoryRepo) Update(ctx context.Context, entity *entities.Category) error { return nil }
func (f fakeCategoryRepo) Delete(ctx context.Context, id uuid.UUID) error              { return nil }
func (f fakeCategoryRepo) CountWardrobeItemsByCategoryAndItemType(ctx context.Context, categoryID uuid.UUID, kind itemtype.ItemType) (int64, error) {
	return 0, nil
}
func (f fakeCategoryRepo) ReassignSystemCatalogItemsToCategory(ctx context.Context, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error {
	return nil
}

type fakeUnitOfWork struct{}

func (f fakeUnitOfWork) Execute(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}
func (f fakeUnitOfWork) Begin(ctx context.Context) (context.Context, error) { return ctx, nil }
func (f fakeUnitOfWork) Commit(ctx context.Context) error                   { return nil }
func (f fakeUnitOfWork) Rollback(ctx context.Context) error                 { return nil }

var _ sharedrepos.IUnitOfWork = fakeUnitOfWork{}

func TestCreateCategoryRejectsLegacyVaySlug(t *testing.T) {
	uc := &CategoryUseCase{
		categoryRepo: fakeCategoryRepo{},
		uow:          fakeUnitOfWork{},
	}

	_, err := uc.CreateCategory(context.Background(), dto.CreateCategoryReq{
		Name: "Váy cũ",
		Slug: "vay",
	})
	if err == nil {
		t.Fatal("expected legacy slug vay to be rejected")
	}
}

func TestUpdateCategoryKeepsSortOrderWhenOmitted(t *testing.T) {
	current := &entities.Category{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Name:            "Áo",
		Slug:            "ao",
		SortOrder:       15,
	}

	uc := &CategoryUseCase{
		categoryRepo: fakeCategoryRepoWithEntity{byID: current},
		uow:          fakeUnitOfWork{},
	}

	res, err := uc.UpdateCategory(context.Background(), current.ID, dto.UpdateCategoryReq{
		Name: "Áo mới",
		Slug: "ao-moi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SortOrder != 15 {
		t.Fatalf("expected sort order to be preserved, got %d", res.SortOrder)
	}
}

type fakeCategoryRepoWithEntity struct {
	fakeCategoryRepo
	byID *entities.Category
}

func (f fakeCategoryRepoWithEntity) GetByID(ctx context.Context, id uuid.UUID) (*entities.Category, error) {
	if f.byID != nil && f.byID.ID == id {
		return f.byID, nil
	}
	return nil, nil
}
