package campaign

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"go.uber.org/zap"
)

// The FR-059 health recording of a sweep run, in its own file (025 T079c, 026).
//
// WHY IT IS SEPARATE FROM THE RUN ITSELF. A sweep does two things that share no
// decision: it WALKS the window and forwards accounts to the shared grant path, and
// it RECORDS what the run saw. Splitting them keeps the run readable end to end —
// every branch of it is about accounts — and it makes the awkward part of the health
// write its own subject: what to do when the registry is empty, when the write
// itself fails, and which fields a failure must NOT advance.
//
// The two also differ in failure semantics, which is the reason they are not one
// function. A grant failure is a fault an operator must act on and the run reports
// it. A health-write failure is a lost RECORD: the grants are already committed, and
// turning it into a returned error would make a successful sweep look failed and
// send the whole window round again.
// recordSuccess writes the health of a run that finished, for EVERY campaign the
// run covered (025 T079c, FR-059).
//
// ONE write for all campaigns rather than one per campaign: a sweep covers every
// open campaign by design, and a per-campaign write would make a partial recording
// â€” half the campaigns marked healthy, half still showing the last fault â€” possible
// in a way a single statement is not.
//
// The degraded flag is CLEARED, not merely left alone: it describes the most
// recent sweep, and a run that completed without error is the answer to "is the
// mechanism granting again" â€” whatever happened an hour ago. `degraded_reason` is
// cleared with it, because a stale reason beside a healthy flag would name a fault
// that no longer exists.
//
// `campaigns` is NIL when there was nothing to sweep: an empty registry produces no
// rows at all, and that is the honest answer. Writing "healthy" rows for campaigns
// that do not exist would invent registry entries.
func (uc *CampaignSweepUseCase) recordSuccess(
	ctx context.Context,
	report uc_interfaces.CampaignSweepReport,
	campaigns []*entities.Campaign,
) {
	uc.persistHealth(ctx, report.SweptAt, true, "", campaigns)
}

// recordFailure writes the health of a run that could not finish (FR-059).
//
// `LastSweepAt` is deliberately NOT advanced: it answers "when did a sweep last
// succeed", and a failed run is not evidence that one did. The repository enforces
// the same asymmetry on the UPDATE, so a failure cannot overwrite it here even if
// a future caller passes the wrong instant. A sweep that fails every fifteen
// minutes for six hours therefore reports a `LastSweepAt` six hours old, which is
// the useful alarm.
func (uc *CampaignSweepUseCase) recordFailure(
	ctx context.Context,
	report uc_interfaces.CampaignSweepReport,
	campaigns []*entities.Campaign,
	err error,
) {
	uc.persistHealth(ctx, report.SweptAt, false, err.Error(), campaigns)
}

// persistHealth writes one row per campaign the run covered.
//
// The failure of the WRITE ITSELF is logged and swallowed rather than returned.
// The sweep has already finished by the time this runs, and its grants are
// already committed â€” turning "could not record that the run was healthy" into a
// returned error would make a successful sweep look failed, and a caller that
// retries on that error would sweep the whole window again for no reason. What is
// lost is only the record; the grants stand, and the next run rewrites the row.
//
// A campaign that has never been swept has NO row until a run happens, and the
// look-up reads that absence as `lastSweepAt` unset rather than as the zero time â€”
// which is the truth, and is exactly what an operator needs to be told.
func (uc *CampaignSweepUseCase) persistHealth(
	ctx context.Context,
	sweptAt time.Time,
	succeeded bool,
	reason string,
	campaigns []*entities.Campaign,
) {
	results := make([]repositories.CampaignSweepResult, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		results = append(results, repositories.CampaignSweepResult{
			CampaignCode:  campaign.Code,
			SweptAt:       sweptAt,
			Succeeded:     succeeded,
			FailureReason: reason,
		})
	}
	if err := uc.sweepRuns.RecordRun(ctx, results); err != nil {
		uc.log.Error("campaign sweep could not record its health; the operator look-up will show the previous state",
			zap.String("degraded_reason", reason),
			zap.Error(err),
		)
	}
}
