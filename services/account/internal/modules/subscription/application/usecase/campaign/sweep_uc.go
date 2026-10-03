package campaign

import (
	"context"
	"time"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// sweepBatchSize is how many accounts one `ListCreatedInWindow` call returns.
	//
	// It is the shared list ceiling (`shared_dto.PaginationQuery.Normalize`
	// clamps anything above 100 to 100), so asking for more would silently read
	// fewer and lie about the batch. 100 is also the right order of magnitude: a
	// sweep over a window with hundreds of thousands of accounts would take long
	// enough to matter, and the shared path takes an advisory lock PER ACCOUNT,
	// so a bigger batch buys nothing but a longer transaction-free stretch.
	sweepBatchSize = 100

	// campaignLateThreshold is the FR-078 commitment restated as a threshold the
	// sweep can OBSERVE. Accounts older than this that the message path has not
	// settled are exactly the accounts FR-082 asks to be recorded, and 5 minutes
	// is the number the specification commits to in two places (FR-078 and SC-015)
	// — so it is a named constant rather than a literal at the comparison, and
	// nobody has to go looking for where the number came from.
	campaignLateThreshold = 5 * time.Minute

	// maxSweepConsecutiveFailures bounds a run that keeps failing (FR-062).
	//
	// A single failed account must NOT end the run: the next sweep would pick it
	// up, but ending the run there would abandon every account behind it for no
	// reason. The opposite failure is real too — with the database down, every
	// account errors, and an unbounded loop turns one outage into a hundred
	// thousand log lines in one run. Ten in a row is far enough to distinguish
	// "one bad account" from "the system is down", and the run stops with an
	// error so the health snapshot reports the fault (FR-059).
	maxSweepConsecutiveFailures = 10
)

// CampaignSweepUseCase is the catch-up sweep: the safety net under a message that
// was never delivered (025 T097, FR-079).
//
// WHY IT EXISTS. The grant path is publisher-driven and deliberately has no
// durability of its own (FR-089): the message is fast, nothing waits in a retry
// loop, and correctness rests on the sweep. Without it a lost message is a
// customer who signed up, qualified, and was never told — permanently and
// silently, which is the one outcome worse than being late.
//
// WHY IT HOLDS NO RULES OF ITS OWN. Every decision — which campaign, whether the
// account is eligible, whether the budget allows it — lives in
// `CampaignUseCase.ProcessUserCreated`, and the sweep reaches it by calling THAT
// method (FR-080). This type walks a list and counts. If it grew a second
// eligibility check, the same account could be granted here and refused there, and
// nothing would report the disagreement: both answers would look individually
// reasonable. That is the whole reason the sweep exists, and also the only way it
// is allowed to be written.
//
// The one predicate it does call is `adminSkipReason`, and calling it is what
// keeps that promise rather than breaking it — see the condition 3 block below.
// A filter is a rule; the shared predicate is the same rule. What must never
// appear here is a rule of its own.
type CampaignSweepUseCase struct {
	cfg       *sharedconfig.Config
	accounts  identitycontract.CampaignAccountLister
	campaigns repositories.ICampaignRepository
	claimRepo repositories.ICampaignClaimRepository
	// sweepRuns is where the run's HEALTH is recorded (025 T079c, FR-059).
	//
	// It replaced an in-memory flag on this type, and the reason is not
	// persistence for its own sake. This service runs as several replicas, each
	// with its own worker set, so an in-memory flag answers two questions
	// wrongly, both in the dangerous direction:
	//
	//  1. An HTTP request landing on replica B reads replica B's flag, while the
	//     sweep that actually failed was run by replica A. The operator page then
	//     reports HEALTHY while the mechanism is broken — the one answer FR-059
	//     exists to prevent.
	//  2. Under `APP_MODE=http` no sweep worker runs at all, so the flag would be
	//     permanently false: a false all-clear that is structurally guaranteed.
	//
	// A row per campaign in the database is read identically by every replica and
	// is reachable by the read-only look-up with no worker running at all.
	sweepRuns  repositories.ICampaignSweepRunRepository
	campaignUC uc_interfaces.ICampaignUseCase
	log        logger.Interface
	// now is the clock, for the same reason `CampaignUseCase.now` is a field: one
	// read per run, pin-able by a test, never `time.Now()` (R6, constitution §VIII).
	now func() time.Time
}

func NewCampaignSweepUseCase(
	cfg *sharedconfig.Config,
	accounts identitycontract.CampaignAccountLister,
	// campaigns is the registry (026). It replaced `cfg.Campaigns`, and it is the
	// same collaborator the grant path uses — so the sweep and the queue read the
	// SAME list through the same query, and a campaign created a second ago is
	// picked up by the next run without a deploy (FR-071).
	campaigns repositories.ICampaignRepository,
	claimRepo repositories.ICampaignClaimRepository,
	sweepRuns repositories.ICampaignSweepRunRepository,
	campaignUC uc_interfaces.ICampaignUseCase,
	log logger.Interface,
) uc_interfaces.ICampaignSweepUseCase {
	return &CampaignSweepUseCase{
		cfg:        cfg,
		accounts:   accounts,
		campaigns:  campaigns,
		claimRepo:  claimRepo,
		sweepRuns:  sweepRuns,
		campaignUC: campaignUC,
		log:        log,
		now:        func() time.Time { return timeutils.GetNow(cfg.Database.TimeZone) },
	}
}

// Sweep runs one catch-up pass.
//
// The window is [the earliest campaign start, now). One pass, not one per
// campaign: the accounts that belong to campaign B are also inside campaign A's
// span whenever A starts earlier, and the shared path re-runs condition 7 for
// every account anyway — which is what keeps the two paths from disagreeing about
// WHICH campaign an account belongs to (FR-011/FR-080).
//
// `now` is read ONCE. Every comparison in the run — the window, the lateness —
// must see the same instant, or an account on the boundary is judged against two
// different "now"s in one run (R6).
func (uc *CampaignSweepUseCase) Sweep(ctx context.Context) (uc_interfaces.CampaignSweepReport, error) {
	now := uc.now()
	report := uc_interfaces.CampaignSweepReport{SweptAt: now}

	// THE REGISTRY IS RE-READ ON EVERY RUN (026, FR-070/FR-071). This is the whole
	// reason the list may live in the database: a campaign created five minutes
	// ago, from a browser, is swept by the run that starts in one sweep period —
	// with no deploy, no restart and no in-memory list that would need invalidating.
	//
	// `ListOpen` returns only what is not force-closed (FR-031), so a closed
	// campaign is not swept while its health row and its claim ledger stay untouched
	// (spec.md Câu 9). The window is decided by `anyCampaignAwake` below, through
	// the SAME predicate the message path's condition 2 uses (FR-080).
	campaigns, err := uc.campaigns.ListOpen(ctx)
	if err != nil {
		// No health row is written, and that is deliberate rather than an oversight:
		// the set of campaigns this run would have covered is exactly what the failed
		// query was supposed to return, so any row written here would be a guess. The
		// error travels to the worker, which logs it, and the NEXT run records the
		// campaigns that exist by then. Inventing rows for campaigns we could not read
		// would be the false all-clear FR-059 exists to prevent.
		return report, subscriptionerrors.ErrCampaignRegistryReadFailed()
	}

	// Conditions 1 and 2, BORROWED rather than restated. `anyCampaignAwake` is the
	// same function the message path's condition 2 calls, so the sweep cannot
	// conclude that a campaign is open while the message path concludes it is
	// closed. Deciding for itself here is precisely how two paths drift (FR-080).
	if len(campaigns) == 0 || !newCampaignEvaluator(campaigns, now).anyCampaignAwake() {
		uc.recordSuccess(ctx, report, nil)
		uc.log.Info("campaign sweep skipped; no campaign is inside its window",
			zap.Int("campaigns", len(campaigns)))
		return report, nil
	}

	claimed, err := uc.alreadyClaimed(ctx, campaigns)
	if err != nil {
		uc.recordFailure(ctx, report, campaigns, err)
		return report, err
	}

	filter := identitycontract.CampaignAccountFilter{
		From:  earliestCampaignStart(campaigns),
		To:    now,
		Limit: sweepBatchSize,
	}

	consecutiveFailures := 0
	for {
		if err := ctx.Err(); err != nil {
			uc.recordFailure(ctx, report, campaigns, err)
			return report, err
		}

		page, err := uc.accounts.ListCreatedInWindow(ctx, filter)
		if err != nil {
			failure := subscriptionerrors.ErrCampaignAccountsQueryFailed(windowLabel(campaigns))
			uc.recordFailure(ctx, report, campaigns, failure)
			return report, failure
		}
		if page == nil || len(page.Items) == 0 {
			break
		}

		consecutiveFailures, err = uc.processBatch(ctx, page.Items, claimed, now, &report, consecutiveFailures)
		if err != nil {
			uc.recordFailure(ctx, report, campaigns, err)
			return report, err
		}

		// Advance by the COMPOSITE key (created_at, id) — the last row of THIS
		// batch. Paging on the timestamp alone is the bug the contract names: many
		// accounts share one `created_at` (millisecond resolution), so the
		// remainder of that instant is skipped or the instant repeats forever.
		last := page.Items[len(page.Items)-1]
		if !last.CreatedAt.After(filter.CursorTime) && last.ID == filter.CursorID {
			// The cursor did not move, so the query ignored it. Stop rather than
			// loop on the same batch; a missed batch is recoverable on the next
			// run, an infinite loop is not.
			uc.log.Warn("campaign sweep cursor did not advance; stopping this run",
				zap.Int("batch_size", len(page.Items)))
			break
		}
		filter.CursorTime = last.CreatedAt
		filter.CursorID = last.ID

		// A short batch means the window had nothing after it.
		if len(page.Items) < sweepBatchSize {
			break
		}
	}

	uc.recordSuccess(ctx, report, campaigns)
	return report, nil
}

// processBatch forwards one batch to the shared grant path and updates the report.
//
// Accounts are handled SEQUENTIALLY, matching the message consumer's
// concurrency of 1 (FR-089). It buys nothing to run them in parallel here: the
// quota decision takes a transaction-scoped advisory lock PER ACCOUNT, so
// parallel evaluation would not make grants faster, it would only make them
// queue on the same lock.
func (uc *CampaignSweepUseCase) processBatch(
	ctx context.Context,
	items []*identitydto.UserBriefContractDTO,
	claimed map[uuid.UUID]struct{},
	now time.Time,
	report *uc_interfaces.CampaignSweepReport,
	consecutiveFailures int,
) (int, error) {
	for _, account := range items {
		if account == nil || account.ID == uuid.Nil || account.CreatedAt.IsZero() {
			// The window query cannot produce this: `id` is the primary key and
			// `created_at` is NOT NULL. Counting it as a skip rather than panicking
			// keeps an unattended worker alive against a row shape nobody expected.
			report.Scanned++
			report.Skipped++
			continue
		}
		report.Scanned++

		// Condition 3 through the SHARED predicate. The sweep reads the flag from
		// `identity.users` where the message reads it from the payload — the same
		// fact from two sources — so the rule itself must be the same function
		// (FR-050, FR-080, T096). An administrator's account is skipped here for
		// the same reason and with the same code it would be skipped at the message
		// path. Note that the message path would refuse it anyway: `payloadFor`
		// copies `CreatedByAdmin` into the payload, so this check is not the only
		// guard, it is a cheaper one — an administrator's account does not cost a
		// transaction, an advisory lock and two counts to be turned away, and the
		// report attributes the skip where the operator can see it.
		if reason := adminSkipReason(account.CreatedByAdmin); reason != "" {
			report.Skipped++
			uc.logSkippedAccount(account.ID, reason)
			continue
		}

		// Condition 4. An account that already holds a claim can never be granted
		// again (FR-012/FR-025), so the shared path's answer for it is
		// `already_claimed` every time. Recognising it here costs one map lookup
		// instead of a transaction, an advisory lock and two bucket counts — which
		// matters because on a full campaign EVERY account in the window is in this
		// set, every fifteen minutes.
		if _, already := claimed[account.ID]; already {
			report.Skipped++
			uc.logSkippedAccount(account.ID, SkipAlreadyClaimed)
			continue
		}

		uc.countLateness(account, now, report)
		report.Forwarded++

		err := uc.campaignUC.ProcessUserCreated(uc.sweepContext(ctx), uc.payloadFor(account))
		if err != nil {
			report.Failed++
			consecutiveFailures++
			uc.log.Error("campaign sweep could not evaluate account; the next run retries it",
				zap.String("user_id", account.ID.String()),
				zap.Error(err),
			)
			if consecutiveFailures >= maxSweepConsecutiveFailures {
				return consecutiveFailures, subscriptionerrors.ErrCampaignSweepAborted(consecutiveFailures)
			}
			continue
		}
		consecutiveFailures = 0
	}
	return consecutiveFailures, nil
}

// payloadFor rebuilds the message the sweep stands in for.
//
// It carries the THREE fields the rules read — the account, the registration
// instant, and who created it — and nothing else. FR-087 keeps personally
// identifiable data out of the message and FR-089 calls the event a trigger
// rather than a contract, so the sweep supplies exactly what the rules consume
// and invents nothing: `Provider` is left empty because no condition reads it,
// and a made-up value would be a fact the message path never had.
//
// `CreatedAt` becomes `RegisteredAt`, and that mapping is the reason FR-011's
// priority rule is pure: conditions 6.1 and 6.2 compare the account's CREATION
// instant, so an account that arrives through the sweep four hours late is
// ranked exactly where the message would have ranked it.
func (uc *CampaignSweepUseCase) payloadFor(account *identitydto.UserBriefContractDTO) identitydto.UserCreatedEventPayload {
	return identitydto.UserCreatedEventPayload{
		UserID:         account.ID,
		RegisteredAt:   account.CreatedAt,
		CreatedByAdmin: account.CreatedByAdmin,
	}
}

// sweepContext tags the call as coming from the catch-up run.
//
// The tag changes no rule — it only reaches the audit payload and the grant log,
// where it is the operator's proof that this account was recovered by the sweep
// rather than settled by the queue (FR-082). See `grant_uc.go`.
func (uc *CampaignSweepUseCase) sweepContext(ctx context.Context) context.Context {
	return withGrantOrigin(ctx, grantOriginSweep)
}

// countLateness records the FR-082 signal for one account.
//
// "Late" means the account is older than the five-minute commitment and is STILL
// unsettled, because the sweep had to consider it at all. It is deliberately
// counted before the shared path answers, and it is therefore an UPPER bound on
// late grants: an account the sweep forwards may still turn out to be ineligible
// or out of budget. The bound is the safe direction — an over-reported slowdown
// costs an operator one glance, while an under-reported one hides a queue that is
// not draining.
func (uc *CampaignSweepUseCase) countLateness(account *identitydto.UserBriefContractDTO, now time.Time, report *uc_interfaces.CampaignSweepReport) {
	delay := now.Sub(account.CreatedAt)
	if delay <= campaignLateThreshold {
		return
	}
	report.LateAccounts++
	if seconds := int(delay / time.Second); seconds > report.MaxDelaySeconds {
		report.MaxDelaySeconds = seconds
	}
	uc.log.Warn("campaign grant is late: the message path had not settled this account",
		zap.String("user_id", account.ID.String()),
		zap.Int("delay_seconds", int(delay/time.Second)),
		zap.Time("registered_at", account.CreatedAt),
		zap.String("grant_trigger", string(grantOriginSweep)),
	)
}

// alreadyClaimed loads the accounts that already hold a claim, so the run can skip
// them without opening a transaction each.
//
// ONE query per campaign, capped by that campaign's own ceiling. A campaign can
// never hold more than `quota + reserve` claims — that is what the ceiling means —
// so asking for exactly that many rows returns the COMPLETE set, not a recent
// sample. If a campaign is ever configured above the ledger's 100-row read cap,
// the set is partial: the accounts missing from it are simply evaluated by the
// shared path, which answers `already_claimed` for them. That is slower, never
// wrong, which is the only acceptable direction for an optimisation.
//
// The budget comes from the REGISTRY ROW (026), never from configuration: after the
// move there is nowhere else it could come from, and reading it from the same row
// the grant path reads is what keeps the two paths from disagreeing about how many
// claims a campaign can hold.
//
// A failure here is NOT downgraded to an empty set. An empty set would mean "no
// account has ever been granted", and the sweep would then push every one of them
// through the shared path — correct, but it converts one failed query into a
// full-window storm of transactions.
func (uc *CampaignSweepUseCase) alreadyClaimed(
	ctx context.Context,
	campaigns []*entities.Campaign,
) (map[uuid.UUID]struct{}, error) {
	claimed := make(map[uuid.UUID]struct{})
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		// The ceiling, not the batch size: asking for more than a campaign can hold
		// would be asking for rows that cannot exist.
		limit := campaign.Quota + campaign.Reserve
		if limit <= 0 {
			// `quota` is CHECKed to be >= 1, so this is unreachable. Guarded rather
			// than assumed: a zero limit would be read as "no limit" by the ledger
			// and would ask for its own default instead of skipping.
			continue
		}
		claims, err := uc.claimRepo.ListByCampaign(ctx, campaign.Code, limit)
		if err != nil {
			return nil, subscriptionerrors.ErrCampaignListClaimsFailed(campaign.Code)
		}
		for _, claim := range claims {
			if claim != nil {
				claimed[claim.UserID] = struct{}{}
			}
		}
	}
	return claimed, nil
}

// earliestCampaignStart is the lower bound of the sweep window.
//
// The EARLIEST start, not the latest: the window has to contain every account
// that could belong to any open campaign, and a later start would silently exclude
// accounts from a campaign that opened earlier. Accounts before the earliest start
// cannot qualify for anything, and the query excludes them for free.
func earliestCampaignStart(campaigns []*entities.Campaign) time.Time {
	earliest := campaigns[0].StartsAt
	for _, campaign := range campaigns {
		if campaign.StartsAt.Before(earliest) {
			earliest = campaign.StartsAt
		}
	}
	return earliest
}

// windowLabel names the campaigns in a log line or error cause. Registry codes are
// CHECKed against `^[a-z0-9][a-z0-9-]{2,63}$`, so joining them cannot put a path
// separator or a control character into a log message.
func windowLabel(campaigns []*entities.Campaign) string {
	if len(campaigns) == 1 {
		return campaigns[0].Code
	}
	return "multiple"
}

// logSkippedAccount records a skip the sweep decided on its own (FR-061).
//
// Debug rather than Info: on a full campaign every account in the window is
// skipped this way every fifteen minutes, and an Info line per account would bury
// the grants and the failures under thousands of routine lines. The COUNT is not
// lost — the worker logs it once per run — so the reason is still on the record.
func (uc *CampaignSweepUseCase) logSkippedAccount(userID uuid.UUID, reason SkipReason) {
	uc.log.Debug("campaign sweep account skipped",
		zap.String("user_id", userID.String()),
		zap.String("reason", string(reason)),
		zap.String("grant_trigger", string(grantOriginSweep)),
	)
}
