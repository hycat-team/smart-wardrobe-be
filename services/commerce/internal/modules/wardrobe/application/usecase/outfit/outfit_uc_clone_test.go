package outfit

import (
	"context"
	"errors"
	"testing"
	"time"

	contract "smart-wardrobe-be/services/account/contract/subscription"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// 033 T049 - the save/update path must clone catalog rows atomically, reject the
// whole save when the wardrobe quota would be exceeded, reuse an existing owned
// row, reject duplicate input, and never start an AI analysis task for a clone.
//
// The unit of work is faked with snapshots: on error every write is reverted, so
// "no partial clone" is asserted exactly like a real transaction would behave.

// --- fakes ------------------------------------------------------------------

type cloneWardrobeState struct {
	bulkBatches  [][]*entities.WardrobeItem
	currentCount int64
	touchCalls   int
	touchIDs     []uuid.UUID
}

type cloneWardrobeRepo struct {
	repositories.IWardrobeItemRepository

	userItems    []*entities.WardrobeItem
	ownedRows    []*entities.WardrobeItem
	catalogRows  []*entities.WardrobeItem
	currentCount int64
	bulkErr      error
	touchErr     error

	bulkBatches [][]*entities.WardrobeItem
	touchCalls  int
	touchIDs    []uuid.UUID
}

func (r *cloneWardrobeRepo) GetByUserID(context.Context, uuid.UUID, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return r.userItems, nil
}

func (r *cloneWardrobeRepo) GetByUserIDAndFashionItemIDs(context.Context, uuid.UUID, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return r.ownedRows, nil
}

func (r *cloneWardrobeRepo) GetActiveSystemCatalogItemsByIDs(context.Context, []uuid.UUID) ([]*entities.WardrobeItem, error) {
	return r.catalogRows, nil
}

func (r *cloneWardrobeRepo) CountByUserID(context.Context, uuid.UUID) (int64, error) {
	return r.currentCount, nil
}

func (r *cloneWardrobeRepo) BulkCreate(_ context.Context, items []*entities.WardrobeItem) error {
	if r.bulkErr != nil {
		return r.bulkErr
	}
	r.bulkBatches = append(r.bulkBatches, items)
	r.currentCount += int64(len(items))
	return nil
}

func (r *cloneWardrobeRepo) TouchLastUsedAt(_ context.Context, ids []uuid.UUID, _ time.Time) error {
	if r.touchErr != nil {
		return r.touchErr
	}
	r.touchCalls++
	r.touchIDs = ids
	return nil
}

func (r *cloneWardrobeRepo) snapshot() cloneWardrobeState {
	return cloneWardrobeState{
		bulkBatches:  append([][]*entities.WardrobeItem(nil), r.bulkBatches...),
		currentCount: r.currentCount,
		touchCalls:   r.touchCalls,
		touchIDs:     r.touchIDs,
	}
}

func (r *cloneWardrobeRepo) restore(state cloneWardrobeState) {
	r.bulkBatches = state.bulkBatches
	r.currentCount = state.currentCount
	r.touchCalls = state.touchCalls
	r.touchIDs = state.touchIDs
}

type cloneOutfitState struct {
	existing     []*entities.Outfit
	created      *entities.Outfit
	createdItems []*entities.OutfitItem
	updated      *entities.Outfit
	updatedItems []*entities.OutfitItem
	createCalls  int
	updateCalls  int
}

type cloneOutfitRepo struct {
	repositories.IOutfitRepository

	existing    []*entities.Outfit
	detail      *entities.Outfit
	detailItems []*entities.OutfitItem
	createErr   error
	updateErr   error

	created      *entities.Outfit
	createdItems []*entities.OutfitItem
	updated      *entities.Outfit
	updatedItems []*entities.OutfitItem
	createCalls  int
	updateCalls  int
}

func (r *cloneOutfitRepo) GetByUserID(context.Context, uuid.UUID) ([]*entities.Outfit, error) {
	return r.existing, nil
}

func (r *cloneOutfitRepo) GetByID(context.Context, uuid.UUID) (*entities.Outfit, error) {
	return r.detail, nil
}

func (r *cloneOutfitRepo) CreateWithItems(_ context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createCalls++
	outfit.ID = uuid.New() // the database generates the id
	r.created = outfit
	r.createdItems = items
	r.existing = append(r.existing, outfit)
	return nil
}

func (r *cloneOutfitRepo) UpdateWithItems(_ context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updateCalls++
	r.updated = outfit
	r.updatedItems = items
	return nil
}

func (r *cloneOutfitRepo) snapshot() cloneOutfitState {
	return cloneOutfitState{
		existing:     append([]*entities.Outfit(nil), r.existing...),
		created:      r.created,
		createdItems: r.createdItems,
		updated:      r.updated,
		updatedItems: r.updatedItems,
		createCalls:  r.createCalls,
		updateCalls:  r.updateCalls,
	}
}

func (r *cloneOutfitRepo) restore(state cloneOutfitState) {
	r.existing = state.existing
	r.created = state.created
	r.createdItems = state.createdItems
	r.updated = state.updated
	r.updatedItems = state.updatedItems
	r.createCalls = state.createCalls
	r.updateCalls = state.updateCalls
}

type cloneSubContract struct {
	contract.IUserSubscriptionContract
	overview *sub_dto.UserSubscriptionContractDTO
	err      error
	calls    int
}

func (c *cloneSubContract) GetUserSubscriptionOverview(context.Context, uuid.UUID) (*sub_dto.UserSubscriptionContractDTO, error) {
	c.calls++
	return c.overview, c.err
}

type clonePublisherState struct {
	topics   []string
	payloads []any
}

type clonePublisher struct {
	topics   []string
	payloads []any
	err      error
}

func (p *clonePublisher) Publish(_ context.Context, topic string, payload any) error {
	if p.err != nil {
		return p.err
	}
	p.topics = append(p.topics, topic)
	p.payloads = append(p.payloads, payload)
	return nil
}

func (p *clonePublisher) snapshot() clonePublisherState {
	return clonePublisherState{
		topics:   append([]string(nil), p.topics...),
		payloads: append([]any(nil), p.payloads...),
	}
}

func (p *clonePublisher) restore(state clonePublisherState) {
	p.topics = state.topics
	p.payloads = state.payloads
}

type cloneFashionContract struct {
	fashion_contract.IFashionContract
	briefs []*fashion_dto.FashionItemBriefRes
}

func (f *cloneFashionContract) ListFashionItemsBrief(context.Context, []uuid.UUID) ([]*fashion_dto.FashionItemBriefRes, error) {
	return f.briefs, nil
}

// snapshotUnitOfWork models a database transaction: a failed callback reverts
// every write performed through the fakes.
type snapshotUnitOfWork struct {
	wardrobe  *cloneWardrobeRepo
	outfit    *cloneOutfitRepo
	publisher *clonePublisher
	executes  int
	rollbacks int
}

func (u *snapshotUnitOfWork) Execute(ctx context.Context, fn func(context.Context) error) error {
	u.executes++
	wardrobeState := u.wardrobe.snapshot()
	outfitState := u.outfit.snapshot()
	publisherState := u.publisher.snapshot()
	if err := fn(ctx); err != nil {
		u.rollbacks++
		u.wardrobe.restore(wardrobeState)
		u.outfit.restore(outfitState)
		u.publisher.restore(publisherState)
		return err
	}
	return nil
}

type cloneFixture struct {
	uc        *OutfitUseCase
	wardrobe  *cloneWardrobeRepo
	outfits   *cloneOutfitRepo
	sub       *cloneSubContract
	publisher *clonePublisher
	uow       *snapshotUnitOfWork
}

func newCloneFixture(wardrobe *cloneWardrobeRepo, outfits *cloneOutfitRepo, maxWardrobeItems int) *cloneFixture {
	sub := &cloneSubContract{overview: &sub_dto.UserSubscriptionContractDTO{
		MaxWardrobeItems: maxWardrobeItems,
		MaxOutfits:       10,
	}}
	publisher := &clonePublisher{}
	uow := &snapshotUnitOfWork{wardrobe: wardrobe, outfit: outfits, publisher: publisher}
	return &cloneFixture{
		uc: &OutfitUseCase{
			cfg:             &config.Config{},
			outfitRepo:      outfits,
			wardrobeRepo:    wardrobe,
			userSubContract: sub,
			fashionContract: &cloneFashionContract{},
			eventPublisher:  publisher,
			uow:             uow,
		},
		wardrobe:  wardrobe,
		outfits:   outfits,
		sub:       sub,
		publisher: publisher,
		uow:       uow,
	}
}

// --- helpers ----------------------------------------------------------------

// catalogTemplate builds a system catalog template row for the given fashion item.
func catalogTemplate(fashionItemID uuid.UUID, price *float64) *entities.WardrobeItem {
	item := &entities.WardrobeItem{
		FashionItemID: &fashionItemID,
		Price:         price,
		ItemType:      itemtype.SystemCatalogItem,
		Status:        wardrobestatus.InWardrobe,
	}
	item.ID = uuid.New()
	return item
}

// ownedRow builds a user wardrobe row for the given fashion item.
func ownedRow(fashionItemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) *entities.WardrobeItem {
	item := &entities.WardrobeItem{
		FashionItemID: &fashionItemID,
		ItemType:      itemtype.UserItem,
		Status:        status,
	}
	item.ID = uuid.New()
	return item
}

func cloneAssertSameError(t *testing.T, got, want error) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected error %q, got nil", want.Error())
	}
	if got.Error() != want.Error() {
		t.Fatalf("error = %q, want %q", got.Error(), want.Error())
	}
}

// --- tests -------------------------------------------------------------------

func TestSaveOutfitClonesCatalogItemsWithoutAnalysisTask(t *testing.T) {
	userID := uuid.New()
	shirtID := uuid.New()
	shoesID := uuid.New()
	price := 199000.0

	wardrobe := &cloneWardrobeRepo{catalogRows: []*entities.WardrobeItem{
		catalogTemplate(shirtID, &price),
		catalogTemplate(shoesID, nil),
	}}
	outfits := &cloneOutfitRepo{}
	fixture := newCloneFixture(wardrobe, outfits, 10)
	systemCatalog := itemContextSystemCatalog

	res, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
		Name: "Bộ đi làm",
		Items: []dto.SaveOutfitItemReq{
			{FashionItemID: shirtID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 1},
			{FashionItemID: shoesID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 2},
		},
	})
	if err != nil {
		t.Fatalf("SaveOutfit: %v", err)
	}
	if res == nil || len(res.Items) != 2 {
		t.Fatalf("response items = %+v, want 2", res)
	}
	for _, item := range res.Items {
		if item.ItemContext != string(outfititemcontext.UserWardrobe) {
			t.Fatalf("response item context = %q, want user_wardrobe", item.ItemContext)
		}
	}

	if len(wardrobe.bulkBatches) != 1 || len(wardrobe.bulkBatches[0]) != 2 {
		t.Fatalf("clone batches = %d (%v), want one batch of 2", len(wardrobe.bulkBatches), wardrobe.bulkBatches)
	}
	clones := wardrobe.bulkBatches[0]
	for _, clone := range clones {
		if clone.UserID != userID {
			t.Fatalf("clone user = %s, want %s", clone.UserID, userID)
		}
		if clone.ItemType != itemtype.UserItem || clone.Status != wardrobestatus.InWardrobe {
			t.Fatalf("clone type/status = %v/%v, want UserItem/InWardrobe", clone.ItemType, clone.Status)
		}
		if clone.TaskID != nil {
			t.Fatal("a cloned catalog row must not start an AI analysis task")
		}
	}
	if clones[0].Price == nil || *clones[0].Price != price {
		t.Fatalf("clone price = %v, want %v", clones[0].Price, price)
	}

	if outfits.createCalls != 1 || len(outfits.createdItems) != 2 {
		t.Fatalf("outfit created %d times with %d items, want 1/2", outfits.createCalls, len(outfits.createdItems))
	}
	if fixture.uow.executes != 1 || fixture.uow.rollbacks != 0 {
		t.Fatalf("uow executes/rollbacks = %d/%d, want 1/0", fixture.uow.executes, fixture.uow.rollbacks)
	}

	if wardrobe.touchCalls != 1 || len(wardrobe.touchIDs) != 2 {
		t.Fatalf("touch calls/ids = %d/%d, want 1/2", wardrobe.touchCalls, len(wardrobe.touchIDs))
	}

	// Exactly one signal: the learning event. No analysis job/task topic.
	if len(fixture.publisher.topics) != 1 || fixture.publisher.topics[0] != eventconstants.TopicOutfitSaved {
		t.Fatalf("published topics = %v, want [%s]", fixture.publisher.topics, eventconstants.TopicOutfitSaved)
	}
	payload, ok := fixture.publisher.payloads[0].(dto.OutfitSavedEventPayload)
	if !ok {
		t.Fatalf("payload type = %T, want dto.OutfitSavedEventPayload", fixture.publisher.payloads[0])
	}
	if len(payload.FashionItemIDs) != 2 {
		t.Fatalf("signal fashion ids = %v, want 2", payload.FashionItemIDs)
	}
}

func TestSaveOutfitRejectsCloneOverWardrobeLimitWithoutPartialClone(t *testing.T) {
	userID := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()

	wardrobe := &cloneWardrobeRepo{
		catalogRows:  []*entities.WardrobeItem{catalogTemplate(firstID, nil), catalogTemplate(secondID, nil)},
		currentCount: 9,
	}
	outfits := &cloneOutfitRepo{}
	fixture := newCloneFixture(wardrobe, outfits, 10)
	systemCatalog := itemContextSystemCatalog

	res, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
		Name: "Bộ vượt hạn mức",
		Items: []dto.SaveOutfitItemReq{
			{FashionItemID: firstID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 1},
			{FashionItemID: secondID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 2},
		},
	})

	if res != nil {
		t.Fatalf("response = %+v, want nil on a rejected save", res)
	}
	cloneAssertSameError(t, err, wardrobeerrors.ErrWardrobeLimitExceededForOutfit([]uuid.UUID{firstID, secondID}))
	if len(wardrobe.bulkBatches) != 0 {
		t.Fatalf("clone batches = %d, want 0 (nothing may be written)", len(wardrobe.bulkBatches))
	}
	if outfits.createCalls != 0 {
		t.Fatalf("outfit create calls = %d, want 0", outfits.createCalls)
	}
	if len(fixture.publisher.topics) != 0 {
		t.Fatalf("published topics = %v, want none", fixture.publisher.topics)
	}
	if fixture.uow.rollbacks != 1 {
		t.Fatalf("rollbacks = %d, want 1", fixture.uow.rollbacks)
	}
}

func TestSaveOutfitRollsBackWhenCloneInsertFails(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()

	insertErr := errors.New("insert wardrobe item failed")
	wardrobe := &cloneWardrobeRepo{
		catalogRows: []*entities.WardrobeItem{catalogTemplate(fashionID, nil)},
		bulkErr:     insertErr,
	}
	outfits := &cloneOutfitRepo{}
	fixture := newCloneFixture(wardrobe, outfits, 10)
	systemCatalog := itemContextSystemCatalog

	res, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
		Name:  "Bộ lỗi",
		Items: []dto.SaveOutfitItemReq{{FashionItemID: fashionID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 1}},
	})

	if res != nil {
		t.Fatalf("response = %+v, want nil when the clone insert fails", res)
	}
	if !errors.Is(err, insertErr) {
		t.Fatalf("error = %v, want %v", err, insertErr)
	}
	if fixture.uow.rollbacks != 1 {
		t.Fatalf("rollbacks = %d, want 1", fixture.uow.rollbacks)
	}
	// Rollback semantics: the outfit row, the clone and the outbox signal are all gone.
	if outfits.created != nil || len(outfits.existing) != 0 || outfits.createCalls != 0 {
		t.Fatalf("outfit writes survived the rollback: created=%v existing=%d calls=%d",
			outfits.created, len(outfits.existing), outfits.createCalls)
	}
	if len(wardrobe.bulkBatches) != 0 {
		t.Fatalf("clone batches = %d, want 0 after rollback", len(wardrobe.bulkBatches))
	}
	if len(fixture.publisher.topics) != 0 {
		t.Fatalf("published topics = %v, want none after rollback", fixture.publisher.topics)
	}
}

func TestSaveOutfitReusesOwnedRowInsteadOfCloning(t *testing.T) {
	cases := []struct {
		name    string
		status  wardrobestatus.WardrobeItemStatus
		context string
	}{
		{"in wardrobe via user_wardrobe", wardrobestatus.InWardrobe, itemContextUserWardrobe},
		{"in wardrobe via system_catalog", wardrobestatus.InWardrobe, itemContextSystemCatalog},
		// A non-deleted row that is not InWardrobe (e.g. still processing) is
		// still the user's own row, so it must be reused rather than cloned.
		{"processing via user_wardrobe", wardrobestatus.Processing, itemContextUserWardrobe},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID := uuid.New()
			fashionID := uuid.New()
			owned := ownedRow(fashionID, tc.status)

			wardrobe := &cloneWardrobeRepo{
				ownedRows:   []*entities.WardrobeItem{owned},
				catalogRows: []*entities.WardrobeItem{catalogTemplate(fashionID, nil)},
			}
			outfits := &cloneOutfitRepo{}
			fixture := newCloneFixture(wardrobe, outfits, 10)
			requested := tc.context

			_, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
				Name:  "Bộ tái sử dụng",
				Items: []dto.SaveOutfitItemReq{{FashionItemID: fashionID, ItemContext: &requested, Scale: 1, LayerOrder: 1}},
			})
			if err != nil {
				t.Fatalf("SaveOutfit: %v", err)
			}
			if len(wardrobe.bulkBatches) != 0 {
				t.Fatalf("clone batches = %d, want 0: an owned row must be reused", len(wardrobe.bulkBatches))
			}
			if len(wardrobe.touchIDs) != 1 || wardrobe.touchIDs[0] != owned.ID {
				t.Fatalf("touch ids = %v, want the owned row %s", wardrobe.touchIDs, owned.ID)
			}
			if len(outfits.createdItems) != 1 || outfits.createdItems[0].ItemContext != outfititemcontext.UserWardrobe {
				t.Fatalf("outfit items = %+v, want one user_wardrobe item", outfits.createdItems)
			}
		})
	}
}

func TestSaveOutfitRejectsDuplicateInputBeforeAnyLookup(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()

	fixture := newCloneFixture(&cloneWardrobeRepo{}, &cloneOutfitRepo{}, 10)

	res, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
		Name: "Bộ trùng",
		Items: []dto.SaveOutfitItemReq{
			{FashionItemID: fashionID, Scale: 1, LayerOrder: 1},
			{FashionItemID: fashionID, Scale: 1, LayerOrder: 2},
		},
	})

	if res != nil {
		t.Fatalf("response = %+v, want nil", res)
	}
	cloneAssertSameError(t, err, wardrobeerrors.ErrDuplicateOutfitItem(fashionID))
	if fixture.sub.calls != 0 {
		t.Fatalf("subscription lookups = %d, want 0 (reject before any dependency call)", fixture.sub.calls)
	}
	if fixture.uow.executes != 0 {
		t.Fatalf("uow executes = %d, want 0", fixture.uow.executes)
	}
}

func TestSaveOutfitItemContextExplicitVersusAuto(t *testing.T) {
	userID := uuid.New()

	t.Run("auto clones a catalog item the user does not own", func(t *testing.T) {
		fashionID := uuid.New()
		wardrobe := &cloneWardrobeRepo{catalogRows: []*entities.WardrobeItem{catalogTemplate(fashionID, nil)}}
		outfits := &cloneOutfitRepo{}
		fixture := newCloneFixture(wardrobe, outfits, 10)

		_, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
			Name:  "Bộ tự động",
			Items: []dto.SaveOutfitItemReq{{FashionItemID: fashionID, Scale: 1, LayerOrder: 1}},
		})
		if err != nil {
			t.Fatalf("SaveOutfit: %v", err)
		}
		if len(wardrobe.bulkBatches) != 1 {
			t.Fatalf("clone batches = %d, want 1", len(wardrobe.bulkBatches))
		}
	})

	t.Run("explicit catalog rejects an item outside the system catalog", func(t *testing.T) {
		fashionID := uuid.New()
		wardrobe := &cloneWardrobeRepo{}
		outfits := &cloneOutfitRepo{}
		fixture := newCloneFixture(wardrobe, outfits, 10)
		systemCatalog := itemContextSystemCatalog

		res, err := fixture.uc.SaveOutfit(context.Background(), userID, dto.SaveOutfitReq{
			Name:  "Bộ danh mục sai",
			Items: []dto.SaveOutfitItemReq{{FashionItemID: fashionID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 1}},
		})
		if res != nil {
			t.Fatalf("response = %+v, want nil", res)
		}
		cloneAssertSameError(t, err, wardrobeerrors.ErrCatalogItemUnavailable())
		if len(wardrobe.bulkBatches) != 0 || outfits.createCalls != 0 {
			t.Fatalf("no write may happen for an unavailable catalog item")
		}
	})
}

func TestUpdateOutfitClonesOnlyTheNewItemAndSignalsOnlyIt(t *testing.T) {
	userID := uuid.New()
	outfitID := uuid.New()
	keepFashionID := uuid.New()
	newFashionID := uuid.New()

	outfit := &entities.Outfit{UserID: userID, Name: "Bộ cũ"}
	outfit.ID = outfitID
	outfit.Items = []*entities.OutfitItem{{
		OutfitID:      outfitID,
		FashionItemID: keepFashionID,
		ItemContext:   outfititemcontext.UserWardrobe,
	}}

	wardrobe := &cloneWardrobeRepo{
		userItems:   []*entities.WardrobeItem{ownedRow(keepFashionID, wardrobestatus.InWardrobe)},
		ownedRows:   []*entities.WardrobeItem{ownedRow(keepFashionID, wardrobestatus.InWardrobe)},
		catalogRows: []*entities.WardrobeItem{catalogTemplate(newFashionID, nil)},
	}
	outfits := &cloneOutfitRepo{detail: outfit, detailItems: outfit.Items}
	fixture := newCloneFixture(wardrobe, outfits, 10)
	systemCatalog := itemContextSystemCatalog

	res, err := fixture.uc.UpdateOutfit(context.Background(), userID, outfitID, dto.SaveOutfitReq{
		Name: "Bộ mới",
		Items: []dto.SaveOutfitItemReq{
			{FashionItemID: keepFashionID, Scale: 1, LayerOrder: 1},
			{FashionItemID: newFashionID, ItemContext: &systemCatalog, Scale: 1, LayerOrder: 2},
		},
	})
	if err != nil {
		t.Fatalf("UpdateOutfit: %v", err)
	}
	if res == nil || len(res.Items) != 2 {
		t.Fatalf("response items = %+v, want 2", res)
	}
	if outfits.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", outfits.updateCalls)
	}
	if len(wardrobe.bulkBatches) != 1 || len(wardrobe.bulkBatches[0]) != 1 {
		t.Fatalf("clone batches = %v, want exactly the new item", wardrobe.bulkBatches)
	}
	// Only the newly added item is touched (FR-062).
	if len(wardrobe.touchIDs) != 1 || wardrobe.touchIDs[0] != wardrobe.bulkBatches[0][0].ID {
		t.Fatalf("touch ids = %v, want only the cloned row", wardrobe.touchIDs)
	}
	payload, ok := fixture.publisher.payloads[0].(dto.OutfitSavedEventPayload)
	if !ok {
		t.Fatalf("payload type = %T, want dto.OutfitSavedEventPayload", fixture.publisher.payloads[0])
	}
	if len(payload.FashionItemIDs) != 1 || payload.FashionItemIDs[0] != newFashionID {
		t.Fatalf("signal ids = %v, want only the new fashion item", payload.FashionItemIDs)
	}
}

func TestUpdateOutfitPublishesNothingWhenMetadataOnlyChanges(t *testing.T) {
	userID := uuid.New()
	outfitID := uuid.New()
	keepFashionID := uuid.New()

	outfit := &entities.Outfit{UserID: userID, Name: "Bộ cũ"}
	outfit.ID = outfitID
	outfit.Items = []*entities.OutfitItem{{
		OutfitID:      outfitID,
		FashionItemID: keepFashionID,
		ItemContext:   outfititemcontext.UserWardrobe,
	}}

	wardrobe := &cloneWardrobeRepo{ownedRows: []*entities.WardrobeItem{ownedRow(keepFashionID, wardrobestatus.InWardrobe)}}
	outfits := &cloneOutfitRepo{detail: outfit, detailItems: outfit.Items}
	fixture := newCloneFixture(wardrobe, outfits, 10)

	if _, err := fixture.uc.UpdateOutfit(context.Background(), userID, outfitID, dto.SaveOutfitReq{
		Name:  "Bộ đổi tên",
		Items: []dto.SaveOutfitItemReq{{FashionItemID: keepFashionID, Scale: 1, LayerOrder: 1}},
	}); err != nil {
		t.Fatalf("UpdateOutfit: %v", err)
	}

	if len(fixture.publisher.topics) != 0 {
		t.Fatalf("published topics = %v, want none for a metadata-only update", fixture.publisher.topics)
	}
	if len(wardrobe.touchIDs) != 0 {
		t.Fatalf("touch ids = %v, want none for unchanged items", wardrobe.touchIDs)
	}
}
