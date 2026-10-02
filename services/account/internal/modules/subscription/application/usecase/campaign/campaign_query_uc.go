package campaign

import (
	"context"
	"time"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
)

// CampaignQueryUseCase is the operator look-up (025 T079, FR-065 to FR-072).
//
// WHY IT IS A SEPARATE TYPE FROM `CampaignUseCase`. Two reasons, and the second
// is the one that matters at three in the morning.
//
// The first is layering: this one READS and the other one WRITES. Sharing a type
// would mean the read path held a unit of work and a notification contract, and
// FR-069 — read-only, changes nothing — would become a rule kept by discipline
// rather than by the type system. Separated, the read use case cannot open a
// transaction because it does not have one.
//
// The second is that the two have genuinely different inputs. The grant path is
// handed a message and decides about an account it has never seen. This one is
// handed a campaign code and an account id and must reconstruct what the OTHER
// path concluded — which means it has to read the ledger, the sweep table and the
// account's own record, none of which the grant path reads at the same time.
//
// EVERY NUMBER IS DERIVED AT THE MOMENT OF THE CALL. Nothing here is cached,
// memoised or stored (data-model.md §4.3, FR-065/FR-066). A derived count that
// were stored would need synchronising along a second path, and the value an
// operator reads would be as stale as the slowest of them — which for a number
// that decides whether a customer gets told "we ran out" is not an acceptable
// failure mode.
//
// IT HOLDS NO RULES ABOUT ELIGIBILITY. `GetAccountEligibility` calls the SAME
// `campaignEvaluator` the grant path calls (FR-080). A look-up that decided
// eligibility itself would drift from the path that actually grants, and the drift
// would be invisible: the look-up would confidently report "eligible" for an
// account the grant path refused, and support would tell the customer so.
type CampaignQueryUseCase struct {
	cfg          *sharedconfig.Config
	claimRepo    repositories.ICampaignClaimRepository
	sweepRunRepo repositories.ICampaignSweepRunRepository
	planRepo     repositories.ISubscriptionPlanRepository
	subRepo      repositories.IUserSubscriptionRepository
	accounts     identitycontract.IUserBriefContract
	log          logger.Interface
	// now is the clock, for the same reason it is a field on every other campaign
	// type: one read per call, pin-able by a test, never `time.Now()` (R6,
	// constitution §VIII).
	now func() time.Time
}

// NewCampaignQueryUseCase builds the operator look-up.
//
// It takes the CLAIM repository rather than `ICampaignUseCase` because it reads
// the ledger for itself: routing the reads back through the grant use case would
// mean the only component holding the budget counts also holds the writes, and
// the read path would inherit its dependencies it must not have (FR-069).
//
// It does NOT take a unit of work. That omission is the read-only guarantee:
// there is no transaction to open, and therefore no way for this use case to
// write anything even by accident.
func NewCampaignQueryUseCase(
	cfg *sharedconfig.Config,
	claimRepo repositories.ICampaignClaimRepository,
	sweepRunRepo repositories.ICampaignSweepRunRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	subRepo repositories.IUserSubscriptionRepository,
	accounts identitycontract.IUserBriefContract,
	log logger.Interface,
) uc_interfaces.ICampaignQueryUseCase {
	return &CampaignQueryUseCase{
		cfg:          cfg,
		claimRepo:    claimRepo,
		sweepRunRepo: sweepRunRepo,
		planRepo:     planRepo,
		subRepo:      subRepo,
		accounts:     accounts,
		log:          log,
		now:          func() time.Time { return timeutils.GetNow(cfg.Database.TimeZone) },
	}
}

// ListCampaigns returns every configured campaign with its derived state (FR-065).
//
// ONE clock read for the whole list: a campaign rendered as `running` and the next
// one rendered as `not_started` from two different readings of the clock is a
// response that is internally inconsistent, and an operator comparing two rows of
// the same page would have no way to tell which to believe.
func (uc *CampaignQueryUseCase) ListCampaigns(ctx context.Context, page, limit int) ([]*sub_dto.CampaignSummaryRes, int64, error) {
	page, limit = uc_interfaces.NormalizeCampaignPagination(page, limit)
	now := uc.now()

	// The sweep health of every configured campaign in ONE query (FR-059). Read
	// from `campaign_sweep_runs` and never from the sweep worker's memory: the
	// service runs several replicas, each with its own worker set, so an in-memory
	// flag read by this replica says nothing about a failure another replica
	// produced — and under APP_MODE=http there is no worker at all, so the flag
	// would be permanently false.
	codes := make([]string, 0, len(uc.cfg.Campaigns))
	for i := range uc.cfg.Campaigns {
		codes = append(codes, uc.cfg.Campaigns[i].Code)
	}
	sweepRuns, err := uc.sweepRunRepo.GetByCampaigns(ctx, codes)
	if err != nil {
		return nil, 0, subscriptionerrors.ErrCampaignListSweepRunsFailed()
	}

	// Every campaign is summarised, including the ones past this page.
	//
	// That is more work than the page needs, and it is still the right call: the
	// counts each summary needs are two indexed counts, and skipping the
	// unrendered campaigns would mean a second code path — one for "in the page"
	// and one for "in the total" — which is how the two drift and an operator sees
	// a total that disagrees with the pages.
	all := make([]*sub_dto.CampaignSummaryRes, 0, len(uc.cfg.Campaigns))
	for i := range uc.cfg.Campaigns {
		campaign := uc.cfg.Campaigns[i]
		summary, err := uc.summarise(ctx, campaign, sweepRuns[campaign.Code], now)
		if err != nil {
			return nil, 0, err
		}
		all = append(all, summary)
	}

	total := int64(len(all))
	start := (page - 1) * limit
	if start >= len(all) {
		// An empty page rather than an error: asking for page 9 of a
		// two-campaign list is a normal thing for an operator's pager to do, and
		// an error would look like a fault.
		return []*sub_dto.CampaignSummaryRes{}, total, nil
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

// GetCampaign returns one campaign by code (FR-065, FR-072).
func (uc *CampaignQueryUseCase) GetCampaign(ctx context.Context, campaignCode string) (*sub_dto.CampaignSummaryRes, error) {
	campaign, ok := uc.findCampaign(campaignCode)
	if !ok {
		return nil, subscriptionerrors.ErrCampaignNotFound(campaignCode)
	}
	sweepRuns, err := uc.sweepRunRepo.GetByCampaigns(ctx, []string{campaign.Code})
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignListSweepRunsFailed()
	}
	return uc.summarise(ctx, *campaign, sweepRuns[campaign.Code], uc.now())
}

// summarise derives one campaign's whole visible state from the ledger, the
// configuration and the sweep table.
//
// The four inputs are the whole story of FR-065: the configuration says what the
// budget SHOULD be, the ledger says what it IS, the clock says whether the window
// is open, and the sweep table says whether the mechanism is working. Nothing is
// read twice and nothing is stored.
func (uc *CampaignQueryUseCase) summarise(
	ctx context.Context,
	campaign sharedconfig.CampaignConfig,
	sweepRun *entities.CampaignSweepRun,
	now time.Time,
) (*sub_dto.CampaignSummaryRes, error) {
	// TWO INDEPENDENT COUNTS, one per budget part (research.md R2b).
	//
	// Never derived from `NOT fills_main_quota`: that flag marks exactly ONE row,
	// so its negation is "total minus one" and folds the reserve in as well. The
	// consequence would not be a cosmetic overcount — `grantedMainCount` is what
	// tells `running` from `compensation`, so a wrong count reports the campaign
	// in the wrong state as well as the wrong number.
	grantedMain, err := uc.claimRepo.CountByBucket(ctx, campaign.Code, entities.GrantBucketMain)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignCountGrantsFailed(campaign.Code, string(entities.GrantBucketMain))
	}
	grantedReserve, err := uc.claimRepo.CountByBucket(ctx, campaign.Code, entities.GrantBucketReserve)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignCountGrantsFailed(campaign.Code, string(entities.GrantBucketReserve))
	}

	// The watermark is a THIRD read, and it is not derivable from either count:
	// it names WHICH claim filled the main budget, and that is the instant every
	// later priority comparison uses (FR-030).
	watermarkAt, watermarkSet, err := uc.claimRepo.GetWatermark(ctx, campaign.Code)
	if err != nil {
		return nil, subscriptionerrors.ErrCampaignReadWatermarkForLookupFailed(campaign.Code)
	}

	grantedTotal := grantedMain + grantedReserve
	quota := int64(campaign.Quota)
	reserve := int64(campaign.Reserve)

	summary := &sub_dto.CampaignSummaryRes{
		Code:                     campaign.Code,
		PlanSlug:                 campaign.PlanSlug,
		Quota:                    campaign.Quota,
		Reserve:                  campaign.Reserve,
		GrantedMainCount:         grantedMain,
		GrantedCompensationCount: grantedReserve,
		GrantedTotalCount:        grantedTotal,
		RemainingMain:            maxInt64(0, quota-grantedMain),
		RemainingCompensation:    maxInt64(0, reserve-grantedReserve),
		HardCap:                  quota + reserve,
		StartsAt:                 campaign.StartsAtTime,
		EndsAt:                   campaign.EndsAtTime,
		Status:                   deriveCampaignStatus(campaign, grantedMain, grantedTotal, now),
	}
	if watermarkSet {
		summary.WatermarkAt = &watermarkAt
	}
	if sweepRun != nil {
		summary.Degraded = sweepRun.Degraded
		if sweepRun.DegradedReason != nil {
			summary.DegradedReason = *sweepRun.DegradedReason
		}
		summary.LastSweepAt = sweepRun.LastSweepAt
	}

	// The plan's display name is read, not configured: the campaign holds a
	// REFERENCE to the plan and copies none of its attributes
	// (data-model.md §8.2), so a catalogue rename must show up here immediately.
	// A missing plan is not an error — FR-010 checks `plan_slug` at runtime rather
	// than at startup, and this response is where an operator sees that it is
	// wrong.
	if plan, err := uc.planRepo.GetBySlug(ctx, campaign.PlanSlug); err == nil && plan != nil {
		summary.PlanName = plan.Name
	}
	return summary, nil
}

// deriveCampaignStatus is the campaign state machine of `data-model.md` §5 (T079b).
//
// WHY IT LIVES HERE AND NOT IN `admission_uc.go`. That file is a PURE evaluator —
// its own header says so, and the specification describes it as "pure function of
// its inputs" (data-model.md §6) — which is what makes conditions 1 to 7 testable
// without a database. This function needs `grantedMainCount`, which is a COUNT on
// `campaign_claims`: a repository read. Putting it next to the evaluator would
// either make the evaluator impure, which costs the eligibility tests their
// point, or give it a repository, which is the same cost in a different shape.
// Separated, the evaluator stays pure and the state machine is free to read.
//
// WHY THE STATE IS NEVER STORED. A stored status would have to be kept in step
// with the claim ledger along a second path — and two paths that must agree are
// precisely where the duplicate grants this mechanism is built to prevent come
// from. Recomputing it on every read costs two indexed counts an operator is
// already waiting on, and it cannot be stale.
//
// THE FIVE STATES, AND WHY EACH IS DISTINGUISHABLE. `running` and `compensation`
// are separate because `compensation` is the one in which the reserve is still
// usable — the budget that exists for the accounts the main budget turned away
// (FR-031). `exhausted` and `expired` are separate because they need different
// people to act: one is "we gave everything away", the other is "we closed it",
// and an operator who cannot tell them apart either keeps sending a closed
// campaign or reopens a finished one.
func deriveCampaignStatus(
	campaign sharedconfig.CampaignConfig,
	grantedMain, grantedTotal int64,
	now time.Time,
) string {
	// `expired` is checked FIRST, ahead of the budget states.
	//
	// Not because time outranks money, but because a campaign past `ends_at` has
	// stopped for a reason no grant can change, and reporting it as
	// `compensation` — which reads as "still granting" — would send an operator
	// to watch a budget that can never be spent again.
	if campaign.EndsAtTime != nil && !now.Before(*campaign.EndsAtTime) {
		return sub_dto.CampaignStatusExpired
	}
	if now.Before(campaign.StartsAtTime) {
		return sub_dto.CampaignStatusNotStarted
	}
	// The hard ceiling is checked before the watermark: at `quota + reserve` grants
	// the campaign is finished whether or not the reserve is empty, and reporting
	// `compensation` there would promise giveaways that cannot be handed out.
	if grantedTotal >= int64(campaign.Quota)+int64(campaign.Reserve) {
		return sub_dto.CampaignStatusExhausted
	}
	// The main budget is full — the watermark is frozen (FR-030) — and the reserve
	// still has room. That is `compensation`, and it is the only state in which a
	// reserve grant is still possible.
	if grantedMain >= int64(campaign.Quota) {
		return sub_dto.CampaignStatusCompensation
	}
	return sub_dto.CampaignStatusRunning
}

// findCampaign looks a campaign up by code in the loaded configuration.
//
// Configuration is the ONLY source of which campaigns exist: they are declared in
// a YAML file (data-model.md §4), not in a table, so a code that is not in it is
// not merely ungranted but undeclared — which is the honest answer and the one
// FR-072 asks for.
func (uc *CampaignQueryUseCase) findCampaign(campaignCode string) (*sharedconfig.CampaignConfig, bool) {
	for i := range uc.cfg.Campaigns {
		if uc.cfg.Campaigns[i].Code == campaignCode {
			return &uc.cfg.Campaigns[i], true
		}
	}
	return nil, false
}

// maxInt64 is `max(0, n)` for the derived remainders.
//
// A negative "slots remaining" would be nonsense on a page an operator reads at a
// glance, and the arithmetic that produces it (a quota lowered on a live campaign
// below what it already granted) is exactly the situation where an operator most
// needs the number to be a number rather than a complaint.
func maxInt64(floor, value int64) int64 {
	if value < floor {
		return floor
	}
	return value
}

// unusedUUID keeps the uuid import honest for the account-id signature below.
var _ = uuid.Nil
