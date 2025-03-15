package main

import (
	"flag"
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/monitor"
	"shm/internal/notifier"
	"shm/internal/storage"
)

var configPath = flag.String("config", "configs/config.json", "path to config file")

func main() {
	flag.Parse()

	config, err := config.ParseConfig(*configPath)
	if err != nil {
		slog.Error("failed to parse config file", slog.String("error", err.Error()))
		os.Exit(1)
	}

	storage, err := storage.NewStorage(config.DBFile)
	if err != nil {
		slog.Error("failed to create storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer storage.Close()

	var notif notifier.Notifier
	if config.Token != "" {
		tgbot, err := notifier.NewTGBot(config.Token, storage)
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
