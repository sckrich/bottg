package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	ctx    = context.Background()
)

func Init() error {
	Client = redis.NewClient(&redis.Options{
		Addr:         "redis:6379",
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
	})

	_, err := Client.Ping(ctx).Result()
	return err
}

func Close() error {
	return Client.Close()
}

func SetChatState(chatID int64, botToken string, state string, ttl time.Duration) error {
	key := chatStateKey(chatID, botToken)
	return Client.Set(ctx, key, state, ttl).Err()
}

func GetChatState(chatID int64, botToken string) (string, error) {
	key := chatStateKey(chatID, botToken)
	return Client.Get(ctx, key).Result()
}

func DeleteChatState(chatID int64, botToken string) error {
	key := chatStateKey(chatID, botToken)
	return Client.Del(ctx, key).Err()
}

func chatStateKey(chatID int64, botToken string) string {
	return fmt.Sprintf("chat:%d:%s:state", chatID, botToken)
}

func CheckRateLimit(userID int64, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("rate:%d", userID)
	count, err := Client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		Client.Expire(ctx, key, window)
	}

	return count > int64(limit), nil
}
