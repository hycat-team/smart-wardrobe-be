package money

import (
	"fmt"
	"math"

	"smart-wardrobe-be/internal/shared/domain/constants/currency"

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

func ToFloatForDTO(amount decimal.Decimal) float64 {
	return amount.InexactFloat64()
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
