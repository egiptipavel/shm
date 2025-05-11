package main

import (
	"log/slog"
	"shm/internal/checker"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/service"
)

func main() {
	cfg := config.NewCheckerConfig()

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	broker := setup.ConnectToMessageBroker(cfg.MessageBroker)
	defer broker.Close()

	resultsRepo := db.ResultsRepo()
	resultsService := service.NewResultsService(resultsRepo, cfg.CommonConfig)

	sitesRepo := db.SitesRepo()
	sitesService := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	checker := checker.New(broker, resultsService, sitesService, cfg)
	slog.Info("starting checker service")
	checker.Start()
}
