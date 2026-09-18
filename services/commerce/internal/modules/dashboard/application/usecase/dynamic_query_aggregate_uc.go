package usecase

import (
	"context"
	"fmt"
	"time"

	dashboard_app "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/granularity"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/metrickey"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/targetdashboard"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"
)

func (uc *DynamicQueryUseCase) sumPreviousPeriod(ctx context.Context, target targetdashboard.TargetDashboard, req *dto.DynamicQueryReq, prevFrom, prevTo time.Time) map[metrickey.MetricKey]float64 {
	prevSum := map[metrickey.MetricKey]float64{}
	prevCount := map[metrickey.MetricKey]int{}

	switch target {
	case targetdashboard.Admin:
		prevSnapshots, err := uc.dashboardRepo.GetAdminSnapshots(ctx, prevFrom, prevTo)
		if err != nil || len(prevSnapshots) == 0 {
			return prevSum
		}
		for _, s := range prevSnapshots {
			for _, m := range req.Metrics {
				val := metricValueFromAdminSnapshot(s, m)
				if val == nil {
					continue
				}
				switch getMetricAggType(m) {
				case metricCumulative:
					prevSum[m] = *val // last value
				case metricNonCumulative:
					prevSum[m] += *val
				case metricRate:
					prevSum[m] += *val
					prevCount[m]++
				}
			}
		}

	case targetdashboard.Brand:
		if req.BrandID == nil {
			return prevSum
		}
		prevSnapshots, err := uc.dashboardRepo.GetBrandSnapshots(ctx, *req.BrandID, prevFrom, prevTo)
		if err != nil || len(prevSnapshots) == 0 {
			return prevSum
		}
		for _, s := range prevSnapshots {
			for _, m := range req.Metrics {
				val := metricValueFromBrandSnapshot(s, m)
				if val == nil {
					continue
				}
				switch getMetricAggType(m) {
				case metricCumulative:
					prevSum[m] = *val // last value
				case metricNonCumulative:
					prevSum[m] += *val
				case metricRate:
					prevSum[m] += *val
					prevCount[m]++
				}
			}
		}

	case targetdashboard.User:
		// User dashboard is a point-in-time snapshot, not a time series,
		// so previous period comparison is not applicable.
	}

	// Average rate metrics
	for k := range prevSum {
		if getMetricAggType(k) == metricRate && prevCount[k] > 0 {
			prevSum[k] /= float64(prevCount[k])
		}
	}
	return prevSum
}

func metricValueFromAdminSnapshot(s *entities.DashboardAdminDailySnapshot, m metrickey.MetricKey) *float64 {
	var v float64
	switch m {
	case metrickey.TotalUsers:
		v = float64(s.TotalUsers)
	case metrickey.ActiveUsersDau:
		v = float64(s.ActiveUsersDau)
	case metrickey.TotalBrands:
		v = float64(s.TotalBrands)
	case metrickey.PendingReviewBrands:
		v = float64(s.PendingReviewBrands)
	case metrickey.AIRequestsCount:
		v = float64(s.AIRequestsCount)
	case metrickey.SubscriptionRevenueVND:
		v = s.SubscriptionRevenueVND
	case metrickey.NetAIMarginVND:
		v = s.NetAIMarginVND
	case metrickey.NewUsers:
		v = float64(s.NewUsers)
	case metrickey.AIErrorCount:
		v = float64(s.AIErrorCount)
	case metrickey.AIErrorRate:
		if s.AIRequestsCount > 0 {
			v = float64(s.AIErrorCount) / float64(s.AIRequestsCount) * 100.0
		}
	default:
		return nil
	}
	return &v
}

func metricValueFromBrandSnapshot(s *entities.DashboardBrandDailySnapshot, m metrickey.MetricKey) *float64 {
	var v float64
	switch m {
	case metrickey.TotalMembers:
		v = float64(s.TotalMembers)
	case metrickey.PointsIssued:
		v = float64(s.PointsIssued)
	case metrickey.PointsRedeemed:
		v = float64(s.PointsRedeemed)
	case metrickey.ActiveVouchers:
		v = float64(s.ActiveVouchers)
	case metrickey.VouchersRedeemed:
		v = float64(s.VouchersRedeemed)
	case metrickey.DigitalSampleVotesCount:
		v = float64(s.DigitalSampleVotesCount)
	case metrickey.DigitalSampleAvgRating:
		v = s.DigitalSampleAvgRating
	case metrickey.OpenCSTickets:
		v = float64(s.OpenCSTickets)
	case metrickey.PointsExpiring30d:
		v = float64(s.PointsExpiring30d)
	case metrickey.PointsLiabilityValueVND:
		v = s.PointsLiabilityValueVND
	default:
		return nil
	}
	return &v
}

type metricAggType int

const (
	metricCumulative metricAggType = iota
	metricNonCumulative
	metricRate
)

func getMetricAggType(m metrickey.MetricKey) metricAggType {
	switch m {
	case metrickey.TotalUsers, metrickey.TotalBrands, metrickey.PendingReviewBrands,
		metrickey.TotalMembers, metrickey.ActiveVouchers,
		metrickey.PointsExpiring30d, metrickey.PointsLiabilityValueVND,
		metrickey.TotalWardrobeItems, metrickey.TotalWardrobeValueVND:
		return metricCumulative
	case metrickey.AIRequestsCount, metrickey.SubscriptionRevenueVND,
		metrickey.PointsIssued, metrickey.PointsRedeemed,
		metrickey.VouchersRedeemed, metrickey.DigitalSampleVotesCount,
		metrickey.OpenCSTickets, metrickey.NewUsers, metrickey.AIErrorCount,
		metrickey.NewCustomers, metrickey.BenefitRedemptionCount,
		metrickey.BrandItemCount:
		return metricNonCumulative
	case metrickey.ActiveUsersDau, metrickey.NetAIMarginVND,
		metrickey.DigitalSampleAvgRating, metrickey.AIErrorRate,
		metrickey.CustomerClaimRate:
		return metricRate
	default:
		return metricNonCumulative
	}
}

func periodKey(t time.Time, gran granularity.Granularity) string {
	switch gran {
	case granularity.Week:
		y, w := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case granularity.Month:
		return t.Format("2006-01")
	case granularity.Year:
		return t.Format("2006")
	default:
		return t.Format("2006-01-02")
	}
}

func periodStart(t time.Time, gran granularity.Granularity) time.Time {
	switch gran {
	case granularity.Week:
		wd := t.Weekday()
		if wd == time.Sunday {
			wd = 7
		}
		return t.AddDate(0, 0, -int(wd-time.Monday)).Truncate(24 * time.Hour)
	case granularity.Month:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case granularity.Year:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return t.Truncate(24 * time.Hour)
	}
}

func aggregateByGranularity(series []dto.TimeSeriesPointDTO, gran granularity.Granularity, metrics []metrickey.MetricKey) []dto.TimeSeriesPointDTO {
	if len(series) == 0 || (gran != granularity.Week && gran != granularity.Month && gran != granularity.Year) {
		return series
	}

	type periodData struct {
		points []dto.TimeSeriesPointDTO
		start  time.Time
	}
	periods := map[string]*periodData{}
	order := make([]string, 0)

	for _, pt := range series {
		key := periodKey(pt.Timestamp, gran)
		if _, ok := periods[key]; !ok {
			periods[key] = &periodData{start: periodStart(pt.Timestamp, gran)}
			order = append(order, key)
		}
		periods[key].points = append(periods[key].points, pt)
	}

	result := make([]dto.TimeSeriesPointDTO, 0, len(order))
	for _, key := range order {
		pd := periods[key]
		aggMetrics := map[metrickey.MetricKey]float64{}

		for _, m := range metrics {
			aggType := getMetricAggType(m)
			switch aggType {
			case metricCumulative:
				lastPt := pd.points[len(pd.points)-1]
				if v, ok := lastPt.Metrics[m]; ok {
					aggMetrics[m] = v
				}
			case metricNonCumulative:
				var sum float64
				for _, pt := range pd.points {
					sum += pt.Metrics[m]
				}
				aggMetrics[m] = sum
			case metricRate:
				var sum float64
				count := 0
				for _, pt := range pd.points {
					if _, ok := pt.Metrics[m]; ok {
						sum += pt.Metrics[m]
						count++
					}
				}
				if count > 0 {
					aggMetrics[m] = sum / float64(count)
				}
			}
		}

		result = append(result, dto.TimeSeriesPointDTO{
			Timestamp: pd.start,
			Metrics:   aggMetrics,
		})
	}
	return result
}

// enforceMaxGranularity auto-downgrades granularity when the date range exceeds limits.
func enforceMaxGranularity(fromDate, toDate time.Time, gran granularity.Granularity) granularity.Granularity {
	days := int(toDate.Sub(fromDate).Hours() / 24)
	if days < 0 {
		days = -days
	}

	if days > dashboard_app.MaxDateRangeDays && gran == granularity.Day {
		return granularity.Month
	}
	return gran
}
