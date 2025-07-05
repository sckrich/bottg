package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Redis      RedisConfig
	Webhook    WebhookConfig
	MTProto    MTProtoConfig
	Panel      PanelConfig
	WorkerBots WorkerBotsConfig
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type WebhookConfig struct {
	URL         string
	Token       string
	ListenAddr  string
	Certificate string
	SecretToken string
}

type PanelConfig struct {
	URL       string
	AuthToken string
}

type WorkerBotsConfig struct {
	DefaultRefCode string
	BlockedPrefix  string
}

type MTProtoConfig struct {
	APIID      int    `json:"api_id"`
	APIHash    string `json:"api_hash"`
	SessionDir string `json:"session_dir"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Webhook: WebhookConfig{
			URL:         strings.TrimSuffix(getEnv("WEBHOOK_URL", ""), "/"),
			Token:       getEnv("BOT_TOKEN", ""),
			ListenAddr:  getEnv("LISTEN_ADDR", ":8080"),
			Certificate: getEnv("WEBHOOK_CERT", ""),
			SecretToken: getEnv("WEBHOOK_SECRET", ""),
		},
		MTProto: MTProtoConfig{
			APIID:   getEnvAsInt("API_ID", 0),
			APIHash: getEnv("API_HASH", ""),
		},
		Panel: PanelConfig{
			URL:       getEnv("PANEL_URL", ""),
			AuthToken: getEnv("PANEL_AUTH_TOKEN", ""),
		},
		WorkerBots: WorkerBotsConfig{
			DefaultRefCode: getEnv("DEFAULT_REF_CODE", generateDefaultRefCode()),
			BlockedPrefix:  getEnv("BLOCKED_PREFIX", "blocked:"),
		},
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateConfig(cfg *Config) error {
	required := map[string]string{
		"BOT_TOKEN":   cfg.Webhook.Token,
		"WEBHOOK_URL": cfg.Webhook.URL,
		"API_ID":      strconv.Itoa(cfg.MTProto.APIID),
		"API_HASH":    cfg.MTProto.APIHash,
		"PANEL_URL":   cfg.Panel.URL,
	}

	for field, value := range required {
		if value == "" {
			return newConfigError(field, "не может быть пустым")
		}
	}

	if cfg.Webhook.SecretToken == "" {
		return newConfigError("WEBHOOK_SECRET", "секретный токен обязателен для безопасности")
	}

	return nil
}

type ConfigError struct {
	Field  string
	Reason string
}

func (e ConfigError) Error() string {
	return "конфигурационная ошибка: поле " + e.Field + " - " + e.Reason
}

func newConfigError(field, reason string) ConfigError {
	return ConfigError{
		Field:  field,
		Reason: reason,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func generateDefaultRefCode() string {
	return "ref_" + strconv.FormatInt(int64(os.Getpid()), 36)
}
