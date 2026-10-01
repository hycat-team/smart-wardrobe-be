package campaign

import (
	"context"
	"time"

	admindto "smart-wardrobe-be/services/account/dto/admin"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	notificationcontract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// grantOrigin names WHICH path handed the account to the shared rules.
//
// It is provenance, never a decision. The same account reaching `applyGrant` from
// the queue and from the catch-up sweep must produce the same plan, the same
// claim and the same audit content for every RULE — and it does, because both
// paths call `ProcessUserCreated` and nothing here branches on the value. What
// the value carries is the one fact an operator cannot infer from the grant
// itself: a grant that arrived four hours late because the queue was backed up
// looks identical to one that arrived in fifty milliseconds once it is written to
// `audit_logs` (FR-082).
//
// FR-092 forbids a second event type, and FR-087 forbids widening the message
// payload, so the audit row is the only place left that can carry the
// distinction — which is also the artefact FR-038/FR-094 already made the answer
// to "who granted this, from where".
type grantOrigin string

const (
	// grantOriginMessage is the normal path: the `identity.user.created` message.
	grantOriginMessage grantOrigin = "campaign"
	// grantOriginSweep is the catch-up sweep (Phase 10), which reaches accounts
	// the message path never settled.
	grantOriginSweep grantOrigin = "campaign_sweep"
)

// grantOriginKey is a private type so nothing outside this package can collide
// with the key or read the value by accident.
type grantOriginKey struct{}

// withGrantOrigin tags the context with the path the account arrived on.
//
// The context rather than a parameter because `ProcessUserCreated` has one
// signature, and that signature is the point of FR-080: a second entry point
// taking a second flag is a second door onto the rules, and rules behind two
// doors drift. The value travels with the work instead of being spelled out
// again at every call site, and `InjectTx` preserves it across the unit of work.
func withGrantOrigin(ctx context.Context, origin grantOrigin) context.Context {
	return context.WithValue(ctx, grantOriginKey{}, origin)
}

// grantOriginOf reads the tag back, defaulting to the message path so a caller
// that forgets to tag is the safe interpretation rather than a blank field.
func grantOriginOf(ctx context.Context) grantOrigin {
	if origin, ok := ctx.Value(grantOriginKey{}).(grantOrigin); ok {
		return origin
	}
	return grantOriginMessage
}

// applyGrant turns a granted quota decision into an actual plan (025 T051).
//
// It runs on the CALLER's transaction — the one `ProcessUserCreated` opened — so
// the claim written by condition 9 and the plan written here commit together or
// not at all (FR-039). The two failure states that matter are both silent
// otherwise: a claim with no plan is granted later by the catch-up sweep, and a
// plan with no claim is granted a second time by the next message.
//
// THE ORDER IS THE REQUIREMENT (R13, FR-036, FR-037):
//
//  1. ApplySubscriptionPlanWithEvent  — writes the plan, returns the history row
//  2. RE-READ the subscription         — because step 1 does not return it
//  3. PublishPlanChange               — builds the payload from the RE-READ row
//  4. record the audit entry           — names the campaign, no administrator
//  5. enqueue the email                — inside the same transaction
//
// Step 2 is not an optimisation. `BuildPlanChangedPayload` takes `effectiveTo`
// from `sub.ExpiresAt` (application/mapper/plan_changed_payload.go), and
// `ApplySubscriptionPlanWithEvent` returns only the EVENT ROW, never the updated
// subscription. Passing the pre-apply row — which is what the naive call does —
// sends an empty `effectiveTo`, and the AI service reads that as a LIFETIME
// grant: one month of Premium handed to an account as unlimited AI access. The
// administrator grant path recorded the same trap in prose at
// `admin_grant_batch_uc.go:442-447`; this is the second door onto it.
func (uc *CampaignUseCase) applyGrant(
	txCtx context.Context,
	payload identitydto.UserCreatedEventPayload,
	campaign sharedconfig.CampaignConfig,
	plan *entities.SubscriptionPlan,
	outcome quotaOutcome,
	now time.Time,
) error {
	// The grant instant is the account's REGISTRATION, not the processing clock,
	// so a message that arrives four days late still grants the term the account
	// was promised (FR-078). The plan row's own duration supplies the length
	// (FR-019); nothing here hard-codes 30.
	event, err := subscription.ApplySubscriptionPlanWithEvent(txCtx, uc.subRepo, payload.UserID, plan, payload.RegisteredAt)
	if err != nil {
		return subscriptionerrors.ErrCampaignApplyPlanFailed(campaign.Code)
	}

	// Re-read. R13: the payload MUST describe the row as it now is. Without this,
	// `effectiveTo` is empty and the account receives unlimited AI access for a
	// one-month plan.
	updated, err := uc.subRepo.GetByUserID(txCtx, payload.UserID)
	if err != nil {
		return subscriptionerrors.ErrCampaignApplyPlanFailed(campaign.Code)
	}
	if updated == nil {
		// `ApplySubscriptionPlanWithEvent` reported success and there is no row.
		// The two cannot both be true, so this is a loud failure rather than a
		// publish built from a nil subscription.
		return subscriptionerrors.ErrCampaignApplyPlanFailed(campaign.Code)
	}

	// FR-035: a promotional grant NEVER leaves auto-renew on. Nobody consented to
	// being charged, and with the flag set the renewal worker would attempt a
	// payment for a customer who never bought anything.
	//
	// `ApplySubscriptionPlanWithEvent` already clears it when the plan differs from
	// the current one or the previous term has expired, and condition 5 already
	// refused the one case where it does not (the account is already on this plan
	// with a live term). Asserting it here rather than trusting that chain of
	// reasoning is one line, and the consequence of being wrong is a charge.
	if updated.IsAutoRenewEnabled {
		updated.IsAutoRenewEnabled = false
		if err := uc.subRepo.Update(txCtx, updated); err != nil {
			return subscriptionerrors.ErrCampaignApplyPlanFailed(campaign.Code)
		}
	}

	// FR-036/FR-037: the plan change is published through the SINGLE publish point
	// (FR-011), so the AI service learns about the grant through the same channel as
	// every other plan change — no new event type (FR-092), no change to
	// services/ai (FR-097).
	if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, event, updated, plan); err != nil {
		// Fatal to the whole transaction (FR-039): a plan that was applied with no
		// event published leaves the AI service holding the old policy, and the
		// customer is charged Premium prices while getting free-tier AI. Nothing
		// anywhere would report it.
		return subscriptionerrors.ErrCampaignPublishPlanChangeFailed(campaign.Code)
	}

	// FR-038/FR-094: the origin of a campaign grant is recorded in the AUDIT
	// TRAIL, not in the plan-change history — which is a cross-service contract
	// and may not grow a field for this (FR-058). The payload names the campaign
	// code, and `ActorUserID` is the zero UUID because NO administrator performed
	// this grant; the field is NOT NULL and a sentinel is what says "nobody".
	if uc.auditLogContract != nil {
		if err := uc.auditLogContract.Record(txCtx, admindto.RecordAuditLogInput{
			ActorUserID: uuid.Nil,
			Action:      string(adminaudit.SubscriptionGrant),
			TargetType:  string(adminaudit.UserSubscription),
			TargetID:    &payload.UserID,
			Payload: map[string]any{
				// `grantSource` names the PATH, not the rule: the rules are the same
				// on both paths (FR-080), so "campaign" vs "campaign_sweep" tells
				// an operator that a grant came from the catch-up run — and
				// therefore that the message path did not settle it (FR-082).
				"grantSource":   string(grantOriginOf(txCtx)),
				"campaignCode":  campaign.Code,
				"planSlug":      plan.Slug,
				"planName":      plan.Name,
				"targetUserId":  payload.UserID.String(),
				"grantBucket":   string(outcome.Bucket),
				"expiresAt":     updated.ExpiresAt.UTC().Format(time.RFC3339),
				"registeredAt":  payload.RegisteredAt.UTC().Format(time.RFC3339),
				"isSelfService": false,
			},
		}); err != nil {
			// The audit row is what makes the grant answerable later, so its failure
			// aborts the transaction rather than producing a grant nobody can
			// attribute.
			return subscriptionerrors.ErrCampaignRecordAuditFailed(campaign.Code)
		}
	}

	// FR-044/FR-049: the email goes out inside the same transaction — it is an
	// outbox row, so it commits with the grant and cannot describe a rolled-back
	// one. A failure to enqueue it is LOGGED AND IGNORED: reporting it as a failed
	// grant would send the catch-up sweep round again and hand the customer a
	// second plan. FR-049 asks for a trace, which the log line is.
	uc.enqueueGrantEmail(txCtx, payload.UserID, plan, updated)

	// FR-040 is handled by the CALLER, after this closure returns: the cached
	// overview must be dropped only once the transaction has COMMITTED. Doing it
	// here would be wrong in the other direction — a rollback would leave the
	// customer with no cached overview at all, which is harmless but pointless,
	// whereas a stale entry that survives a successful grant is not.
	return nil
}

// enqueueGrantEmail writes the notification. Never fails the grant (FR-049).
//
// FR-047: an account with no address is not a failed grant. `SendByUserID` reports
// that, and the log line distinguishes it from a real delivery problem.
func (uc *CampaignUseCase) enqueueGrantEmail(txCtx context.Context, userID uuid.UUID, plan *entities.SubscriptionPlan, sub *entities.UserSubscription) {
	if uc.notifContract == nil {
		return
	}
	expiresText := "Không thời hạn"
	if sub.ExpiresAt != nil {
		expiresText = sub.ExpiresAt.In(planTimeZone(uc)).Format("02/01/2006")
	}
	if err := uc.notifContract.SendByUserID(
		txCtx,
		notificationcontract.ChannelEmail,
		notificationcontract.TemplateCampaignGrant,
		map[string]any{
			"PlanName":  plan.Name,
			"ExpiresAt": expiresText,
		},
		userID,
	); err != nil {
		uc.log.Warn("campaign grant email could not be enqueued; the grant stands",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
	}
}

// invalidateOverviewAfterCommit drops the cached overview (FR-040).
//
// It delegates to the SHARED `InvalidateOverviewCache` rather than calling
// `Invalidate` itself, and that is not a style preference. Five plan-change paths
// once each carried their own `_ = cache.Invalidate(ctx, ...)`, and every one of
// them had the same defect: the caller's context may already be cancelled by the
// time the transaction commits, so `Del` fails immediately, the cache keeps
// describing the OLD plan until its TTL expires, and the customer sees the
// previous plan after a successful grant — with nothing anywhere reporting it
// (024 T151/T154). The shared function applies `context.WithoutCancel` plus a
// bounded timeout, and the boundary test
// `TestNoCallSiteBypassesTheSharedCacheInvalidation` fails if a sixth copy
// appears.
//
// T058's "no `_ =` on the caller's ctx" is satisfied by making this the ONE call
// site: the caller's context IS passed, and the helper is what detaches it.
func (uc *CampaignUseCase) invalidateOverviewAfterCommit(ctx context.Context, userID uuid.UUID) {
	if uc.overviewCache == nil {
		return
	}
	subscription.InvalidateOverviewCache(ctx, uc.log, uc.overviewCache, userID)
}

// planTimeZone is the configured zone, used only to FORMAT the expiry date for
// the customer. Reading the clock is `timeutils.GetNow`'s job; this is a label.
func planTimeZone(uc *CampaignUseCase) *time.Location {
	loc, err := time.LoadLocation(uc.cfg.Database.TimeZone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// planTimeZoneDoc keeps the time-zone note attached to the only place it applies.
// Formatting a date for a customer is the one thing in this package that is not a
// "current time" question, so it reads the configured zone directly instead of
// going through `timeutils.GetNow` — that function answers "what time is it",
// and there is no clock involved here.
