package worker

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_worker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// campaignSweepRunTimeout bounds one sweep run.
//
// It is LONGER than the default sweep interval on purpose. A run that overruns its
// slot must still be allowed to finish rather than be cut off half way through a
// window: the accounts it has already evaluated are settled and the rest are
// simply picked up next tick, so a truncated run wastes work. The timeout exists
// to stop a run that can never finish from holding the goroutine forever.
const campaignSweepRunTimeout = 15 * time.Minute

// CampaignSweepWorker runs the catch-up sweep on a schedule (025 T098, FR-083).
//
// It is the ONLY thing that exists because a message was lost: the grant path
// deliberately has no durability of its own (FR-089), so this worker is what makes
// "signed up, qualified, never told" an outcome the system cannot produce
// silently (FR-079).
//
// The cadence is READ FROM CONFIGURATION, never written here (FR-083). Two
// consequences of the same decision: a development instance can sweep every
// minute while production sweeps every fifteen, with no code difference; and there
// is no environment variable, because `campaign_sweep` lives in the same YAML as
// the campaigns themselves and a second way to configure the same thing is a
// second answer waiting to be given (R10).
type CampaignSweepWorker struct {
	*shared_worker.BaseWorker
	sweepUC    uc_interfaces.ICampaignSweepUseCase
	schedule   string
	runOnStart bool
	cronEngine *cron.Cron
	log        logger.Interface
	// inFlight is the "one run at a time" guard, held for the duration of a run
	// rather than for the duration of the tick.
	inFlight atomic.Bool
}

func NewCampaignSweepWorker(
	sweepUC uc_interfaces.ICampaignSweepUseCase,
	cfg *sharedconfig.Config,
	log logger.Interface,
) *CampaignSweepWorker {
	interval := cfg.CampaignSweep.IntervalSeconds
	if interval <= 0 {
		// The loader defaults this key and validation rejects anything under the
		// floor, so production cannot reach here. A worker whose schedule were
		// empty would register a cron entry that never fires — a sweep that looks
		// wired and grants nothing — so the floor is used as the fallback rather
		// than trusting that a config path was always taken.
		interval = sharedconfig.DefaultCampaignSweepIntervalSeconds
	}
	return &CampaignSweepWorker{
		BaseWorker: shared_worker.NewBaseWorker("campaign_sweep"),
		sweepUC:    sweepUC,
		// `@every <n>s` rather than a six-field cron expression: the requirement is
		// a PERIOD, and writing it as one keeps "900" in the YAML meaning 900
		// seconds rather than depending on which cron field it landed in.
		schedule:   fmt.Sprintf("@every %ds", interval),
		runOnStart: cfg.CampaignSweep.RunOnStart,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *CampaignSweepWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

	// One run at startup, which closes the gap the last sweep before the service
	// stopped left open (FR-079, SC-016). It is configurable because a deployment
	// that starts many replicas at once can reasonably not want every replica to
	// scan the window immediately.
	if w.runOnStart {
		go w.executeSweep(workerlog.TriggerStartup)
	}

	if _, err := w.cronEngine.AddFunc(w.schedule, func() {
		w.executeSweep(workerlog.TriggerCron)
	}); err != nil {
		w.log.Error("Failed to register campaign sweep cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
	w.log.Info("Campaign sweep worker started",
		zap.String("schedule", w.schedule),
		zap.Bool("run_on_start", w.runOnStart),
	)
}

func (w *CampaignSweepWorker) Stop() {
	if w.cronEngine != nil {
		// Stops the scheduler. A run already in flight finishes under its own
		// timeout: cancelling it half way would leave accounts for the next
		// startup to find, which is exactly what the startup run is for.
		w.cronEngine.Stop()
	}
	w.BaseWorker.Stop()
}

// executeSweep runs one pass and reports what it saw.
//
// A failure is LOGGED, never returned and never fatal: FR-041 requires the
// campaign mechanism not to be on the critical path of anything, and a worker
// that took the process down over a sweep would take the message consumer and the
// grant path down with it. The failure is still visible — it sets the degraded
// flag the health line below reports (FR-059) — it is simply not an outage.
func (w *CampaignSweepWorker) executeSweep(triggerType string) {
	// A run that overruns its slot must not be joined by a second one. cron offers
	// `SkipIfStillRunning`, but it wants a `cron.Logger` and the only place in this
	// worker that cares about a skip is the service log, so the guard is these two
	// lines rather than an adapter whose output nobody reads. Overlapping runs
	// would also gain nothing: the sweep is idempotent, so the second pass would
	// find the accounts already granted and repeat the same work.
	if !w.inFlight.CompareAndSwap(false, true) {
		w.log.Warn("Previous campaign sweep is still running; skipping this tick",
			zap.String("schedule", w.schedule))
		return
	}
	defer w.inFlight.Store(false)

	// A background context, not the Start context: `Start`'s context is cancelled
	// on shutdown, and a sweep that stopped halfway would leave a window
	// unexamined until the next process started.
	ctx, cancel := context.WithTimeout(context.Background(), campaignSweepRunTimeout)
	defer cancel()

	run := workerlog.New("campaign_sweep", triggerType)

	report, err := w.sweepUC.Sweep(ctx)
	if err != nil {
		run.LogFailure(w.log, err)
		w.logSweepHealth(report, err)
		return
	}

	run.AddTotal(report.Scanned)
	run.AddSuccess(report.Forwarded)
	run.AddSkipped(report.Skipped)
	run.AddError(report.Failed)
	run.LogSuccessIfData(w.log,
		zap.Int("forwarded", report.Forwarded),
		zap.Int("skipped", report.Skipped),
		zap.Int("failed", report.Failed),
		zap.Int("late_accounts", report.LateAccounts),
		zap.Int("max_delay_seconds", report.MaxDelaySeconds),
	)

	w.logSweepHealth(report, nil)
}

// logSweepHealth writes the FR-059 line for ONE run.
//
// It is logged on EVERY run, including a clean one, and that is the point: the
// three states an operator must tell apart are "running", "asleep" and "running
// but not granting", and only a periodic line can answer "since when". A line
// that appears only on failure can say that something broke but never that it
// stopped breaking.
//
// IT REPORTS WHAT THIS RUN SAW, and it says so, because it is no longer the
// system-wide health (025 T079c). The authoritative state lives in
// `subscription.campaign_sweep_runs` and is read by the operator look-up; this
// line is the log-side record of the run that just happened on THIS replica. The
// two can legitimately disagree for one interval — replica A's failing sweep is
// written to the table, and replica B's succeeding sweep writes its own verdict —
// and that is the correct behaviour rather than a defect: two replicas running at
// the same moment genuinely both happened.
//
// `runErr` is the failure that ended the run, or nil. It is a parameter rather
// than a field because the worker's own view of "did this run fail" is the error
// the run returned, and reading it from anywhere else would let the log line
// disagree with what `Sweep` reported to the scheduler.
func (w *CampaignSweepWorker) logSweepHealth(report uc_interfaces.CampaignSweepReport, runErr error) {
	fields := []zap.Field{
		zap.Bool("degraded", runErr != nil),
		zap.Int("late_account_count", report.LateAccounts),
		zap.Int("max_delay_seconds", report.MaxDelaySeconds),
	}
	if !report.SweptAt.IsZero() {
		fields = append(fields, zap.Time("swept_at", report.SweptAt))
	}
	if runErr != nil {
		fields = append(fields, zap.String("degraded_reason", runErr.Error()))
		w.log.Error("campaign sweep run failed", fields...)
		return
	}
	w.log.Info("campaign sweep run completed", fields...)
}
