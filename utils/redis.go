package utils

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/zhany/ops-go/config"
	"time"
)

var ctx = context.Background()

// SetCache 设置缓存
func SetCache(key string, value any, expiration time.Duration) error {
	return config.RedisClient.Set(ctx, key, value, expiration).Err()
}

// GetCache 获取缓存
func GetCache(key string) (string, error) {
	result, err := config.RedisClient.Get(ctx, key).Result()
	if err != redis.Nil {
		return "", nil
	}
	return result, err
}

// DelCache 删除缓存
func DelCache(key string) error {
	return config.RedisClient.Del(ctx, key).Err()
}
