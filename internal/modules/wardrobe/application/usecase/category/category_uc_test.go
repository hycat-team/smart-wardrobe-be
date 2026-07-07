package category

import (
	"context"
	"testing"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	sharedrepos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type fakeFashionContract struct{}

func (f fakeFashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*fashion_dto.FashionItemRes, error) {
	return nil, nil
}
func (f fakeFashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.FashionItemRes, error) {
	return nil, nil
}
func (f fakeFashionContract) ListFashionItemsBrief(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.FashionItemBriefRes, error) {
	return nil, nil
}
func (f fakeFashionContract) GetCategory(ctx context.Context, id uuid.UUID) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (f fakeFashionContract) ListCategories(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (f fakeFashionContract) GetAllCategories(ctx context.Context) ([]*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (f fakeFashionContract) GetCategoryBySlug(ctx context.Context, slug string) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (f fakeFashionContract) GetCategoryByName(ctx context.Context, name string) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (f fakeFashionContract) CreateCategory(ctx context.Context, input fashion_dto.CreateCategoryInput) (*fashion_dto.CategoryRes, error) {
	return &fashion_dto.CategoryRes{
		ID:        uuid.New(),
		Name:      input.Name,
		Slug:      input.Slug,
		SortOrder: input.SortOrder,
	}, nil
}
func (f fakeFashionContract) UpdateCategory(ctx context.Context, id uuid.UUID, input fashion_dto.UpdateCategoryInput) (*fashion_dto.CategoryRes, error) {
	var name, slug string
	var sortOrder int
	if input.Name != nil {
		name = *input.Name
	}
	if input.Slug != nil {
		slug = *input.Slug
	}
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	return &fashion_dto.CategoryRes{
		ID:        id,
		Name:      name,
		Slug:      slug,
		SortOrder: sortOrder,
	}, nil
}
func (f fakeFashionContract) DeleteCategory(ctx context.Context, id uuid.UUID) error {
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
		fashionContract: fakeFashionContract{},
		uow:             fakeUnitOfWork{},
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
	current := &fashion_dto.CategoryRes{
		ID:        uuid.New(),
		Name:      "Áo",
		Slug:      "ao",
		SortOrder: 15,
	}

	fc := fakeFashionContractWithEntity{byID: current}

	uc := &CategoryUseCase{
		fashionContract: fc,
		uow:             fakeUnitOfWork{},
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

type fakeFashionContractWithEntity struct {
	fakeFashionContract
	byID *fashion_dto.CategoryRes
}

func (f fakeFashionContractWithEntity) GetCategory(ctx context.Context, id uuid.UUID) (*fashion_dto.CategoryRes, error) {
	if f.byID != nil && f.byID.ID == id {
		return f.byID, nil
	}
	return nil, nil
}

func (f fakeFashionContractWithEntity) GetCategoryByName(ctx context.Context, name string) (*fashion_dto.CategoryRes, error) {
	if f.byID != nil && f.byID.Name == name {
		return f.byID, nil
	}
	return nil, nil
}

func (f fakeFashionContractWithEntity) GetCategoryBySlug(ctx context.Context, slug string) (*fashion_dto.CategoryRes, error) {
	if f.byID != nil && f.byID.Slug == slug {
		return f.byID, nil
	}
	return nil, nil
}

func (f fakeFashionContractWithEntity) UpdateCategory(ctx context.Context, id uuid.UUID, input fashion_dto.UpdateCategoryInput) (*fashion_dto.CategoryRes, error) {
	if f.byID == nil || f.byID.ID != id {
		return f.fakeFashionContract.UpdateCategory(ctx, id, input)
	}
	if input.Name != nil {
		f.byID.Name = *input.Name
	}
	if input.Slug != nil {
		f.byID.Slug = *input.Slug
	}
	if input.SortOrder != nil {
		f.byID.SortOrder = *input.SortOrder
	}
	return f.byID, nil
}
