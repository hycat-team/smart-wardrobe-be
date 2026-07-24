package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	dashboard_app "smart-wardrobe-be/internal/modules/dashboard/application"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/granularity"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/metrickey"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/targetdashboard"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type DynamicQueryUseCase struct {
	dashboardRepo dashboardRepos.IDashboardRepository
	brandReader   brandContract.IBrandDashboardReader
}

func NewDynamicQueryUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	brandReader brandContract.IBrandDashboardReader,
) uc_interfaces.IDynamicQueryUseCase {
	return &DynamicQueryUseCase{
		dashboardRepo: dashboardRepo,
		brandReader:   brandReader,
	}
}

func (uc *DynamicQueryUseCase) ExecuteDynamicQuery(
	ctx context.Context,
	userID uuid.UUID,
	userRole roleslug.RoleSlug,
	req *dto.DynamicQueryReq,
) (*dto.DynamicQueryRes, error) {
	if req == nil {
		return nil, dashboarderrors.ErrDashboardDataNotFound()
	}

	target := targetdashboard.TargetDashboard(strings.ToLower(string(req.TargetDashboard)))

	// RBAC Security Enforcement
	switch target {
	case targetdashboard.Admin:
		if userRole != roleslug.Admin {
			return nil, apperror.NewForbidden("Bạn không có quyền truy cập Admin Dashboard.")
		}
	case targetdashboard.Brand:
		if req.BrandID == nil {
			return nil, apperror.NewBadRequest("Thiếu thông tin brandId để truy vấn Brand Dashboard.")
		}
		role, err := uc.brandReader.GetUserBrandRole(ctx, userID, *req.BrandID)
		if err != nil {
			return nil, apperror.NewInternalError("Không thể xác thực quyền truy cập nhãn hàng. Vui lòng thử lại sau.")
		}
		if role == "" {
			return nil, apperror.NewForbidden("Bạn không có quyền truy cập thông tin nhãn hàng này.")
		}
	case targetdashboard.User:
		// User queries scoped to self
	default:
		return nil, apperror.NewBadRequest("TargetDashboard không hợp lệ.")
	}

	fromDate, err := time.Parse("2006-01-02", req.Timeframe.FromDate)
	if err != nil {
		return nil, apperror.NewBadRequest("Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
	}

	toDate, err := time.Parse("2006-01-02", req.Timeframe.ToDate)
	if err != nil {
		return nil, apperror.NewBadRequest("Định dạng ngày kết thúc (toDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
	}

	if fromDate.After(toDate) {
		return nil, apperror.NewBadRequest("Ngày bắt đầu (fromDate) không được sau ngày kết thúc (toDate).")
	}

	days := int(toDate.Sub(fromDate).Hours() / 24)
	if days > dashboard_app.MaxDateRangeDays {
		return nil, apperror.NewBadRequest("Khoảng thời gian truy vấn không được vượt quá 730 ngày. Vui lòng chọn khoảng thời gian ngắn hơn.")
	}

	gran := granularity.Granularity(strings.ToLower(string(req.Granularity)))
	switch gran {
	case granularity.Day, granularity.Week, granularity.Month, granularity.Year:
	case "":
		gran = granularity.Day
	default:
		return nil, apperror.NewBadRequest("Giá trị granularity không hợp lệ. Giá trị hợp lệ: \"day\", \"week\", \"month\", \"year\".")
	}

	series := make([]dto.TimeSeriesPointDTO, 0)

	switch target {
	case targetdashboard.Admin:
		snapshots, err := uc.dashboardRepo.GetAdminSnapshots(ctx, fromDate, toDate)
		if err != nil {
			return nil, err
		}

		for _, s := range snapshots {
			metricValues := map[metrickey.MetricKey]float64{}
			for _, m := range req.Metrics {
				switch m {
				case metrickey.TotalUsers:
					metricValues[m] = float64(s.TotalUsers)
				case metrickey.ActiveUsersDau:
					metricValues[m] = float64(s.ActiveUsersDau)
				case metrickey.TotalBrands:
					metricValues[m] = float64(s.TotalBrands)
				case metrickey.PendingReviewBrands:
					metricValues[m] = float64(s.PendingReviewBrands)
				case metrickey.AIRequestsCount:
					metricValues[m] = float64(s.AIRequestsCount)
				case metrickey.SubscriptionRevenueVND:
					metricValues[m] = s.SubscriptionRevenueVND
				case metrickey.NetAIMarginVND:
					metricValues[m] = s.NetAIMarginVND
				case metrickey.NewUsers:
					metricValues[m] = float64(s.NewUsers)
				case metrickey.AIErrorCount:
					metricValues[m] = float64(s.AIErrorCount)
				case metrickey.AIErrorRate:
					if s.AIRequestsCount > 0 {
						metricValues[m] = float64(s.AIErrorCount) / float64(s.AIRequestsCount) * 100.0
					} else {
						metricValues[m] = 0
					}
				}
			}
			series = append(series, dto.TimeSeriesPointDTO{
				Timestamp: s.SnapshotDate,
				Metrics:   metricValues,
			})
		}

	case targetdashboard.Brand:
		if req.BrandID != nil {
			snapshots, err := uc.dashboardRepo.GetBrandSnapshots(ctx, *req.BrandID, fromDate, toDate)
			if err != nil {
				return nil, err
			}

			for _, s := range snapshots {
				metricValues := map[metrickey.MetricKey]float64{}
				for _, m := range req.Metrics {
					switch m {
					case metrickey.TotalMembers:
						metricValues[m] = float64(s.TotalMembers)
					case metrickey.PointsIssued:
						metricValues[m] = float64(s.PointsIssued)
					case metrickey.PointsRedeemed:
						metricValues[m] = float64(s.PointsRedeemed)
					case metrickey.ActiveVouchers:
						metricValues[m] = float64(s.ActiveVouchers)
					case metrickey.VouchersRedeemed:
						metricValues[m] = float64(s.VouchersRedeemed)
					case metrickey.DigitalSampleVotesCount:
						metricValues[m] = float64(s.DigitalSampleVotesCount)
					case metrickey.DigitalSampleAvgRating:
						metricValues[m] = s.DigitalSampleAvgRating
					case metrickey.OpenCSTickets:
						metricValues[m] = float64(s.OpenCSTickets)
					case metrickey.CSFirstResponseAvgSeconds:
						metricValues[m] = float64(s.CSFirstResponseAvgSeconds)
					case metrickey.PointsExpiring30d:
						metricValues[m] = float64(s.PointsExpiring30d)
					case metrickey.PointsLiabilityValueVND:
						metricValues[m] = s.PointsLiabilityValueVND
					case metrickey.NewCustomers:
						val, _ := uc.dashboardRepo.GetBrandNewCustomerCount(ctx, *req.BrandID, fromDate, toDate)
						metricValues[m] = val
					case metrickey.CustomerClaimRate:
						val, _ := uc.dashboardRepo.GetBrandCustomerClaimRate(ctx, *req.BrandID)
						metricValues[m] = val
					case metrickey.BenefitRedemptionCount:
						val, _ := uc.dashboardRepo.GetBrandRedemptionCount(ctx, *req.BrandID, fromDate, toDate)
						metricValues[m] = val
					case metrickey.BrandItemCount:
						val, _ := uc.dashboardRepo.GetBrandItemCount(ctx, *req.BrandID)
						metricValues[m] = val
					case metrickey.BrandItemCountByType:
						rows, _ := uc.dashboardRepo.GetBrandItemCountByType(ctx, *req.BrandID)
						for _, row := range rows {
							itemType, _ := row["item_type"].(string)
							count, _ := row["count"].(int64)
							metricValues[metrickey.MetricKey("brand_item_count_"+itemType)] = float64(count)
						}
					}
				}
				series = append(series, dto.TimeSeriesPointDTO{
					Timestamp: s.SnapshotDate,
					Metrics:   metricValues,
				})
			}
		}

	case targetdashboard.User:
		summary, err := uc.dashboardRepo.GetUserWardrobeSummary(ctx, userID)
		if err != nil {
			return nil, err
		}
		if summary != nil {
			metricValues := map[metrickey.MetricKey]float64{}
			for _, m := range req.Metrics {
				switch m {
				case metrickey.TotalWardrobeItems:
					metricValues[m] = float64(summary.TotalItems)
				case metrickey.TotalWardrobeValueVND:
					metricValues[m] = summary.TotalWardrobeValueVND
				}
			}
			series = append(series, dto.TimeSeriesPointDTO{
				Timestamp: summary.UpdatedAt,
				Metrics:   metricValues,
			})
		}
	}

	if len(series) == 0 {
		if target == targetdashboard.Brand {
			return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
		}
		return nil, dashboarderrors.ErrDashboardDataNotFound()
	}

	var comparison *dto.ComparisonDTO
	if req.CompareWithPrevious {
		duration := toDate.Sub(fromDate)
		prevToDate := fromDate.AddDate(0, 0, -1)
		prevFromDate := prevToDate.Add(-duration)

		currSum := map[metrickey.MetricKey]float64{}
		currCount := map[metrickey.MetricKey]int{}
		for _, pt := range series {
			for k, v := range pt.Metrics {
				switch getMetricAggType(k) {
				case metricCumulative:
					currSum[k] = v // last value
				case metricNonCumulative:
					currSum[k] += v
				case metricRate:
					currSum[k] += v
					currCount[k]++
				}
			}
		}
		for k := range currSum {
			if getMetricAggType(k) == metricRate && currCount[k] > 0 {
				currSum[k] /= float64(currCount[k])
			}
		}

		// Query previous period snapshots from DB for accurate comparison
		prevSum := uc.sumPreviousPeriod(ctx, target, req, prevFromDate, prevToDate)

		deltaMap := map[metrickey.MetricKey]float64{}
		for _, m := range req.Metrics {
			prevVal := prevSum[m]
			if prevVal == 0 {
				continue // Không thể tính delta % khi kỳ trước không có dữ liệu
			}
			currVal := currSum[m]
			deltaMap[m] = ((currVal - prevVal) / prevVal) * 100.0
		}

		comparison = &dto.ComparisonDTO{
			PreviousPeriodFrom:     prevFromDate.Format("2006-01-02"),
			PreviousPeriodTo:       prevToDate.Format("2006-01-02"),
			MetricDeltaPercentages: deltaMap,
		}
	}

	// Auto-downgrade granularity for large date ranges (trước aggregation)
	effectiveGran := enforceMaxGranularity(fromDate, toDate, gran)

	// Aggregation: gộp daily series theo effective granularity nếu là week/month/year
	if effectiveGran == granularity.Week || effectiveGran == granularity.Month || effectiveGran == granularity.Year {
		series = aggregateByGranularity(series, effectiveGran, req.Metrics)
		gran = effectiveGran
	}

	return &dto.DynamicQueryRes{
		TargetDashboard: target,
		Granularity:     gran,
		Series:          series,
		Comparison:      comparison,
	}, nil
}

// sumPreviousPeriod queries snapshot data for the previous period and computes aggregated metric values.
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
	case metrickey.CSFirstResponseAvgSeconds:
		v = float64(s.CSFirstResponseAvgSeconds)
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
		metrickey.DigitalSampleAvgRating, metrickey.CSFirstResponseAvgSeconds,
		metrickey.AIErrorRate, metrickey.CustomerClaimRate:
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
