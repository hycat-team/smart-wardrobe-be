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
// granted without becoming a second decision path. What it CAN say, and what an
// operator needs, is how much it looked at, how much it passed on, how much it
// passed on late, and what went wrong.
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
	LateAccounts int
	// MaxDelaySeconds is the worst lateness seen this run (FR-082: "độ trễ lần
	// nhất"). Zero when nothing was late.
	MaxDelaySeconds int
}

// CampaignSweepHealth is the operator's view of the sweep as a SYSTEM rather than
// as a single run (FR-059).
//
// `data-model.md` §5.1 asks for three things an operator must be able to tell
// apart: a campaign is running, a campaign has gone to sleep, and a campaign is
// RUNNING BUT BROKEN — configured, in its window, and not granting anything. Only
// the third is a fault, and no per-account log line can show it: every skip looks
// locally correct. This snapshot is the answer, and it is kept IN MEMORY rather
// than persisted because a stale persisted flag outlives the process that set it
// and would report a healthy sweep as broken after a restart.
type CampaignSweepHealth struct {
	// Degraded is set when the most recent sweep failed, and reset by the next
	// sweep that completes. FR-059's third state: running, but not granting.
	Degraded bool
	// DegradedReason is the most recent failure, so the operator does not have to
	// correlate a timestamp with a log search.
	DegradedReason string
	// LastSweepAt is the instant of the most recent SUCCESSFUL sweep (FR-059).
	// A failure deliberately does not move it: "the last sweep that worked" and
	// "the last sweep that ran" are different questions and only the first one
	// answers "is the mechanism still running".
	LastSweepAt time.Time
	// LateAccountCount is how many accounts the last successful sweep found
	// unsettled past five minutes (FR-082). It is what separates "the system is
	// SLOW" from "the system is BROKEN": a non-zero count with `Degraded = false`
	// means the queue is congested, not that the mechanism is failing.
	LateAccountCount int
	// MaxDelaySeconds is the worst lateness from the last successful sweep.
	MaxDelaySeconds int
}

// ICampaignSweepUseCase is the catch-up sweep's application entry point
// (025 T097).
//
// It deliberately does NOT sit on `ICampaignUseCase`. There is one rule
// implementation — `CampaignUseCase.ProcessUserCreated` — and this interface owns
// only the work AROUND it: walking the window in batches, refusing to re-open a
// transaction for an account that already holds a claim, and reporting what the
// run saw.
type ICampaignSweepUseCase interface {
	// Sweep runs one catch-up pass over the campaign window.
	//
	// A skip is a success: an account that turns out to be ineligible, asleep,
	// or already granted returns no error. An error means the sweep could not
	// finish, and the next run repeats whatever it did not reach — which is safe
	// because the shared grant path is idempotent (FR-012/FR-025).
	Sweep(ctx context.Context) (CampaignSweepReport, error)

	// Health returns the operator-facing snapshot described above. It is a
	// snapshot of IN-MEMORY state, so it is safe to call concurrently with a run.
	Health() CampaignSweepHealth
}
