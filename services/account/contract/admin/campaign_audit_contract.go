package contract

import (
	"context"

	"smart-wardrobe-be/services/account/dto/admin"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/admin/application/interface/usecase"
)

// 026 T064 (FR-063, research.md R8) — the narrow view the campaign module needs from the
// ADMIN module's audit trail.
//
// WHY IT IS NOT A METHOD ON `IAuditLogContract`, and this is the load-bearing reason the
// whole design rests on. `IAuditLogContract` is served over gRPC: it has a client adapter
// (`contracts/admin/v1/client/audit_log_client.go`) and a server adapter
// (`admin/presentation/grpc/server.go`). Adding a method to it would force a change to
// `admin.proto` — a CROSS-SERVICE CONTRACT change — for a need that exists only inside
// this service. FR-058 forbids exactly that, and 024 (`AdminGrantLookup`) and 025
// (`CampaignAccountLister`) each solved the same problem by declaring a narrow view of
// their own. This is the third of that series.
//
// WHY THE FILTER HAD TO BE ADDED RATHER THAN REUSED. The existing audit filter takes
// `actor_user_id`, `action`, `target_type`, `target_id`, and a date range — and
// `target_id` is a `*uuid.UUID`. A campaign is keyed by a TEXT code, so the filter that
// answers "who changed this subscription" cannot answer "who changed this campaign" at
// all. The campaign rows therefore carry `target_type = 'campaign'`, `target_id = NULL`
// and the code in `payload.campaignCode` (FR-063), and this method is the only reader of
// that.
//
// WHAT IT RETURNS, AND WHY IT IS A DTO. `dto/dto.AdminAuditLogDTO`, never an
// `entities.AdminAuditLog`: constitution §II — cross-module communication carries DTOs,
// never domain entities. The consumer then maps it to its OWN response shape
// (`sub_dto.CampaignAuditLogRes`), because the shared DTO declares
// `json:"targetId,omitempty"` and would therefore OMIT the null `target_id` that every
// campaign row has — see that type's header.
type ICampaignAuditLookup interface {
	// ListByCampaignCode returns one page of the audit trail of EXACTLY ONE campaign,
	// newest first, plus the total number of rows behind it.
	//
	// THE CAMPAIGN'S EXISTENCE IS NOT THIS METHOD'S BUSINESS, and the caller has to
	// establish it first. This module does not own `subscription.campaigns` and must not
	// read it (constitution §II), so a code nobody typed produces an empty page here
	// rather than a not-found error — and an operator who mistyped a code would read
	// "this campaign has never been changed" as a fact about the audit mechanism. The
	// `subscription` handler therefore checks the code against the registry BEFORE
	// calling this, and answers 404 itself (contract §6.4).
	//
	// `page < 1` and `limit <= 0` are CLAMPED rather than refused, exactly as the general
	// audit list does, because "give me more than the maximum" has the right answer and
	// it is the maximum plus a total.
	ListByCampaignCode(
		ctx context.Context,
		campaignCode string,
		page, limit int,
	) ([]*dto.AdminAuditLogDTO, int64, error)
}

// CampaignAuditLookup is the IN-PROCESS adapter for `ICampaignAuditLookup`.
//
// IT IS A DISTINCT TYPE rather than a narrowing of `AuditLogContract`, and that is forced
// rather than chosen: the concrete contract implements `Record` and nothing else, because
// everything else about it is a gRPC wire format. This type exists so the composition root
// can hand the campaign module an interface it may depend on without either of them
// touching the wire (the same shape `NewAdminGrantLookup` has, one service over).
type CampaignAuditLookup struct {
	auditUC uc_interfaces.ICampaignAuditReader
}

// NewCampaignAuditLookup adapts the admin module's reader to the narrow campaign view.
//
// IT TAKES THE INTERFACE, not the concrete `*usecase.AuditLogUseCase`, so the contract
// package's reach into the admin module stops at its own interface package. That is the
// one place in this tree allowed to know the other module's application shape, and
// keeping the dependency on the narrowest declaration is what makes that exception
// auditable.
func NewCampaignAuditLookup(auditUC uc_interfaces.ICampaignAuditReader) ICampaignAuditLookup {
	return &CampaignAuditLookup{auditUC: auditUC}
}

// ListByCampaignCode serves the campaign trail in-process (026 T064).
func (c *CampaignAuditLookup) ListByCampaignCode(
	ctx context.Context,
	campaignCode string,
	page, limit int,
) ([]*dto.AdminAuditLogDTO, int64, error) {
	return c.auditUC.ListByCampaignCode(ctx, campaignCode, page, limit)
}

var _ ICampaignAuditLookup = (*CampaignAuditLookup)(nil)
