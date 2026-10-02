package entities

import (
	"time"
)

// CampaignSweepRun is one campaign's health of the FR-079 catch-up sweep
// (025 T079a, FR-059).
//
// WHY A ROW AND NOT A FLAG ON THE WORKER. FR-059 asks an operator to tell three
// states apart: a campaign is running, a campaign has gone to sleep, and a
// campaign is RUNNING BUT BROKEN. The earlier design kept that flag in the
// sweep worker's memory, which is only correct while there is exactly ONE worker.
// This service runs as several replicas, each with its own worker set, and two
// failures follow — both in the dangerous direction:
//
//  1. An HTTP request landing on replica B reads replica B's flag, while the sweep
//     that actually failed was run by replica A. The page then reports HEALTHY
//     while the mechanism is broken, which is the one answer FR-059 exists to
//     prevent.
//  2. Under `APP_MODE=http` no sweep worker runs at all, so the flag would be
//     permanently false — a false all-clear that is structurally guaranteed.
//
// A row keyed on the campaign code is read identically by every replica and is
// reachable by the read-only operator look-up (FR-065) with no worker at all.
//
// The key is `campaign_code` ALONE rather than a surrogate id: a campaign is the
// only thing this row is about, `campaign_claims` is keyed the same way, and a
// second identity column would be one more value that could disagree with it.
type CampaignSweepRun struct {
	// CampaignCode is the primary key. It matches `CampaignClaim.CampaignCode`
	// rather than a foreign key: campaigns are declared in a YAML file
	// (data-model.md §4), not in a table, so there is no row to point at.
	CampaignCode string `gorm:"type:varchar(64);primaryKey"`
	// LastSweepAt is the instant of the most recent sweep that COMPLETED.
	//
	// Nullable on purpose, and deliberately NOT advanced by a failed run: the
	// column answers "when did a sweep last succeed", and a run that failed is
	// not evidence that one did. A sweep failing every fifteen minutes for six
	// hours therefore reports a six-hour-old timestamp, which is the useful
	// alarm. A nil value means the campaign has never been swept successfully
	// since it was declared — itself worth knowing.
	LastSweepAt *time.Time `gorm:"type:timestamp with time zone"`
	// Degraded is set when the most recent sweep failed and cleared by the next
	// one that completes. It describes the most recent run, so "it stopped
	// breaking" is as much of an event as "it broke".
	Degraded bool `gorm:"type:boolean;not null;default:false"`
	// DegradedReason is the most recent failure, so an operator does not have to
	// correlate a timestamp with a log search. Cleared on success, because a
	// stale reason beside `degraded = false` would name a fault that no longer
	// exists.
	//
	// TEXT rather than a code from an enum table: it is the rendered `Error()`
	// of whatever went wrong, and constraining it to a fixed vocabulary would
	// make the table the place that has to be edited every time a new failure
	// mode is described.
	DegradedReason *string `gorm:"type:text"`
	// UpdatedAt is GORM's own bookkeeping column and is NOT part of the contract:
	// nothing reads it, because `last_sweep_at` already answers "when".
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

// TableName pins the GORM table name. Declared here rather than in the module's
// shared `table_name.go`, matching `campaign_claim.go`: the entity is readable on
// its own and H2 exempts `TableName()` boilerplate from the struct-per-file rule.
func (CampaignSweepRun) TableName() string { return "campaign_sweep_runs" }
