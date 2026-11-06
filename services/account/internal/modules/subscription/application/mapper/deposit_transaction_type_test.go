package mapper

import (
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
)

// TestParseDepositTransactionTypeRoundTripsEveryColumnValue is the same
// invariant ParseDepositStatus has: every value the API can print back as
// `transactionType` must be accepted again as a filter input. The column is a
// plain varchar with no CHECK constraint, so the parser is the only guard.
func TestParseDepositTransactionTypeRoundTripsEveryColumnValue(t *testing.T) {
	cases := []struct {
		name string
		want deposittransactiontype.DepositTransactionType
	}{
		{"direct_purchase", deposittransactiontype.DirectPurchase},
		{"wallet_topup", deposittransactiontype.WalletTopup},
	}

	for _, tc := range cases {
		parsed, ok := ParseDepositTransactionType(tc.name)
		if !ok {
			t.Errorf("ParseDepositTransactionType(%q) was rejected", tc.name)
			continue
		}
		if parsed != tc.want {
			t.Errorf("ParseDepositTransactionType(%q) = %q, want %q", tc.name, parsed, tc.want)
		}
	}
}

// TestParseDepositTransactionTypeRejectsUnknownValues pins the other side: a
// value outside the two column values is not a filter, so the handler drops it
// instead of building `transaction_type = 'refund_credit'` and answering an
// empty page that looks like "no such transactions".
func TestParseDepositTransactionTypeRejectsUnknownValues(t *testing.T) {
	for _, name := range []string{"", "DIRECT_PURCHASE", "refund_credit", "wallet_topup "} {
		if got, ok := ParseDepositTransactionType(name); ok {
			t.Errorf("ParseDepositTransactionType(%q) accepted as %q", name, got)
		}
	}
}
