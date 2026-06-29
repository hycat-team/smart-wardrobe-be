package aiusageeventstatus

type AIUsageEventStatus string

const (
	Reserved          AIUsageEventStatus = "reserved"
	InFlight          AIUsageEventStatus = "in_flight"
	Confirmed         AIUsageEventStatus = "confirmed"
	Released          AIUsageEventStatus = "released"
	UnknownUsage      AIUsageEventStatus = "unknown_usage"
	ExpiredUnverified AIUsageEventStatus = "expired_unverified"
)
