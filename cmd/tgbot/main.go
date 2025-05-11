package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/notifier/telegram"
	"shm/internal/service"
)

func main() {
	cfg := config.NewTelegramBotConfig()
	if cfg.Token == "" {
		slog.Error("telegram token is not found")
		os.Exit(1)
	}

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	broker := setup.ConnectToMessageBroker(cfg.MessageBroker)
	defer broker.Close()

	chatsRepo := db.ChatsRepo()
	chatsService := service.NewChatsService(chatsRepo, cfg.CommonConfig)

	sitesRepo := db.SitesRepo()
	sitesService := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	tgbot, err := telegram.New(broker, chatsService, sitesService, cfg)
	if err != nil {
		slog.Error("failed to create tg bot", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting telegram bot")
	tgbot.Start()
}
