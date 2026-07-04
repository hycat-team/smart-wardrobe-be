package subscriptionrenewalstatus

type SubscriptionRenewalStatus string

const (
	Processing SubscriptionRenewalStatus = "processing"
	Succeeded  SubscriptionRenewalStatus = "succeeded"
	Skipped    SubscriptionRenewalStatus = "skipped"
	Failed     SubscriptionRenewalStatus = "failed"
	Renewed    SubscriptionRenewalStatus = "renewed"
	Downgraded SubscriptionRenewalStatus = "downgraded"
)
