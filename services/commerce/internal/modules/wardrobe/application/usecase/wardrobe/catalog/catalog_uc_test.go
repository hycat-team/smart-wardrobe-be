package catalog

import (
	"context"
	"errors"
	"testing"

	contract "smart-wardrobe-be/services/account/contract/subscription"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

// 033 T049 - InitClosetFromCatalog may only clone genuine system catalog
// templates. Supplying another user's private item id (or a deleted / non
// InWardrobe catalog id) must be rejected before anything is written (FR-035).

type catalogWardrobeRepo struct {
	repositories.IWardrobeItemRepository

	currentCount int64
	countErr     error
	templates    []*entities.WardrobeItem
	templateErr  error
	bulkErr      error

	countCalls    int
	getByIDsCalls int
	bulkCalls     int
	bulkCreated   []*entities.WardrobeItem
}

func (r *catalogWardrobeRepo) CountByUserID(context.Context, uuid.UUID) (int64, error) {
	r.countCalls++
	return r.currentCount, r.countErr
}

func (r *catalogWardrobeRepo) GetByIDs(context.Context, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	r.getByIDsCalls++
	return r.templates, r.templateErr
}

func (r *catalogWardrobeRepo) BulkCreate(_ context.Context, items []*entities.WardrobeItem) error {
	if r.bulkErr != nil {
		return r.bulkErr
	}
	r.bulkCalls++
	r.bulkCreated = append(r.bulkCreated, items...)
	return nil
}

type catalogSubContract struct {
	contract.IUserSubscriptionContract
	overview *sub_dto.UserSubscriptionContractDTO
}

func (c *catalogSubContract) GetUserSubscriptionOverview(context.Context, uuid.UUID) (*sub_dto.UserSubscriptionContractDTO, error) {
	return c.overview, nil
}

type catalogFashionContract struct {
	fashion_contract.IFashionContract
	briefs []*fashion_dto.FashionItemBriefRes
}

func (f *catalogFashionContract) ListFashionItemsBrief(context.Context, []uuid.UUID) ([]*fashion_dto.FashionItemBriefRes, error) {
	return f.briefs, nil
}

func newCatalogUseCase(repo *catalogWardrobeRepo, maxWardrobeItems int) *WardrobeCatalogUseCase {
	return &WardrobeCatalogUseCase{
		wardrobeRepo:    repo,
		userSubContract: &catalogSubContract{overview: &sub_dto.UserSubscriptionContractDTO{MaxWardrobeItems: maxWardrobeItems}},
		fashionContract: &catalogFashionContract{},
	}
}

func catalogRow(fashionItemID *uuid.UUID, itemType itemtype.ItemType, status wardrobestatus.WardrobeItemStatus, owner uuid.UUID) *entities.WardrobeItem {
	row := &entities.WardrobeItem{
		UserID:        owner,
		FashionItemID: fashionItemID,
		ItemType:      itemType,
		Status:        status,
	}
	row.ID = uuid.New()
	return row
}

func catalogAssertError(t *testing.T, got, want error) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected error %q, got nil", want.Error())
	}
	if got.Error() != want.Error() {
		t.Fatalf("error = %q, want %q", got.Error(), want.Error())
	}
}

func TestInitClosetFromCatalogRejectsIneligibleTemplates(t *testing.T) {
	requesterID := uuid.New()
	otherUserID := uuid.New()
	fashionID := uuid.New()

	deleted := catalogRow(&fashionID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)
	deleted.IsDeleted = true

	cases := []struct {
		name     string
		template *entities.WardrobeItem
	}{
		{
			// FR-035 security fix: a private wardrobe row of another user must
			// never be cloned into the caller's closet.
			name:     "another user's private item",
			template: catalogRow(&fashionID, itemtype.UserItem, wardrobestatus.InWardrobe, otherUserID),
		},
		{
			name:     "soft-deleted catalog item",
			template: deleted,
		},
		{
			name:     "catalog item that is no longer InWardrobe",
			template: catalogRow(&fashionID, itemtype.SystemCatalogItem, wardrobestatus.Sold, uuid.Nil),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &catalogWardrobeRepo{templates: []*entities.WardrobeItem{tc.template}}
			uc := newCatalogUseCase(repo, 10)

			got, err := uc.InitClosetFromCatalog(context.Background(), requesterID, []uuid.UUID{tc.template.ID})

			if got != nil {
				t.Fatalf("result = %+v, want nil", got)
			}
			catalogAssertError(t, err, wardrobeerrors.ErrCatalogItemNotFound())
			if repo.bulkCalls != 0 {
				t.Fatalf("bulk create calls = %d, want 0 (nothing may be cloned)", repo.bulkCalls)
			}
		})
	}
}

// A genuine system catalog template always references a fashion item. Cloning a
// template with a nil FashionItemID used to fall back to the catalog row id,
// producing a wardrobe row whose fashion_item_id does not exist. It must be
// rejected, and a valid template in the same batch must not be cloned either.
func TestInitClosetFromCatalogRejectsTemplateWithoutFashionItem(t *testing.T) {
	requesterID := uuid.New()
	validFashionID := uuid.New()

	valid := catalogRow(&validFashionID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)
	withoutFashionItem := catalogRow(nil, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)

	repo := &catalogWardrobeRepo{templates: []*entities.WardrobeItem{valid, withoutFashionItem}}
	uc := newCatalogUseCase(repo, 10)

	got, err := uc.InitClosetFromCatalog(context.Background(), requesterID, []uuid.UUID{valid.ID, withoutFashionItem.ID})

	if got != nil {
		t.Fatalf("result = %+v, want nil", got)
	}
	catalogAssertError(t, err, wardrobeerrors.ErrCatalogItemUnavailable())
	if repo.bulkCalls != 0 || len(repo.bulkCreated) != 0 {
		t.Fatalf("bulk create calls/rows = %d/%d, want 0/0 (no clone may be created)", repo.bulkCalls, len(repo.bulkCreated))
	}
}

func TestInitClosetFromCatalogRejectsMixedBatchWithoutPartialClone(t *testing.T) {
	requesterID := uuid.New()
	fashionID := uuid.New()

	valid := catalogRow(&fashionID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)
	private := catalogRow(&fashionID, itemtype.UserItem, wardrobestatus.InWardrobe, uuid.New())

	repo := &catalogWardrobeRepo{templates: []*entities.WardrobeItem{valid, private}}
	uc := newCatalogUseCase(repo, 10)

	_, err := uc.InitClosetFromCatalog(context.Background(), requesterID, []uuid.UUID{valid.ID, private.ID})

	catalogAssertError(t, err, wardrobeerrors.ErrCatalogItemNotFound())
	if repo.bulkCalls != 0 || len(repo.bulkCreated) != 0 {
		t.Fatalf("bulk create calls/rows = %d/%d, want 0/0", repo.bulkCalls, len(repo.bulkCreated))
	}
}

func TestInitClosetFromCatalogClonesValidTemplates(t *testing.T) {
	requesterID := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()
	price := 259000.0

	first := catalogRow(&firstID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)
	first.Price = &price
	second := catalogRow(&secondID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)

	repo := &catalogWardrobeRepo{templates: []*entities.WardrobeItem{first, second}}
	uc := newCatalogUseCase(repo, 10)

	got, err := uc.InitClosetFromCatalog(context.Background(), requesterID, []uuid.UUID{first.ID, second.ID})
	if err != nil {
		t.Fatalf("InitClosetFromCatalog: %v", err)
	}
	if len(got) != 2 || len(repo.bulkCreated) != 2 {
		t.Fatalf("result/clones = %d/%d, want 2/2", len(got), len(repo.bulkCreated))
	}
	for i, clone := range repo.bulkCreated {
		if clone.UserID != requesterID {
			t.Fatalf("clone %d user = %s, want the requester", i, clone.UserID)
		}
		if clone.ItemType != itemtype.UserItem || clone.Status != wardrobestatus.InWardrobe {
			t.Fatalf("clone %d type/status = %v/%v, want UserItem/InWardrobe", i, clone.ItemType, clone.Status)
		}
	}
	if repo.bulkCreated[0].FashionItemID == nil || *repo.bulkCreated[0].FashionItemID != firstID {
		t.Fatalf("clone fashion item = %v, want %s", repo.bulkCreated[0].FashionItemID, firstID)
	}
	if repo.bulkCreated[0].Price == nil || *repo.bulkCreated[0].Price != price {
		t.Fatalf("clone price = %v, want %v", repo.bulkCreated[0].Price, price)
	}
}

func TestInitClosetFromCatalogRejectsEmptyIDs(t *testing.T) {
	repo := &catalogWardrobeRepo{}
	uc := newCatalogUseCase(repo, 10)

	_, err := uc.InitClosetFromCatalog(context.Background(), uuid.New(), nil)

	catalogAssertError(t, err, wardrobeerrors.ErrCatalogItemIDsEmpty())
	if repo.countCalls != 0 || repo.getByIDsCalls != 0 {
		t.Fatalf("repository called (count=%d, getByIDs=%d) for an empty request", repo.countCalls, repo.getByIDsCalls)
	}
}

func TestInitClosetFromCatalogRejectsOverWardrobeLimit(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	repo := &catalogWardrobeRepo{
		currentCount: 9,
		templates: []*entities.WardrobeItem{
			catalogRow(&firstID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil),
			catalogRow(&secondID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil),
		},
	}
	uc := newCatalogUseCase(repo, 10)

	_, err := uc.InitClosetFromCatalog(context.Background(), uuid.New(), []uuid.UUID{firstID, secondID})

	catalogAssertError(t, err, wardrobeerrors.ErrWardrobeLimitExceededForCatalog(9, 10, 2))
	if repo.bulkCalls != 0 {
		t.Fatalf("bulk create calls = %d, want 0", repo.bulkCalls)
	}
	if repo.getByIDsCalls != 0 {
		t.Fatalf("templates were read before the quota check (%d calls)", repo.getByIDsCalls)
	}
}

func TestInitClosetFromCatalogRejectsWhenNothingMatches(t *testing.T) {
	repo := &catalogWardrobeRepo{}
	uc := newCatalogUseCase(repo, 10)

	_, err := uc.InitClosetFromCatalog(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	catalogAssertError(t, err, wardrobeerrors.ErrCatalogItemNotFound())
	if repo.bulkCalls != 0 {
		t.Fatalf("bulk create calls = %d, want 0", repo.bulkCalls)
	}
}

func TestInitClosetFromCatalogPropagatesBulkCreateError(t *testing.T) {
	fashionID := uuid.New()
	writeErr := errors.New("insert failed")
	repo := &catalogWardrobeRepo{
		templates: []*entities.WardrobeItem{catalogRow(&fashionID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)},
		bulkErr:   writeErr,
	}
	uc := newCatalogUseCase(repo, 10)

	got, err := uc.InitClosetFromCatalog(context.Background(), uuid.New(), []uuid.UUID{repo.templates[0].ID})

	if got != nil {
		t.Fatalf("result = %+v, want nil", got)
	}
	if !errors.Is(err, writeErr) {
		t.Fatalf("error = %v, want %v", err, writeErr)
	}
}
