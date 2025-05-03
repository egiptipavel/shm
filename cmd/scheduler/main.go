package main

import (
	"log/slog"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/repository/postgres"
	"shm/internal/scheduler"
	"shm/internal/service"
)

func main() {
	cfg := config.NewSchedulerConfig()

	db := setup.ConnectToPostgreSQL(config.NewPostgreSQLConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	sitesRepo := postgres.NewSitesRepo(db)
	sitesService := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	slog.Info("starting scheduler service")
	scheduler.New(broker, sitesService, cfg).Start()
}
