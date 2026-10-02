package usecase

import (
	"context"
	"time"
)

// CampaignSweepReport is what ONE catch-up sweep run observed (025 T097, FR-079).
//
// It is a report of the sweep's OWN work, not a report of the mechanism's
// decisions. The sweep does not decide anything — it hands accounts to the same
// use case the message path uses (FR-080) — so it cannot say how many plans were
// granted without becoming a second decision path. What it CAN say, and what a log
// line can usefully record, is how much it looked at, how much it passed on, how
// much it passed on late, and what went wrong.
//
// IT IS NOT THE OPERATOR'S VIEW OF SWEEP HEALTH. That used to be a second type on
// this interface — a live flag an HTTP handler could read — and it was removed at
// 025 T079c, with the reasons worth keeping:
//
// The flag lived in the sweep worker's memory, which is only correct while there is
// exactly ONE worker, and there is not: this service runs as several replicas,
// each with its own worker set. Two failures follow, both in the dangerous
// direction. An HTTP request landing on replica B read replica B's flag while the
// sweep that actually failed was run by replica A — so the operator page reported
// HEALTHY while the mechanism was broken, which is the single answer FR-059 exists
// to prevent. And under `APP_MODE=http` no sweep worker runs at all, so the flag
// was permanently false: a false all-clear that is structurally guaranteed rather
// than observed.
//
// The state therefore lives in `subscription.campaign_sweep_runs`, one row per
// campaign, written by the sweep itself and read by the operator look-up (FR-065)
// — identically on every replica, and with no worker running at all.
type CampaignSweepReport struct {
	// SweptAt is the instant the run read once and used for every comparison, so
	// every account in the run was judged against the same "now" (R6).
	SweptAt time.Time
	// Scanned is how many accounts the run read out of the window.
	Scanned int
	// Forwarded is how many accounts were handed to the shared grant path.
	Forwarded int
	// Skipped is how many were dropped without being handed over: administrator
	// accounts (condition 3) and accounts that already hold a claim (condition 4).
	Skipped int
	// Failed is how many could not be evaluated at all — a transient error, so
	// the account is still unsettled and the NEXT sweep tries again.
	Failed int
	// LateAccounts is the FR-082 signal: accounts older than the five-minute
	// commitment that the message path had not settled, so this run had to.
	//
	// It counts accounts the sweep CONSIDERED, not accounts it granted — the
	// sweep has no way to know the latter without a second decision path — so it
	// is an upper bound on late grants, which is the safe direction for an alert.
	//
	// It is a per-run figure rather than a persisted flag for the same reason the
	// health is persisted: it describes what THIS pass saw, and persisting it would
	// invite a reader to treat "no late accounts last time" as "the queue is
	// healthy", which is a different claim.
	LateAccounts int
	// MaxDelaySeconds is the worst lateness seen this run (FR-082: "độ trễ lần
	// nhất"). Zero when nothing was late.
	MaxDelaySeconds int
}

// ICampaignSweepUseCase is the catch-up sweep's application entry point
// (025 T097).
//
// It deliberately does NOT sit on `ICampaignUseCase`. There is one rule
// implementation — `CampaignUseCase.ProcessUserCreated` — and this interface owns
// only the work AROUND it: walking the window in batches, refusing to re-open a
// transaction for an account that already holds a claim, and recording what the
// run saw.
type ICampaignSweepUseCase interface {
	// Sweep runs one catch-up pass over the campaign window.
	//
	// A skip is a success: an account that turns out to be ineligible, asleep,
	// or already granted returns no error. An error means the sweep could not
	// finish, and the next run repeats whatever it did not reach — which is safe
	// because the shared grant path is idempotent (FR-012/FR-025).
	//
	// It RECORDS ITS OWN HEALTH before returning, into
	// `subscription.campaign_sweep_runs` (025 T079c, FR-059). Recording is part of
	// the run rather than the caller's job because the caller is a cron tick: a
	// failure to record would then be logged by the scheduler and forgotten, and
	// the operator page would keep reporting the previous state indefinitely.
	Sweep(ctx context.Context) (CampaignSweepReport, error)
}
