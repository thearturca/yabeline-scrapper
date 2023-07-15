package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"yabeline-tg/telegram"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	TG_TOKEN, exists := os.LookupEnv("TG_TOKEN")

	if !exists {
		panic("TG_TOKEN not found")
	}

	log.Printf("Bot started")
	telegram.StartBot(ctx, TG_TOKEN)

}
