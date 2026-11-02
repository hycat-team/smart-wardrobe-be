package money

import (
	"fmt"
	"math"
	"strings"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"

	"github.com/shopspring/decimal"
)

var (
	Zero = decimal.Zero
)

func FromFloatAmount(amount float64) (decimal.Decimal, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return decimal.Zero, fmt.Errorf("Số tiền không hợp lệ")
	}
	return decimal.NewFromFloat(amount), nil
}

func ToFloat(amount decimal.Decimal) float64 {
	return amount.InexactFloat64()
}

// FormatVND renders an amount for customer-facing text such as a transactional
// email: grouped thousands with the đồng suffix, never an exponent.
//
// It exists because `ToFloat` used to be handed straight to an html/template,
// and Go renders a float64 with %v semantics: 1000000 becomes "1e+06". An email
// telling a customer their wallet holds "1.5e+06" is worse than no email.
//
// VND only on purpose: `ValidateSupportedCurrency` rejects every other
// currency, so a grouping rule for a decimal currency would be untested code.
// When a second currency is supported this MUST take the currency instead of
// assuming đồng.
func FormatVND(amount decimal.Decimal) string {
	digits := amount.StringFixed(0)

	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign = "-"
		digits = digits[1:]
	}

	var b strings.Builder
	b.Grow(len(digits) + len(digits)/3 + 2)
	b.WriteString(sign)
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	b.WriteString("đ")
	return b.String()
}

func CurrencyExponent(code currency.Currency) (int32, error) {
	switch code {
	case currency.VND:
		return 0, nil
	case currency.USD:
		return 2, nil
	default:
		return 0, fmt.Errorf("unsupported currency: %s", code)
	}
}

func ValidateScale(amount decimal.Decimal, code currency.Currency) error {
	exponent, err := CurrencyExponent(code)
	if err != nil {
		return err
	}
	if !amount.Equal(amount.Truncate(exponent)) {
		return fmt.Errorf("amount has invalid scale for currency %s", code)
	}
	return nil
}

func ToMinorUnits(amount decimal.Decimal, code currency.Currency) (int64, error) {
	exponent, err := CurrencyExponent(code)
	if err != nil {
		return 0, err
	}
	if err := ValidateScale(amount, code); err != nil {
		return 0, err
	}
	scaleFactor := decimal.New(1, exponent)
	return amount.Mul(scaleFactor).IntPart(), nil
}

const (
	ErrInvalidAmount         = "Số tiền không hợp lệ"
	ErrUnsupportedCurrency   = "Hệ thống hiện chỉ hỗ trợ thanh toán bằng VND"
	ErrInvalidVNDGatewayUnit = "Số tiền phải là số nguyên VND theo yêu cầu của cổng thanh toán"
)

func ValidateSupportedCurrency(cur currency.Currency) error {
	if cur != currency.VND {
		return apperror.NewBadRequest(ErrUnsupportedCurrency)
	}
	return nil
}

func ValidateSupportedCurrencyText(cur string) error {
	normalized := strings.ToLower(strings.TrimSpace(cur))
	if normalized == "" {
		return apperror.NewBadRequest(ErrUnsupportedCurrency)
	}
	return ValidateSupportedCurrency(currency.Currency(normalized))
}
