package usecase

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
)

type BrandLoyaltyUseCase struct {
	brandRepo    repositories.IBrandRepository
	memberRepo   repositories.IBrandMemberRepository
	customerRepo repositories.IBrandCustomerRepository
	userRepo     identity_repos.IUserRepository
	programRepo  repositories.ILoyaltyProgramRepository
	tierRepo     repositories.ILoyaltyTierRepository
	accountRepo  repositories.ILoyaltyAccountRepository
	txRepo       repositories.ILoyaltyPointTransactionRepository
	lotRepo      repositories.ILoyaltyPointLotRepository
	uow          shared_repos.IUnitOfWork
	cfg          *config.Config
}

func NewBrandLoyaltyUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	userRepo identity_repos.IUserRepository,
	programRepo repositories.ILoyaltyProgramRepository,
	tierRepo repositories.ILoyaltyTierRepository,
	accountRepo repositories.ILoyaltyAccountRepository,
	txRepo repositories.ILoyaltyPointTransactionRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.IBrandLoyaltyUseCase {
	return &BrandLoyaltyUseCase{
		brandRepo:    brandRepo,
		memberRepo:   memberRepo,
		customerRepo: customerRepo,
		userRepo:     userRepo,
		programRepo:  programRepo,
		tierRepo:     tierRepo,
		accountRepo:  accountRepo,
		txRepo:       txRepo,
		lotRepo:      lotRepo,
		uow:          uow,
		cfg:          cfg,
	}
}

func (uc *BrandLoyaltyUseCase) GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandCustomersQueryReq) (*dto.BrandCustomerListRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

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

	return &dto.BrandCustomerListRes{
		Items:    mapper.MapBrandCustomers(result.Customers),
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandLoyaltyUseCase) GetBrandCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.BrandCustomerRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	return mapper.MapBrandCustomer(customer), nil
}

func (uc *BrandLoyaltyUseCase) JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerRes, error) {
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
		return mapper.MapBrandCustomer(existing), nil
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
	return mapper.MapBrandCustomer(customer), nil
}

func (uc *BrandLoyaltyUseCase) CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error) {
	if strings.TrimSpace(input.PhoneE164) == "" {
		return nil, branderrors.ErrPhoneRequired()
	}
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
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
		return mapper.MapBrandCustomer(existing), nil
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
	return mapper.MapBrandCustomer(customer), nil
}

func (uc *BrandLoyaltyUseCase) GrantLoyaltyPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*dto.LoyaltyPointsTransactionRes, error) {
	if err := validateGrantLoyaltyPointsInput(input); err != nil {
		return nil, err
	}
	var response *dto.LoyaltyPointsTransactionRes

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := requireBrandRoleShared(txCtx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
			return err
		}

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

	res := make([]*dto.BrandLoyaltyRes, 0, len(customers))
	for _, customer := range activeCustomers {
		account := accountByCustomerID[customer.ID]
		if account == nil {
			continue
		}
		lot, err := uc.lotRepo.GetNearestExpiringActiveLot(ctx, account.ID, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		res = append(res, mapper.MapBrandLoyalty(account, customer, customer.Brand, lot))
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

func (uc *BrandLoyaltyUseCase) GetUserBrandLoyaltyTransactions(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyPointTransactionDetailRes, error) {
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
		return []*dto.LoyaltyPointTransactionDetailRes{}, nil
	}
	transactions, err := uc.txRepo.GetByLoyaltyAccountID(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTransactions(transactions), nil
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
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

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
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
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
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
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
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner); err != nil {
		return nil, err
	}

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
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	tiers, err := uc.tierRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyTiers(tiers), nil
}

func (uc *BrandLoyaltyUseCase) ProcessExpiredLoyaltyPointLots(ctx context.Context, now time.Time, batchSize int) (int, error) {
	accountIDs, err := uc.lotRepo.ListAccountsWithExpiredLots(ctx, now, batchSize)
	if err != nil {
		return 0, err
	}
	totalExpired := 0
	for _, accountID := range accountIDs {
		var expired int
		if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
			var err error
			expired, err = uc.expireDueLotsForAccount(txCtx, accountID, now, nil)
			return err
		}); err != nil {
			return totalExpired, err
		}
		totalExpired += expired
	}
	return totalExpired, nil
}
