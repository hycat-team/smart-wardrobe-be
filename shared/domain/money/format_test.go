package money

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// TestFormatVNDNeverRendersAnExponent pins the defect the helper was written
// for. Before it existed, templates received `ToFloat(amount)` - a float64 - and
// html/template formats a float64 with %v semantics, so 1.000.000 rendered as
// "1e+06". A wallet receipt that reads "Số dư hiện tại: 1.5e+06" is a message no
// customer can act on.
func TestFormatVNDNeverRendersAnExponent(t *testing.T) {
	cases := []struct {
		amount decimal.Decimal
		want   string
	}{
		{decimal.Zero, "0đ"},
		{decimal.NewFromInt(5000), "5.000đ"},
		{decimal.NewFromInt(99000), "99.000đ"},
		{decimal.NewFromInt(500000), "500.000đ"},
		{decimal.NewFromInt(1000000), "1.000.000đ"},
		{decimal.NewFromInt(1500000), "1.500.000đ"},
		{decimal.NewFromInt(1234567890), "1.234.567.890đ"},
	}

	for _, tc := range cases {
		got := FormatVND(tc.amount)
		if got != tc.want {
			t.Errorf("FormatVND(%s) = %q, want %q", tc.amount, got, tc.want)
		}
		if strings.ContainsAny(got, "eE") {
			t.Errorf("FormatVND(%s) = %q must never use exponent notation", tc.amount, got)
		}
	}
}

// TestFormatVNDGroupsNegativeAmounts covers the wallet-statement direction a
// future caller could pass: a debit is negative, and the sign must not disturb
// the grouping.
func TestFormatVNDGroupsNegativeAmounts(t *testing.T) {
	if got, want := FormatVND(decimal.NewFromInt(-1000000)), "-1.000.000đ"; got != want {
		t.Errorf("FormatVND(-1000000) = %q, want %q", got, want)
	}
}

// TestFormatVNDRoundsToWholeDong documents that VND has no minor unit, so a
// fractional amount (which the payment path rejects upstream) cannot render a
// stray ".5" into an email.
func TestFormatVNDRoundsToWholeDong(t *testing.T) {
	amount := decimal.RequireFromString("5000.4")
	if got, want := FormatVND(amount), "5.000đ"; got != want {
		t.Errorf("FormatVND(5000.4) = %q, want %q", got, want)
	}
}
