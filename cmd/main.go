package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/monitor"
	"shm/internal/notifier"
	"shm/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found")
	}

	config := config.New()

	storage, err := storage.NewStorage(config.DatabaseFile)
	if err != nil {
		slog.Error("failed to create storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer storage.Close()

	var notif notifier.Notifier
	if config.TelegramToken != "" {
		tgbot, err := notifier.NewTGBot(config.TelegramToken, storage)
		if err != nil {
			slog.Error("failed to create tg bot", slog.String("error", err.Error()))
			os.Exit(1)
		}
		go func() {
			tgbot.Start()
		}()
		notif = tgbot
	}

	monitor.New(storage, notif, *config).Start()
}
