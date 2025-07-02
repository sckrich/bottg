package main

import (
	"log"
	"net/http"
	"worker-bot/mtproto"
	"worker-bot/webhook"
	"shared/redis"
)

func main() {
	redis.Init()

	if err := mtproto.Init(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/webhook", webhook.Handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}