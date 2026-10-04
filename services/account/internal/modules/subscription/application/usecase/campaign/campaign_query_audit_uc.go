package campaign

import (
	"context"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
)

// 026 T067 (FR-063, FR-072, contract §6) — the audit trail of ONE campaign, as the
// operator's screen reads it.
//
// IT IS A SEPARATE FILE from `campaign_query_uc.go` for the same reason
// `campaign_query_claims_uc.go` is: that use case already declares the look-up's
// interface assertion and its shared helpers, and a fourth concern appended to it would
// make "what does the look-up read" a question about a file whose name no longer answers
// it. One method, one file — and the file's name says what the method is for.
//
// WHAT IT HOLDS. The narrow cross-module contract and nothing else: no unit of work, no
// writer, no clock. That is what keeps this method provably read-only, which is the whole
// content of FR-069 for the look-up side — the trail is a record of what somebody else
// did, and reading it changes nothing about the campaign.

// ListCampaignAudit returns one page of a campaign's audit trail (026 T067, FR-063).
//
// THE ORDER OF THE TWO ANSWERS IS THE REQUIREMENT, and it is the same argument
// `contract/admin.ICampaignAuditLookup` makes about itself: the registry is asked FIRST,
// and only then is the trail.
//
// The reason is that the two modules hold different halves of the answer. This module owns
// `subscription.campaigns` and is the only one that may say "this code names nothing"; the
// `admin` module owns the trail and is the only one that may read it. Asking the trail
// first would answer an unknown code with an empty page and a total of zero, which is
// indistinguishable from "nobody has ever touched this campaign" — and an operator reading
// that would conclude the audit mechanism is broken rather than that they typed a code
// wrong. FR-072 already settles that an unknown code is a 404 and never an empty list, and
// this endpoint has no licence to be the exception.
//
// PAGINATION IS NORMALISED HERE, with the look-up's own helper, so the trail obeys the
// same ceiling as the other three paginated reads. Clamping rather than refusing is that
// helper's decision, not this method's: "give me more than the maximum" has the right
// answer, and it is the maximum plus a total.
func (uc *CampaignQueryUseCase) ListCampaignAudit(
	ctx context.Context,
	campaignCode string,
	page, limit int,
) ([]*sub_dto.CampaignAuditLogRes, int64, error) {
	if _, err := uc.requireCampaign(ctx, campaignCode); err != nil {
		return nil, 0, err
	}
	page, limit = uc_interfaces.NormalizeCampaignPagination(page, limit)

	items, total, err := uc.auditLookup.ListByCampaignCode(ctx, campaignCode, page, limit)
	if err != nil {
		// SURFACED, NOT SWALLOWED INTO AN EMPTY PAGE: a failed read of the trail would
		// otherwise be indistinguishable from "this campaign was never changed", and an
		// operator would report a broken screen for a campaign nobody had ever opened.
		// The message names the campaign, because that is what the reader was looking at
		// when the page failed to render.
		return nil, 0, subscriptionerrors.ErrCampaignAuditListFailed(campaignCode)
	}
	// The mapping is the mapper's, not this file's: constitution §"Code Style" puts
	// conversions in `application/mapper`, and it is also what keeps `targetId` emitted
	// as `null` rather than omitted — the whole reason this response has a shape of its
	// own instead of reusing `dto/admin.AdminAuditLogDTO`.
	return mapper.MapCampaignAuditLogResList(items), total, nil
}
