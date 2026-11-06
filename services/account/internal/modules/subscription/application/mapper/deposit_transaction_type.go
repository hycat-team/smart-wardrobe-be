package mapper

import "smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"

// ParseDepositTransactionType converts the admin API's `transactionType` filter
// value into the domain constant. The domain type is already a string on the
// wire (`DepositTransactionRes.transactionType`), so unlike DepositStatus there
// is no second serialisation to keep in sync — the only job here is to refuse
// anything outside the two values the column can hold, so an unknown filter
// cannot silently match zero rows.
//
// A value outside the contract is reported as "not a valid filter" and the
// handler drops it, mirroring how the existing `status` filter behaves.
func ParseDepositTransactionType(
	name string,
) (deposittransactiontype.DepositTransactionType, bool) {
	switch name {
	case string(deposittransactiontype.DirectPurchase):
		return deposittransactiontype.DirectPurchase, true
	case string(deposittransactiontype.WalletTopup):
		return deposittransactiontype.WalletTopup, true
	default:
		return "", false
	}
}
