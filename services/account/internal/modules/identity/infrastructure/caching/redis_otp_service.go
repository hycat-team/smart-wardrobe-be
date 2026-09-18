package caching

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/application/constants/otpconstants"
	shared_application_caching "smart-wardrobe-be/shared/application/interface/caching"
	"smart-wardrobe-be/shared/config"
	shared_infra_caching "smart-wardrobe-be/shared/infrastructure/caching"
)

type RedisOtpService struct {
	cache    shared_infra_caching.ICacheService
	cooldown shared_application_caching.ICooldownService
	cfg      *config.Config
}

func NewRedisOtpService(cache shared_infra_caching.ICacheService, cooldown shared_application_caching.ICooldownService, cfg *config.Config) identity.IOtpService {
	return &RedisOtpService{
		cache:    cache,
		cooldown: cooldown,
		cfg:      cfg,
	}
}

func (s *RedisOtpService) GenerateOtp(ctx context.Context, email string, tempUserDataJson string, purpose string) (string, error) {
	otpCode := s.generateSecureOTP()

	otpKey := otpconstants.BuildKey(purpose, otpconstants.KeyValue, email)
	attemptKey := otpconstants.BuildKey(purpose, otpconstants.KeyAttempts, email)
	cooldownKey := otpconstants.BuildKey(purpose, otpconstants.KeyCooldown, email)
	dataKey := otpconstants.BuildKey(purpose, otpconstants.KeyData, email)

	expiry := time.Duration(s.cfg.Otp.ExpiryMinutes) * time.Minute
	cooldownTTL := time.Duration(s.cfg.Otp.ResendIntervalSeconds) * time.Second

	err := s.cache.SetRaw(ctx, otpKey, otpCode, expiry)
	if err != nil {
		return "", cacheUnavailable(err)
	}

	err = s.cache.SetRaw(ctx, attemptKey, "0", expiry)
	if err != nil {
		return "", cacheUnavailable(err)
	}

	err = s.cache.SetRaw(ctx, dataKey, tempUserDataJson, expiry)
	if err != nil {
		return "", cacheUnavailable(err)
	}

	err = s.cooldown.SetCooldown(ctx, cooldownKey, cooldownTTL)
	if err != nil {
		return "", cacheUnavailable(err)
	}

	return otpCode, nil
}

func (s *RedisOtpService) VerifyOtp(ctx context.Context, email string, otpCode string, purpose string) (string, error) {
	otpKey := otpconstants.BuildKey(purpose, otpconstants.KeyValue, email)
	attemptKey := otpconstants.BuildKey(purpose, otpconstants.KeyAttempts, email)
	dataKey := otpconstants.BuildKey(purpose, otpconstants.KeyData, email)

	savedOtp, err := s.cache.GetRaw(ctx, otpKey)
	if err != nil {
		if errors.Is(err, shared_infra_caching.ErrKeyNotFound) {
			return "", apperror.NewBadRequest("Mã OTP đã hết hạn hoặc không tồn tại. Vui lòng yêu cầu mã mới.")
		}
		return "", cacheUnavailable(err)
	}

	tempUserData, err := s.cache.GetRaw(ctx, dataKey)
	if err != nil {
		if errors.Is(err, shared_infra_caching.ErrKeyNotFound) {
			return "", apperror.NewBadRequest("Mã OTP đã hết hạn hoặc không tồn tại. Vui lòng yêu cầu mã mới.")
		}
		return "", cacheUnavailable(err)
	}

	attemptsStr, err := s.cache.GetRaw(ctx, attemptKey)
	if err != nil && !errors.Is(err, shared_infra_caching.ErrKeyNotFound) {
		return "", cacheUnavailable(err)
	}
	if errors.Is(err, shared_infra_caching.ErrKeyNotFound) {
		attemptsStr = "0"
	}

	currentAttempts, err := strconv.Atoi(attemptsStr)
	if err != nil {
		currentAttempts = 0
	}

	if currentAttempts >= s.cfg.Otp.MaxAttempts {
		_ = s.cache.Del(ctx, otpKey)
		_ = s.cache.Del(ctx, dataKey)
		return "", apperror.NewTooManyRequest("Bạn đã nhập sai quá nhiều lần. Mã OTP này đã bị hủy để bảo mật. Vui lòng thử lại sau 1 phút.")
	}

	if savedOtp != otpCode {
		currentAttempts++
		expiry := time.Duration(s.cfg.Otp.ExpiryMinutes) * time.Minute
		_ = s.cache.SetRaw(ctx, attemptKey, strconv.Itoa(currentAttempts), expiry)

		remainingAttempts := s.cfg.Otp.MaxAttempts - currentAttempts
		return "", apperror.NewBadRequest(fmt.Sprintf("Mã OTP không chính xác. Bạn còn %d lần thử.", remainingAttempts))
	}

	_ = s.cache.Del(ctx, otpKey)
	_ = s.cache.Del(ctx, attemptKey)
	_ = s.cache.Del(ctx, dataKey)

	return tempUserData, nil
}

func (s *RedisOtpService) IsInResendCooldown(ctx context.Context, email string, purpose string) (bool, error) {
	key := otpconstants.BuildKey(purpose, otpconstants.KeyCooldown, email)
	inCooldown, err := s.cooldown.IsInCooldown(ctx, key)
	if err != nil {
		return false, cacheUnavailable(err)
	}
	return inCooldown, nil
}

func (s *RedisOtpService) GetData(ctx context.Context, email string, purpose string) (string, error) {
	dataKey := otpconstants.BuildKey(purpose, otpconstants.KeyData, email)
	tempUserData, err := s.cache.GetRaw(ctx, dataKey)
	if err != nil {
		if errors.Is(err, shared_infra_caching.ErrKeyNotFound) {
			return "", nil
		}
		return "", cacheUnavailable(err)
	}
	return tempUserData, nil
}

// cacheUnavailable maps raw Redis errors to a client-safe 503 so internal
// network detail (e.g. dial tcp addresses) never reaches handlers.
func cacheUnavailable(err error) error {
	return apperror.NewUpstreamUnavailable(err)
}

func (s *RedisOtpService) generateSecureOTP() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "749717"
	}
	return fmt.Sprintf("%06d", nBig.Int64())
}
