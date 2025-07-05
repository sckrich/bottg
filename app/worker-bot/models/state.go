package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	*redis.Client
}

func NewRedisClient(url string) (*RedisClient, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	return &RedisClient{
		Client: redis.NewClient(opt),
	}, nil
}

func (r *RedisClient) GetBotState(ctx context.Context, userID int64) (*BotState, error) {
	key := "bot_state:" + strconv.FormatInt(userID, 10)
	data, err := r.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get state from Redis: %w", err)
	}

	var state BotState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bot state: %w", err)
	}

	return &state, nil
}

func (r *RedisClient) SaveBotState(ctx context.Context, state *BotState) error {
	state.LastActive = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal bot state: %w", err)
	}

	key := "bot_state:" + strconv.FormatInt(state.UserID, 10)
	if err := r.Set(ctx, key, data, 7*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save state to Redis: %w", err)
	}

	return nil
}

func (r *RedisClient) BlockUser(ctx context.Context, userID int64) error {
	state, err := r.GetBotState(ctx, userID)
	if err != nil {
		return err
	}

	if state == nil {
		state = &BotState{
			UserID:     userID,
			IsBlocked:  true,
			LastActive: time.Now(),
		}
	} else {
		state.IsBlocked = true
	}

	return r.SaveBotState(ctx, state)
}
