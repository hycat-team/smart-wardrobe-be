package mapper

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func MapLoyaltyTransactionResponse(tx *entities.LoyaltyPointTransaction, account *entities.LoyaltyAccount, customer *entities.BrandCustomer) *dto.LoyaltyPointsTransactionRes {
	if account == nil || customer == nil {
		return nil
	}
	var transactionID uuid.UUID
	var pointsDelta int
	var balanceAfter = account.CurrentPoints
	if tx != nil {
		transactionID = tx.ID
		pointsDelta = tx.PointsDelta
		balanceAfter = tx.BalanceAfter
	}
	return &dto.LoyaltyPointsTransactionRes{
		TransactionID:   transactionID,
		BrandID:         account.BrandID,
		BrandCustomerID: customer.ID,
		UserID:          customer.UserID,
		CustomerStatus:  customer.Status,
		PointsDelta:     pointsDelta,
		BalanceAfter:    balanceAfter,
		TotalSpend:      account.TotalSpend,
		CurrentTier:     MapLoyaltyTierBrief(account.CurrentTier),
	}
}

func MapBrandLoyalty(account *entities.LoyaltyAccount, customer *entities.BrandCustomer, brand *entities.Brand, lot *entities.LoyaltyPointLot) *dto.BrandLoyaltyRes {
	if account == nil || customer == nil {
		return nil
	}
	return &dto.BrandLoyaltyRes{
		BrandID:                 account.BrandID,
		Brand:                   MapBrand(brand, -1),
		BrandCustomerID:         customer.ID,
		LoyaltyAccountID:        account.ID,
		CurrentPoints:           account.CurrentPoints,
		LifetimePoints:          account.LifetimePoints,
		TotalSpend:              account.TotalSpend,
		CurrentTier:             MapLoyaltyTierBrief(account.CurrentTier),
		NearestExpiringPointLot: MapLoyaltyPointLot(lot),
	}
}

func MapLoyaltyTierBrief(tier *entities.LoyaltyTier) *dto.LoyaltyTierBriefRes {
	if tier == nil {
		return nil
	}
	return &dto.LoyaltyTierBriefRes{
		ID:   tier.ID,
		Name: tier.Name,
	}
}

func MapLoyaltyPointLot(lot *entities.LoyaltyPointLot) *dto.LoyaltyPointLotRes {
	if lot == nil {
		return nil
	}
	return &dto.LoyaltyPointLotRes{
		ID:                lot.ID,
		EarnedPoints:      lot.EarnedPoints,
		RemainingPoints:   lot.RemainingPoints,
		ExpiresAt:         lot.ExpiresAt,
		Status:            string(lot.Status),
		EarnTransactionID: lot.EarnTransactionID,
		CreatedAt:         lot.CreatedAt,
	}
}

func MapLoyaltyPointLots(lots []*entities.LoyaltyPointLot) []*dto.LoyaltyPointLotRes {
	res := make([]*dto.LoyaltyPointLotRes, 0, len(lots))
	for _, lot := range lots {
		res = append(res, MapLoyaltyPointLot(lot))
	}
	return res
}

func MapLoyaltyTransaction(tx *entities.LoyaltyPointTransaction) *dto.LoyaltyPointTransactionDetailRes {
	if tx == nil {
		return nil
	}
	return &dto.LoyaltyPointTransactionDetailRes{
		ID:               tx.ID,
		LoyaltyAccountID: tx.LoyaltyAccountID,
		BrandID:          tx.BrandID,
		BrandCustomerID:  tx.BrandCustomerID,
		UserID:           tx.UserID,
		PointsDelta:      tx.PointsDelta,
		BalanceAfter:     tx.BalanceAfter,
		TransactionType:  string(tx.TransactionType),
		Reason:           tx.Reason,
		SpendAmount:      tx.SpendAmount,
		ReferenceType:    tx.ReferenceType,
		ReferenceID:      tx.ReferenceID,
		ExpiresAt:        tx.ExpiresAt,
		IdempotencyKey:   tx.IdempotencyKey,
		CreatedByUserID:  tx.CreatedByUserID,
		CreatedAt:        tx.CreatedAt,
	}
}

func MapLoyaltyTransactions(transactions []*entities.LoyaltyPointTransaction) []*dto.LoyaltyPointTransactionDetailRes {
	res := make([]*dto.LoyaltyPointTransactionDetailRes, len(transactions))
	for i, tx := range transactions {
		res[i] = MapLoyaltyTransaction(tx)
	}
	return res
}

func MapLoyaltyProgram(program *entities.LoyaltyProgram) *dto.LoyaltyProgramRes {
	if program == nil {
		return nil
	}
	return &dto.LoyaltyProgramRes{
		ID:              program.ID,
		BrandID:         program.BrandID,
		Name:            program.Name,
		AmountPerPoint:  program.AmountPerPoint,
		PointExpiryDays: program.PointExpiryDays,
		RoundingMode:    string(program.RoundingMode),
		IsActive:        program.IsActive,
		CreatedAt:       program.CreatedAt,
		UpdatedAt:       program.UpdatedAt,
	}
}

func MapLoyaltyTier(tier *entities.LoyaltyTier) *dto.LoyaltyTierRes {
	if tier == nil {
		return nil
	}
	return &dto.LoyaltyTierRes{
		ID:            tier.ID,
		BrandID:       tier.BrandID,
		Name:          tier.Name,
		Rank:          tier.Rank,
		MinTotalSpend: tier.MinTotalSpend,
		Description:   tier.Description,
		CreatedAt:     tier.CreatedAt,
		UpdatedAt:     tier.UpdatedAt,
	}
}

func MapLoyaltyTiers(tiers []*entities.LoyaltyTier) []*dto.LoyaltyTierRes {
	res := make([]*dto.LoyaltyTierRes, len(tiers))
	for i, tier := range tiers {
		res[i] = MapLoyaltyTier(tier)
	}
	return res
}
