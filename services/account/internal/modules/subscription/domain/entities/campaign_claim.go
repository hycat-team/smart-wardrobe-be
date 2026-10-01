package entities

import (
	"time"

	"github.com/google/uuid"
)

// GrantBucket names which part of a campaign's budget a claim was drawn from.
//
// 025 T018 / research.md R2b. It is a frozen enum written at grant time on
// purpose: the alternative — inferring the part from the row's position, or from
// `NOT FillsMainQuota` — makes the meaning of historical rows depend on a value
// that lives in the YAML config, so editing `quota` on a live campaign would
// silently reclassify every earlier claim, corrupt the admin numbers, and move the
// FR-030 watermark.
type GrantBucket string

const (
	// GrantBucketMain is drawn from the campaign's main budget (`quota`).
	GrantBucketMain GrantBucket = "main"
	// GrantBucketReserve is drawn from the campaign's reserve budget (`reserve`).
	GrantBucketReserve GrantBucket = "reserve"
)

// CampaignClaim is one grant of a promotional plan under a campaign (025 T018).
//
// The row is three mechanisms at once, on purpose (data-model.md §3.1): the
// record of a grant, the duplicate guard, and the storage of the FR-030 priority
// watermark. They belong together because they have to commit together — split
// across two tables they would be two sources of truth that have to be
// reconciled, and the reconciliation is exactly where a duplicate grant comes
// from.
type CampaignClaim struct {
	// ID is the primary key. The claim is uniquely identified by UserID, which
	// the UNIQUE constraint enforces at the data layer (FR-025) — this surrogate
	// exists because other tables in this module key on uuid.UUID through the
	// generic repository.
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	// UserID is UNIQUE ALONE, not (CampaignCode, UserID): FR-012 gives an account
	// at most one participation across the whole mechanism, so a later campaign
	// has to skip an account an earlier one already granted. A composite key
	// would let that account collect a second promotional plan and extend its
	// term each time.
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	CampaignCode string    `gorm:"type:varchar(64);not null"`
	PlanSlug     string    `gorm:"type:varchar(100);not null"`
	PlanID       uuid.UUID `gorm:"type:uuid;not null"`
	// RegisteredAt snapshots the account's creation instant at grant time so
	// FR-031 can compare it against the frozen watermark without reading
	// identity.users — the campaign module may not read another module's table.
	RegisteredAt time.Time `gorm:"type:timestamp with time zone;not null"`
	// ExpiresAt is NOT NULL: the granted plan is finite, and the plan table
	// already CHECKs that a finite plan has an expiry. A nullable column here
	// would let a claim record a grant that could never have committed.
	ExpiresAt time.Time `gorm:"type:timestamp with time zone;not null"`
	GrantedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
	// GrantBucket says which part of the budget this grant came from, on EVERY
	// row. Use it for both counts: grantedMainCount and grantedCompensationCount
	// are two independent numbers read from it, never one number and the
	// remainder of another (data-model.md §4.3).
	GrantBucket GrantBucket `gorm:"type:varchar(16);not null"`
	// FillsMainQuota is true on EXACTLY ONE row per campaign: the grant that
	// filled the main budget. That is the FR-030 watermark, and it is not the
	// same question as GrantBucket — 49 other rows are also "main".
	//
	// Do NOT use it to count the main part. Its negation equals "total minus
	// one", which folds the reserve in as well, and the reserve branch then
	// trips ux_campaign_claims_watermark instead of falling through
	// (research.md R2b).
	FillsMainQuota bool `gorm:"type:boolean;not null;default:false"`
}

// TableName pins the GORM table name. Declared here rather than in the module's
// shared `table_name.go` so the entity is readable on its own; H2 exempts
// `TableName()` boilerplate from the struct-per-file rule either way.
func (CampaignClaim) TableName() string { return "campaign_claims" }
