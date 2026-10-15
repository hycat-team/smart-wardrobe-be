package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
)

type IAICostRepository interface {
	// ResolvePolicy returns the grant currently in force for an account plus the
	// live policy configuration, or (nil, nil, nil, nil) when the account has no
	// usable grant. A missing grant is NOT an error: the caller degrades through
	// the fallback resolver (FR-006).
	//
	// The parameter list has no plan id on purpose (FR-044). The AI service must
	// not read the Account service's subscription table, and the plan
	// cross-check it used to perform was a WRITE — it closed a grant whose plan
	// did not match, inside a read path. The plan id is still carried on the
	// grant for diagnostics.
	ResolvePolicy(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, now time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, error)
	// Reserve applies the admission decision and writes the ledger row in one
	// transaction (constitution VI).
	//
	// `maxPaidAttemptsPerDay` is the per-day cap as RESOLVED BY THE CALLER, and the
	// caller must have taken it from the grant SNAPSHOT when a grant exists
	// (FR-029b: the snapshot decides what the customer bought, the live
	// configuration decides what the system risks). It is passed in rather than
	// read from `operation` because the repository cannot see the snapshot, and
	// reading the live column here is what made an administrator's edit take
	// effect instantly on accounts that had already paid for a different number.
	//
	// nil means "no cap is available": either the account has no grant at all (the
	// tier-1 fallback, where the live column is the only source), or the snapshot
	// carries none. The repository does not guess between those - it refuses the
	// paid route, which is the safe direction, and the caller escalates to tier 2.
	Reserve(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant, policy *aiexecutor_entities.AICostPolicy, operation *aiexecutor_entities.AICostPolicyOperation, maxPaidAttemptsPerDay *int, event *aiexecutor_entities.AIUsageEvent, now time.Time) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID, at time.Time) error
	// Confirm records the real usage of a request and its actual cost.
	//
	// errorCode carries the reconciliation code for confirmed events with no
	// billable amount (e.g. "pricing_config_missing", "usage_unreported"); it
	// is empty when the confirmation carries no such condition.
	Confirm(ctx context.Context, requestID uuid.UUID, promptTokens, outputTokens, thinkingTokens, actualCost int64, provider, model, finishReason, errorCode string, at time.Time) error
	Release(ctx context.Context, requestID uuid.UUID, reason string, at time.Time) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string, expiresAt time.Time) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
	// CloseActiveGrants closes every currently effective grant at cutoff.
	CloseActiveGrants(ctx context.Context, userID uuid.UUID, cutoff, at time.Time) error
	// DeleteFutureGrants removes pending rows outright. They are deleted rather
	// than closed because a closed row would still be returned by a query that
	// does not filter on status.
	DeleteFutureGrants(ctx context.Context, userID uuid.UUID) error
	// TruncateGrantsBefore moves the end of every grant that spans cutoff, so an
	// account is never left without a grant between two periods.
	TruncateGrantsBefore(ctx context.Context, userID uuid.UUID, cutoff, at time.Time) error
	// GetCurrentGrant returns the grant currently in force for an account, or
	// nil. It answers "what applies right now" — the resolution question.
	GetCurrentGrant(ctx context.Context, userID uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error)
	// GetNewestGrantForOrdering returns the account's most recently SCHEDULED grant,
	// whatever its status and whatever window it sits in, or nil if the account has
	// none. It answers the different question "what is the newest thing we have
	// already applied" — the ordering question.
	//
	// 024 T164. These two look interchangeable and are not, which is the whole
	// defect this method exists to close.
	//
	// The ordering check used to read GetCurrentGrant, which filters
	// `effective_from <= now`. That makes a `future` grant invisible to it, and a
	// renewal legitimately creates one: the customer paid for the next period
	// before it starts. So an event that arrived afterwards but was older saw no
	// current grant, passed the check, and was written. Its CloseActiveGrants used a
	// cutoff from its own earlier timestamp, which by design does not touch a future
	// grant, so it closed nothing either — and the account was left holding two
	// overlapping active entitlements, which FR-034 forbids and which no resolver
	// can disambiguate.
	//
	// Every status is included deliberately, `closed` among them. A closed grant
	// still carries the newest occurredAt, and dropping it would let a straggler
	// resurrect an entitlement that was revoked — the exact failure
	// rejectStaleEvent exists to prevent.
	GetNewestGrantForOrdering(ctx context.Context, userID uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error)
	// GetGrantBySourceEventID is the duplicate key lookup. The source_event_id
	// index added by the 024 migration exists for exactly this.
	GetGrantBySourceEventID(ctx context.Context, sourceEventID uuid.UUID) (*aiexecutor_entities.AIPolicyGrant, error)
	// LockGrantsForAccount serialises every grant write for one account for the
	// rest of the current transaction (024 T163, T164).
	//
	// It is a PostgreSQL transaction-scoped advisory lock keyed on the account,
	// which is the only mechanism that works on the very first grant: a
	// `SELECT ... FOR UPDATE` over the account's rows locks nothing when the
	// account has no rows yet, and a brand-new account is exactly when the first
	// event arrives.
	//
	// FR-034 forbids a unique constraint at the data layer - one account may hold
	// several `active` rows as long as they do not overlap in time - so "at most one
	// active grant" has to be established by making the read-decide-write sequence
	// atomic per account. That is this call.
	//
	// The lock is released automatically at commit or rollback; the caller must not
	// hold it longer than its own transaction.
	LockGrantsForAccount(ctx context.Context, userID uuid.UUID) error
	CreateGrant(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant) error
	CreateEvent(ctx context.Context, event *aiexecutor_entities.AIUsageEvent) error
	GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*AIUsageAdminStatsDTO, error)
	GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]AIEventDrillDownDTO, int64, error)
	GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]AIUsageByOperationDTO, error)
	GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error)
	// GetAIErrorBreakdown groups released errors by code/provider/model. With a
	// period it filters `created_at >= from AND created_at < to` (029 R1/R9):
	// the same window the caller uses for the request count, so the two halves
	// of the error rate can never diverge. Both bounds zero keeps the all-time
	// behaviour old callers rely on; a zero bound leaves that side open.
	GetAIErrorBreakdown(ctx context.Context, from, to time.Time) ([]AIErrorBreakdownDTO, error)
	GetUsageByUser(ctx context.Context, userID uuid.UUID, fromDate, toDate *time.Time, page, limit int) ([]*aiexecutor_entities.AIUsageEvent, int64, error)
	ListGrantsByUser(ctx context.Context, userID uuid.UUID) ([]*aiexecutor_entities.AIPolicyGrant, error)
	GetPolicyByID(ctx context.Context, policyID uuid.UUID) (*aiexecutor_entities.AICostPolicy, error)
	// GetPolicyByCode resolves a policy by its configuration code, operations
	// preloaded. ai_cost_policies has no tier_rank or is_default column, so the
	// default policy can only be selected by a code supplied through
	// configuration (research.md R8) — it cannot be derived from the data.
	GetPolicyByCode(ctx context.Context, code string) (*aiexecutor_entities.AICostPolicy, error)
	// ListPolicies returns every cost policy with its operations preloaded —
	// the read path behind GET /admin/ai/cost-policies (024 FR-004).
	ListPolicies(ctx context.Context) ([]*aiexecutor_entities.AICostPolicy, error)
	// GetPolicyOperationByID loads a single operation row together with its
	// parent policy, because the caller must know the enforcement mode and the
	// hard cap before it is allowed to write (FR-005e).
	GetPolicyOperationByID(ctx context.Context, operationID uuid.UUID) (*aiexecutor_entities.AICostPolicyOperation, *aiexecutor_entities.AICostPolicy, error)
	// UpdatePolicyOperation persists a changed per-day paid attempt limit.
	// A nil limit writes SQL NULL, which is the sanctioned way to say "this
	// policy does not use a per-day attempt cap".
	UpdatePolicyOperation(ctx context.Context, op *aiexecutor_entities.AICostPolicyOperation) error
	// CountGrants returns how many grant rows exist, and how many of those have
	// effective_to IS NULL. The second number is the dangerous one: those rows
	// are only recreated when the account changes plan, so a frozen
	// entitlement value can never refresh on its own (FR-004b(2)).
	CountGrants(ctx context.Context) (total int64, openEnded int64, err error)
}
