package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/admin"
	"smart-wardrobe-be/services/account/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/admin/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/admin/domain/repositories"
)

// CampaignAuditLookupUseCase serves ONE campaign's audit trail (026 T066, FR-063).
//
// WHY A SEPARATE USE CASE AND NOT A METHOD ON `AuditLogUseCase`. That type is the
// administration screen's collaborator and it answers questions about the WHOLE trail;
// this one answers a question about a single campaign's trail, which is asked from another
// module through a narrow contract (`contract/admin.ICampaignAuditLookup`). Adding the
// method there would widen the screen's use case with a campaign-specific shape, and the
// campaign module would then have to depend on that wider interface to reach it — which
// is the thing the narrow contract exists to avoid (research.md R8).
//
// IT HOLDS NO RULES. The filter is the repository's, the mapping is the admin module's
// own `mapper`, and the "does this campaign exist" question is NOT answered here because
// this module does not own `subscription.campaigns` and must not read it (constitution
// §II). The caller checks the code against the registry first and turns an empty trail
// into a 404 of its own (contract §6.4).
type CampaignAuditLookupUseCase struct {
	auditRepo repositories.IAdminAuditLogRepository
}

func NewCampaignAuditLookupUseCase(
	auditRepo repositories.IAdminAuditLogRepository,
) usecase.ICampaignAuditReader {
	return &CampaignAuditLookupUseCase{auditRepo: auditRepo}
}

// ListByCampaignCode returns one page of a campaign's trail, newest first.
//
// THE MAPPER IS THE MODULE'S OWN, and that is deliberate even though the rows never leave
// this service. `MapAdminAuditLogsToDTO` already turns `entities.AdminAuditLog` into
// `dto.AdminAuditLogDTO` for the administration screen, and reusing it means one
// conversion of one row type rather than two that could drift — a screen comparing the
// general audit list with the campaign trail would otherwise see two renderings of one
// row. What the CONSUMER does next with that DTO is its own business, and it maps it to a
// response shape of its own because `targetId` must be emitted rather than omitted.
func (uc *CampaignAuditLookupUseCase) ListByCampaignCode(
	ctx context.Context,
	campaignCode string,
	page, limit int,
) ([]*dto.AdminAuditLogDTO, int64, error) {
	logs, total, err := uc.auditRepo.ListByCampaignCode(ctx, campaignCode, page, limit)
	if err != nil {
		// THE ERROR IS NOT TRANSLATED. The caller wraps it in its own
		// `ErrCampaignAuditListFailed`, because the message an operator reads has to
		// name the campaign they asked about and this module does not know what a
		// campaign is. Swallowing it here and re-wrapping there would leave the cause
		// two hops from the client.
		return nil, 0, err
	}
	return mapper.MapAdminAuditLogsToDTO(logs), total, nil
}
