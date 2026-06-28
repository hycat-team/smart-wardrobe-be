package brandcustomerjoinedsource

type BrandCustomerJoinedSource string

const (
	SelfJoin        BrandCustomerJoinedSource = "SELF_JOIN"
	OfflinePurchase BrandCustomerJoinedSource = "OFFLINE_PURCHASE"
	Import          BrandCustomerJoinedSource = "IMPORT"
)
