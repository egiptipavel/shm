package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/notifier/telegram"
)

func main() {
	cfg := config.NewTelegramBotConfig()
	if cfg.Token == "" {
		slog.Error("telegram token is not found")
		os.Exit(1)
	}

	db := setup.ConnectToSQLite(config.NewSQLiteConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	tgbot, err := telegram.New(db, broker, cfg)
	if err != nil {
		slog.Error("failed to create tg bot", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting telegram bot")
	tgbot.Start()
}
