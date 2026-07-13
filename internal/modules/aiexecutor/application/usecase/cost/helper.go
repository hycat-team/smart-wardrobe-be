package cost

import "github.com/shopspring/decimal"

func microCost(inputTokens, outputTokens int64, inputRate, outputRate, fx decimal.Decimal) int64 {
	return decimal.NewFromInt(inputTokens).Mul(inputRate).Mul(fx).Add(decimal.NewFromInt(outputTokens).Mul(outputRate).Mul(fx)).Ceil().IntPart()
}
