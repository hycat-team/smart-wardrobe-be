package usecase

import (
	"context"
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

	gran := granularity.Granularity(strings.ToLower(string(req.Granularity)))
	if gran == "" {
		gran = granularity.Day
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
				case metrickey.AIRequestsCount:
					metricValues[m] = float64(s.AIRequestsCount)
				case metrickey.SubscriptionRevenueVND:
					metricValues[m] = s.SubscriptionRevenueVND
				case metrickey.NetAIMarginVND:
					metricValues[m] = s.NetAIMarginVND
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
		for _, pt := range series {
			for k, v := range pt.Metrics {
				currSum[k] += v
			}
		}

		// Query previous period snapshots from DB for accurate comparison
		prevSum := uc.sumPreviousPeriod(ctx, target, req, prevFromDate, prevToDate)

		deltaMap := map[metrickey.MetricKey]float64{}
		for _, m := range req.Metrics {
			currVal := currSum[m]
			prevVal := prevSum[m]
			if currVal == 0 && prevVal == 0 {
				deltaMap[m] = 0.0
			} else if prevVal == 0 {
				deltaMap[m] = 100.0
			} else {
				deltaMap[m] = ((currVal - prevVal) / prevVal) * 100.0
			}
		}

		comparison = &dto.ComparisonDTO{
			PreviousPeriodFrom:     prevFromDate.Format("2006-01-02"),
			PreviousPeriodTo:       prevToDate.Format("2006-01-02"),
			MetricDeltaPercentages: deltaMap,
		}
	}

	// Auto-downgrade granularity for large date ranges
	gran = enforceMaxGranularity(fromDate, toDate, gran)
	fromDate = enforceMaxDateRange(fromDate, toDate)

	return &dto.DynamicQueryRes{
		TargetDashboard: target,
		Granularity:     gran,
		Series:          series,
		Comparison:      comparison,
	}, nil
}

// sumPreviousPeriod queries snapshot data for the previous period and sums metric values.
func (uc *DynamicQueryUseCase) sumPreviousPeriod(ctx context.Context, target targetdashboard.TargetDashboard, req *dto.DynamicQueryReq, prevFrom, prevTo time.Time) map[metrickey.MetricKey]float64 {
	prevSum := map[metrickey.MetricKey]float64{}

	switch target {
	case targetdashboard.Admin:
		prevSnapshots, err := uc.dashboardRepo.GetAdminSnapshots(ctx, prevFrom, prevTo)
		if err != nil {
			return prevSum
		}
		if len(prevSnapshots) == 0 {
			return prevSum
		}
		for _, s := range prevSnapshots {
			for _, m := range req.Metrics {
				switch m {
				case metrickey.TotalUsers:
					prevSum[m] += float64(s.TotalUsers)
				case metrickey.ActiveUsersDau:
					prevSum[m] += float64(s.ActiveUsersDau)
				case metrickey.AIRequestsCount:
					prevSum[m] += float64(s.AIRequestsCount)
				case metrickey.SubscriptionRevenueVND:
					prevSum[m] += s.SubscriptionRevenueVND
				case metrickey.NetAIMarginVND:
					prevSum[m] += s.NetAIMarginVND
				}
			}
		}

	case targetdashboard.Brand:
		if req.BrandID == nil {
			return prevSum
		}
		prevSnapshots, err := uc.dashboardRepo.GetBrandSnapshots(ctx, *req.BrandID, prevFrom, prevTo)
		if err != nil {
			return prevSum
		}
		if len(prevSnapshots) == 0 {
			return prevSum
		}
		for _, s := range prevSnapshots {
			for _, m := range req.Metrics {
				switch m {
				case metrickey.TotalMembers:
					prevSum[m] += float64(s.TotalMembers)
				case metrickey.PointsIssued:
					prevSum[m] += float64(s.PointsIssued)
				case metrickey.PointsRedeemed:
					prevSum[m] += float64(s.PointsRedeemed)
				case metrickey.ActiveVouchers:
					prevSum[m] += float64(s.ActiveVouchers)
				}
			}
		}

	case targetdashboard.User:
		// User dashboard is a point-in-time snapshot, not a time series,
		// so previous period comparison is not applicable.
	}

	return prevSum
}

// enforceMaxDateRange caps the date range to prevent excessive DB queries.
func enforceMaxDateRange(fromDate, toDate time.Time) time.Time {
	days := int(toDate.Sub(fromDate).Hours() / 24)
	if days < 0 {
		days = -days
	}
	if days > dashboard_app.MaxDateRangeDays {
		return toDate.AddDate(0, 0, -dashboard_app.MaxDateRangeDays)
	}
	return fromDate
}

// enforceMaxGranularity auto-downgrades granularity when the date range exceeds limits.
func enforceMaxGranularity(fromDate, toDate time.Time, gran granularity.Granularity) granularity.Granularity {
	days := int(toDate.Sub(fromDate).Hours() / 24)
	if days < 0 {
		days = -days
	}

	if days > dashboard_app.MaxDateRangeDays && (gran == granularity.Hour || gran == granularity.Day) {
		return granularity.Month
	}
	if days > dashboard_app.MaxDateRangeDays/2 && gran == granularity.Hour {
		return granularity.Day
	}
	return gran
}
