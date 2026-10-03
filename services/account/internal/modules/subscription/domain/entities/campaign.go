package entities

import (
	"time"
)

// Campaign is ONE row of the campaign registry `subscription.campaigns`
// (026, FR-001/FR-010, data-model.md §2.5).
//
// WHY THIS IS A TABLE AND NOT A CONFIG ENTRY. Feature 025 already made a campaign
// "data, not code", but that data lived in `configs/config*.yaml` — compiled INTO
// the image. Opening one therefore still required a commit, a tag and a deploy,
// and closing one early had no path at all. The registry is the only place a
// campaign can live and be changed without shipping new binaries, so it moved
// here and the `campaigns` block was deleted from both YAML files (FR-001/FR-002).
//
// The six fields 025 declared are the six that are here: `closed_at` and
// `version` were added by 026 and neither is a business attribute an operator
// invented — see the two comments below.
type Campaign struct {
	// Code identifies the campaign and is the PRIMARY KEY.
	//
	// Immutable after creation (FR-024): `campaign_claims.campaign_code` and
	// `campaign_sweep_runs.campaign_code` are historical TEXT keys with no
	// foreign key, so renaming a campaign would orphan its whole ledger without
	// anything failing. There is no method anywhere that writes `code`.
	//
	// Restricted to `^[a-z0-9][a-z0-9-]{2,63}$` by a CHECK on the column: the code
	// reaches an audit payload and possibly a queue or file name, so an unrestricted
	// alphabet turns the first concatenation into a path traversal.
	Code string `gorm:"type:varchar(64);primaryKey"`
	// PlanSlug is the plan granted to an eligible account. It is a REFERENCE and
	// never a copy of the plan's attributes (data-model.md §8.2).
	//
	// Non-blank is enforced by a CHECK at write time. Whether the plan EXISTS and
	// is ACTIVE is checked at RUNTIME, not at write time and not at startup:
	// loading configuration must not reach into the database (constitution §I/§VIII,
	// FR-003), and a wrong slug must never be able to crash a service. A grant to a
	// missing plan is skipped with `plan_unavailable` (FR-014).
	PlanSlug string `gorm:"type:varchar(100);not null"`
	// Quota is the main budget: the number of grants that fill it and freeze the
	// priority watermark. Must be >= 1 (FR-012) — zero would be a campaign that
	// grants nothing, i.e. meaningless rather than invalid.
	Quota int `gorm:"type:integer;not null"`
	// Reserve is the compensation budget, a CEILING and not a target (FR-032).
	// Must be >= 0 (FR-012); zero means the hard ceiling is exactly Quota.
	Reserve int `gorm:"type:integer;not null;default:0"`
	// StartsAt is the instant the campaign opens. TIMESTAMPTZ, and it is a VALUE
	// rather than the RFC3339 string 025 stored: the column carries the zone, so
	// there is no zone-less instant left to interpret (FR-013, R6).
	StartsAt time.Time `gorm:"type:timestamp with time zone;not null"`
	// EndsAt is the instant the campaign closes, or nil for an open-ended campaign
	// (FR-013). When set it must be strictly AFTER StartsAt, which a CHECK enforces.
	EndsAt *time.Time `gorm:"type:timestamp with time zone"`
	// ClosedAt is the operator's force-close instant (FR-031).
	//
	// NULL means "not force-closed". That is NOT the same as being past EndsAt: an
	// open-ended campaign can be closed early, and a closed campaign is asleep
	// regardless of where the clock is (FR-032).
	//
	// Who closed it and why are NOT stored here — they live in the audit trail
	// (spec.md Câu 4), which is the only place that keeps an append-only history.
	ClosedAt *time.Time `gorm:"type:timestamp with time zone"`
	// Version is the optimistic-concurrency token (FR-055).
	//
	// It is not a business attribute: it exists so a stale PATCH or close is
	// refused with 412 instead of silently overwriting another administrator's
	// change, which would leave the audit trail describing a chain of edits that
	// does not match the final state. Incremented only on a write that changed at
	// least one field. Precedent in this module: `user_subscriptions.version`.
	Version int `gorm:"type:integer;not null;default:1"`
	// CreatedAt and UpdatedAt are GORM's own bookkeeping columns. Nothing reads
	// them for a decision — `starts_at`, `closed_at` and the claim ledger answer
	// every question an operator asks.
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

// TableName pins the GORM table name. Declared here rather than in the module's
// shared `table_name.go`, matching `campaign_claim.go` and
// `campaign_sweep_run.go`: the entity is readable on its own and H2 exempts
// `TableName()` boilerplate from the struct-per-file rule either way.
func (Campaign) TableName() string { return "campaigns" }
