package main

import (
	"log/slog"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/scheduler"
	"shm/internal/service"
)

func main() {
	cfg := config.NewSchedulerConfig()

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	sitesRepo := db.SitesRepo()
	sitesService := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	slog.Info("starting scheduler service")
	scheduler.New(broker, sitesService, cfg).Start()
}
