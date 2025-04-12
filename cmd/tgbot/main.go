package main

import (
	"log/slog"
	"os"
	"shm/internal/broker/rabbitmq"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/notifier/telegram"
	"shm/internal/storage/sqlite"
)

func main() {
	config := config.New()
	if config.TelegramToken == "" {
		slog.Error("telegram token is not found")
		os.Exit(1)
	}

	slog.Info("connecting to database")
	db, err := sqlite.New(config.DatabaseFile)
	if err != nil {
		slog.Error("failed to create database", logger.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("connecting to message broker")
	broker, err := rabbitmq.New(config.RabbitMQ)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", logger.Error(err))
		os.Exit(1)
	}
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
