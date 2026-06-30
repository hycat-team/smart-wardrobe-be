package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type BrandClaimUseCase struct {
	brandRepo    repositories.IBrandRepository
	memberRepo   repositories.IBrandMemberRepository
	customerRepo repositories.IBrandCustomerRepository
	accountRepo  repositories.ILoyaltyAccountRepository
	claimRepo    repositories.IBrandCustomerClaimRepository
	redisClient  *redis.Client
	uow          shared_repos.IUnitOfWork
	cfg          *config.Config
}

func NewBrandClaimUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	accountRepo repositories.ILoyaltyAccountRepository,
	claimRepo repositories.IBrandCustomerClaimRepository,
	redisClient *redis.Client,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.IBrandClaimUseCase {
	return &BrandClaimUseCase{
		brandRepo:    brandRepo,
		memberRepo:   memberRepo,
		customerRepo: customerRepo,
		accountRepo:  accountRepo,
		claimRepo:    claimRepo,
		redisClient:  redisClient,
		uow:          uow,
		cfg:          cfg,
	}
}

func (uc *BrandClaimUseCase) CreateBrandCustomerClaim(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.CreateClaimTokenRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
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

	activeClaims, err := uc.claimRepo.GetActiveByCustomerID(ctx, customerID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	reason := "Rotated by new claim token"
	for _, active := range activeClaims {
		active.RevokedAt = &now
		active.RevokedByUserID = &staffUserID
		active.RevokedReason = &reason
		if err := uc.claimRepo.Update(ctx, active); err != nil {
			return nil, err
		}
	}

	token, err := generateClaimToken()
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])

	expiresAt := now.Add(24 * time.Hour)
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

func (uc *BrandClaimUseCase) ListBrandCustomerClaims(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) ([]*dto.ClaimTokenRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	claims, err := uc.claimRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return mapper.MapClaimTokens(claims), nil
}

func (uc *BrandClaimUseCase) RevokeBrandCustomerClaim(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID, claimID uuid.UUID, input dto.RevokeClaimTokenReq) (*dto.ClaimTokenRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.BrandID != brandID {
		return nil, branderrors.ErrCustomerNotFound()
	}
	claim, err := uc.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, err
	}
	if claim == nil || claim.BrandCustomerID != customerID {
		return nil, branderrors.ErrInvalidToken()
	}
	now := time.Now().UTC()
	claim.RevokedAt = &now
	claim.RevokedByUserID = &staffUserID
	claim.RevokedReason = input.Reason
	if err := uc.claimRepo.Update(ctx, claim); err != nil {
		return nil, err
	}
	return mapper.MapClaimToken(claim), nil
}

func (uc *BrandClaimUseCase) ClaimBrandCustomer(ctx context.Context, userID uuid.UUID, claimToken string, clientIP string) (*dto.BrandCustomerRes, error) {
	token := strings.TrimSpace(claimToken)
	if token == "" {
		return nil, branderrors.ErrInvalidToken()
	}
	if err := uc.checkClaimRateLimit(ctx, userID, token, clientIP); err != nil {
		return nil, err
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
	if claim.RevokedAt != nil {
		return nil, branderrors.ErrTokenRevoked()
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

	existing, err := uc.customerRepo.GetByBrandAndUser(ctx, customer.BrandID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrUserAlreadyHasCustomer()
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

func generateClaimToken() (string, error) {
	buf := make([]byte, 9)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	if len(token) > 12 {
		token = token[:12]
	}
	return token, nil
}

func (uc *BrandClaimUseCase) checkClaimRateLimit(ctx context.Context, userID uuid.UUID, claimToken string, clientIP string) error {
	if uc.redisClient == nil || uc.cfg == nil {
		return branderrors.ErrClaimRateLimitUnavailable()
	}
	tokenHash := sha256.Sum256([]byte(claimToken))
	limits := []struct {
		key   string
		limit int
	}{
		{key: "claim:ip:" + strings.TrimSpace(clientIP), limit: uc.cfg.ClaimRateLimit.IPLimit},
		{key: "claim:user:" + userID.String(), limit: uc.cfg.ClaimRateLimit.UserLimit},
		{key: "claim:token:" + hex.EncodeToString(tokenHash[:]), limit: uc.cfg.ClaimRateLimit.TokenLimit},
	}
	window := uc.cfg.ClaimRateLimit.WindowSeconds
	for _, item := range limits {
		allowed, err := uc.consumeClaimRateLimit(ctx, item.key, item.limit, window)
		if err != nil {
			return branderrors.ErrClaimRateLimitUnavailable()
		}
		if !allowed {
			return branderrors.ErrClaimRateLimited()
		}
	}
	return nil
}

func (uc *BrandClaimUseCase) consumeClaimRateLimit(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	script := redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)
	res, err := script.Run(ctx, uc.redisClient, []string{key}, windowSeconds, limit).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
