package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/useractivitytype"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/validation"

	"github.com/google/uuid"
)

// ListActivityEvents returns the paginated admin activity list (029 FR-001,
// FR-002, FR-005), read-only.
//
// The query string is validated HERE rather than in the handler, beside the
// repository call it constrains: a malformed userId, an activity type outside
// the two existing constants, or a malformed/inverted period are refusals any
// caller must get, not HTTP-only behaviour. The handler only binds and hands
// the query over.
func (uc *UserUseCase) ListActivityEvents(
	ctx context.Context,
	query dto.GetActivityEventsQueryReq,
) (*dto.ActivityEventListRes, error) {
	filter, err := buildAdminActivityEventFilter(query)
	if err != nil {
		return nil, err
	}

	pagination := query.PaginationQuery.Normalize()
	rows, total, err := uc.activityRepo.ListAdmin(ctx, filter, pagination.Page, pagination.Limit)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.ActivityEventRes, len(rows))
	for idx := range rows {
		items[idx] = mapper.MapToActivityEventRes(rows[idx])
	}

	return &dto.ActivityEventListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(pagination, total),
	}, nil
}

// buildAdminActivityEventFilter validates the query and turns it into the
// repository filter. A present-but-invalid value is a 400 with a Vietnamese
// message; an absent one leaves the range open (data-model §3).
func buildAdminActivityEventFilter(query dto.GetActivityEventsQueryReq) (repositories.AdminActivityEventFilter, error) {
	var filter repositories.AdminActivityEventFilter

	if query.UserID != "" {
		userID, err := uuid.Parse(query.UserID)
		if err != nil {
			return filter, apperror.NewBadRequest("Mã người dùng (userId) không hợp lệ. Vui lòng sử dụng định dạng UUID.")
		}
		filter.UserID = &userID
	}

	if query.ActivityType != "" {
		activityType := useractivitytype.UserActivityType(query.ActivityType)
		if activityType != useractivitytype.Login && activityType != useractivitytype.Refresh {
			return filter, apperror.NewBadRequest("Loại hoạt động (activityType) không hợp lệ. Chỉ chấp nhận: login, refresh.")
		}
		filter.ActivityType = &activityType
	}

	from, to, err := validation.ParseVNDateRangeQuery(query.FromDate, query.ToDate)
	if err != nil {
		return filter, err
	}
	filter.From = from
	filter.To = to

	return filter, nil
}
