package utils

import (
	"context"
	"errors"
	"github.com/zhany/ops-go/config"
	"log"
	"time"
)

var ctx = context.Background()

// SetCache 设置缓存
func SetCache(key string, value any, expiration time.Duration) error {
	if config.RedisClient == nil {
		log.Println("Redis client is not initialized")
		return errors.New("redis client is not initialized")
	}
	return config.RedisClient.Set(ctx, key, value, expiration).Err()
}

// GetCache 获取缓存
func GetCache(key string) (string, error) {
	result, err := config.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return "", errors.New("获取缓存异常")
	}
	return result, nil
}

// DelCache 删除缓存
func DelCache(key string) error {
	return config.RedisClient.Del(ctx, key).Err()
}
