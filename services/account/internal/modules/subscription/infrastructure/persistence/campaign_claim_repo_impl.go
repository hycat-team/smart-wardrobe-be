package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// campaignClaimListDefaultLimit bounds an unbounded ListByCampaign.
//
// A campaign's claim count is bounded by its configured quota plus reserve, so
// this is a guard against a caller that forgets to pass a limit rather than a
// number derived from the mechanism.
const campaignClaimListDefaultLimit = 100

// CampaignClaimRepository implements the campaign claim ledger (025 T034).
//
// Every method resolves its handle through the shared repository base, so a
// transaction injected into ctx by `IUnitOfWork.Execute` is reused. That matters
// here more than elsewhere: the quota decision reads a count and then writes a
// claim, and the two have to be atomic — two handles would make that impossible.
//
// It embeds the generic base for its `GetDB`, but does NOT satisfy
// `IGenericRepository`: the ledger's `Create` reports whether it inserted, which
// the generic `Create(ctx, entity) error` cannot express (FR-025).
type CampaignClaimRepository struct {
	*shared_repos.GenericRepository[entities.CampaignClaim, uuid.UUID]
}

func NewCampaignClaimRepository(dbConn *gorm.DB) repositories.ICampaignClaimRepository {
	return &CampaignClaimRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.CampaignClaim, uuid.UUID](dbConn, nil),
	}
}

// LockClaimWindow serialises quota decisions for one campaign (R3).
//
// An advisory lock rather than a row lock, because the case that matters is a
// campaign with NO claims yet: a row lock locks nothing, so two concurrent
// messages both read a count of zero and both grant. That is how the quota was
// overrun.
//
// Keyed on the table name, not on the campaign code, deliberately: FR-022/FR-024
// cap the MECHANISM's budget, and two campaigns running at once must not be able
// to spend each other's allowance by holding different locks.
//
// `hashtextextended` maps the name onto a 64-bit key. The hash is not a security
// boundary — a collision only makes two unrelated campaigns serialise against each
// other for a few milliseconds.
//
// It MUST be called inside `IUnitOfWork.Execute`: the lock is released at COMMIT
// or ROLLBACK, so a worker that dies mid-way leaves nothing behind. Calling it
// outside a transaction would take a session-level lock on a pooled connection
// and hold it for the connection's lifetime.
func (r *CampaignClaimRepository) LockClaimWindow(ctx context.Context, campaignCode string) error {
	// `campaignCode` is deliberately unused in the key, but the parameter is kept
	// because callers pass the campaign they are about to decide on, and dropping
	// it would make the lock look per-campaign at the call site.
	_ = campaignCode
	return r.GetDB(ctx).Exec(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		"subscription.campaign_claims",
	).Error
}

// CountByBucket counts ONE budget part of one campaign.
//
// Reads `ix_campaign_claims_campaign_bucket`, so it scans the main part or the
// reserve part rather than the whole table.
//
// It counts on `grant_bucket`, never on `NOT fills_main_quota`: that flag marks
// exactly one row, so its negation is "total minus one" and folds the reserve
// into the main count (research.md R2b).
func (r *CampaignClaimRepository) CountByBucket(ctx context.Context, campaignCode string, bucket entities.GrantBucket) (int64, error) {
	var count int64
	err := r.GetDB(ctx).
		Model(&entities.CampaignClaim{}).
		Where("campaign_code = ? AND grant_bucket = ?", campaignCode, string(bucket)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetWatermark returns the FR-030 priority watermark for a campaign.
//
// `found` is returned alongside the value because `registered_at` is NOT NULL
// while the ROW itself may be absent — the normal state before the main budget
// fills. Without the flag those two are indistinguishable, and the reserve branch
// would grant against a watermark that does not exist.
func (r *CampaignClaimRepository) GetWatermark(ctx context.Context, campaignCode string) (time.Time, bool, error) {
	var claim entities.CampaignClaim
	err := r.GetDB(ctx).
		Select("registered_at").
		Where("campaign_code = ? AND fills_main_quota = ?", campaignCode, true).
		First(&claim).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return claim.RegisteredAt, true, nil
}

// Create writes the claim unless the account already has one, and reports whether
// it inserted.
//
// `ON CONFLICT (user_id) DO NOTHING` is what enforces FR-025 / FR-012, and it is
// the INSERT that does it rather than a check that runs first: two concurrent
// callers both pass any pre-check, and only one of them can win the constraint.
// The loser sees zero rows affected and reports "already claimed", so the caller
// re-reads instead of granting twice.
//
// The count and this insert run under the same advisory lock (LockClaimWindow), so
// the counted total and the written row agree.
func (r *CampaignClaimRepository) Create(ctx context.Context, claim *entities.CampaignClaim) (bool, error) {
	if claim.ID == uuid.Nil {
		claim.ID = uuid.New()
	}
	result := r.GetDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(claim)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ListByCampaign returns a campaign's claims newest-first (FR-062).
//
// Reads `ix_campaign_claims_campaign_granted`, so "who was skipped and when" is a
// bounded index walk instead of a sort over the ledger.
func (r *CampaignClaimRepository) ListByCampaign(ctx context.Context, campaignCode string, limit int) ([]*entities.CampaignClaim, error) {
	if limit <= 0 || limit > campaignClaimListDefaultLimit {
		limit = campaignClaimListDefaultLimit
	}
	var claims []*entities.CampaignClaim
	err := r.GetDB(ctx).
		Where("campaign_code = ?", campaignCode).
		Order("granted_at DESC").
		Limit(limit).
		Find(&claims).Error
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ListPageByCampaign returns one page of a campaign's claims plus the total for
// the same filter (FR-066, FR-067).
//
// The count and the page are read from the same handle, so they describe the same
// database: a campaign being granted to while this runs can make the total one
// higher than the rows shown, which is a snapshot an operator will understand,
// rather than a total from a different moment that no page can reconcile.
func (r *CampaignClaimRepository) ListPageByCampaign(
	ctx context.Context,
	campaignCode string,
	filter repositories.CampaignClaimsFilter,
	page, limit int,
) ([]*entities.CampaignClaim, int64, error) {
	page, limit = normalizeCampaignClaimPage(page, limit)
	query := r.GetDB(ctx).Model(&entities.CampaignClaim{}).Where("campaign_code = ?", campaignCode)
	if filter.From != nil {
		query = query.Where("granted_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("granted_at < ?", *filter.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var claims []*entities.CampaignClaim
	err := query.
		Order("granted_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&claims).Error
	if err != nil {
		return nil, 0, err
	}
	return claims, total, nil
}

// normalizeCampaignClaimPage keeps a caller-supplied page and limit inside the
// bounds the shared pagination ceiling allows.
//
// The upper bound is `PaginationQuery.Normalize`'s 100, not a number derived from
// the campaign: a campaign cannot hold more than `quota + reserve` claims, so the
// only way a limit above 100 could be useful is a campaign configured above it —
// and if that ever happens, the right answer is 100 rows a page and a correct
// total, not a silently truncated list that reports itself as complete.
func normalizeCampaignClaimPage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = campaignClaimListDefaultLimit
	}
	if limit > campaignClaimListDefaultLimit {
		limit = campaignClaimListDefaultLimit
	}
	return page, limit
}

// GetByUser returns the account's claim whatever campaign wrote it (FR-012).
//
// `found` is returned alongside the value for the same reason `GetWatermark`
// returns it: the row may legitimately be ABSENT, and an absent row must not be
// reported as a claim whose fields happen to be zero.
func (r *CampaignClaimRepository) GetByUser(ctx context.Context, userID uuid.UUID) (*entities.CampaignClaim, bool, error) {
	var claim entities.CampaignClaim
	err := r.GetDB(ctx).Where("user_id = ?", userID).First(&claim).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &claim, true, nil
}

var _ repositories.ICampaignClaimRepository = (*CampaignClaimRepository)(nil)
