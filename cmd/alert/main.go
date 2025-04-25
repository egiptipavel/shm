package main

import (
	"log/slog"
	"os"
	"shm/internal/alert"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
)

func main() {
	cfg := config.NewAlertServiceConfig()

	db := setup.ConnectToSQLite(config.NewSQLiteConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	alert, err := alert.New(db, broker, cfg)
	if err != nil {
		slog.Error("failed to create alert service", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting alert service")
	alert.Start()
}
