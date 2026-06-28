package loyaltypointlotstatus

type LoyaltyPointLotStatus string

const (
	Active   LoyaltyPointLotStatus = "ACTIVE"
	Consumed LoyaltyPointLotStatus = "CONSUMED"
	Expired  LoyaltyPointLotStatus = "EXPIRED"
)
