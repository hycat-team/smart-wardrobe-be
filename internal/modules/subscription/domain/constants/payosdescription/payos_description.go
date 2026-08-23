package payosdescription

import (
	"fmt"

	"github.com/shopspring/decimal"
)

const (
	BrandPrefix       = "CLOSY"
	DirectPurchaseFmt = BrandPrefix + " #%d %s %s"
	WalletTopupFmt    = BrandPrefix + " #%d Nap vi %s"
)

func FormatDirectPurchase(orderCode int64, planName, amount string) string {
	return fmt.Sprintf(DirectPurchaseFmt, orderCode, planName, amount)
}

func FormatWalletTopup(orderCode int64, amount string) string {
	return fmt.Sprintf(WalletTopupFmt, orderCode, amount)
}

func FormatAmountShort(amount decimal.Decimal) string {
	amountInt := amount.IntPart()
	if amountInt >= 1000000 {
		return fmt.Sprintf("%dm", amountInt/1000000)
	}
	if amountInt >= 1000 {
		return fmt.Sprintf("%dk", amountInt/1000)
	}
	return fmt.Sprintf("%d", amountInt)
}
