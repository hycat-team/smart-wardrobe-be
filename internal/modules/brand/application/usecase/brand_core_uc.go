package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/internal/shared/domain/constants/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandchat/senderrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/branditem/votetype"
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
	brandRepo       repositories.IBrandRepository
	memberRepo      repositories.IBrandMemberRepository
	customerRepo    repositories.IBrandCustomerRepository
	userRepo        identity_repos.IUserRepository
	programRepo     repositories.ILoyaltyProgramRepository
	tierRepo        repositories.ILoyaltyTierRepository
	accountRepo     repositories.ILoyaltyAccountRepository
	txRepo          repositories.ILoyaltyPointTransactionRepository
	lotRepo         repositories.ILoyaltyPointLotRepository
	benefitRepo     repositories.IBrandBenefitRepository
	redemptionRepo  repositories.IBenefitRedemptionRepository
	convRepo        repositories.IBrandConversationRepository
	msgRepo         repositories.IBrandConversationMessageRepository
	brandItemRepo   repositories.IBrandItemRepository
	feedbackRepo    repositories.IDigitalSampleResponseRepository
	claimRepo       repositories.IBrandCustomerClaimRepository
	fashionContract fashion_contract.IFashionContract
	uow             shared_repos.IUnitOfWork
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
	benefitRepo repositories.IBrandBenefitRepository,
	redemptionRepo repositories.IBenefitRedemptionRepository,
	convRepo repositories.IBrandConversationRepository,
	msgRepo repositories.IBrandConversationMessageRepository,
	brandItemRepo repositories.IBrandItemRepository,
	feedbackRepo repositories.IDigitalSampleResponseRepository,
	claimRepo repositories.IBrandCustomerClaimRepository,
	fashionContract fashion_contract.IFashionContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandCoreUseCase {
	return &BrandCoreUseCase{
		brandRepo:       brandRepo,
		memberRepo:      memberRepo,
		customerRepo:    customerRepo,
		userRepo:        userRepo,
		programRepo:     programRepo,
		tierRepo:        tierRepo,
		accountRepo:     accountRepo,
		txRepo:          txRepo,
		lotRepo:         lotRepo,
		benefitRepo:     benefitRepo,
		redemptionRepo:  redemptionRepo,
		convRepo:        convRepo,
		msgRepo:         msgRepo,
		brandItemRepo:   brandItemRepo,
		feedbackRepo:    feedbackRepo,
		claimRepo:       claimRepo,
		fashionContract: fashionContract,
		uow:             uow,
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

func (uc *BrandCoreUseCase) CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
		return nil, err
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	bType := benefittype.BenefitType(strings.ToUpper(input.BenefitType))
	uType := benefitunlocktype.BenefitUnlockType(strings.ToUpper(input.UnlockType))

	benefit := &entities.BrandBenefit{
		BrandID:     brandID,
		Name:        strings.TrimSpace(input.Name),
		Description: input.Description,
		BenefitType: bType,
		UnlockType:  uType,
		Status:      benefitstatus.Active,
	}

	if uType == benefitunlocktype.PointRedemption {
		if input.RequiredPoints == nil || *input.RequiredPoints <= 0 {
			return nil, branderrors.ErrPurchaseAmountOrPointsRequired() // generic bad request
		}
		benefit.RequiredPoints = input.RequiredPoints
	} else if uType == benefitunlocktype.TierPrivilege {
		if input.RequiredTierID == nil {
			return nil, branderrors.ErrInvalidBrandMemberRole("Yeu cau phai co requiredTierId") // generic bad request
		}
		tier, err := uc.tierRepo.GetByID(ctx, *input.RequiredTierID)
		if err != nil {
			return nil, err
		}
		if tier == nil || tier.BrandID != brandID {
			return nil, branderrors.ErrBenefitInvalidStatus()
		}
		benefit.RequiredTierID = input.RequiredTierID
	}

	if input.FeatureCode != nil {
		fCode := benefitfeaturecode.BenefitFeatureCode(strings.ToUpper(*input.FeatureCode))
		benefit.FeatureCode = &fCode
	}

	if input.FeatureConfig != nil {
		bytes, err := json.Marshal(input.FeatureConfig)
		if err != nil {
			return nil, branderrors.ErrBenefitInvalidStatus()
		}
		benefit.FeatureConfig = entities.JSONDocument(bytes)
	}

	if err := uc.benefitRepo.Create(ctx, benefit); err != nil {
		return nil, err
	}

	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandCoreUseCase) ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	benefits, err := uc.benefitRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandBenefits(benefits), nil
}

func (uc *BrandCoreUseCase) ListActiveBenefitsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandBenefits(benefits), nil
}

func (uc *BrandCoreUseCase) UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager); err != nil {
		return nil, err
	}

	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil || benefit.BrandID != brandID {
		return nil, branderrors.ErrBenefitNotFound()
	}

	bStatus := benefitstatus.BenefitStatus(strings.ToUpper(status))
	if bStatus != benefitstatus.Active && bStatus != benefitstatus.Inactive && bStatus != benefitstatus.Archived {
		return nil, branderrors.ErrBenefitInvalidStatus()
	}

	benefit.Status = bStatus
	if err := uc.benefitRepo.Update(ctx, benefit); err != nil {
		return nil, err
	}

	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandCoreUseCase) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return false, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return false, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	fCode := benefitfeaturecode.BenefitFeatureCode(strings.ToUpper(featureCode))

	for _, benefit := range benefits {
		if benefit.BenefitType != benefittype.FeatureAccess || benefit.FeatureCode == nil || *benefit.FeatureCode != fCode {
			continue
		}

		if benefit.UnlockType == benefitunlocktype.TierPrivilege {
			// Check user current tier
			account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
			if err != nil || account == nil || account.CurrentTierID == nil {
				continue
			}

			userTier, err := uc.tierRepo.GetByID(ctx, *account.CurrentTierID)
			if err != nil || userTier == nil {
				continue
			}

			requiredTier, err := uc.tierRepo.GetByID(ctx, *benefit.RequiredTierID)
			if err != nil || requiredTier == nil {
				continue
			}

			if userTier.Rank >= requiredTier.Rank {
				return true, nil
			}
		} else if benefit.UnlockType == benefitunlocktype.PointRedemption || benefit.UnlockType == benefitunlocktype.ManualGrant {
			// Check if active redemption exists
			red, err := uc.redemptionRepo.GetActiveRedemptionByFeature(ctx, customer.ID, featureCode, now)
			if err != nil {
				return false, err
			}
			if red != nil {
				return true, nil
			}
		}
	}

	return false, nil
}

func (uc *BrandCoreUseCase) RedeemBenefit(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil || benefit.BrandID != brandID {
		return nil, branderrors.ErrBenefitNotFound()
	}
	if benefit.Status != benefitstatus.Active {
		return nil, branderrors.ErrBenefitNotActive()
	}

	now := time.Now().UTC()
	var redemption *entities.BenefitRedemption

	if benefit.UnlockType == benefitunlocktype.PointRedemption {
		// MULTI-WRITE USECASE -> Apply Unit of Work (uow) pattern
		err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
			account, err := uc.accountRepo.GetByBrandCustomerIDForUpdate(txCtx, customer.ID)
			if err != nil {
				return err
			}
			if account == nil {
				return branderrors.ErrInsufficientLoyaltyPoints()
			}

			// Check double redemption
			var featStr string
			if benefit.FeatureCode != nil {
				featStr = string(*benefit.FeatureCode)
			}
			existing, err := uc.redemptionRepo.GetActiveRedemptionByFeature(txCtx, customer.ID, featStr, now)
			if err != nil {
				return err
			}
			if existing != nil {
				return branderrors.ErrBenefitRedemptionExists()
			}

			reqPoints := 0
			if benefit.RequiredPoints != nil {
				reqPoints = *benefit.RequiredPoints
			}

			reason := "Doi quyen loi: " + benefit.Name
			refType := "BENEFIT_REDEMPTION"
			refID := benefit.ID
			_, err = uc.redeemLoyaltyPointsFromLots(txCtx, account.ID, reqPoints, now, &reason, &refType, &refID, &userID)
			if err != nil {
				return err
			}

			var expiresAt *time.Time
			durationDays := parseValidDurationDays(benefit.FeatureConfig)
			if durationDays > 0 {
				t := now.Add(time.Duration(durationDays) * 24 * time.Hour)
				expiresAt = &t
			}

			redemption = &entities.BenefitRedemption{
				BenefitID:       benefit.ID,
				BrandID:         brandID,
				BrandCustomerID: customer.ID,
				UserID:          &userID,
				PointsSpent:     reqPoints,
				Status:          benefitredemptionstatus.Redeemed,
				RedeemedAt:      now,
				ExpiresAt:       expiresAt,
			}

			return uc.redemptionRepo.Create(txCtx, redemption)
		})
		if err != nil {
			return nil, err
		}
	} else if benefit.UnlockType == benefitunlocktype.TierPrivilege {
		// SINGLE-WRITE (checks tier, writes once to benefit_redemptions). No UoW needed.
		account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
		if err != nil {
			return nil, err
		}
		if account == nil || account.CurrentTierID == nil {
			return nil, branderrors.ErrBenefitRequiredTierNotMet()
		}

		userTier, err := uc.tierRepo.GetByID(ctx, *account.CurrentTierID)
		if err != nil || userTier == nil {
			return nil, branderrors.ErrBenefitRequiredTierNotMet()
		}

		requiredTier, err := uc.tierRepo.GetByID(ctx, *benefit.RequiredTierID)
		if err != nil || requiredTier == nil {
			return nil, branderrors.ErrBenefitRequiredTierNotMet()
		}

		if userTier.Rank < requiredTier.Rank {
			return nil, branderrors.ErrBenefitRequiredTierNotMet()
		}

		var expiresAt *time.Time
		durationDays := parseValidDurationDays(benefit.FeatureConfig)
		if durationDays > 0 {
			t := now.Add(time.Duration(durationDays) * 24 * time.Hour)
			expiresAt = &t
		}

		redemption = &entities.BenefitRedemption{
			BenefitID:       benefit.ID,
			BrandID:         brandID,
			BrandCustomerID: customer.ID,
			UserID:          &userID,
			PointsSpent:     0,
			Status:          benefitredemptionstatus.Redeemed,
			RedeemedAt:      now,
			ExpiresAt:       expiresAt,
		}

		if err := uc.redemptionRepo.Create(ctx, redemption); err != nil {
			return nil, err
		}
	} else {
		return nil, branderrors.ErrBenefitUnlockTypeNotSupported()
	}

	return mapper.MapBenefitRedemption(redemption), nil
}

func parseValidDurationDays(doc entities.JSONDocument) int {
	if len(doc) == 0 {
		return 0
	}
	var config map[string]interface{}
	if err := json.Unmarshal(doc, &config); err != nil {
		return 0
	}
	if val, ok := config["valid_duration_days"]; ok {
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return 0
}

func (uc *BrandCoreUseCase) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, filter interface{}) (interface{}, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Status != userstatus.Active {
		return []*entities.BrandItem{}, nil
	}

	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var eligibleBrandItems []*entities.BrandItem
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active {
			continue
		}

		brand, err := uc.brandRepo.GetByID(ctx, customer.BrandID)
		if err != nil {
			return nil, err
		}
		if brand == nil || brand.Status != brandstatus.Active {
			continue
		}

		// Check SAMPLE_MIX_ACCESS feature access for this brand
		hasSampleAccess, err := uc.CheckBrandFeatureAccess(ctx, userID, customer.BrandID, "SAMPLE_MIX_ACCESS")
		if err != nil {
			return nil, err
		}

		brandItems, err := uc.brandItemRepo.GetByBrandID(ctx, customer.BrandID)
		if err != nil {
			return nil, err
		}

		for _, item := range brandItems {
			if item.Status != branditemstatus.Active {
				continue
			}

			if item.ItemType == branditemtype.Product {
				eligibleBrandItems = append(eligibleBrandItems, item)
			} else if item.ItemType == branditemtype.Sample {
				if hasSampleAccess {
					eligibleBrandItems = append(eligibleBrandItems, item)
				}
			}
		}
	}

	return eligibleBrandItems, nil
}

func (uc *BrandCoreUseCase) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *entities.BrandItem, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, nil, err
	}
	if user == nil || user.Status != userstatus.Active {
		return false, nil, nil
	}

	brandItem, err := uc.brandItemRepo.GetByFashionItemID(ctx, fashionItemID)
	if err != nil {
		return false, nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return false, nil, nil
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandItem.BrandID)
	if err != nil {
		return false, nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandItem.BrandID, userID)
	if err != nil {
		return false, nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil, nil
	}

	if brandItem.ItemType == branditemtype.Sample {
		hasSampleAccess, err := uc.CheckBrandFeatureAccess(ctx, userID, brandItem.BrandID, "SAMPLE_MIX_ACCESS")
		if err != nil {
			return false, nil, err
		}
		if !hasSampleAccess {
			return false, nil, nil
		}
	}

	return true, brandItem, nil
}

func (uc *BrandCoreUseCase) GetUserConversation(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandConversationRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	conv, err := uc.convRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		// Return 404 with clear message as optimized in Grill-me
		return nil, branderrors.ErrBrandNotFound() // We will return conversation not found via custom error or a default NotFound
	}

	user, _ := uc.userRepo.GetByID(ctx, userID)
	userDisp := getUserDisplayName(user)
	return mapper.MapBrandConversation(conv, customer.CustomerName, &userDisp), nil
}

func (uc *BrandCoreUseCase) SendUserMessage(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	var messageRes *dto.BrandConversationMessageRes
	now := time.Now().UTC()

	// MULTI-WRITE USECASE -> Apply Unit of Work (uow) pattern
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		conv, err := uc.convRepo.GetByBrandAndUser(txCtx, brandID, userID)
		if err != nil {
			return err
		}

		if conv == nil {
			// First message -> create conversation automatically as requested
			conv = &entities.BrandConversation{
				BrandID:       brandID,
				UserID:        userID,
				Status:        conversationstatus.Open,
				LastMessageAt: &now,
			}
			if err := uc.convRepo.Create(txCtx, conv); err != nil {
				return err
			}
		} else {
			// Update status and last message time
			conv.Status = conversationstatus.Open
			conv.LastMessageAt = &now
			if err := uc.convRepo.Update(txCtx, conv); err != nil {
				return err
			}
		}

		msg := &entities.BrandConversationMessage{
			ConversationID: conv.ID,
			SenderUserID:   &userID,
			SenderRole:     senderrole.Customer,
			Message:        strings.TrimSpace(input.Message),
			CreatedAt:      now,
		}
		if err := uc.msgRepo.Create(txCtx, msg); err != nil {
			return err
		}

		messageRes = mapper.MapBrandConversationMessage(msg)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return messageRes, nil
}

func (uc *BrandCoreUseCase) ListBrandConversations(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandConversationRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff); err != nil {
		return nil, err
	}

	conversations, err := uc.convRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	var res []*dto.BrandConversationRes
	for _, conv := range conversations {
		customer, _ := uc.customerRepo.GetByBrandAndUser(ctx, brandID, conv.UserID)
		user, _ := uc.userRepo.GetByID(ctx, conv.UserID)
		userDisp := getUserDisplayName(user)
		var custName *string
		if customer != nil {
			custName = customer.CustomerName
		}
		res = append(res, mapper.MapBrandConversation(conv, custName, &userDisp))
	}

	return res, nil
}

func (uc *BrandCoreUseCase) ListConversationMessages(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) ([]*dto.BrandConversationMessageRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff); err != nil {
		return nil, err
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}

	messages, err := uc.msgRepo.GetByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	return mapper.MapBrandConversationMessages(messages), nil
}

func (uc *BrandCoreUseCase) SendStaffMessage(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff); err != nil {
		return nil, err
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}

	var messageRes *dto.BrandConversationMessageRes
	now := time.Now().UTC()

	// MULTI-WRITE USECASE -> Apply Unit of Work (uow) pattern
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		lockedConv, err := uc.convRepo.GetByIDForUpdate(txCtx, conversationID)
		if err != nil {
			return err
		}
		if lockedConv == nil {
			return branderrors.ErrBrandNotFound()
		}

		lockedConv.Status = conversationstatus.Open
		lockedConv.LastMessageAt = &now
		if err := uc.convRepo.Update(txCtx, lockedConv); err != nil {
			return err
		}

		msg := &entities.BrandConversationMessage{
			ConversationID: lockedConv.ID,
			SenderUserID:   &staffUserID,
			SenderRole:     senderrole.BrandStaff,
			Message:        strings.TrimSpace(input.Message),
			CreatedAt:      now,
		}
		if err := uc.msgRepo.Create(txCtx, msg); err != nil {
			return err
		}

		messageRes = mapper.MapBrandConversationMessage(msg)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return messageRes, nil
}

func getUserDisplayName(u *entities.User) string {
	if u == nil {
		return ""
	}
	var parts []string
	if u.FirstName != nil && *u.FirstName != "" {
		parts = append(parts, *u.FirstName)
	}
	if u.LastName != nil && *u.LastName != "" {
		parts = append(parts, *u.LastName)
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return u.Username
	}
	return name
}

func mapToBrandItemRes(item *entities.BrandItem) *dto.BrandItemRes {
	if item == nil {
		return nil
	}
	return &dto.BrandItemRes{
		ID:            item.ID,
		BrandID:       item.BrandID,
		FashionItemID: item.FashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		Description:   item.Description,
		Price:         item.Price,
		ItemType:      string(item.ItemType),
		Status:        string(item.Status),
		FashionItem:   item.FashionItem,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func mapToDigitalSampleResponseRes(res *entities.DigitalSampleResponse) *dto.DigitalSampleResponseRes {
	if res == nil {
		return nil
	}
	var voteStr *string
	if res.VoteType != nil {
		s := string(*res.VoteType)
		voteStr = &s
	}
	return &dto.DigitalSampleResponseRes{
		ID:           res.ID,
		BrandItemID:  res.BrandItemID,
		UserID:       res.UserID,
		OutfitID:     res.OutfitID,
		VoteType:     voteStr,
		Rating:       res.Rating,
		FeedbackText: res.FeedbackText,
		CreatedAt:    res.CreatedAt,
	}
}

func (uc *BrandCoreUseCase) CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}

	brandItemID := uuid.New()
	fashionItemID, err := uc.fashionContract.CreateFashionItem(
		ctx,
		staffUserID,
		brandItemID,
		"brand",
		input.CategoryID,
		input.ImageUrl,
		input.ImagePublicID,
	)
	if err != nil {
		return nil, err
	}

	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID:        brandItemID,
				CreatedAt: time.Now().UTC(),
			},
			UpdatedAt: time.Now().UTC(),
		},
		BrandID:       brandID,
		FashionItemID: fashionItemID,
		ProductCode:   input.ProductCode,
		Name:          input.Name,
		Description:   input.Description,
		Price:         input.Price,
		ItemType:      branditemtype.BrandItemType(input.ItemType),
		Status:        branditemstatus.BrandItemStatus(input.Status),
	}
	if item.Status == "" {
		item.Status = branditemstatus.Draft
	}

	if err := uc.brandItemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, fashionItemID)
	item.FashionItem = fashionItem

	return mapToBrandItemRes(item), nil
}

func (uc *BrandCoreUseCase) GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	items, err := uc.brandItemRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.BrandItemRes, len(items))
	for i, item := range items {
		res[i] = mapToBrandItemRes(item)
	}
	return res, nil
}

func (uc *BrandCoreUseCase) UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	item.Name = input.Name
	item.Description = input.Description
	item.Price = input.Price
	item.Status = branditemstatus.BrandItemStatus(input.Status)
	item.UpdatedAt = time.Now().UTC()

	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, item.FashionItemID)
	item.FashionItem = fashionItem

	return mapToBrandItemRes(item), nil
}

func (uc *BrandCoreUseCase) GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff, brandmemberrole.Marketer); err != nil {
		return nil, err
	}
	feedbacks, err := uc.feedbackRepo.GetByBrandItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.DigitalSampleResponseRes, len(feedbacks))
	for i, f := range feedbacks {
		res[i] = mapToDigitalSampleResponseRes(f)
	}
	return res, nil
}

func (uc *BrandCoreUseCase) ListBrandItemsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error) {
	items, err := uc.brandItemRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	var activeItems []*entities.BrandItem
	for _, item := range items {
		if item.Status == branditemstatus.Active {
			activeItems = append(activeItems, item)
		}
	}
	res := make([]*dto.BrandItemRes, len(activeItems))
	for i, item := range activeItems {
		res[i] = mapToBrandItemRes(item)
	}
	return res, nil
}

func (uc *BrandCoreUseCase) SubmitSampleFeedback(ctx context.Context, userID uuid.UUID, brandItemID uuid.UUID, input dto.SubmitSampleFeedbackReq) (*dto.DigitalSampleResponseRes, error) {
	brandItem, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return nil, branderrors.ErrBrandNotFound()
	}

	var vote *votetype.VoteType
	if input.VoteType != nil {
		v := votetype.VoteType(*input.VoteType)
		vote = &v
	}

	feedback := &entities.DigitalSampleResponse{
		BrandItemID:  brandItemID,
		UserID:       userID,
		OutfitID:     input.OutfitID,
		VoteType:     vote,
		Rating:       input.Rating,
		FeedbackText: input.FeedbackText,
		CreatedAt:    time.Now().UTC(),
	}

	if err := uc.feedbackRepo.Create(ctx, feedback); err != nil {
		return nil, err
	}

	return mapToDigitalSampleResponseRes(feedback), nil
}

func (uc *BrandCoreUseCase) CreateBrandCustomerClaim(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.CreateClaimTokenRes, error) {
	if err := uc.RequireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Manager, brandmemberrole.SupportStaff); err != nil {
		return nil, err
	}
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	if customer.UserID != nil {
		return nil, branderrors.ErrCustomerAlreadyLinked()
	}

	token := uuid.New().String()
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])

	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	claim := &entities.BrandCustomerClaim{
		BrandCustomerID: customerID,
		ClaimTokenHash:  hashStr,
		ExpiresAt:       expiresAt,
	}

	if err := uc.claimRepo.Create(ctx, claim); err != nil {
		return nil, err
	}

	return &dto.CreateClaimTokenRes{
		ClaimToken: token,
		ExpiresAt:  expiresAt,
	}, nil
}

func (uc *BrandCoreUseCase) ClaimBrandCustomer(ctx context.Context, userID uuid.UUID, claimToken string) (*dto.BrandCustomerRes, error) {
	token := strings.TrimSpace(claimToken)
	if token == "" {
		return nil, branderrors.ErrInvalidToken()
	}
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])

	claim, err := uc.claimRepo.GetByTokenHash(ctx, hashStr)
	if err != nil {
		return nil, err
	}
	if claim == nil {
		return nil, branderrors.ErrInvalidToken()
	}
	if claim.ConsumedAt != nil {
		return nil, branderrors.ErrTokenAlreadyUsed()
	}
	if time.Now().UTC().After(claim.ExpiresAt) {
		return nil, branderrors.ErrTokenExpired()
	}

	customer, err := uc.customerRepo.GetByID(ctx, claim.BrandCustomerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, branderrors.ErrCustomerNotFound()
	}
	if customer.UserID != nil {
		return nil, branderrors.ErrCustomerAlreadyLinked()
	}

	var updatedCustomer *entities.BrandCustomer
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		now := time.Now().UTC()

		customer.UserID = &userID
		customer.ClaimedAt = &now
		customer.UpdatedAt = now
		if err := uc.customerRepo.Update(txCtx, customer); err != nil {
			return err
		}

		account, err := uc.accountRepo.GetByBrandCustomerID(txCtx, customer.ID)
		if err != nil {
			return err
		}
		if account != nil {
			account.UserID = &userID
			account.UpdatedAt = now
			if err := uc.accountRepo.Update(txCtx, account); err != nil {
				return err
			}
		}

		claim.ConsumedAt = &now
		if err := uc.claimRepo.Update(txCtx, claim); err != nil {
			return err
		}

		updatedCustomer = customer
		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapper.MapBrandCustomer(updatedCustomer), nil
}
