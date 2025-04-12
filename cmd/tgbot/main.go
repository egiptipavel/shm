package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/lib/setup"
	"shm/internal/notifier/telegram"
)

func main() {
	config := config.New()
	if config.TelegramToken == "" {
		slog.Error("telegram token is not found")
		os.Exit(1)
	}

	db := setup.ConnectToSQLite(config.DatabaseFile)
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.RabbitMQ)
	defer broker.Close()

	tgbot, err := telegram.New(config.TelegramToken, db, broker)
	if err != nil {
		slog.Error("failed to create tg bot", logger.Error(err))
		os.Exit(1)
	}

	slog.Info("starting telegram bot")
	if err = tgbot.Start(); err != nil {
		slog.Error("error from telegram bot", logger.Error(err))
	}
}
