package main

import (
	"log/slog"
	"os"
	"shm/internal/checker"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/service"
)

func main() {
	cfg := config.NewCheckerConfig()

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	resultsRepo := db.ResultsRepo()
	resultsService := service.NewResultsService(resultsRepo, cfg.CommonConfig)

	sitesRepo := db.SitesRepo()
	sitesService := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	slog.Info("creating checker service")
	checker, err := checker.New(broker, resultsService, sitesService, cfg)
	if err != nil {
		slog.Error("failed to create checker", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting checker service")
	checker.Start()
}
