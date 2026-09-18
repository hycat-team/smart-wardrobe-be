package usersubscriptionstatus

type UserSubscriptionStatus string

const (
	Active  UserSubscriptionStatus = "active"
	Expired UserSubscriptionStatus = "expired"
)

func ParseUserSubscriptionStatus(s string) (UserSubscriptionStatus, bool) {
	switch UserSubscriptionStatus(s) {
	case Active:
		return Active, true
	case Expired:
		return Expired, true
	default:
		return "", false
	}
}
