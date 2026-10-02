package persistence

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CampaignSweepRunRepository persists the catch-up sweep's health (025 T079c,
// FR-059).
//
// Every method resolves its handle through the shared repository base, like the
// claim ledger does. There is no transaction here by design: the row is written
// AFTER the sweep has finished, outside the grants it performed, so that a
// failure to record the health of a run can never roll back a plan a customer
// was already given.
type CampaignSweepRunRepository struct {
	*shared_repos.GenericRepository[entities.CampaignSweepRun, string]
}

func NewCampaignSweepRunRepository(dbConn *gorm.DB) repositories.ICampaignSweepRunRepository {
	return &CampaignSweepRunRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.CampaignSweepRun, string](dbConn, nil),
	}
}

// RecordRun upserts one row per campaign after a sweep (FR-059).
//
// The asymmetry between a successful and a failed run is the whole reason the
// columns are separate:
//
//   - SUCCESS writes `last_sweep_at = swept_at`, `degraded = false`, and clears
//     `degraded_reason`. It is the answer to "is the mechanism granting again",
//     and "it stopped breaking" is as much of an event as "it broke".
//   - FAILURE writes `degraded = true` and `degraded_reason`, and LEAVES
//     `last_sweep_at` alone. A run that failed is not evidence that a sweep last
//     succeeded, and moving the column would turn a sweep failing every fifteen
//     minutes for six hours into a timestamp that looks fifteen minutes old —
//     the exact false all-clear FR-059 exists to prevent.
//
// Two replicas writing at once is safe because both write the same shape: the
// later run's verdict is the current one, and neither read-modify-writes a
// counter. There is nothing to lose by interleaving.
func (r *CampaignSweepRunRepository) RecordRun(ctx context.Context, results []repositories.CampaignSweepResult) error {
	if len(results) == 0 {
		return nil
	}
	runs := make([]entities.CampaignSweepRun, 0, len(results))
	for _, result := range results {
		run := entities.CampaignSweepRun{
			CampaignCode: result.CampaignCode,
			Degraded:     !result.Succeeded,
		}
		if result.Succeeded {
			sweptAt := result.SweptAt
			run.LastSweepAt = &sweptAt
			// Left nil deliberately: `degraded_reason` is only meaningful beside a
			// true `degraded`, and a stale reason beside a false flag would name a
			// fault that no longer exists.
		} else {
			reason := result.FailureReason
			run.DegradedReason = &reason
		}
		runs = append(runs, run)
	}

	return r.GetDB(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "campaign_code"}},
			// `last_sweep_at` is EXCLUDED from the update list on purpose: it is
			// only ever advanced by a successful run, and naming it here would let
			// a failure overwrite the timestamp of the last run that worked.
			DoUpdates: clause.AssignmentColumns([]string{"degraded", "degraded_reason"}),
		}).
		Create(&runs).Error
}

// GetByCampaigns returns the recorded health of the named campaigns (FR-065).
//
// One query for the whole campaign list, and the caller matches codes to rows
// itself: the list endpoint renders one row per CONFIGURED campaign, so querying
// per campaign would be one round trip per rendered row.
//
// A campaign with no row is simply absent from the map. That absence is the
// answer "no sweep has ever completed here" and the caller must render it as an
// unset `lastSweepAt` — never as the zero time, which would read as "swept at the
// year 1".
func (r *CampaignSweepRunRepository) GetByCampaigns(ctx context.Context, campaignCodes []string) (map[string]*entities.CampaignSweepRun, error) {
	out := make(map[string]*entities.CampaignSweepRun, len(campaignCodes))
	if len(campaignCodes) == 0 {
		return out, nil
	}

	var runs []entities.CampaignSweepRun
	err := r.GetDB(ctx).
		Where("campaign_code IN ?", campaignCodes).
		Find(&runs).Error
	if err != nil {
		return nil, err
	}
	for i := range runs {
		out[runs[i].CampaignCode] = &runs[i]
	}
	return out, nil
}

var _ repositories.ICampaignSweepRunRepository = (*CampaignSweepRunRepository)(nil)
