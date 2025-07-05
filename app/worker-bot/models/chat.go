package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	KeyBotState    = "bot:%d:state"
	KeyBlockedBots = "blocked:bots:%d"
	KeyUserSession = "user:%d:session:%s"
)

var (
	ErrBotBlocked     = errors.New("бот заблокирован")
	ErrUserBlocked    = errors.New("пользователь заблокирован")
	ErrSessionExpired = errors.New("сессия истекла")
)

type RedisStorage struct {
	cli *redis.Client
}

func NewRedisStorage(addr, password string, db int) *RedisStorage {
	return &RedisStorage{
		cli: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (r *RedisStorage) Close() error {
	return r.cli.Close()
}

type BotState struct {
	UserID      int64     `json:"user_id"`
	CurrentStep string    `json:"current_step"`
	RefCode     string    `json:"ref_code"`
	LastActive  time.Time `json:"last_active"`
	IsBlocked   bool      `json:"is_blocked"`
}

func (r *RedisStorage) SaveState(ctx context.Context, botID int64, state *BotState) error {
	key := fmt.Sprintf(KeyBotState, botID)
	state.LastActive = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state error: %w", err)
	}
	return r.cli.Set(ctx, key, data, 7*24*time.Hour).Err()
}

func (r *RedisStorage) GetState(ctx context.Context, botID int64) (*BotState, error) {
	key := fmt.Sprintf(KeyBotState, botID)
	data, err := r.cli.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var state BotState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state error: %w", err)
	}

	if state.IsBlocked {
		return nil, ErrBotBlocked
	}

	return &state, nil
}

func (r *RedisStorage) BlockBot(ctx context.Context, botID int64, reason string) error {
	state, err := r.GetState(ctx, botID)
	if err != nil {
		return err
	}

	if state == nil {
		state = &BotState{}
	}

	state.IsBlocked = true
	if err := r.SaveState(ctx, botID, state); err != nil {
		return err
	}

	blockKey := fmt.Sprintf(KeyBlockedBots, botID)
	return r.cli.HSet(ctx, blockKey, map[string]interface{}{
		"blocked_at": time.Now().Format(time.RFC3339),
		"reason":     reason,
	}).Err()
}

func GenerateRefCode() string {
	return "ref_" + uuid.New().String()[:8]
}

type UserSession struct {
	ID        string    `json:"id"`
	MTProtoID string    `json:"mtproto_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (r *RedisStorage) SaveSession(ctx context.Context, userID int64, session *UserSession) error {
	key := fmt.Sprintf(KeyUserSession, userID, session.ID)
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session error: %w", err)
	}
	ttl := time.Until(session.ExpiresAt)
	return r.cli.Set(ctx, key, data, ttl).Err()
}

func (r *RedisStorage) GetSession(ctx context.Context, userID int64, sessionID string) (*UserSession, error) {
	key := fmt.Sprintf(KeyUserSession, userID, sessionID)
	data, err := r.cli.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionExpired
		}
		return nil, err
	}

	var session UserSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session error: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}
