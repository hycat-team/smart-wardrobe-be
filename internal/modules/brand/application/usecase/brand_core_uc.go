package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltyroundingmode"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandCoreUseCase struct {
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
}

func NewBrandCoreUseCase(
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
) uc_interfaces.IBrandCoreUseCase {
	return &BrandCoreUseCase{
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
	}
}

func (uc *BrandCoreUseCase) CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	return uc.createBrandWithOwner(ctx, userID, input, brandstatus.PendingReview, nil, nil)
}

func (uc *BrandCoreUseCase) CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	now := time.Now().UTC()
	return uc.createBrandWithOwner(ctx, adminID, input, brandstatus.Active, &adminID, &now)
}

func (uc *BrandCoreUseCase) createBrandWithOwner(ctx context.Context, creatorID uuid.UUID, input dto.CreateBrandReq, status brandstatus.BrandStatus, approverID *uuid.UUID, approvedAt *time.Time) (*dto.BrandRes, error) {
	slug := strings.TrimSpace(strings.ToLower(input.Slug))
	if existing, err := uc.brandRepo.GetBySlug(ctx, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, branderrors.ErrBrandSlugExists()
	}

	brand := &entities.Brand{
		Slug:             slug,
		Name:             strings.TrimSpace(input.Name),
		Description:      input.Description,
		LogoURL:          input.LogoURL,
		Status:           status,
		CreatedByUserID:  creatorID,
		ApprovedByUserID: approverID,
		ApprovedAt:       approvedAt,
	}
	member := &entities.BrandMember{
		UserID: creatorID,
		Role:   brandmemberrole.Owner,
		Status: brandmemberstatus.Active,
	}

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.brandRepo.Create(txCtx, brand); err != nil {
			return err
		}
		member.BrandID = brand.ID
		return uc.memberRepo.Create(txCtx, member)
	}); err != nil {
		return nil, err
	}

	return mapper.MapBrand(brand), nil
}

func (uc *BrandCoreUseCase) UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error) {
	if !isValidBrandStatus(input.Status) {
		return nil, branderrors.ErrInvalidBrandStatus(input.Status)
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	brand.Status = input.Status
	if input.Status == brandstatus.Active {
		now := time.Now().UTC()
		brand.ApprovedByUserID = &adminID
		brand.ApprovedAt = &now
	}
	if err := uc.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand), nil
}

func (uc *BrandCoreUseCase) GetActiveBrands(ctx context.Context) ([]*dto.BrandRes, error) {
	brands, err := uc.brandRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrands(brands), nil
}

func (uc *BrandCoreUseCase) GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand), nil
}

func (uc *BrandCoreUseCase) AddBrandMember(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMemberReq) (*dto.BrandMemberRes, error) {
	if !isValidBrandMemberRole(input.Role) {
		return nil, branderrors.ErrInvalidBrandMemberRole(input.Role)
	}
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
		return nil, err
	}
	existing, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, input.UserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.Role = input.Role
		existing.Status = brandmemberstatus.Active
		if err := uc.memberRepo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return mapper.MapBrandMember(existing), nil
	}

	member := &entities.BrandMember{
		BrandID: brandID,
		UserID:  input.UserID,
		Role:    input.Role,
		Status:  brandmemberstatus.Active,
	}
	if err := uc.memberRepo.Create(ctx, member); err != nil {
		return nil, err
	}
	return mapper.MapBrandMember(member), nil
}

func (uc *BrandCoreUseCase) GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
		return nil, err
	}
	members, err := uc.memberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandMembers(members), nil
}

func (uc *BrandCoreUseCase) GetBrandCustomers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandCustomerRes, error) {
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	customers, err := uc.customerRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandCustomers(customers), nil
}

func (uc *BrandCoreUseCase) JoinLoyalty(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, brandID uuid.UUID) (*dto.BrandCustomerRes, error) {
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

func (uc *BrandCoreUseCase) CreateOfflineCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.CreateOfflineBrandCustomerReq) (*dto.BrandCustomerRes, error) {
	if strings.TrimSpace(input.PhoneE164) == "" {
		return nil, branderrors.ErrPhoneRequired()
	}
	if err := uc.RequireBrandRole(ctx, userID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff); err != nil {
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

func (uc *BrandCoreUseCase) GrantLoyaltyPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*dto.LoyaltyPointsTransactionRes, error) {
	if err := validateGrantLoyaltyPointsInput(input); err != nil {
		return nil, err
	}
	var response *dto.LoyaltyPointsTransactionRes

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.RequireBrandRole(txCtx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
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

func (uc *BrandCoreUseCase) expireDueLotsForAccount(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time, createdByUserID *uuid.UUID) (int, error) {
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

func (uc *BrandCoreUseCase) ProcessExpiredLoyaltyPointLots(ctx context.Context, now time.Time, batchSize int) (int, error) {
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

func (uc *BrandCoreUseCase) redeemLoyaltyPointsFromLots(ctx context.Context, loyaltyAccountID uuid.UUID, requiredPoints int, now time.Time, reason *string, referenceType *string, referenceID *uuid.UUID, createdByUserID *uuid.UUID) (*entities.LoyaltyPointTransaction, error) {
	if requiredPoints <= 0 {
		return nil, branderrors.ErrPointsDeltaZero()
	}
	account, err := uc.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, branderrors.ErrInsufficientLoyaltyPoints()
	}
	if _, err := uc.expireDueLotsForAccount(ctx, loyaltyAccountID, now, nil); err != nil {
		return nil, err
	}
	account, err = uc.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.CurrentPoints < requiredPoints {
		return nil, branderrors.ErrInsufficientLoyaltyPoints()
	}
	lots, err := uc.lotRepo.ListRedeemableLotsForUpdate(ctx, loyaltyAccountID, now)
	if err != nil {
		return nil, err
	}
	remainingToRedeem := requiredPoints
	for _, lot := range lots {
		take := minInt(lot.RemainingPoints, remainingToRedeem)
		nextRemaining := lot.RemainingPoints - take
		nextStatus := loyaltypointlotstatus.Active
		if nextRemaining == 0 {
			nextStatus = loyaltypointlotstatus.Consumed
		}
		if err := uc.lotRepo.UpdateLotRemainingAndStatus(ctx, lot.ID, nextRemaining, nextStatus); err != nil {
			return nil, err
		}
		remainingToRedeem -= take
		if remainingToRedeem == 0 {
			break
		}
	}
	if remainingToRedeem > 0 {
		return nil, branderrors.ErrInsufficientLoyaltyPoints()
	}
	account.CurrentPoints -= requiredPoints
	if err := uc.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	tx := &entities.LoyaltyPointTransaction{
		LoyaltyAccountID: loyaltyAccountID,
		BrandID:          account.BrandID,
		BrandCustomerID:  account.BrandCustomerID,
		UserID:           account.UserID,
		PointsDelta:      -requiredPoints,
		BalanceAfter:     account.CurrentPoints,
		TransactionType:  loyaltytransactiontype.Redeem,
		Reason:           reason,
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		CreatedByUserID:  createdByUserID,
	}
	if err := uc.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (uc *BrandCoreUseCase) ensureLoyaltyAccount(ctx context.Context, customer *entities.BrandCustomer) error {
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

func (uc *BrandCoreUseCase) resolveBrandCustomerForPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*entities.BrandCustomer, error) {
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

func (uc *BrandCoreUseCase) resolvePointsDelta(input dto.GrantLoyaltyPointsReq, program *entities.LoyaltyProgram) (int, error) {
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

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

func (uc *BrandCoreUseCase) RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return err
	}
	if brand == nil {
		return branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return branderrors.ErrBrandNotActive()
	}
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return err
	}
	if member == nil || member.Status != brandmemberstatus.Active {
		return branderrors.ErrBrandPortalForbidden()
	}
	for _, allowedRole := range allowedRoles {
		if member.Role == allowedRole {
			return nil
		}
	}
	return branderrors.ErrBrandPortalForbidden()
}

func isValidBrandStatus(status brandstatus.BrandStatus) bool {
	switch status {
	case brandstatus.PendingReview, brandstatus.Active, brandstatus.Suspended, brandstatus.Archived:
		return true
	default:
		return false
	}
}

func isValidBrandMemberRole(role brandmemberrole.BrandMemberRole) bool {
	switch role {
	case brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer:
		return true
	default:
		return false
	}
}

func hashPhone(phone string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(phone)))
	return hex.EncodeToString(sum[:])
}
