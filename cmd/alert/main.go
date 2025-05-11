package main

import (
	"log/slog"
	"os"
	"shm/internal/alert"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/service"
)

func main() {
	cfg := config.NewAlertServiceConfig()

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	broker := setup.ConnectToMessageBroker(cfg.MessageBroker)
	defer broker.Close()

	resultsRepo := db.ResultsRepo()
	resultsService := service.NewResultsService(resultsRepo, cfg.CommonConfig)

	alert, err := alert.New(broker, resultsService, cfg)
	if err != nil {
		slog.Error("failed to create alert service", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting alert service")
	alert.Start()
}
