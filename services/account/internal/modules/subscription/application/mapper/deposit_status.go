package mapper

import "smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"

// DepositStatusToString renders the admin API's string form of a deposit
// status. The type itself stays numeric (R4): it also appears in the
// user-facing `PaymentLinkDTO.PaymentStatus`, which must keep its number, so
// the translation lives here rather than in a MarshalJSON on the type.
//
// The 11 values are data-model §2 and FR-008. A value outside them is out of
// contract — the column is a plain smallint with no CHECK constraint — and is
// reported as `unknown` rather than as a blank, so an operator sees that the
// row carries something the API does not know. `unknown` is deliberately not
// accepted by ParseDepositStatus: a value with no numeric meaning cannot be a
// filter input.
func DepositStatusToString(status depositstatus.DepositStatus) string {
	switch status {
	case depositstatus.Pending:
		return "pending"
	case depositstatus.Success:
		return "success"
	case depositstatus.FailedLegacy:
		return "failed_legacy"
	case depositstatus.Creating:
		return "creating"
	case depositstatus.ReconciliationRequired:
		return "reconciliation_required"
	case depositstatus.Reconciling:
		return "reconciling"
	case depositstatus.CreationFailed:
		return "creation_failed"
	case depositstatus.Cancelled:
		return "cancelled"
	case depositstatus.Expired:
		return "expired"
	case depositstatus.InvestigationRequired:
		return "investigation_required"
	case depositstatus.Refunded:
		return "refunded"
	default:
		return "unknown"
	}
}

// ParseDepositStatus converts a status name (lowercase) into a DepositStatus
// value. It accepts exactly the 11 values DepositStatusToString serialises —
// `failed_legacy` included (FR-008), which the handler-local parser this
// replaced did not — so every status the API prints can be fed straight back as
// the `status` filter (data-model §2's round-trip invariant).
//
// It lives beside DepositStatusToString, not in the handler, because two
// switches in two packages are exactly how the round trip broke.
func ParseDepositStatus(name string) (depositstatus.DepositStatus, bool) {
	switch name {
	case "pending":
		return depositstatus.Pending, true
	case "success":
		return depositstatus.Success, true
	case "failed_legacy":
		return depositstatus.FailedLegacy, true
	case "creating":
		return depositstatus.Creating, true
	case "reconciliation_required":
		return depositstatus.ReconciliationRequired, true
	case "reconciling":
		return depositstatus.Reconciling, true
	case "creation_failed":
		return depositstatus.CreationFailed, true
	case "cancelled":
		return depositstatus.Cancelled, true
	case "expired":
		return depositstatus.Expired, true
	case "investigation_required":
		return depositstatus.InvestigationRequired, true
	case "refunded":
		return depositstatus.Refunded, true
	default:
		return 0, false
	}
}
