package main

import (
	"log"
	"worker-bot/config"
	"worker-bot/models"
	mtproto "worker-bot/mt-proto"
	"worker-bot/webhook"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load error: %v", err)
	}

	redis, err := models.NewRedisClient(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("Redis init error: %v", err)
	}
	defer func() {
		if err := redis.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		}
	}()

	mtProtoConfig := mtproto.MTProtoConfig{
		APIID:      cfg.MTProto.APIID,
		APIHash:    cfg.MTProto.APIHash,
		SessionDir: cfg.MTProto.SessionDir,
	}

	mtpClient, err := mtproto.NewSession(mtProtoConfig)
	if err != nil {
		log.Fatalf("MTProto init error: %v", err)
	}

	webhookConfig := webhook.WebhookConfig{
		Token:      cfg.Webhook.Token,
		URL:        cfg.Webhook.URL,
		ListenAddr: cfg.Webhook.ListenAddr,
	}

	log.Printf("Starting worker bot with config: %+v", cfg)
	webhook.Start(webhookConfig, redis, mtpClient)
}
