package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/notifier/telegram"
	"shm/internal/repository/postgres"
	"shm/internal/service"
)

func main() {
	cfg := config.NewTelegramBotConfig()
	if cfg.Token == "" {
		slog.Error("telegram token is not found")
		os.Exit(1)
	}

	db := setup.ConnectToPostgreSQL(config.NewPostgreSQLConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	chatsRepo := postgres.NewChatsRepo(db)
	chatsService := service.NewChatsService(chatsRepo, cfg.CommonConfig)

	sitesRepo := postgres.NewSitesRepo(db)
	sitesService := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	tgbot, err := telegram.New(broker, chatsService, sitesService, cfg)
	if err != nil {
		slog.Error("failed to create tg bot", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting telegram bot")
	tgbot.Start()
}
