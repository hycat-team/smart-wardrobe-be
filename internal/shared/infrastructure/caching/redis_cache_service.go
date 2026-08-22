package caching

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCacheService struct {
	client *redis.Client
}

func NewRedisCacheService(client *redis.Client) ICacheService {
	return &RedisCacheService{client: client}
}

func (s *RedisCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	cached, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrKeyNotFound
		}
		return err
	}
	return json.Unmarshal([]byte(cached), dest)
}

func (s *RedisCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *RedisCacheService) GetRaw(ctx context.Context, key string) (string, error) {
	result, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrKeyNotFound
		}
		return "", err
	}
	return result, nil
}

func (s *RedisCacheService) SetRaw(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *RedisCacheService) Del(ctx context.Context, keys ...string) error {
	return s.client.Del(ctx, keys...).Err()
}

func (s *RedisCacheService) Exists(ctx context.Context, keys ...string) (bool, error) {
	result, err := s.client.Exists(ctx, keys...).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *RedisCacheService) MGet(ctx context.Context, dest interface{}, keys ...string) error {
	results, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}
	strResults := make([]string, 0, len(results))
	for _, r := range results {
		if r != nil {
			if str, ok := r.(string); ok {
				strResults = append(strResults, str)
			}
		}
	}
	data, err := json.Marshal(strResults)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
