package loyalty

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

func (uc *BrandLoyaltyUseCase) ListUserBrandLoyalties(ctx context.Context, userID uuid.UUID) ([]*dto.BrandLoyaltyRes, error) {
	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	activeCustomerIDs := make([]uuid.UUID, 0, len(customers))
	activeCustomers := make([]*entities.BrandCustomer, 0, len(customers))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active {
			continue
		}
		activeCustomerIDs = append(activeCustomerIDs, customer.ID)
		activeCustomers = append(activeCustomers, customer)
	}
	accounts, err := uc.accountRepo.GetByBrandCustomerIDs(ctx, activeCustomerIDs)
	if err != nil {
		return nil, err
	}
	accountByCustomerID := make(map[uuid.UUID]*entities.LoyaltyAccount, len(accounts))
	for _, account := range accounts {
		accountByCustomerID[account.BrandCustomerID] = account
	}

	accountIDs := make([]uuid.UUID, 0, len(accounts))
	for _, account := range accounts {
		accountIDs = append(accountIDs, account.ID)
	}
	lotMap, err := uc.lotRepo.GetNearestExpiringActiveLots(ctx, accountIDs, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	res := make([]*dto.BrandLoyaltyRes, 0, len(customers))
	for _, customer := range activeCustomers {
		account := accountByCustomerID[customer.ID]
		if account == nil {
			continue
		}
		res = append(res, mapper.MapBrandLoyalty(account, customer, customer.Brand, lotMap[account.ID]))
	}
	return res, nil
}

func (uc *BrandLoyaltyUseCase) GetUserBrandLoyalty(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandLoyaltyRes, error) {
	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrCustomerNotFound()
	}
	account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, branderrors.ErrActiveLoyaltyProgramRequired()
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	lot, err := uc.lotRepo.GetNearestExpiringActiveLot(ctx, account.ID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandLoyalty(account, customer, brand, lot), nil
}

func (uc *BrandLoyaltyUseCase) GetUserBrandLoyaltyTransactions(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyPointTransactionBriefRes, error) {
	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrCustomerNotFound()
	}
	account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return []*dto.LoyaltyPointTransactionBriefRes{}, nil
	}
	transactions, err := uc.txRepo.GetByLoyaltyAccountID(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyPointTransactionsBrief(transactions), nil
}

func (uc *BrandLoyaltyUseCase) GetUserBrandLoyaltyLots(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error) {
	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrCustomerNotFound()
	}
	account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return []*dto.LoyaltyPointLotRes{}, nil
	}
	return uc.listLoyaltyLots(ctx, account.ID, query)
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyAccountTransactionsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID, query dto.GetLoyaltyTransactionsQueryReq) (*dto.LoyaltyTransactionListRes, error) {
	account, err := uc.accountRepo.GetByID(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.BrandID != brandID {
		return nil, apperror.NewNotFound("Không tìm thấy tài khoản tích điểm này")
	}

	filter := repositories.LoyaltyTransactionFilter{
		LoyaltyAccountID: account.ID,
		Page:             query.Page,
		Limit:            query.Limit,
	}

	result, err := uc.txRepo.GetByLoyaltyAccountIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &dto.LoyaltyTransactionListRes{
		Items:    mapper.MapLoyaltyTransactions(result.Transactions),
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyAccountLotsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID, query dto.GetLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error) {
	account, err := uc.accountRepo.GetByID(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	return uc.listLoyaltyLots(ctx, loyaltyAccountID, query)
}
