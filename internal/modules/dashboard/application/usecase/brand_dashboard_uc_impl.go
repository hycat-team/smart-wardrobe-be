package usecase

import (
	"context"
	"sort"
	"time"

	brand_dto "smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/application/mapper"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/feedbacksentiment"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	fashionContract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	wardrobeContract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

	"github.com/google/uuid"
)

type BrandDashboardUseCase struct {
	dashboardRepo   dashboardRepos.IDashboardRepository
	brandReader     brandContract.IBrandDashboardReader
	wardrobeReader  wardrobeContract.IWardrobeDashboardReader
	fashionContract fashionContract.IFashionContract
	brandItem       brandContract.IBrandContract
}

func NewBrandDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	brandReader brandContract.IBrandDashboardReader,
	wardrobeReader wardrobeContract.IWardrobeDashboardReader,
	fashionContract fashionContract.IFashionContract,
	brandItem brandContract.IBrandContract,
) uc_interfaces.IBrandDashboardUseCase {
	return &BrandDashboardUseCase{
		dashboardRepo:   dashboardRepo,
		brandReader:     brandReader,
		wardrobeReader:  wardrobeReader,
		fashionContract: fashionContract,
		brandItem:       brandItem,
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
				TotalMembers:            stats.TotalMembers,
				PointsIssued:            stats.PointsIssued,
				PointsRedeemed:          stats.PointsRedeemed,
				ActiveVouchers:          stats.ActiveVouchers,
				VouchersRedeemed:        stats.VouchersRedeemed,
				DigitalSampleVotesCount: stats.DigitalSampleVotesCount,
				DigitalSampleAvgRating:  stats.DigitalSampleAvgRating,
				OpenCSTickets:           stats.OpenCSTickets,
			}, nil
		}
		return &dto.BrandKPICardsRes{}, nil
	}

	return &dto.BrandKPICardsRes{
		TotalMembers:            latest.TotalMembers,
		PointsIssued:            latest.PointsIssued,
		PointsRedeemed:          latest.PointsRedeemed,
		ActiveVouchers:          latest.ActiveVouchers,
		VouchersRedeemed:        latest.VouchersRedeemed,
		DigitalSampleVotesCount: latest.DigitalSampleVotesCount,
		DigitalSampleAvgRating:  latest.DigitalSampleAvgRating,
		OpenCSTickets:           latest.OpenCSTickets,
	}, nil
}

func (uc *BrandDashboardUseCase) GetPointsLiability(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, role brandmemberrole.BrandMemberRole) (*dto.BrandPointsLiabilityRes, error) {
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
	rows, err := uc.brandReader.GetCustomerAcquisition(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalCustomers := 0
	sources := make([]dto.CustomerAcquisitionDTO, 0, len(rows))
	for _, r := range rows {
		totalCustomers += int(r.CustomerCount)
		sources = append(sources, dto.CustomerAcquisitionDTO{
			Source:        r.Source,
			CustomerCount: int(r.CustomerCount),
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
	rows, err := uc.brandReader.GetLoyaltyTierDistribution(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalMembers := 0
	tiers := make([]dto.TierDistributionDTO, 0, len(rows))
	for _, r := range rows {
		totalMembers += int(r.MemberCount)
		tiers = append(tiers, dto.TierDistributionDTO{
			TierID:      r.TierID,
			TierName:    r.TierName,
			TierRank:    r.TierRank,
			MemberCount: int(r.MemberCount),
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
	rows, err := uc.brandReader.GetPointExpiryForecast(ctx, brandID)
	if err != nil {
		return nil, err
	}

	forecast := make([]dto.PointExpiryForecastDTO, 0, len(rows))
	for _, r := range rows {
		forecast = append(forecast, dto.PointExpiryForecastDTO{
			Month:          r.Month.Format("2006-01"),
			PointsExpiring: int(r.PointsExpiring),
		})
	}

	return &dto.PointExpiryForecastRes{
		BrandID:  brandID,
		Forecast: forecast,
	}, nil
}

func (uc *BrandDashboardUseCase) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) (*dto.BenefitRedemptionAnalyticsRes, error) {
	rows, err := uc.brandReader.GetBenefitRedemptionAnalytics(ctx, brandID)
	if err != nil {
		return nil, err
	}

	benefitMap := map[uuid.UUID]*dto.BenefitRedemptionDTO{}
	trendMap := map[string]int{}

	for _, r := range rows {
		monthKey := r.Month.Format("2006-01")
		trendMap[monthKey] += int(r.RedemptionCount)

		if _, ok := benefitMap[r.BenefitID]; !ok {
			benefitMap[r.BenefitID] = &dto.BenefitRedemptionDTO{
				BenefitID:       r.BenefitID,
				BenefitName:     r.BenefitName,
				BenefitType:     r.BenefitType,
				RedemptionCount: 0,
			}
		}
		benefitMap[r.BenefitID].RedemptionCount += int(r.RedemptionCount)
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
	stats, err := uc.brandReader.GetCatalogStats(ctx, brandID)
	if err != nil {
		return nil, err
	}

	calcPercent := func(items []brand_dto.CatalogStatGroupDTO) []dto.CatalogStatGroupDTO {
		result := make([]dto.CatalogStatGroupDTO, 0, len(items))
		for _, item := range items {
			pct := 0.0
			if stats.TotalCount > 0 {
				pct = float64(item.Count) / float64(stats.TotalCount) * 100
			}
			result = append(result, dto.CatalogStatGroupDTO{
				Label:      item.Label,
				Count:      int(item.Count),
				Percentage: pct,
			})
		}
		return result
	}

	return &dto.CatalogStatsRes{
		BrandID:    brandID,
		TotalItems: int(stats.TotalCount),
		ByStatus:   calcPercent(stats.ByStatus),
		ByType:     calcPercent(stats.ByType),
	}, nil
}

func (uc *BrandDashboardUseCase) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) (*dto.SpendSegmentationRes, error) {
	rows, err := uc.brandReader.GetCustomerSpendSegmentation(ctx, brandID)
	if err != nil {
		return nil, err
	}

	totalCustomers := 0
	segments := make([]dto.SpendSegmentDTO, 0, 4)
	for _, r := range rows {
		totalCustomers += int(r.CustomerCount)
		segments = append(segments, dto.SpendSegmentDTO{
			Segment:       r.Segment,
			CustomerCount: int(r.CustomerCount),
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

	var brandItemName *string
	var brandItemImageURL *string
	brandItemID := &itemID
	if uc.brandItem != nil {
		if bi, err := uc.brandItem.GetBrandItemByID(ctx, itemID); err == nil && bi != nil {
			name := bi.Name
			brandItemName = &name
		}
	}

	return &dto.DigitalSampleLabAnalyticsRes{
		ItemID:              itemID,
		BrandItemID:         brandItemID,
		BrandItemName:       brandItemName,
		BrandItemImageURL:   brandItemImageURL,
		VotesCount:          feedback.VotesCount,
		AvgRating:           feedback.AvgRating,
		FeedbackSentiment:   feedbacksentiment.FeedbackSentiment(feedback.FeedbackSentiment),
		TopPairedCategories: pairedCategories,
	}, nil
}
