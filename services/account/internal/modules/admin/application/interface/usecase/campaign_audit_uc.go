package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/admin"
)

// ICampaignAuditReader reads ONE campaign's audit trail, and nothing else (026 T064,
// FR-063).
//
// WHY A NEW INTERFACE INSTEAD OF A METHOD ON `IAuditLogUseCase`. The general interface
// exposes `Record` and `List`, and `List` filters by `action`, `target_type`, `target_id`
// and a date range. None of those can select a campaign: `target_id` is a UUID and a
// campaign is keyed by a text code, so the code lives in `payload.campaignCode`
// (FR-063) and no combination of the existing columns reaches it.
//
// Adding the method there would also be the wrong SHAPE for it. `IAuditLogUseCase` is the
// shape the audit CONSUMER implements and the administration screen drives; a per-campaign
// read is a narrow view another module needs, and the precedent for that is
// `AdminGrantLookup` (024) and `CampaignAccountLister` (025), not a wider shared
// interface.
type ICampaignAuditReader interface {
	// ListByCampaignCode returns one page of the trail of EXACTLY ONE campaign, newest
	// first, plus the total number of rows behind the page.
	//
	// IT DOES NOT CHECK THAT THE CAMPAIGN EXISTS, and cannot: this module does not own
	// `subscription.campaigns` and must not read it (constitution §II). A code nobody
	// typed therefore answers with an empty page and a zero total, and the caller — the
	// `subscription` module, which does own the registry — is what turns that into a 404
	// before ever asking (contract §6.4).
	//
	// An EMPTY PAGE IS NOT "NOBODY CHANGED IT", and that is why the caller checks first:
	// the general audit screen has the same trap, and an operator reading it would
	// conclude the audit mechanism is broken rather than that they typed a code wrong.
	ListByCampaignCode(
		ctx context.Context,
		campaignCode string,
		page, limit int,
	) ([]*dto.AdminAuditLogDTO, int64, error)
}
