package main

import (
	"log/slog"
	"os"
	"shm/internal/alert"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/lib/setup"
)

func main() {
	config := config.New()

	db := setup.ConnectToSQLite(config.DatabaseFile)
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.RabbitMQ)
	defer broker.Close()

	alert := alert.New(db, broker)
	slog.Info("starting alert service")
	if err := alert.Start(); err != nil {
		slog.Error("error from alert service", logger.Error(err))
		os.Exit(1)
	}
}
