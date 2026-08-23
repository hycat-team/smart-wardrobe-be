package loyalty

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/stringutils"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BrandLoyaltyUseCase struct {
	brandRepo    repositories.IBrandRepository
	memberRepo   repositories.IBrandMemberRepository
	customerRepo repositories.IBrandCustomerRepository
	userContract identity_contract.IUserContract
	programRepo  repositories.ILoyaltyProgramRepository
	tierRepo     repositories.ILoyaltyTierRepository
	accountRepo  repositories.ILoyaltyAccountRepository
	txRepo       repositories.ILoyaltyPointTransactionRepository
	lotRepo      repositories.ILoyaltyPointLotRepository
	benefitRepo  repositories.IBrandBenefitRepository
	uow          shared_repos.IUnitOfWork
	cfg          *config.Config
	logger       logger.Interface
}

func NewBrandLoyaltyUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	userContract identity_contract.IUserContract,
	programRepo repositories.ILoyaltyProgramRepository,
	tierRepo repositories.ILoyaltyTierRepository,
	accountRepo repositories.ILoyaltyAccountRepository,
	txRepo repositories.ILoyaltyPointTransactionRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	benefitRepo repositories.IBrandBenefitRepository,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
	l logger.Interface,
) uc_interfaces.IBrandLoyaltyUseCase {
	return &BrandLoyaltyUseCase{
		brandRepo:    brandRepo,
		memberRepo:   memberRepo,
		customerRepo: customerRepo,
		userContract: userContract,
		programRepo:  programRepo,
		tierRepo:     tierRepo,
		accountRepo:  accountRepo,
		txRepo:       txRepo,
		lotRepo:      lotRepo,
		benefitRepo:  benefitRepo,
		uow:          uow,
		cfg:          cfg,
		logger:       l,
	}
}

func (uc *BrandLoyaltyUseCase) GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandCustomersQueryReq) (*dto.BrandCustomerListRes, error) {
	var statusFilter *brandcustomerstatus.BrandCustomerStatus
	if query.Status != nil {
		val := brandcustomerstatus.BrandCustomerStatus(*query.Status)
		statusFilter = &val
	}

	filter := repositories.BrandCustomerFilter{
		BrandID: brandID,
		Status:  statusFilter,
		Query:   query.Query,
		Page:    query.Page,
		Limit:   query.Limit,
	}

	result, err := uc.customerRepo.GetByBrandIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}

	userMap := make(map[uuid.UUID]*identity_dto.UserRes)
	if len(result.Customers) > 0 {
		userIDs := make([]uuid.UUID, 0, len(result.Customers))
		for _, c := range result.Customers {
			if c.UserID != nil {
				userIDs = append(userIDs, *c.UserID)
			}
		}
		if len(userIDs) > 0 {
			users, err := uc.userContract.GetByIDs(ctx, userIDs)
			if err == nil {
				for _, u := range users {
					userMap[u.ID] = u
				}
			}
		}
	}

	return &dto.BrandCustomerListRes{
		Items:    mapper.MapBrandCustomersWithUsers(result.Customers, userMap),
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetBrandCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.BrandCustomerRes, error) {
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	var user *identity_dto.UserRes
	if customer.UserID != nil {
		u, err := uc.userContract.GetByID(ctx, *customer.UserID)
		if err == nil {
			user = u
		}
	}
	return mapper.MapBrandCustomer(customer, user), nil
}

func (uc *BrandLoyaltyUseCase) JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerBriefRes, error) {
	if currentRole != roleslug.User {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}
	if existing, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID); err != nil {
		return nil, err
	} else if existing != nil {
		if err := uc.ensureLoyaltyAccount(ctx, existing); err != nil {
			return nil, err
		}
		return mapper.MapBrandCustomerBrief(existing, nil), nil
	}

	customer := &entities.BrandCustomer{
		BrandID:      brandID,
		UserID:       &userID,
		JoinedSource: brandcustomerjoinedsource.SelfJoin,
		Status:       brandcustomerstatus.Active,
		JoinedAt:     time.Now().UTC(),
	}
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.customerRepo.Create(txCtx, customer); err != nil {
			return err
		}
		return uc.ensureLoyaltyAccount(txCtx, customer)
	}); err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomerBrief(customer, nil), nil
}

func (uc *BrandLoyaltyUseCase) CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error) {
	if strings.TrimSpace(input.PhoneE164) == "" {
		return nil, branderrors.ErrPhoneRequired()
	}
	creator, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	phone := strings.TrimSpace(input.PhoneE164)
	phoneHash := stringutils.HashSHA256(phone)
	if existing, err := uc.customerRepo.GetByBrandAndPhoneHash(ctx, brandID, phoneHash); err != nil {
		return nil, err
	} else if existing != nil {
		if err := uc.ensureLoyaltyAccount(ctx, existing); err != nil {
			return nil, err
		}
		return mapper.MapBrandCustomer(existing, nil), nil
	}

	customer := &entities.BrandCustomer{
		BrandID:              brandID,
		CustomerName:         input.CustomerName,
		PhoneE164:            &phone,
		PhoneHash:            &phoneHash,
		ExternalCustomerCode: input.ExternalCustomerCode,
		JoinedSource:         brandcustomerjoinedsource.OfflinePurchase,
		Status:               brandcustomerstatus.Active,
		JoinedAt:             time.Now().UTC(),
		CreatedByMemberID:    &creator.ID,
	}
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.customerRepo.Create(txCtx, customer); err != nil {
			return err
		}
		return uc.ensureLoyaltyAccount(txCtx, customer)
	}); err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomer(customer, nil), nil
}

func (uc *BrandLoyaltyUseCase) GrantLoyaltyPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*dto.LoyaltyPointsTransactionRes, error) {
	if err := validateGrantLoyaltyPointsInput(input); err != nil {
		return nil, err
	}
	var response *dto.LoyaltyPointsTransactionRes

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if input.IdempotencyKey != nil && strings.TrimSpace(*input.IdempotencyKey) != "" {
			existingTx, err := uc.txRepo.GetByBrandAndIdempotencyKey(txCtx, brandID, strings.TrimSpace(*input.IdempotencyKey))
			if err != nil {
				return err
			}
			if existingTx != nil {
				account, err := uc.accountRepo.GetByID(txCtx, existingTx.LoyaltyAccountID)
				if err != nil {
					return err
				}
				customer, err := uc.customerRepo.GetByID(txCtx, existingTx.BrandCustomerID)
				if err != nil {
					return err
				}
				response = mapper.MapLoyaltyTransactionResponse(existingTx, account, customer)
				return nil
			}
		}

		customer, err := uc.resolveBrandCustomerForPoints(txCtx, staffUserID, brandID, input)
		if err != nil {
			return err
		}

		if err := uc.ensureLoyaltyAccount(txCtx, customer); err != nil {
			return err
		}
		account, err := uc.accountRepo.GetByBrandCustomerIDForUpdate(txCtx, customer.ID)
		if err != nil {
			return err
		}
		if account == nil {
			return branderrors.ErrActiveLoyaltyProgramRequired()
		}

		program, err := uc.programRepo.GetActiveByBrandID(txCtx, brandID)
		if err != nil {
			return err
		}

		pointsDelta, err := uc.resolvePointsDelta(input, program)
		if err != nil {
			return err
		}
		newBalance := account.CurrentPoints + pointsDelta
		if newBalance < 0 {
			return branderrors.ErrInsufficientLoyaltyPoints()
		}

		spendAmount := input.PurchaseAmount
		if input.PurchaseAmount != nil && input.TransactionType == loyaltytransactiontype.Earn {
			account.TotalSpend += *input.PurchaseAmount
		}
		if input.TransactionType == loyaltytransactiontype.Earn && pointsDelta > 0 {
			account.LifetimePoints += pointsDelta
		}
		account.CurrentPoints = newBalance
		account.UserID = customer.UserID

		tier, err := uc.tierRepo.GetHighestEligibleBySpend(txCtx, brandID, account.TotalSpend)
		if err != nil {
			return err
		}
		if tier != nil {
			account.CurrentTierID = &tier.ID
			account.CurrentTier = tier
		} else {
			account.CurrentTierID = nil
			account.CurrentTier = nil
		}
		if err := uc.accountRepo.Update(txCtx, account); err != nil {
			return err
		}

		if pointsDelta == 0 {
			response = mapper.MapLoyaltyTransactionResponse(nil, account, customer)
			return nil
		}

		expiresAt := calculateTransactionExpiry(input.TransactionType, program)
		idempotencyKey := stringutils.TrimmedPtr(input.IdempotencyKey)
		tx := &entities.LoyaltyPointTransaction{
			LoyaltyAccountID: account.ID,
			BrandID:          brandID,
			BrandCustomerID:  customer.ID,
			UserID:           customer.UserID,
			PointsDelta:      pointsDelta,
			BalanceAfter:     newBalance,
			TransactionType:  input.TransactionType,
			Reason:           input.Reason,
			SpendAmount:      spendAmount,
			ReferenceType:    input.ReferenceType,
			ReferenceID:      input.ReferenceID,
			ExpiresAt:        expiresAt,
			IdempotencyKey:   idempotencyKey,
			CreatedByUserID:  &staffUserID,
		}
		if err := uc.txRepo.Create(txCtx, tx); err != nil {
			return err
		}
		if input.TransactionType == loyaltytransactiontype.Earn && pointsDelta > 0 {
			lot := &entities.LoyaltyPointLot{
				LoyaltyAccountID:  account.ID,
				BrandID:           brandID,
				BrandCustomerID:   customer.ID,
				UserID:            customer.UserID,
				EarnTransactionID: tx.ID,
				EarnedPoints:      pointsDelta,
				RemainingPoints:   pointsDelta,
				ExpiresAt:         expiresAt,
				Status:            loyaltypointlotstatus.Active,
			}
			if err := uc.lotRepo.Create(txCtx, lot); err != nil {
				return err
			}
		}
		response = mapper.MapLoyaltyTransactionResponse(tx, account, customer)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

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

func (uc *BrandLoyaltyUseCase) GetUserBrandLoyaltyLots(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.ListLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error) {
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

func (uc *BrandLoyaltyUseCase) GetLoyaltyAccountLotsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID, query dto.ListLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error) {
	account, err := uc.accountRepo.GetByID(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	return uc.listLoyaltyLots(ctx, loyaltyAccountID, query)
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyProgramForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) (*dto.LoyaltyProgramRes, error) {
	program, err := uc.programRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if program == nil {
		return nil, branderrors.ErrActiveLoyaltyProgramRequired()
	}
	return mapper.MapLoyaltyProgram(program), nil
}

func (uc *BrandLoyaltyUseCase) UpsertLoyaltyProgram(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.UpsertLoyaltyProgramReq) (*dto.LoyaltyProgramRes, error) {
	program, err := uc.programRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	if program == nil {
		program = &entities.LoyaltyProgram{
			BrandID:         brandID,
			Name:            input.Name,
			AmountPerPoint:  input.AmountPerPoint,
			PointExpiryDays: input.PointExpiryDays,
			RoundingMode:    input.RoundingMode,
			IsActive:        true,
		}
		if input.IsActive != nil {
			program.IsActive = *input.IsActive
		}
		if err := uc.programRepo.Create(ctx, program); err != nil {
			return nil, err
		}
	} else {
		program.Name = input.Name
		program.AmountPerPoint = input.AmountPerPoint
		program.PointExpiryDays = input.PointExpiryDays
		program.RoundingMode = input.RoundingMode
		if input.IsActive != nil {
			program.IsActive = *input.IsActive
		}
		if err := uc.programRepo.Update(ctx, program); err != nil {
			return nil, err
		}
	}

	return mapper.MapLoyaltyProgram(program), nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTiersForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyTierRes, error) {
	tiers, err := uc.tierRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTiers(tiers), nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTierDetailForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, tierID uuid.UUID) (*dto.LoyaltyTierDetailRes, error) {
	tier, err := uc.tierRepo.GetByID(ctx, tierID)
	if err != nil {
		return nil, err
	}
	if tier == nil || tier.BrandID != brandID {
		return nil, branderrors.ErrTierNotFound()
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandIDAndTierID(ctx, brandID, tierID)
	if err != nil {
		return nil, err
	}

	return &dto.LoyaltyTierDetailRes{
		LoyaltyTierRes: *mapper.MapLoyaltyTier(tier),
		Benefits:       mapper.MapBrandBenefits(benefits),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTierDetailForUser(ctx context.Context, brandID uuid.UUID, tierID uuid.UUID) (*dto.LoyaltyTierDetailRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	tier, err := uc.tierRepo.GetByID(ctx, tierID)
	if err != nil {
		return nil, err
	}
	if tier == nil || tier.BrandID != brandID {
		return nil, branderrors.ErrTierNotFound()
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandIDAndTierID(ctx, brandID, tierID)
	if err != nil {
		return nil, err
	}

	return &dto.LoyaltyTierDetailRes{
		LoyaltyTierRes: *mapper.MapLoyaltyTier(tier),
		Benefits:       mapper.MapBrandBenefits(benefits),
	}, nil
}

func (uc *BrandLoyaltyUseCase) CreateLoyaltyTier(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateLoyaltyTierReq) (*dto.LoyaltyTierRes, error) {
	existing, err := uc.tierRepo.GetByBrandAndName(ctx, brandID, input.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrTierNameExists(input.Name)
	}

	existing, err = uc.tierRepo.GetByBrandAndRank(ctx, brandID, input.Rank)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrTierRankExists(input.Rank)
	}

	tier := &entities.LoyaltyTier{
		BrandID:       brandID,
		Name:          input.Name,
		Rank:          input.Rank,
		MinTotalSpend: input.MinTotalSpend,
		Description:   input.Description,
	}
	if err := uc.tierRepo.Create(ctx, tier); err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTier(tier), nil
}

func (uc *BrandLoyaltyUseCase) UpdateLoyaltyTier(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, tierID uuid.UUID, input dto.UpdateLoyaltyTierReq) (*dto.LoyaltyTierRes, error) {
	tier, err := uc.tierRepo.GetByID(ctx, tierID)
	if err != nil {
		return nil, err
	}
	if tier == nil || tier.BrandID != brandID {
		return nil, branderrors.ErrTierNotFound()
	}

	if input.Name != nil {
		existing, err := uc.tierRepo.GetByBrandAndName(ctx, brandID, *input.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != tierID {
			return nil, branderrors.ErrTierNameExists(*input.Name)
		}
		tier.Name = *input.Name
	}

	if input.Rank != nil {
		existing, err := uc.tierRepo.GetByBrandAndRank(ctx, brandID, *input.Rank)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != tierID {
			return nil, branderrors.ErrTierRankExists(*input.Rank)
		}
		tier.Rank = *input.Rank
	}

	if input.MinTotalSpend != nil {
		tier.MinTotalSpend = *input.MinTotalSpend
	}

	if input.Description != nil {
		tier.Description = input.Description
	}

	if err := uc.tierRepo.Update(ctx, tier); err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTier(tier), nil
}

func (uc *BrandLoyaltyUseCase) ProcessExpiredLoyaltyPointLots(ctx context.Context, run *workerlog.Run, now time.Time, batchSize int) (int, error) {
	accountIDs, err := uc.lotRepo.ListAccountsWithExpiredLots(ctx, now, batchSize)
	if err != nil {
		run.ChildError(uc.logger, "Failed to list accounts with expired lots", zap.Error(err))
		return 0, err
	}
	run.AddTotal(len(accountIDs))
	totalExpired := 0
	for _, accountID := range accountIDs {
		var expired int
		if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
			var err error
			expired, err = uc.expireDueLotsForAccount(txCtx, accountID, now, nil)
			return err
		}); err != nil {
			run.ChildError(uc.logger, "Failed to expire lots for account", zap.String("accountID", accountID.String()), zap.Error(err))
			return totalExpired, err
		}
		totalExpired += expired
	}
	run.AddSuccess(totalExpired)
	return totalExpired, nil
}
