package brandcustomerjoinedsource

type BrandCustomerJoinedSource string

const (
	SelfJoin        BrandCustomerJoinedSource = "self_join"
	OfflinePurchase BrandCustomerJoinedSource = "offline_purchase"
	Import          BrandCustomerJoinedSource = "import"
)
