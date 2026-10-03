package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CampaignRepository is the registry repository (026, T006).
//
// Every method resolves its handle through the shared repository base, like the
// claim ledger does, so a transaction injected into ctx by `IUnitOfWork.Execute`
// is reused. `GetByCodeForShare` depends on that: a `FOR SHARE` taken on a handle
// outside the caller's transaction would lock nothing the close could wait on.
//
// It embeds the generic base for its `GetDB` but does NOT satisfy
// `IGenericRepository`, for the same reason the claim ledger does not: this table
// has no delete path (FR-005) and its writes report whether they changed a row
// (FR-055), neither of which the generic CRUD set can express.
type CampaignRepository struct {
	*shared_repos.GenericRepository[entities.Campaign, string]
}

func NewCampaignRepository(dbConn *gorm.DB) repositories.ICampaignRepository {
	return &CampaignRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.Campaign, string](dbConn, nil),
	}
}

// ListAll returns every campaign, newest-starting first.
//
// `code DESC` is the deterministic tie-break (FR-040): two campaigns may share a
// `starts_at`, and without a second ordering key the same registry could render
// in a different order on two consecutive calls, which on the admin list looks
// like rows moving under the operator's cursor.
//
// It serves `ORDER BY starts_at DESC` from `ix_campaigns_starts_at`.
func (r *CampaignRepository) ListAll(ctx context.Context) ([]*entities.Campaign, error) {
	return r.listOrdered(ctx, nil)
}

// ListOpen returns the campaigns that are not force-closed (026, FR-031/FR-070).
//
// `closed_at IS NULL` is the ONLY term, and the comment on the interface method
// explains why the window is applied in Go instead: `campaignEvaluator.isAwake` is
// specified to be the same predicate as `data-model.md` §4.1, and it cannot be
// both — dropping the window here would turn every asleep campaign into
// `campaign_not_configured` and lose the `campaign_sleeping` reason code
// (spec.md Câu 4, FR-042).
func (r *CampaignRepository) ListOpen(ctx context.Context) ([]*entities.Campaign, error) {
	return r.listOrdered(ctx, func(query *gorm.DB) *gorm.DB {
		return query.Where("closed_at IS NULL")
	})
}

// listOrdered is the one query both listings share.
//
// One implementation rather than two near-identical ones: the ordering IS part of
// FR-040's contract, and a copy that dropped `code DESC` would keep every
// behavioural test green while making the result non-deterministic.
func (r *CampaignRepository) listOrdered(
	ctx context.Context,
	scoped func(*gorm.DB) *gorm.DB,
) ([]*entities.Campaign, error) {
	query := r.GetDB(ctx).Model(&entities.Campaign{})
	if scoped != nil {
		query = scoped(query)
	}
	var campaigns []*entities.Campaign
	if err := query.Order("starts_at DESC, code DESC").Find(&campaigns).Error; err != nil {
		return nil, err
	}
	return campaigns, nil
}

// GetByCode returns one campaign, or nil when the code does not exist.
//
// `nil, nil` for "absent" rather than an error, so the caller answers 404 for a
// code nobody typed and 500 for a database that is down — two situations an
// operator has to tell apart (FR-054).
func (r *CampaignRepository) GetByCode(ctx context.Context, campaignCode string) (*entities.Campaign, error) {
	var campaign entities.Campaign
	if err := r.GetDB(ctx).Where("code = ?", campaignCode).First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &campaign, nil
}

// GetByCodeForShare re-reads the row under a SHARE lock (FR-045).
//
// `FOR SHARE` and NOT `FOR UPDATE`, and the difference is the point: a close or a
// budget edit has to WAIT for this transaction, while other grants reading the
// same row do not — so a burst of signups against a live campaign is not
// serialised behind one another, which `FOR UPDATE` would do.
//
// Deadlock-free by construction: every grant already serialises on the campaign
// advisory lock first, so the order is always "advisory → row share", and the
// close path takes only the row lock (research.md R12).
func (r *CampaignRepository) GetByCodeForShare(ctx context.Context, campaignCode string) (*entities.Campaign, error) {
	var campaign entities.Campaign
	err := r.GetDB(ctx).
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where("code = ?", campaignCode).
		First(&campaign).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &campaign, nil
}

// Create inserts a new campaign.
//
// The duplicate-code case is left to the `PRIMARY KEY` rather than pre-checked:
// a read-then-write has a window between the two statements, and two
// administrators creating the same code at the same moment is exactly the case
// where the loser must be told 409 (FR-054) rather than be inserted under a name
// that already exists.
func (r *CampaignRepository) Create(ctx context.Context, campaign *entities.Campaign) error {
	return r.GetDB(ctx).Create(campaign).Error
}

// Update writes the mutable fields and reports whether a row changed (FR-055).
//
// THE COLUMN LIST IS THE POINT. It names `plan_slug`, `quota`, `reserve`,
// `starts_at` and `ends_at` one by one rather than saving the struct, so `code`
// can never be rewritten (FR-024) and `closed_at` cannot be cleared by an
// accidental field added to the entity later (FR-034).
//
// `version = version + 1` is computed by the STATEMENT, which is why no call site
// has to remember it: an increment written by hand is an increment a later method
// forgets, and a forgotten increment turns every subsequent write into a silent
// overwrite.
//
// Zero rows means "the row is not there at the expected version". The caller
// answers 412 for the version case and 404 for the missing one, having read the
// code first — the two are different facts and FR-054 gives them different codes.
func (r *CampaignRepository) Update(
	ctx context.Context,
	campaign *entities.Campaign,
	expectedVersion int,
) (bool, error) {
	result := r.GetDB(ctx).
		Model(&entities.Campaign{}).
		Where("code = ? AND version = ?", campaign.Code, expectedVersion).
		Updates(map[string]any{
			"plan_slug":  campaign.PlanSlug,
			"quota":      campaign.Quota,
			"reserve":    campaign.Reserve,
			"starts_at":  campaign.StartsAt,
			"ends_at":    campaign.EndsAt,
			"version":    gorm.Expr("version + 1"),
			"updated_at": gorm.Expr("now()"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// Close force-closes a campaign and reports whether a row changed (FR-030/FR-031).
//
// A separate statement rather than an `Update` with `closed_at` set, so the
// immutability trigger `trg_campaigns_immutable_budget` has nothing to compare
// against a budget change, and so "close" reads at the call site as the
// irreversible thing it is (FR-034).
//
// `closed_at = now()` is the DATABASE clock rather than an instant passed in: the
// value answers "when was this campaign closed" for an operator reading the audit
// trail weeks later, and a caller's clock is not a trustworthy witness to its own
// request. The expected version is still compared, so a close racing a budget edit
// still resolves to one winner.
func (r *CampaignRepository) Close(
	ctx context.Context,
	campaignCode string,
	expectedVersion int,
) (bool, error) {
	result := r.GetDB(ctx).
		Model(&entities.Campaign{}).
		Where("code = ? AND version = ?", campaignCode, expectedVersion).
		Updates(map[string]any{
			"closed_at":  gorm.Expr("now()"),
			"version":    gorm.Expr("version + 1"),
			"updated_at": gorm.Expr("now()"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ExistsClaims reports whether the campaign has granted at least once (FR-020).
//
// `EXISTS` rather than `COUNT(*) > 0`: the answer is a boolean, the ledger holds a
// campaign's whole lifetime of grants, and a count would read every row to
// discard all but the first.
func (r *CampaignRepository) ExistsClaims(ctx context.Context, campaignCode string) (bool, error) {
	var count int64
	err := r.GetDB(ctx).
		Model(&entities.CampaignClaim{}).
		Where("campaign_code = ?", campaignCode).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

var _ repositories.ICampaignRepository = (*CampaignRepository)(nil)
