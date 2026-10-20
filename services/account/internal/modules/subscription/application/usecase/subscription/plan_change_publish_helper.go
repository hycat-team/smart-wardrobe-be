package subscription

import (
	"context"
	"fmt"

	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	sharedevent "smart-wardrobe-be/shared/application/event"

	"github.com/google/uuid"
)

// PlanChangePublisher is the single point that turns a subscription history row
// into a published subscription.plan.changed event.
//
// It exists because before feature 024 there were nine places that could change
// a plan and exactly ONE of them published an event. The other eight — upgrade,
// downgrade, PayOS settlement, manual grant, manual revoke, default
// provisioning, auto-renew toggle — wrote no history row at all, so nothing
// downstream could learn about them. The AI service therefore had no policy
// grant for a large share of paying customers, and nothing anywhere reported an
// error.
//
// FR-011 makes the history row the only source of the event, and FR-012 makes
// publishing mandatory for every change type, including the ones a receiver may
// choose to ignore. Routing every call site through one function turns both from
// reminders into something the compiler enforces.
type PlanChangePublisher struct {
	eventRepo      repositories.IUserSubscriptionEventRepository
	eventPublisher sharedevent.IEventPublisher
	// 031 D1/R2: the validity-interval history is written here, at the single
	// point every plan mutation already passes through. It runs inside the
	// caller's transaction, so the interval and the business change commit or
	// roll back together.
	periodRecorder PlanPeriodRecorder
}

func NewPlanChangePublisher(
	eventRepo repositories.IUserSubscriptionEventRepository,
	eventPublisher sharedevent.IEventPublisher,
	periodRecorder PlanPeriodRecorder,
) *PlanChangePublisher {
	return &PlanChangePublisher{
		eventRepo:      eventRepo,
		eventPublisher: eventPublisher,
		periodRecorder: periodRecorder,
	}
}

// PublishPlanChange writes the history row and the outbox record in the caller's
// transaction, then returns the event.
//
// Three things are load-bearing:
//
//   - Both writes happen on the caller's transaction (passed as ctx), so if the
//     outbox write fails the history row is rolled back with it. A history row
//     without an event is a change nobody will ever hear about, and an event
//     without a history row has no event id to de-duplicate on.
//   - GetByKey is checked first so a retry is a cheap no-op. It is NOT the thing
//     that makes the write idempotent, because two callers can both read "absent"
//     and then both insert (024 T148). The insert is: `event_key` is UNIQUE and it
//     refuses the loser instead of failing it, so a duplicate cannot become two
//     events no matter how the callers interleave.
//   - The payload is built AFTER the history insert, because GORM assigns the
//     primary key during INSERT and the receiver needs it as sourceEventId.
//
// Returns the stored event without publishing when the key already exists - via
// either the early check or the lost race. That is a deliberate skip, not an error,
// and the caller should not fail the business operation over it.
func (p *PlanChangePublisher) PublishPlanChange(
	ctx context.Context,
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) (*entities.UserSubscriptionEvent, error) {
	if event == nil {
		return nil, nil
	}

	if event.EventKey != "" {
		existing, err := p.eventRepo.GetByKey(ctx, event.EventKey)
		if err != nil {
			return nil, fmt.Errorf("check existing plan-change event: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	inserted, err := p.eventRepo.CreateIfAbsent(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("create plan-change event: %w", err)
	}
	if !inserted {
		// 024 T148: someone else committed this key between the check above and the
		// insert. `event_key` is UNIQUE, so the loser of that race is now here
		// instead of failing with a unique-violation - which surfaced as HTTP 500
		// for an account whose subscription had in fact been created correctly.
		//
		// The winner already published the event, so re-publishing it here would
		// produce a duplicate the receiver would have to de-duplicate. Returning the
		// stored row is exactly what the early-return branch above does, so both
		// paths now behave identically whether the race was won or lost.
		existing, getErr := p.eventRepo.GetByKey(ctx, event.EventKey)
		if getErr != nil {
			return nil, fmt.Errorf("read back plan-change event after conflict: %w", getErr)
		}
		if existing != nil {
			return existing, nil
		}
		// Unreachable in practice: the conflict proves the row exists. Returning the
		// in-memory event keeps the caller working rather than inventing an error
		// for a state the database cannot be in.
		return event, nil
	}

	payload := mapper.BuildPlanChangedPayload(event, sub, plan)
	// 024 T155: refuse to publish a message the receiver is guaranteed to reject.
	//
	// `BuildPlanChangedPayload` cannot fail, so with no plan row it used to produce a
	// payload whose `toPlanId` was empty. The AI service refuses those by design
	// (FR-022/FR-023, T137) - it will not write a grant with a blank plan id - which
	// meant the sender was manufacturing messages that could only ever be discarded,
	// silently. FR-012 requires every change to publish an event; an event that is
	// rejected on arrival does not satisfy it, and nothing anywhere reported the gap.
	//
	// Returning an error puts the refusal inside the caller's transaction, so a
	// subscription is not committed while its event is known-unsendable. The twelve
	// call sites do not each have to remember to check.
	if err := mapper.ValidatePlanChangedPayload(payload); err != nil {
		return nil, fmt.Errorf("build plan-changed payload: %w", err)
	}

	// 031 D1/R2: record the validity interval AFTER the history row exists and
	// the payload is known to be publishable. A retry never reaches this point
	// (both early returns above return the stored event), which is the
	// idempotency guarantee (FR-006 / research.md R9). A recorder failure rolls
	// back the event insert with the business mutation, so no orphan interval
	// can be committed.
	if err := p.periodRecorder.Record(ctx, event, sub, plan); err != nil {
		return nil, fmt.Errorf("record user plan period: %w", err)
	}

	if err := p.eventPublisher.Publish(ctx, eventconstants.TopicSubscriptionPlanChanged, payload); err != nil {
		return nil, fmt.Errorf("publish plan-changed event: %w", err)
	}
	return event, nil
}

// actorAdminIDOf is a small helper for the administrator paths, which are the
// only ones that know who initiated the change. Kept here so the column is
// filled consistently instead of ad hoc per call site. A nil or zero id is
// normalised to nil: the column is a foreign key, and uuid.Nil would be a
// reference to a user that does not exist.
func actorAdminIDOf(actorID *uuid.UUID) *uuid.UUID {
	if actorID == nil || *actorID == uuid.Nil {
		return nil
	}
	copied := *actorID
	return &copied
}
