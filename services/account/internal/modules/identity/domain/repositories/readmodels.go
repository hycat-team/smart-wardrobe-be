package repositories

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"

	"github.com/google/uuid"
)

// Repository read models owned by the domain layer (feature 017);
// application/dto aliases them for handler compatibility.
type UserStyleProfileRes struct {
	UserID          uuid.UUID                 `json:"userId"`
	TasteEmbedding  entities.Vector           `json:"tasteEmbedding,omitempty"`
	PreferredColors *entities.PreferredColors `json:"preferredColors,omitempty"`
}

type UserAdminStatsDTO struct {
	SnapshotDate   time.Time `json:"snapshotDate"`
	TotalUsers     int       `json:"totalUsers"`
	NewUsers       int       `json:"newUsers"`
	ActiveUsersDau int       `json:"activeUsersDau"`
}

// CampaignAccountFilter is the creation-window slice the campaign catch-up sweep
// asks for (025 T020).
//
// Named for the campaign rather than "GetCreatedInWindow" so it does not read as
// a general filter of the gRPC contract: the only consumer is the campaign
// mechanism, which is served in-process.
type CampaignAccountFilter struct {
	// From is inclusive, To is exclusive.
	From time.Time
	To   time.Time
	// CursorTime / CursorID paginate by the COMPOSITE key (created_at, id). A
	// zero pair means the first batch.
	//
	// The composite key is not optional: created_at has millisecond resolution, so
	// accounts registered in one test run routinely share an instant, and
	// paginating by the timestamp alone either skips the rest of that instant or
	// repeats it forever. The id breaks the tie deterministically, and because the
	// two fields are ordered together the batch boundaries cannot drift.
	CursorTime time.Time
	CursorID   uuid.UUID
	Limit      int
}

// CampaignAccountPage is one batch of accounts created inside a window
// (025 T020).
//
// Items are DOMAIN ENTITIES here, on purpose, and the DTO translation happens one
// layer up in the use case.
//
// The task description asked for `[]*dto.UserBriefContractDTO` at this level. That
// is not expressible in Go: `dto/identity` ALREADY imports this package — it
// aliases `UserAdminStatsDTO` (dashboard_dto.go:7) and `UserStyleProfileRes`
// (profile.go:63) — so importing `dto/identity` back from here is an import cycle
// and the whole service stops compiling. The rule the requirement was protecting
// (constitution §II: "using DTOs, never domain entities") is still honoured where
// it actually bites: `entities.User` never crosses the MODULE boundary. The
// contract method returns `[]*dto.UserBriefContractDTO`.
type CampaignAccountPage struct {
	Items []*entities.User
	// TotalCount is the number of accounts in the whole window, NOT the size of
	// this batch. The sweep needs it to know how many batches are left. The FR-065
	// query interface paginates over campaign_claims instead and is unrelated.
	TotalCount int64
}
