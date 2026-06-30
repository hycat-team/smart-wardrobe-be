package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltyroundingmode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

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

func (uc *BrandLoyaltyUseCase) GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandCustomerRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	customers, err := uc.customerRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomers(customers), nil
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
	phoneHash := hashPhone(phone)
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
				response = mapLoyaltyTransactionResponse(existingTx, account, customer)
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

		spendAmount := valueOrNil(input.PurchaseAmount)
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
			response = mapLoyaltyTransactionResponse(nil, account, customer)
			return nil
		}

		expiresAt := calculateTransactionExpiry(input.TransactionType, program)
		idempotencyKey := trimmedStringPtr(input.IdempotencyKey)
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
		response = mapLoyaltyTransactionResponse(tx, account, customer)
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
		res = append(res, mapBrandLoyalty(account, customer, customer.Brand, lot))
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
	return mapBrandLoyalty(account, customer, brand, lot), nil
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
	return mapLoyaltyTransactions(transactions), nil
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

func (uc *BrandLoyaltyUseCase) GetLoyaltyAccountTransactionsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, loyaltyAccountID uuid.UUID) ([]*dto.LoyaltyPointTransactionDetailRes, error) {
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
	transactions, err := uc.txRepo.GetByLoyaltyAccountID(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	return mapLoyaltyTransactions(transactions), nil
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
	return mapLoyaltyProgram(program), nil
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

	return mapLoyaltyProgram(program), nil
}

func (uc *BrandLoyaltyUseCase) GetLoyaltyTiersForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.LoyaltyTierRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	tiers, err := uc.tierRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapLoyaltyTiers(tiers), nil
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

func (uc *BrandLoyaltyUseCase) expireDueLotsForAccount(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time, createdByUserID *uuid.UUID) (int, error) {
	account, err := uc.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return 0, err
	}
	if account == nil {
		return 0, nil
	}
	expiredLots, err := uc.lotRepo.ListExpiredLotsForUpdate(ctx, loyaltyAccountID, now)
	if err != nil {
		return 0, err
	}
	expiredPoints := 0
	for _, lot := range expiredLots {
		expiredPoints += lot.RemainingPoints
	}
	if expiredPoints == 0 {
		return 0, nil
	}
	if expiredPoints > account.CurrentPoints {
		expiredPoints = account.CurrentPoints
	}
	for _, lot := range expiredLots {
		if err := uc.lotRepo.UpdateLotRemainingAndStatus(ctx, lot.ID, 0, loyaltypointlotstatus.Expired); err != nil {
			return 0, err
		}
	}
	account.CurrentPoints -= expiredPoints
	if account.CurrentPoints < 0 {
		account.CurrentPoints = 0
	}
	if err := uc.accountRepo.Update(ctx, account); err != nil {
		return 0, err
	}
	reason := "Expired loyalty points"
	referenceType := "POINT_EXPIRY"
	tx := &entities.LoyaltyPointTransaction{
		LoyaltyAccountID: loyaltyAccountID,
		BrandID:          account.BrandID,
		BrandCustomerID:  account.BrandCustomerID,
		UserID:           account.UserID,
		PointsDelta:      -expiredPoints,
		BalanceAfter:     account.CurrentPoints,
		TransactionType:  loyaltytransactiontype.Expire,
		Reason:           &reason,
		ReferenceType:    &referenceType,
		CreatedByUserID:  createdByUserID,
	}
	if err := uc.txRepo.Create(ctx, tx); err != nil {
		return 0, err
	}
	return expiredPoints, nil
}

func (uc *BrandLoyaltyUseCase) listLoyaltyLots(ctx context.Context, loyaltyAccountID uuid.UUID, query dto.ListLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error) {
	var status *loyaltypointlotstatus.LoyaltyPointLotStatus
	if query.Status != nil && strings.TrimSpace(*query.Status) != "" {
		value := loyaltypointlotstatus.LoyaltyPointLotStatus(strings.ToLower(strings.TrimSpace(*query.Status)))
		status = &value
	}
	lots, err := uc.lotRepo.ListByAccountID(ctx, loyaltyAccountID, status, query.ExpiresAt, query.Page, query.Limit)
	if err != nil {
		return nil, err
	}
	return mapLoyaltyPointLots(lots), nil
}

func (uc *BrandLoyaltyUseCase) ensureLoyaltyAccount(ctx context.Context, customer *entities.BrandCustomer) error {
	if customer == nil {
		return nil
	}
	existing, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil || existing != nil {
		return err
	}
	account := &entities.LoyaltyAccount{
		BrandID:         customer.BrandID,
		BrandCustomerID: customer.ID,
		UserID:          customer.UserID,
		CurrentPoints:   0,
		LifetimePoints:  0,
		TotalSpend:      0,
	}
	return uc.accountRepo.Create(ctx, account)
}

func (uc *BrandLoyaltyUseCase) resolveBrandCustomerForPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*entities.BrandCustomer, error) {
	if input.UserID != nil {
		user, err := uc.userRepo.GetByID(ctx, *input.UserID)
		if err != nil {
			return nil, err
		}
		if user == nil || user.IsDeleted || user.Status != userstatus.Active {
			return nil, branderrors.ErrUserNotFoundOrInactive()
		}
		existing, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, *input.UserID)
		if err != nil || existing != nil {
			return existing, err
		}
		customer := &entities.BrandCustomer{
			BrandID:      brandID,
			UserID:       input.UserID,
			CustomerName: input.CustomerName,
			JoinedSource: brandcustomerjoinedsource.SelfJoin,
			Status:       brandcustomerstatus.Active,
			JoinedAt:     time.Now().UTC(),
		}
		if err := uc.customerRepo.Create(ctx, customer); err != nil {
			return nil, err
		}
		return customer, nil
	}

	if input.Phone != nil && strings.TrimSpace(*input.Phone) != "" {
		phone := normalizePhone(*input.Phone)
		phoneHash := hashPhone(phone)
		existing, err := uc.customerRepo.GetByBrandAndPhoneHash(ctx, brandID, phoneHash)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.CustomerName == nil && input.CustomerName != nil {
				existing.CustomerName = input.CustomerName
				if err := uc.customerRepo.Update(ctx, existing); err != nil {
					return nil, err
				}
			}
			return existing, nil
		}
		member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, staffUserID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, branderrors.ErrBrandPortalForbidden()
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
			CreatedByMemberID:    &member.ID,
		}
		if err := uc.customerRepo.Create(ctx, customer); err != nil {
			return nil, err
		}
		return customer, nil
	}

	externalCode := strings.TrimSpace(valueOrEmpty(input.ExternalCustomerCode))
	existing, err := uc.customerRepo.GetByBrandAndExternalCode(ctx, brandID, externalCode)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, staffUserID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	customer := &entities.BrandCustomer{
		BrandID:              brandID,
		CustomerName:         input.CustomerName,
		ExternalCustomerCode: input.ExternalCustomerCode,
		JoinedSource:         brandcustomerjoinedsource.OfflinePurchase,
		Status:               brandcustomerstatus.Active,
		JoinedAt:             time.Now().UTC(),
		CreatedByMemberID:    &member.ID,
	}
	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (uc *BrandLoyaltyUseCase) resolvePointsDelta(input dto.GrantLoyaltyPointsReq, program *entities.LoyaltyProgram) (int, error) {
	if input.PointsDelta != nil {
		return *input.PointsDelta, nil
	}
	if input.PurchaseAmount == nil || input.TransactionType != loyaltytransactiontype.Earn {
		return 0, branderrors.ErrPurchaseAmountOrPointsRequired()
	}
	if program == nil || program.AmountPerPoint <= 0 {
		return 0, branderrors.ErrActiveLoyaltyProgramRequired()
	}
	points := *input.PurchaseAmount / program.AmountPerPoint
	switch program.RoundingMode {
	case loyaltyroundingmode.Ceil:
		return int(math.Ceil(points)), nil
	case loyaltyroundingmode.Round:
		return int(math.Round(points)), nil
	default:
		return int(math.Floor(points)), nil
	}
}

func validateGrantLoyaltyPointsInput(input dto.GrantLoyaltyPointsReq) error {
	hasUserID := input.UserID != nil && *input.UserID != uuid.Nil
	hasPhone := input.Phone != nil && strings.TrimSpace(*input.Phone) != ""
	hasExternalCode := input.ExternalCustomerCode != nil && strings.TrimSpace(*input.ExternalCustomerCode) != ""
	if !hasUserID && !hasPhone && !hasExternalCode {
		return branderrors.ErrCustomerIdentifierRequired()
	}
	if input.PurchaseAmount == nil && input.PointsDelta == nil {
		return branderrors.ErrPurchaseAmountOrPointsRequired()
	}
	if input.PurchaseAmount != nil && *input.PurchaseAmount < 0 {
		return branderrors.ErrPurchaseAmountOrPointsRequired()
	}
	if input.PointsDelta != nil && *input.PointsDelta == 0 {
		return branderrors.ErrPointsDeltaZero()
	}
	switch input.TransactionType {
	case loyaltytransactiontype.Earn:
		return nil
	default:
		return branderrors.ErrInvalidLoyaltyTransactionType()
	}
}

func calculateTransactionExpiry(transactionType loyaltytransactiontype.LoyaltyTransactionType, program *entities.LoyaltyProgram) *time.Time {
	if transactionType != loyaltytransactiontype.Earn || program == nil || program.PointExpiryDays == nil {
		return nil
	}
	expiresAt := time.Now().UTC().AddDate(0, 0, *program.PointExpiryDays)
	return &expiresAt
}

func mapLoyaltyTransactionResponse(tx *entities.LoyaltyPointTransaction, account *entities.LoyaltyAccount, customer *entities.BrandCustomer) *dto.LoyaltyPointsTransactionRes {
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
	var currentTier *dto.LoyaltyTierBriefRes
	if account.CurrentTier != nil {
		currentTier = &dto.LoyaltyTierBriefRes{
			ID:   account.CurrentTier.ID,
			Name: account.CurrentTier.Name,
		}
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
		CurrentTier:     currentTier,
	}
}

func mapBrandLoyalty(account *entities.LoyaltyAccount, customer *entities.BrandCustomer, brand *entities.Brand, lot *entities.LoyaltyPointLot) *dto.BrandLoyaltyRes {
	if account == nil || customer == nil {
		return nil
	}
	var currentTier *dto.LoyaltyTierBriefRes
	if account.CurrentTier != nil {
		currentTier = &dto.LoyaltyTierBriefRes{
			ID:   account.CurrentTier.ID,
			Name: account.CurrentTier.Name,
		}
	}
	return &dto.BrandLoyaltyRes{
		BrandID:                 account.BrandID,
		Brand:                   mapper.MapBrand(brand),
		BrandCustomerID:         customer.ID,
		LoyaltyAccountID:        account.ID,
		CurrentPoints:           account.CurrentPoints,
		LifetimePoints:          account.LifetimePoints,
		TotalSpend:              account.TotalSpend,
		CurrentTier:             currentTier,
		NearestExpiringPointLot: mapLoyaltyPointLot(lot),
	}
}

func mapLoyaltyPointLot(lot *entities.LoyaltyPointLot) *dto.LoyaltyPointLotRes {
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

func mapLoyaltyPointLots(lots []*entities.LoyaltyPointLot) []*dto.LoyaltyPointLotRes {
	res := make([]*dto.LoyaltyPointLotRes, 0, len(lots))
	for _, lot := range lots {
		res = append(res, mapLoyaltyPointLot(lot))
	}
	return res
}

func mapLoyaltyTransaction(tx *entities.LoyaltyPointTransaction) *dto.LoyaltyPointTransactionDetailRes {
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

func mapLoyaltyTransactions(transactions []*entities.LoyaltyPointTransaction) []*dto.LoyaltyPointTransactionDetailRes {
	res := make([]*dto.LoyaltyPointTransactionDetailRes, len(transactions))
	for i, tx := range transactions {
		res[i] = mapLoyaltyTransaction(tx)
	}
	return res
}

func mapLoyaltyProgram(program *entities.LoyaltyProgram) *dto.LoyaltyProgramRes {
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

func mapLoyaltyTier(tier *entities.LoyaltyTier) *dto.LoyaltyTierRes {
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

func mapLoyaltyTiers(tiers []*entities.LoyaltyTier) []*dto.LoyaltyTierRes {
	res := make([]*dto.LoyaltyTierRes, len(tiers))
	for i, tier := range tiers {
		res[i] = mapLoyaltyTier(tier)
	}
	return res
}

func hashPhone(phone string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(phone)))
	return hex.EncodeToString(sum[:])
}

func normalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

func valueOrNil(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func trimmedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
