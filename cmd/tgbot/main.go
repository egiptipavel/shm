package main

import (
	"log/slog"
	"os"
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

	db, err := sqlite.New(config.DatabaseFile)
	if err != nil {
		slog.Error("failed to create database", logger.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	tgbot, err := telegram.New(config.TelegramToken, db)
	if err != nil {
		slog.Error("failed to create tg bot", logger.Error(err))
		os.Exit(1)
	}

	slog.Info("starting telegram bot")
	tgbot.Start()
}
