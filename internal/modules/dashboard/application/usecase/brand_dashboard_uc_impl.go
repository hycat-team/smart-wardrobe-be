package usecase

import (
	"context"
	"encoding/json"
	"time"

	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/application/mapper"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/feedbacksentiment"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"

	"github.com/google/uuid"
)

type BrandDashboardUseCase struct {
	dashboardRepo dashboardRepos.IDashboardRepository
	brandReader   brandContract.IBrandDashboardReader
}

func NewBrandDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	brandReader brandContract.IBrandDashboardReader,
) uc_interfaces.IBrandDashboardUseCase {
	return &BrandDashboardUseCase{
		dashboardRepo: dashboardRepo,
		brandReader:   brandReader,
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
		return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
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
		if stats == nil {
			return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
		}
		expiring30d = stats.PointsExpiring30d
		valVND = stats.PointsLiabilityValueVND
	}

	res := &dto.BrandPointsLiabilityRes{
		BrandID:                 brandID,
		PointsExpiring30d:       expiring30d,
		PointsLiabilityValueVND: valVND,
	}

	return mapper.RedactPointsLiabilityForRole(res, role), nil
}

func (uc *BrandDashboardUseCase) GetDigitalSampleLabAnalytics(ctx context.Context, brandID uuid.UUID, itemID uuid.UUID) (*dto.DigitalSampleLabAnalyticsRes, error) {
	itemSnapshot, err := uc.dashboardRepo.GetLatestBrandItemSnapshot(ctx, brandID, itemID)
	if err != nil {
		return nil, err
	}

	if itemSnapshot != nil {
		votes := itemSnapshot.VotesCount
		rating := itemSnapshot.AvgRating
		sentiment := feedbacksentiment.FeedbackSentiment(itemSnapshot.FeedbackSentiment)
		pairedCategories := make([]dto.PairedCategoryStat, 0)
		if len(itemSnapshot.TopPairedCategories) > 0 {
			_ = json.Unmarshal([]byte(itemSnapshot.TopPairedCategories), &pairedCategories)
		}
		return &dto.DigitalSampleLabAnalyticsRes{
			ItemID:              itemID,
			VotesCount:          votes,
			AvgRating:           rating,
			FeedbackSentiment:   sentiment,
			TopPairedCategories: pairedCategories,
		}, nil
	}

	// Fallback to brand reader stats if available
	stats, err := uc.brandReader.GetBrandAnalyticsStats(ctx, brandID, time.Now())
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, dashboarderrors.ErrBrandAnalyticsNotFound()
	}

	return &dto.DigitalSampleLabAnalyticsRes{
		ItemID:     itemID,
		VotesCount: stats.DigitalSampleVotesCount,
		AvgRating:  stats.DigitalSampleAvgRating,
	}, nil
}
