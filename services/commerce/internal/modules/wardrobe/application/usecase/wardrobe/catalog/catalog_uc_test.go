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
	lockErr      error

	countCalls    int
	getByIDsCalls int
	bulkCalls     int
	lockCalls     int
	bulkCreated   []*entities.WardrobeItem

	// callOrder records the repository calls in the order they happened, so a
	// test can prove the lock is taken before the quota count.
	callOrder []string
	// inTx mirrors whether the fake unit of work is currently running its
	// callback; countInTx/bulkInTx snapshot it inside the mocked calls.
	inTx      bool
	countInTx bool
	bulkInTx  bool

	uow *catalogUnitOfWork
}

func (r *catalogWardrobeRepo) record(call string) {
	r.callOrder = append(r.callOrder, call)
}

func (r *catalogWardrobeRepo) LockUserWardrobe(context.Context, uuid.UUID) error {
	r.lockCalls++
	r.record("lock")
	return r.lockErr
}

func (r *catalogWardrobeRepo) CountByUserID(context.Context, uuid.UUID) (int64, error) {
	r.countCalls++
	r.record("count")
	r.countInTx = r.inTx
	return r.currentCount, r.countErr
}

func (r *catalogWardrobeRepo) GetByIDs(context.Context, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	r.getByIDsCalls++
	r.record("getByIDs")
	return r.templates, r.templateErr
}

func (r *catalogWardrobeRepo) BulkCreate(_ context.Context, items []*entities.WardrobeItem) error {
	if r.bulkErr != nil {
		return r.bulkErr
	}
	r.bulkCalls++
	r.record("bulk")
	r.bulkInTx = r.inTx
	r.bulkCreated = append(r.bulkCreated, items...)
	return nil
}

// catalogUnitOfWork records how the catalog use case drives the transaction.
type catalogUnitOfWork struct {
	repo      *catalogWardrobeRepo
	executes  int
	commits   int
	rollbacks int
}

func (u *catalogUnitOfWork) Execute(ctx context.Context, fn func(context.Context) error) error {
	u.executes++
	u.repo.inTx = true
	err := fn(ctx)
	u.repo.inTx = false
	if err != nil {
		u.rollbacks++
		return err
	}
	u.commits++
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
	uow := &catalogUnitOfWork{repo: repo}
	repo.uow = uow
	return &WardrobeCatalogUseCase{
		wardrobeRepo:    repo,
		userSubContract: &catalogSubContract{overview: &sub_dto.UserSubscriptionContractDTO{MaxWardrobeItems: maxWardrobeItems}},
		fashionContract: &catalogFashionContract{},
		uow:             uow,
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
	if repo.lockCalls != 0 || repo.uow.executes != 0 {
		t.Fatalf("lock/transaction started (%d/%d) for an empty request", repo.lockCalls, repo.uow.executes)
	}
}

// 033 concurrency fix - the quota check and the insert must share one unit of
// work, behind the per-user wardrobe lock, so two concurrent initializations
// cannot both pass the check.
func TestInitClosetFromCatalogLocksQuotaBeforeCountInsideOneUnitOfWork(t *testing.T) {
	requesterID := uuid.New()
	fashionID := uuid.New()

	repo := &catalogWardrobeRepo{
		currentCount: 1,
		templates:    []*entities.WardrobeItem{catalogRow(&fashionID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)},
	}
	uc := newCatalogUseCase(repo, 10)

	got, err := uc.InitClosetFromCatalog(context.Background(), requesterID, []uuid.UUID{repo.templates[0].ID})
	if err != nil {
		t.Fatalf("InitClosetFromCatalog: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("result = %d items, want 1", len(got))
	}

	if repo.lockCalls != 1 {
		t.Fatalf("lock calls = %d, want 1", repo.lockCalls)
	}
	want := []string{"lock", "count", "getByIDs", "bulk"}
	if len(repo.callOrder) != len(want) {
		t.Fatalf("call order = %v, want %v", repo.callOrder, want)
	}
	for i := range want {
		if repo.callOrder[i] != want[i] {
			t.Fatalf("call order = %v, want %v", repo.callOrder, want)
		}
	}
	if !repo.countInTx || !repo.bulkInTx {
		t.Fatalf("count/insert outside the unit of work (count=%v, insert=%v)", repo.countInTx, repo.bulkInTx)
	}
	if repo.uow.executes != 1 || repo.uow.commits != 1 || repo.uow.rollbacks != 0 {
		t.Fatalf("uow executes/commits/rollbacks = %d/%d/%d, want 1/1/0",
			repo.uow.executes, repo.uow.commits, repo.uow.rollbacks)
	}
}

func TestInitClosetFromCatalogPropagatesLockError(t *testing.T) {
	lockErr := errors.New("advisory lock failed")
	fashionID := uuid.New()
	repo := &catalogWardrobeRepo{
		lockErr:   lockErr,
		templates: []*entities.WardrobeItem{catalogRow(&fashionID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, uuid.Nil)},
	}
	uc := newCatalogUseCase(repo, 10)

	got, err := uc.InitClosetFromCatalog(context.Background(), uuid.New(), []uuid.UUID{repo.templates[0].ID})

	if got != nil {
		t.Fatalf("result = %+v, want nil", got)
	}
	if !errors.Is(err, lockErr) {
		t.Fatalf("error = %v, want %v", err, lockErr)
	}
	if repo.countCalls != 0 || repo.bulkCalls != 0 {
		t.Fatalf("count/insert ran without the lock (count=%d, bulk=%d)", repo.countCalls, repo.bulkCalls)
	}
	if repo.uow.rollbacks != 1 {
		t.Fatalf("rollbacks = %d, want 1", repo.uow.rollbacks)
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
	if repo.lockCalls != 1 {
		t.Fatalf("lock calls = %d, want 1 (the check must be serialized)", repo.lockCalls)
	}
	if len(repo.callOrder) < 2 || repo.callOrder[0] != "lock" || repo.callOrder[1] != "count" {
		t.Fatalf("call order = %v, want the lock before the count", repo.callOrder)
	}
	if !repo.countInTx {
		t.Fatal("the quota count ran outside the unit of work")
	}
	if repo.uow.rollbacks != 1 || repo.uow.commits != 0 {
		t.Fatalf("uow rollbacks/commits = %d/%d, want 1/0", repo.uow.rollbacks, repo.uow.commits)
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
	// The quota check and the insert share the transaction, so a failed insert
	// rolls the whole thing back instead of leaving the clones behind.
	if repo.uow.rollbacks != 1 || repo.uow.commits != 0 {
		t.Fatalf("uow rollbacks/commits = %d/%d, want 1/0", repo.uow.rollbacks, repo.uow.commits)
	}
	if repo.lockCalls != 1 {
		t.Fatalf("lock calls = %d, want 1", repo.lockCalls)
	}
}
