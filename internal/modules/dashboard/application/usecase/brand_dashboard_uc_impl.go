package usecase

import (
	"context"
	"sort"
	"time"

	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/application/mapper"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/feedbacksentiment"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	fashionContract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	wardrobeContract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/internal/pkg/utils/numberutils"

	"github.com/google/uuid"
)

type BrandDashboardUseCase struct {
	dashboardRepo   dashboardRepos.IDashboardRepository
	brandReader     brandContract.IBrandDashboardReader
	wardrobeReader  wardrobeContract.IWardrobeDashboardReader
	fashionContract fashionContract.IFashionContract
}

func NewBrandDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	brandReader brandContract.IBrandDashboardReader,
	wardrobeReader wardrobeContract.IWardrobeDashboardReader,
	fashionContract fashionContract.IFashionContract,
) uc_interfaces.IBrandDashboardUseCase {
	return &BrandDashboardUseCase{
		dashboardRepo:   dashboardRepo,
		brandReader:     brandReader,
		wardrobeReader:  wardrobeReader,
		fashionContract: fashionContract,
	}
}

func (uc *BrandDashboardUseCase) GetBrandKPICards(ctx context.Context, brandID uuid.UUID) (*dto.BrandKPICardsRes, error) {
	latest, err := uc.dashboardRepo.GetLatestBrandSnapshot(ctx, brandID)
	if err != nil {
		return nil, err
	}

	if latest == nil {
		stats, err := uc.brandReader.GetBrandAnalyticsStats(ctx, brandID, time.Now())
		if err != nil {
			return nil, err
		}
		if stats != nil {
			return &dto.BrandKPICardsRes{
				BrandID:                   brandID,
				TotalMembers:              stats.TotalMembers,
				PointsIssued:              stats.PointsIssued,
				PointsRedeemed:            stats.PointsRedeemed,
				ActiveVouchers:            stats.ActiveVouchers,
				VouchersRedeemed:          stats.VouchersRedeemed,
				DigitalSampleVotesCount:   stats.DigitalSampleVotesCount,
				DigitalSampleAvgRating:    stats.DigitalSampleAvgRating,
				OpenCSTickets:             stats.OpenCSTickets,
				CSFirstResponseAvgSeconds: stats.CSFirstResponseAvgSeconds,
				UpdatedAt:                 time.Now(),
			}, nil
		}
		return &dto.BrandKPICardsRes{
			BrandID:   brandID,
			UpdatedAt: time.Now(),
		}, nil
	}

	return &dto.BrandKPICardsRes{
		BrandID:                   latest.BrandID,
		TotalMembers:              latest.TotalMembers,
		PointsIssued:              latest.PointsIssued,
		PointsRedeemed:            latest.PointsRedeemed,
		ActiveVouchers:            latest.ActiveVouchers,
		VouchersRedeemed:          latest.VouchersRedeemed,
		DigitalSampleVotesCount:   latest.DigitalSampleVotesCount,
		DigitalSampleAvgRating:    latest.DigitalSampleAvgRating,
		OpenCSTickets:             latest.OpenCSTickets,
		CSFirstResponseAvgSeconds: latest.CSFirstResponseAvgSeconds,
		UpdatedAt:                 latest.UpdatedAt,
	}, nil
}

func (uc *BrandDashboardUseCase) GetPointsLiability(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandPointsLiabilityRes, error) {
	role, err := uc.brandReader.GetUserBrandRole(ctx, userID, brandID)
	if err != nil {
		return nil, err
	}

	latest, err := uc.dashboardRepo.GetLatestBrandSnapshot(ctx, brandID)
	if err != nil {
		return nil, err
	}

	expiring30d := 0
	valVND := 0.0

	if latest != nil {
		expiring30d = latest.PointsExpiring30d
		valVND = latest.PointsLiabilityValueVND
	} else {
		stats, err := uc.brandReader.GetBrandAnalyticsStats(ctx, brandID, time.Now())
		if err != nil {
			return nil, err
		}
		if stats != nil {
			expiring30d = stats.PointsExpiring30d
			valVND = stats.PointsLiabilityValueVND
		}
	}

	res := &dto.BrandPointsLiabilityRes{
		BrandID:                 brandID,
		PointsExpiring30d:       expiring30d,
		PointsLiabilityValueVND: valVND,
	}

	return mapper.RedactPointsLiabilityForRole(res, role), nil
}

func (uc *BrandDashboardUseCase) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) (*dto.CustomerAcquisitionRes, error) {
	raw, err := uc.dashboardRepo.GetCustomerAcquisition(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalCustomers := 0
	sources := make([]dto.CustomerAcquisitionDTO, 0, len(raw))
	for _, r := range raw {
		source, _ := r["source"].(string)
		customerCount, _ := r["customer_count"].(int64)
		totalCustomers += int(customerCount)
		sources = append(sources, dto.CustomerAcquisitionDTO{
			Source:        source,
			CustomerCount: int(customerCount),
			Percentage:    0,
		})
	}

	for i := range sources {
		if totalCustomers > 0 {
			sources[i].Percentage = float64(sources[i].CustomerCount) / float64(totalCustomers) * 100
		}
	}

	return &dto.CustomerAcquisitionRes{
		BrandID:        brandID,
		TotalCustomers: totalCustomers,
		Sources:        sources,
	}, nil
}

func (uc *BrandDashboardUseCase) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) (*dto.TierDistributionRes, error) {
	raw, err := uc.dashboardRepo.GetLoyaltyTierDistribution(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalMembers := 0
	tiers := make([]dto.TierDistributionDTO, 0, len(raw))
	for _, r := range raw {
		tierIDStr, _ := r["tier_id"].(string)
		tierName, _ := r["tier_name"].(string)
		memberCount, _ := r["member_count"].(int64)
		totalMembers += int(memberCount)

		tierID, _ := uuid.Parse(tierIDStr)
		tierRank := int(numberutils.ToFloat64(r["tier_rank"]))
		tiers = append(tiers, dto.TierDistributionDTO{
			TierID:      tierID,
			TierName:    tierName,
			TierRank:    tierRank,
			MemberCount: int(memberCount),
			Percentage:  0,
		})
	}

	for i := range tiers {
		if totalMembers > 0 {
			tiers[i].Percentage = float64(tiers[i].MemberCount) / float64(totalMembers) * 100
		}
	}

	return &dto.TierDistributionRes{
		BrandID:      brandID,
		TotalMembers: totalMembers,
		Tiers:        tiers,
	}, nil
}

func (uc *BrandDashboardUseCase) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) (*dto.PointExpiryForecastRes, error) {
	raw, err := uc.dashboardRepo.GetPointExpiryForecast(ctx, brandID)
	if err != nil {
		return nil, err
	}

	forecast := make([]dto.PointExpiryForecastDTO, 0, len(raw))
	for _, r := range raw {
		month, _ := r["month"].(time.Time)
		points := int(numberutils.ToFloat64(r["points_expiring"]))
		forecast = append(forecast, dto.PointExpiryForecastDTO{
			Month:          month.Format("2006-01"),
			PointsExpiring: points,
		})
	}

	return &dto.PointExpiryForecastRes{
		BrandID:  brandID,
		Forecast: forecast,
	}, nil
}

func (uc *BrandDashboardUseCase) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) (*dto.BenefitRedemptionAnalyticsRes, error) {
	raw, err := uc.dashboardRepo.GetBenefitRedemptionAnalytics(ctx, brandID)
	if err != nil {
		return nil, err
	}

	benefitMap := map[string]*dto.BenefitRedemptionDTO{}
	trendMap := map[string]int{}

	for _, r := range raw {
		benefitIDStr, _ := r["benefit_id"].(string)
		benefitName, _ := r["benefit_name"].(string)
		benefitType, _ := r["benefit_type"].(string)
		redemptionCount, _ := r["redemption_count"].(int64)
		month, _ := r["month"].(time.Time)

		monthKey := month.Format("2006-01")
		trendMap[monthKey] += int(redemptionCount)

		if _, ok := benefitMap[benefitIDStr]; !ok {
			benefitID, _ := uuid.Parse(benefitIDStr)
			benefitMap[benefitIDStr] = &dto.BenefitRedemptionDTO{
				BenefitID:       benefitID,
				BenefitName:     benefitName,
				BenefitType:     benefitType,
				RedemptionCount: 0,
			}
		}
		benefitMap[benefitIDStr].RedemptionCount += int(redemptionCount)
	}

	topBenefits := make([]dto.BenefitRedemptionDTO, 0, len(benefitMap))
	totalRedemptions := 0
	for _, b := range benefitMap {
		totalRedemptions += b.RedemptionCount
		topBenefits = append(topBenefits, *b)
	}

	// Sort by redemption count desc
	sort.Slice(topBenefits, func(i, j int) bool {
		return topBenefits[i].RedemptionCount > topBenefits[j].RedemptionCount
	})
	if len(topBenefits) > 10 {
		topBenefits = topBenefits[:10]
	}

	for i := range topBenefits {
		if totalRedemptions > 0 {
			topBenefits[i].Percentage = float64(topBenefits[i].RedemptionCount) / float64(totalRedemptions) * 100
		}
	}

	trend := make([]dto.RedemptionTrendDTO, 0, len(trendMap))
	for month, count := range trendMap {
		trend = append(trend, dto.RedemptionTrendDTO{
			Month:           month,
			RedemptionCount: count,
		})
	}
	sort.Slice(trend, func(i, j int) bool {
		return trend[i].Month < trend[j].Month
	})

	return &dto.BenefitRedemptionAnalyticsRes{
		BrandID:          brandID,
		TotalRedemptions: totalRedemptions,
		Trend:            trend,
		TopBenefits:      topBenefits,
	}, nil
}

func (uc *BrandDashboardUseCase) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (*dto.CatalogStatsRes, error) {
	raw, err := uc.dashboardRepo.GetCatalogStats(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalCount, _ := raw["total_count"].(int64)

	calcPercent := func(items []map[string]interface{}) []dto.CatalogStatGroupDTO {
		result := make([]dto.CatalogStatGroupDTO, 0, len(items))
		for _, item := range items {
			label, _ := item["label"].(string)
			countVal, _ := item["count"].(int64)
			pct := 0.0
			if totalCount > 0 {
				pct = float64(countVal) / float64(totalCount) * 100
			}
			result = append(result, dto.CatalogStatGroupDTO{
				Label:      label,
				Count:      int(countVal),
				Percentage: pct,
			})
		}
		return result
	}

	byStatusRaw, _ := raw["by_status"].([]map[string]interface{})
	byTypeRaw, _ := raw["by_type"].([]map[string]interface{})

	return &dto.CatalogStatsRes{
		BrandID:    brandID,
		TotalItems: int(totalCount),
		ByStatus:   calcPercent(byStatusRaw),
		ByType:     calcPercent(byTypeRaw),
	}, nil
}

func (uc *BrandDashboardUseCase) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) (*dto.SpendSegmentationRes, error) {
	raw, err := uc.dashboardRepo.GetCustomerSpendSegmentation(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalCustomers := 0
	segments := make([]dto.SpendSegmentDTO, 0, 4)
	for _, r := range raw {
		segment, _ := r["segment"].(string)
		customerCount, _ := r["customer_count"].(int64)
		totalCustomers += int(customerCount)
		segments = append(segments, dto.SpendSegmentDTO{
			Segment:       segment,
			CustomerCount: int(customerCount),
		})
	}

	segmentRanges := map[string]struct{ min, max float64 }{
		"low":     {0, 1000000},
		"medium":  {1000000, 5000000},
		"high":    {5000000, 10000000},
		"premium": {10000000, 0},
	}

	for i := range segments {
		if r, ok := segmentRanges[segments[i].Segment]; ok {
			segments[i].MinVND = r.min
			if r.max > 0 {
				segments[i].MaxVND = &r.max
			}
		}
		if totalCustomers > 0 {
			segments[i].Percentage = float64(segments[i].CustomerCount) / float64(totalCustomers) * 100
		}
	}

	return &dto.SpendSegmentationRes{
		BrandID:        brandID,
		TotalCustomers: totalCustomers,
		Segments:       segments,
	}, nil
}

func (uc *BrandDashboardUseCase) GetDigitalSampleLabAnalytics(ctx context.Context, brandID uuid.UUID, itemID uuid.UUID) (*dto.DigitalSampleLabAnalyticsRes, error) {
	// Lazy load: đọc feedback summary từ brand reader
	feedback, err := uc.brandReader.GetItemFeedbackSummary(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if feedback == nil {
		return &dto.DigitalSampleLabAnalyticsRes{
			ItemID:              itemID,
			VotesCount:          0,
			AvgRating:           0,
			FeedbackSentiment:   feedbacksentiment.Neutral,
			TopPairedCategories: []dto.PairedCategoryStat{},
		}, nil
	}

	// Lazy load: paired categories từ outfit → fashion item → category chain
	pairedCategories := make([]dto.PairedCategoryStat, 0)
	if len(feedback.OutfitIDs) > 0 {
		fashionItemIDs, err := uc.wardrobeReader.GetOutfitItemFashionItemIDs(ctx, feedback.OutfitIDs)
		if err != nil {
			return nil, err
		}
		if len(fashionItemIDs) > 0 {
			fashionItems, err := uc.fashionContract.ListFashionItems(ctx, fashionItemIDs)
			if err != nil {
				return nil, err
			}
			categoryCounts := map[string]int{}
			for _, fi := range fashionItems {
				if fi.Category != nil {
					categoryCounts[fi.Category.Name]++
				}
			}
			for name, count := range categoryCounts {
				pairedCategories = append(pairedCategories, dto.PairedCategoryStat{
					CategoryName: name,
					PairCount:    count,
				})
			}
			sort.Slice(pairedCategories, func(i, j int) bool {
				return pairedCategories[i].PairCount > pairedCategories[j].PairCount
			})
			if len(pairedCategories) > 5 {
				pairedCategories = pairedCategories[:5]
			}
		}
	}

	return &dto.DigitalSampleLabAnalyticsRes{
		ItemID:              itemID,
		VotesCount:          feedback.VotesCount,
		AvgRating:           feedback.AvgRating,
		FeedbackSentiment:   feedbacksentiment.FeedbackSentiment(feedback.FeedbackSentiment),
		TopPairedCategories: pairedCategories,
	}, nil
}
