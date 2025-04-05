package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
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
		slog.Error("failed to create database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	tgbot, err := telegram.New(config.TelegramToken, db)
	if err != nil {
		slog.Error("failed to create tg bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("starting telegram bot")
	tgbot.Start()
}
