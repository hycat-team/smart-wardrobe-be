package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// CampaignSweepResult is what ONE campaign's row should say after a sweep run
// (025 T079c, FR-059).
//
// It is a value rather than three repository methods (`MarkSucceeded`,
// `MarkFailed`, `Touch`) because the sweep writes a row for EVERY campaign on
// every run, and three methods would make "did this run update all of them?" a
// question with three possible partial answers.
type CampaignSweepResult struct {
	// CampaignCode is the campaign this result is about.
	CampaignCode string
	// SweptAt is the instant the run read once and used for every comparison.
	SweptAt time.Time
	// Succeeded is the whole verdict. There is no "partially failed" case: a run
	// that could not finish reports the failure for every campaign, because a
	// half-recorded run is exactly the state in which the table says something
	// untrue.
	Succeeded bool
	// FailureReason is the failure text. It is read only when Succeeded is false,
	// and MUST be empty when it is true — leaving a stale reason beside
	// `degraded = false` would name a fault that no longer exists.
	FailureReason string
}

// ICampaignSweepRunRepository persists the health of the catch-up sweep
// (025 T079a, FR-059).
//
// WHY IT IS A SEPARATE REPOSITORY rather than a method on the claim ledger: the
// claim ledger is append-only and its row is written inside the grant
// transaction, while this table is a single mutable row per campaign that no
// transaction touches. Merging them would put a mutable health flag into an
// interface whose whole design premise is "nothing updates or deletes a row".
type ICampaignSweepRunRepository interface {
	// RecordRun upserts one row per configured campaign after a sweep.
	//
	// It is an UPSERT rather than an insert because a sweep is periodic and the
	// row must survive: the third run of the day overwrites the second run's row
	// rather than failing because the campaign was already swept.
	//
	// It MUST write a row for EVERY campaign in `results`, including campaigns
	// the run did not touch. A campaign added to the configuration has no row
	// until a sweep runs, and the look-up reads a missing row as "never swept" —
	// which is true, and is exactly what an operator needs to be told.
	RecordRun(ctx context.Context, results []CampaignSweepResult) error

	// GetByCampaigns returns the recorded health of the named campaigns.
	//
	// It returns a MAP rather than a slice, keyed by campaign code, because the
	// caller holds one campaign list and wants to read N codes in one query: a
	// slice would make the caller match codes to rows itself, and a list response
	// is built per campaign, so one query per campaign is the shape that costs a
	// query per list row.
	//
	// A campaign with no row is ABSENT from the map rather than returned as a
	// zero value. The difference is the answer: absent means "no sweep has ever
	// completed for this campaign", while a zero value means "a sweep completed
	// at the zero instant", and an operator must not be shown the second for the
	// first.
	GetByCampaigns(ctx context.Context, campaignCodes []string) (map[string]*entities.CampaignSweepRun, error)
}
