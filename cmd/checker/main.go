package main

import (
	"log/slog"
	"os"
	"shm/internal/checker"
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

	slog.Info("creating checker service")
	checker, err := checker.New(db, broker, config.IntervalMins)
	if err != nil {
		slog.Error("failed to create checker", logger.Error(err))
		os.Exit(1)
	}

	slog.Info("starting checker service")
	checker.Start()
}
