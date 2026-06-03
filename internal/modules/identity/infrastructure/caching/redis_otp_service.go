package caching

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/application/constants/otpconstants"

	"github.com/redis/go-redis/v9"
)

type RedisOtpService struct {
	redisClient *redis.Client
	cfg         *config.Config
}

func NewRedisOtpService(client *redis.Client, cfg *config.Config) identity.IOtpService {
	return &RedisOtpService{
		redisClient: client,
		cfg:         cfg,
	}
}

func (s *RedisOtpService) GenerateOtp(ctx context.Context, email string, tempUserDataJson string, purpose string) (string, error) {
	otpCode := s.generateSecureOTP()

	otpKey := otpconstants.BuildKey(purpose, otpconstants.KeyValue, email)
	attemptKey := otpconstants.BuildKey(purpose, otpconstants.KeyAttempts, email)
	cooldownKey := otpconstants.BuildKey(purpose, otpconstants.KeyCooldown, email)
	dataKey := otpconstants.BuildKey(purpose, otpconstants.KeyData, email)

	expiry := time.Duration(s.cfg.Otp.ExpiryMinutes) * time.Minute
	cooldown := time.Duration(s.cfg.Otp.ResendIntervalSeconds) * time.Second

	err := s.redisClient.Set(ctx, otpKey, otpCode, expiry).Err()
	if err != nil {
		return "", err
	}

	err = s.redisClient.Set(ctx, attemptKey, "0", expiry).Err()
	if err != nil {
		return "", err
	}

	err = s.redisClient.Set(ctx, dataKey, tempUserDataJson, expiry).Err()
	if err != nil {
		return "", err
	}

	err = s.redisClient.Set(ctx, cooldownKey, "locked", cooldown).Err()
	if err != nil {
		return "", err
	}

	return otpCode, nil
}

func (s *RedisOtpService) VerifyOtp(ctx context.Context, email string, otpCode string, purpose string) (string, error) {
	otpKey := otpconstants.BuildKey(purpose, otpconstants.KeyValue, email)
	attemptKey := otpconstants.BuildKey(purpose, otpconstants.KeyAttempts, email)
	dataKey := otpconstants.BuildKey(purpose, otpconstants.KeyData, email)

	savedOtp, err := s.redisClient.Get(ctx, otpKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", errorcode.NewBadRequest("Mã OTP đã hết hạn hoặc không tồn tại. Vui lòng yêu cầu mã mới.")
		}
		return "", err
	}

	tempUserData, err := s.redisClient.Get(ctx, dataKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", errorcode.NewBadRequest("Mã OTP đã hết hạn hoặc không tồn tại. Vui lòng yêu cầu mã mới.")
		}
		return "", err
	}

	attemptsStr, err := s.redisClient.Get(ctx, attemptKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", err
	}
	if errors.Is(err, redis.Nil) {
		attemptsStr = "0"
	}

	currentAttempts, err := strconv.Atoi(attemptsStr)
	if err != nil {
		currentAttempts = 0
	}

	if currentAttempts >= s.cfg.Otp.MaxAttempts {
		_ = s.redisClient.Del(ctx, otpKey).Err()
		_ = s.redisClient.Del(ctx, dataKey).Err()
		return "", errorcode.NewTooManyRequest("Bạn đã nhập sai quá nhiều lần. Mã OTP này đã bị hủy để bảo mật. Vui lòng thử lại sau 1 phút.")
	}

	if savedOtp != otpCode {
		currentAttempts++
		expiry := time.Duration(s.cfg.Otp.ExpiryMinutes) * time.Minute
		_ = s.redisClient.Set(ctx, attemptKey, strconv.Itoa(currentAttempts), expiry).Err()

		remainingAttempts := s.cfg.Otp.MaxAttempts - currentAttempts
		return "", errorcode.NewBadRequest(fmt.Sprintf("Mã OTP không chính xác. Bạn còn %d lần thử.", remainingAttempts))
	}

	_ = s.redisClient.Del(ctx, otpKey).Err()
	_ = s.redisClient.Del(ctx, attemptKey).Err()
	_ = s.redisClient.Del(ctx, dataKey).Err()

	return tempUserData, nil
}

func (s *RedisOtpService) IsInResendCooldown(ctx context.Context, email string, purpose string) (bool, error) {
	key := otpconstants.BuildKey(purpose, otpconstants.KeyCooldown, email)
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisOtpService) generateSecureOTP() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "123456"
	}
	return fmt.Sprintf("%06d", nBig.Int64())
}
