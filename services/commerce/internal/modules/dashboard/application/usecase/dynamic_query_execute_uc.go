package usecase

import (
	"context"
	"strings"
	"time"

	dashboard_app "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/granularity"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/metrickey"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/constants/targetdashboard"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

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

	var brandID *uuid.UUID

	// RBAC Security Enforcement
	switch target {
	case targetdashboard.Admin:
		if userRole != roleslug.Admin {
			return nil, apperror.NewForbidden("Bạn không có quyền truy cập Admin Dashboard.")
		}
	case targetdashboard.Brand:
		if req.BrandID == nil || strings.TrimSpace(*req.BrandID) == "" {
			return nil, apperror.NewBadRequest("Thiếu thông tin brandId để truy vấn Brand Dashboard.")
		}
		resolvedBrandID, err := uc.brandReader.ResolveBrandRef(ctx, *req.BrandID)
		if err != nil {
			return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
		}
		brandID = &resolvedBrandID
		if _, err := uc.brandReader.GetUserBrandRole(ctx, userID, resolvedBrandID); err != nil {
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
		if brandID != nil {
			snapshots, err := uc.dashboardRepo.GetBrandSnapshots(ctx, *brandID, fromDate, toDate)
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
					case metrickey.PointsExpiring30d:
						metricValues[m] = float64(s.PointsExpiring30d)
					case metrickey.PointsLiabilityValueVND:
						metricValues[m] = s.PointsLiabilityValueVND
					case metrickey.NewCustomers:
						val, _ := uc.brandReader.GetBrandNewCustomerCount(ctx, *brandID, fromDate, toDate)
						metricValues[m] = float64(val)
					case metrickey.CustomerClaimRate:
						val, _ := uc.brandReader.GetBrandCustomerClaimRate(ctx, *brandID)
						metricValues[m] = val
					case metrickey.BenefitRedemptionCount:
						val, _ := uc.brandReader.GetBrandRedemptionCount(ctx, *brandID, fromDate, toDate)
						metricValues[m] = float64(val)
					case metrickey.BrandItemCount:
						val, _ := uc.brandReader.GetBrandItemCount(ctx, *brandID)
						metricValues[m] = float64(val)
					case metrickey.BrandItemCountByType:
						rows, _ := uc.brandReader.GetBrandItemCountByType(ctx, *brandID)
						for _, row := range rows {
							metricValues[metrickey.MetricKey("brand_item_count_"+row.Label)] = float64(row.Count)
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
		prevSum := uc.sumPreviousPeriod(ctx, target, req, brandID, prevFromDate, prevToDate)

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
