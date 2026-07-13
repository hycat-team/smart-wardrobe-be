package loyaltypointlotstatus

type LoyaltyPointLotStatus string

const (
	Active   LoyaltyPointLotStatus = "active"
	Consumed LoyaltyPointLotStatus = "consumed"
	Expired  LoyaltyPointLotStatus = "expired"
)
