package payosdescription

import (
	"fmt"
)

const (
	BrandPrefix    = "CLOSY"
	TransactionFmt = BrandPrefix + "%d"
)

func FormatTransactionDescription(orderCode int64) string {
	return fmt.Sprintf(TransactionFmt, orderCode)
}
