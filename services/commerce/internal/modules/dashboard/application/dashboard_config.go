package application

import "time"

const (
	DashboardCacheTTL       = 1 * time.Hour
	AggregationInterval     = 15 * time.Minute
	DefaultLookbackDays     = 30
	UnderutilizedItemsLimit = 10
	MaxDateRangeDays        = 730
)
