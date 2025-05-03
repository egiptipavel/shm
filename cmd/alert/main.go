package main

import (
	"log/slog"
	"os"
	"shm/internal/alert"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/repository/postgres"
	"shm/internal/service"
)

func main() {
	cfg := config.NewAlertServiceConfig()

	db := setup.ConnectToPostgreSQL(config.NewPostgreSQLConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	resultsRepo := postgres.NewResultsRepo(db)
	resultsService := service.NewResultsService(resultsRepo, cfg.CommonConfig)

	alert, err := alert.New(broker, resultsService, cfg)
	if err != nil {
		slog.Error("failed to create alert service", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting alert service")
	alert.Start()
}
