package subscription

import (
	"context"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// plannedVersion is what the batch observed BEFORE it wrote anything.
type plannedVersion struct {
	version int64
	found   bool
}

// loadPlannedVersions reads the subscription versions ONCE for the whole batch
// (FR-069, FR-096).
//
// One query for 500 accounts rather than 500: a pre-pass per item would be a
// second round of I/O that buys nothing. It also makes the conflict real rather
// than theoretical - the last item of a long list is planned early and written
// late, so a purchase in between has to be detected instead of silently
// overwritten.
//
// The identifiers come from `resolved`, the map the resolver already produced, so
// the pre-pass never re-queries identity for accounts that were just resolved. They
// are deduplicated before the query: a list naming the same account three times is
// three lines of output but one row to read.
func (uc *SubscriptionUseCase) loadPlannedVersions(
	ctx context.Context,
	items []sub_dto.AdminGrantBatchItem,
	resolved map[string]*ResolvedUser,
) (map[uuid.UUID]plannedVersion, error) {
	out := make(map[uuid.UUID]plannedVersion, len(items))

	ids := make([]uuid.UUID, 0, len(items))
	seen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		entry := resolved[normalizeIdentifier(item.UserIdentifier)]
		if entry == nil || entry.UserID == uuid.Nil {
			continue
		}
		if _, dup := seen[entry.UserID]; dup {
			continue
		}
		seen[entry.UserID] = struct{}{}
		ids = append(ids, entry.UserID)
	}
	if len(ids) == 0 {
		return out, nil
	}

	subs, err := uc.userSubRepo.GetByUserIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		out[sub.UserID] = plannedVersion{version: sub.Version, found: true}
	}
	return out, nil
}

// checkPlannedVersion reports a conflict when the row changed after the batch
// planned it (FR-069).
//
// The absence of a subscription is NOT a conflict: an account with no
// subscription yet is the normal case for a first grant, and its "planned"
// version is simply nothing. Treating that as a change would refuse every grant
// to a new customer.
//
// 024 T129: the factory return replaces a package-level `errors.New` sentinel.
// `apperror.Error.Is` compares status and message, so the call sites still match
// with `errors.Is` against a fresh instance.
func checkPlannedVersion(req grantRequest, current *entities.UserSubscription) error {
	if !req.plannedFound {
		return nil
	}
	if current == nil {
		// The subscription disappeared between the read and the write. Granting
		// would recreate it and lose whatever the deletion was meant to end.
		return subscriptionerrors.ErrAdminGrantConcurrentModification()
	}
	if current.Version != req.plannedVersion {
		return subscriptionerrors.ErrAdminGrantConcurrentModification()
	}
	return nil
}
